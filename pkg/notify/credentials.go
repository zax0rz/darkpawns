package notify

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
)

// Credentials is accepted only on write-only endpoints. Never embed it in View.
type Credentials struct {
	Token string `json:"token,omitempty"`
	User  string `json:"user,omitempty"`
	URL   string `json:"url,omitempty"`
}

type credentialSnapshot struct {
	values  map[string]Credentials
	sources map[string]string
}
type credentialVault struct {
	keyID [32]byte

	keyPath   string
	aead      cipher.AEAD       // accessed under Service.saveMu after initialization
	encrypted map[string]string // empty ciphertext is an explicit removal, overriding env
	snapshot  atomic.Pointer[credentialSnapshot]
}
type diskConfig struct {
	Config
	Credentials map[string]string `json:"credentials,omitempty"`
}

var (
	ErrCredentials       = errors.New("invalid credentials: use both 30-character Pushover keys, a canonical Discord webhook URL, or a hosted Healthchecks ping URL")
	ErrCredentialStorage = errors.New("notification encryption key is missing, invalid, or unreadable; restore the original private key file")
)

func validCredentials(id string, c Credentials) bool {
	switch id {
	case Pushover:
		return c.URL == "" && keyPattern.MatchString(c.Token) && keyPattern.MatchString(c.User)
	case Discord, Gossip:
		_, ok := providerURL(c.URL, "discord.com", webhookPattern)
		return ok && c.Token == "" && c.User == ""
	case Heartbeat:
		_, ok := providerURL(c.URL, "hc-ping.com", heartbeatPattern)
		return ok && c.Token == "" && c.User == ""
	}
	return false
}

func envCredentials(id string) Credentials {
	switch id {
	case Pushover:
		return Credentials{Token: os.Getenv("DP_NOTIFY_PUSHOVER_TOKEN"), User: os.Getenv("DP_NOTIFY_PUSHOVER_USER")}
	case Discord:
		return Credentials{URL: os.Getenv("DP_NOTIFY_DISCORD_WEBHOOK")}
	case Gossip:
		return Credentials{URL: os.Getenv("DP_NOTIFY_GOSSIP_WEBHOOK")}
	case Heartbeat:
		return Credentials{URL: os.Getenv("DP_NOTIFY_HEARTBEAT_URL")}
	}
	return Credentials{}
}

func newCredentialVault(path string, encrypted map[string]string) (*credentialVault, error) {
	keyPath := os.Getenv("DP_NOTIFICATION_KEY_PATH")
	if keyPath == "" {
		keyPath = path + ".key"
	}
	v := &credentialVault{keyPath: filepath.Clean(keyPath), encrypted: map[string]string{}}
	state := &credentialSnapshot{values: map[string]Credentials{}, sources: map[string]string{}}
	for _, id := range []string{Pushover, Discord, Heartbeat, Gossip} {
		state.values[id] = envCredentials(id)
		state.sources[id] = "environment"
	}
	for id, encoded := range encrypted {
		if id != Pushover && id != Discord && id != Heartbeat && id != Gossip {
			return nil, ErrCredentials
		}
		v.encrypted[id] = encoded
		if encoded == "" {
			state.values[id] = Credentials{}
			state.sources[id] = "none"
			continue
		}
		if v.aead == nil {
			if err := v.loadKey(false); err != nil {
				return nil, err
			}
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(raw) < v.aead.NonceSize() {
			return nil, ErrCredentialStorage
		}
		plain, err := v.aead.Open(nil, raw[:v.aead.NonceSize()], raw[v.aead.NonceSize():], credentialAAD(id))
		if err != nil {
			return nil, ErrCredentialStorage
		}
		var c Credentials
		if json.Unmarshal(plain, &c) != nil || !validCredentials(id, c) {
			return nil, ErrCredentialStorage
		}
		state.values[id] = c
		state.sources[id] = "admin"
	}
	v.snapshot.Store(state)
	return v, nil
}
func credentialAAD(id string) []byte { return []byte("darkpawns-notifications/v1/" + id) }

// loadKey never replaces an existing key. Fresh installations generate a private
// key on first credential save; a missing key with ciphertext always fails closed.
func (v *credentialVault) loadKey(create bool) error {
	key, err := os.ReadFile(v.keyPath) // #nosec G703 -- server-configured key path, never request-derived
	if errors.Is(err, os.ErrNotExist) && create {
		if os.MkdirAll(filepath.Dir(v.keyPath), 0o750) != nil {
			return ErrCredentialStorage
		}
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return ErrCredentialStorage
		}
		f, openErr := os.OpenFile(v.keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G703 -- server-configured key path
		if errors.Is(openErr, os.ErrExist) {
			return v.loadKey(false)
		}
		if openErr != nil {
			return ErrCredentialStorage
		}
		_, err = f.Write(key)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return ErrCredentialStorage
		}
	} else if err != nil {
		return ErrCredentialStorage
	}
	info, err := os.Stat(v.keyPath) // #nosec G703 -- server-configured key path
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || len(key) != 32 {
		return ErrCredentialStorage
	}
	digest := sha256.Sum256(key)
	if v.aead != nil && v.keyID != digest {
		return ErrCredentialStorage
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ErrCredentialStorage
	}
	v.aead, err = cipher.NewGCM(block)
	v.keyID = digest
	if err != nil {
		return ErrCredentialStorage
	}
	return nil
}

