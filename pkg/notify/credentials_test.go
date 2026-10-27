package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emptyEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"DP_NOTIFY_PUSHOVER_TOKEN", "DP_NOTIFY_PUSHOVER_USER", "DP_NOTIFY_DISCORD_WEBHOOK", "DP_NOTIFY_GOSSIP_WEBHOOK", "DP_NOTIFY_HEARTBEAT_URL", "DP_NOTIFICATION_KEY_PATH"} {
		t.Setenv(key, "")
	}
}

func TestEncryptedCredentialLifecycle(t *testing.T) {
	emptyEnv(t)
	path := filepath.Join(t.TempDir(), "notifications.json")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".key"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("empty instance generated a key")
	}
	c := Credentials{URL: "https://discord.com/api/webhooks/123/secret-first"}
	if err := s.SaveCredentials(Discord, 1, &c); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), c.URL) || strings.Contains(string(raw), "secret-first") {
		t.Fatal("plaintext written to settings")
	}
	for _, p := range []string{path, path + ".key"} {
		info, err := os.Stat(p)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatal("private file mode", err)
		}
	}
	view := s.View()
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), c.URL) || view.Status[Discord].Source != "admin" || !view.Status[Discord].Ready {
		t.Fatal("unsafe or incomplete credential view")
	}
	restarted, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.vault.snapshot.Load().values[Discord] != c {
		t.Fatal("encrypted credential did not survive restart")
	}
	first := s.vault.encrypted[Discord]
	if err := s.SaveCredentials(Discord, s.View().Config.Revision, &c); err != nil {
		t.Fatal(err)
	}
	if first == s.vault.encrypted[Discord] {
		t.Fatal("nonce reused")
	}
	if err := s.SaveCredentials(Discord, 1, &c); !errors.Is(err, ErrConflict) {
		t.Fatal("stale replacement accepted", err)
	}
	cfg := s.View().Config
	cfg.Discord = Destination{true, []string{Level}}
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DP_NOTIFY_DISCORD_WEBHOOK", "https://discord.com/api/webhooks/123/old-env-value")
	if err := s.SaveCredentials(Discord, s.View().Config.Revision, nil); err != nil {
		t.Fatal(err)
	}
	view = s.View()
	if view.Status[Discord].Ready || view.Config.Discord.Enabled || len(view.Config.Discord.Events) != 1 {
		t.Fatal("removal did not disable/preserve choices")
	}
	restarted, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.View().Status[Discord].Ready {
		t.Fatal("removal resurrected environment credential")
	}
}

func TestCredentialTamperingAndKeyLossFailClosed(t *testing.T) {
	for _, kind := range []string{"missing-key", "wrong-key", "permissive-key", "corrupt-cipher", "provider-swap"} {
		t.Run(kind, func(t *testing.T) {
			emptyEnv(t)
			path := filepath.Join(t.TempDir(), "notifications.json")
			s, err := New(path)
			if err != nil {
				t.Fatal(err)
			}
			c := Credentials{URL: "https://discord.com/api/webhooks/123/private"}
			if err := s.SaveCredentials(Discord, 1, &c); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing-key":
				if err := os.Remove(path + ".key"); err != nil {
					t.Fatal(err)
				}
			case "wrong-key":
				if err := os.WriteFile(path+".key", []byte(strings.Repeat("x", 32)), 0o600); err != nil {
					t.Fatal(err)
				}
			case "permissive-key":
				if err := os.Chmod(path+".key", 0o644); err != nil {
					t.Fatal(err)
				}
			default:
				var stored diskConfig
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, &stored); err != nil {
					t.Fatal(err)
				}
				if kind == "corrupt-cipher" {
					stored.Credentials[Discord] = "invalid-ciphertext"
				} else {
					stored.Credentials["heartbeat"] = stored.Credentials[Discord]
					delete(stored.Credentials, Discord)
				}
				raw, err = json.Marshal(stored)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := New(path); err == nil {
				t.Fatal("compromised storage accepted")
			}
		})
	}
}

func TestCredentialSaveFailureAndImmediateUse(t *testing.T) {
	emptyEnv(t)
	path := filepath.Join(t.TempDir(), "notifications.json")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	original := providerClient.Transport
	t.Cleanup(func() { providerClient.Transport = original })
	got := ""
	providerClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		got = form.Get("token")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":1}`)), Header: http.Header{}}, nil
	})
	var previousSend func(context.Context, string, string) error
	for _, token := range []string{strings.Repeat("a", 30), strings.Repeat("c", 30)} {
		c := Credentials{Token: token, User: strings.Repeat("b", 30)}
		if err := s.SaveCredentials(Pushover, s.View().Config.Revision, &c); err != nil {
			t.Fatal(err)
		}
		if err := s.send(context.Background(), Pushover, "test"); err != nil {
			t.Fatal(err)
		}
		if got != token {
			t.Fatal("replacement not used immediately")
		}
		if previousSend != nil {
			if err := previousSend(context.Background(), Pushover, "in flight"); err != nil {
				t.Fatal(err)
			}
			if got != strings.Repeat("a", 30) {
				t.Fatal("older job rerouted to replacement destination")
			}
		}
		previousSend = s.send
	}
	before := s.View().Config.Revision
	old := s.vault.snapshot.Load().values[Pushover]
	s.path = filepath.Join(s.path, "impossible.json")
	c := Credentials{Token: strings.Repeat("d", 30), User: strings.Repeat("b", 30)}
	if s.SaveCredentials(Pushover, before, &c) == nil || s.View().Config.Revision != before || s.vault.snapshot.Load().values[Pushover] != old {
		t.Fatal("failed persistence changed credentials")
	}
	if s.SaveCredentials(Pushover, before, &Credentials{Token: "invalid"}) == nil {
		t.Fatal("invalid credentials accepted")
	}
}

func TestLiveKeyChangeCannotReencryptWithDifferentKey(t *testing.T) {
	emptyEnv(t)
	path := filepath.Join(t.TempDir(), "notifications.json")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	c := Credentials{URL: "https://discord.com/api/webhooks/123/secret"}
	if err := s.SaveCredentials(Discord, 1, &c); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".key", []byte(strings.Repeat("z", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	before := s.View().Config.Revision
	if err := s.SaveCredentials(Discord, before, &c); !errors.Is(err, ErrCredentialStorage) {
		t.Fatal("live key replacement accepted", err)
	}
	if s.View().Config.Revision != before {
		t.Fatal("failed key verification mutated settings")
	}
}