func (v *credentialVault) seal(id string, c Credentials) (string, error) {
	if err := v.loadKey(v.aead == nil); err != nil {
		return "", err
	}
	plain, err := json.Marshal(c)
	if err != nil {
		return "", ErrCredentialStorage
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrCredentialStorage
	}
	return base64.StdEncoding.EncodeToString(v.aead.Seal(nonce, nonce, plain, credentialAAD(id))), nil
}

// SaveCredentials replaces a complete provider credential set. Removing it also
// disables that destination, preserves event choices, and suppresses env fallback.
// Disk/encryption operations never hold the game publisher's mutex.
func (s *Service) SaveCredentials(id string, revision int64, c *Credentials) error {
	if id != Pushover && id != Discord && id != Heartbeat && id != Gossip {
		return ErrCredentials
	}
	if c != nil && !validCredentials(id, *c) {
		return ErrCredentials
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	cfg := clone(s.cfg)
	s.mu.Unlock()
	if revision != cfg.Revision {
		return ErrConflict
	}
	next := &credentialSnapshot{values: map[string]Credentials{}, sources: map[string]string{}}
	for key, value := range s.vault.snapshot.Load().values {
		next.values[key] = value
	}
	for key, value := range s.vault.snapshot.Load().sources {
		next.sources[key] = value
	}
	encrypted := map[string]string{}
	for key, value := range s.vault.encrypted {
		encrypted[key] = value
	}
	if c == nil {
		encrypted[id] = ""
		next.values[id] = Credentials{}
		next.sources[id] = "none"
		switch id {
		case Pushover:
			cfg.Pushover.Enabled = false
		case Discord:
			cfg.Discord.Enabled = false
		case Gossip:
			cfg.Gossip = false
		case Heartbeat:
			cfg.Heartbeat = false
		}
	} else {
		encoded, err := s.vault.seal(id, *c)
		if err != nil {
			return err
		}
		encrypted[id] = encoded
		next.values[id] = *c
		next.sources[id] = "admin"
	}
	cfg.Revision++
	if err := s.persist(cfg, encrypted); err != nil {
		return err
	}
	s.vault.encrypted = encrypted
	s.mu.Lock()
	s.vault.snapshot.Store(next)
	s.send = credentialSender(next)
	s.cfg = cfg
	s.status[id] = Status{}
	s.mu.Unlock()
	return nil
}

// Each worker captures this immutable destination snapshot under the same lock as
// its revision check. Replacing credentials cannot route an older job to a new
// recipient between that check and the network request.
func credentialSender(state *credentialSnapshot) func(context.Context, string, string) error {
	return func(ctx context.Context, id, message string) error {
		return deliverWithCredentials(ctx, id, message, state.values[id])
	}
}
