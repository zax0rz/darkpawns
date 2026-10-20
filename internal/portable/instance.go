// Package portable owns release/instance separation. It never changes the game
// defaults: only the new launcher opts into this isolated, loopback-only layout.
package portable

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const Format = 1

// Manifest describes the immutable payload, not the operator's editable world.
// Checksums detect damaged bundles; they are not a substitute for release signing.
type Manifest struct {
	Format       int               `json:"format"`
	UpdateFormat int               `json:"update_format,omitempty"`
	DataFormat   int               `json:"data_format,omitempty"`
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	Files        map[string]string `json:"files"`
}

type Config struct {
	Format          int               `json:"format"`
	HTTPPort        int               `json:"http_port"`
	JWTSecret       string            `json:"jwt_secret"`
	TelnetBind      string            `json:"telnet_bind,omitempty"`
	TelnetTLSBind   string            `json:"telnet_tls_bind,omitempty"`
	WhodBind        string            `json:"whod_bind,omitempty"`
	TelnetPort      int               `json:"telnet_port,omitempty"`
	TelnetTLSPort   int               `json:"telnet_tls_port,omitempty"`
	WhodPort        int               `json:"whod_port,omitempty"`
	ServerEnv       map[string]string `json:"server_env,omitempty"`
	MudletDirectory string            `json:"mudlet_directory,omitempty"`
}

func ReadManifest(bundle string) (Manifest, error) {
	var m Manifest
	if err := readJSON(filepath.Join(bundle, "release-assets", "manifest.json"), &m); err != nil {
		return m, fmt.Errorf("read release manifest: %w", err)
	}
	if m.Format != Format || m.Version == "" || len(m.Files) == 0 {
		return m, errors.New("unsupported or incomplete release manifest")
	}
	for _, name := range []string{"lib/world/wld/index", "web/index.html", "admin-ui/index.html"} {
		if _, ok := m.Files[name]; !ok {
			return m, fmt.Errorf("release manifest missing %s", name)
		}
	}
	return m, nil
}

func Verify(bundle string, m Manifest) (result error) {
	root, err := os.OpenRoot(filepath.Join(bundle, "release-assets"))
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, root.Close()) }()
	for name, want := range m.Files {
		if !fs.ValidPath(name) || strings.Contains(name, "\\") {
			return fmt.Errorf("invalid release path %q", name)
		}
		f, err := root.Open(filepath.FromSlash(name))
		if err != nil {
			return fmt.Errorf("release file %s: %w", name, err)
		}
		sum := sha256.New()
		_, copyErr := io.Copy(sum, f)
		closeErr := f.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return err
		}
		if hex.EncodeToString(sum.Sum(nil)) != want {
			return fmt.Errorf("release checksum mismatch: %s", name)
		}
	}
	return nil
}

// Init refuses existing destinations, including empty directories. It copies
// only the manifest's starter lib files; saved players/rent are never seeded.
func Init(bundle, instance string, port int) error {
	return InitWithTelnet(bundle, instance, port, 7777)
}

// InitWithTelnet creates a new local instance with an explicit loopback telnet
// port (zero disables it). Existing configurations are never rewritten.
func InitWithTelnet(bundle, instance string, port, telnetPort int) error {
	if port < 1 || port > 65535 {
		return errors.New("HTTP port must be between 1 and 65535")
	}
	if err := (Config{HTTPPort: port, TelnetPort: telnetPort, TelnetBind: "127.0.0.1"}).validateHosting(); err != nil {
		return err
	}
	bundle, err := filepath.Abs(bundle)
	if err != nil {
		return err
	}
	instance, err = filepath.Abs(instance)
	if err != nil {
		return err
	}
	// Resolve the parent so a symlink cannot put mutable state inside the release.
	parent, err := filepath.EvalSymlinks(filepath.Dir(instance))
	if err != nil {
		return fmt.Errorf("instance parent must exist: %w", err)
	}
	instance = filepath.Join(parent, filepath.Base(instance))
	realBundle, err := filepath.EvalSymlinks(bundle)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(realBundle, instance)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("instance must be outside the replaceable release directory")
	}
	m, err := ReadManifest(bundle)
	if err != nil {
		return err
	}
	if err := Verify(bundle, m); err != nil {
		return err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	if err := os.Mkdir(instance, 0o700); err != nil {
		return fmt.Errorf("create new instance (existing directories are never overwritten): %w", err)
	}
	// A failed initialization leaves an identifiable partial directory. Never
	// recursively delete an operator-selected path as error recovery.
	for name := range m.Files {
		if !strings.HasPrefix(name, "lib/") {
			continue
		}
		dst := filepath.Join(instance, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		if err := copyNew(filepath.Join(bundle, "release-assets", filepath.FromSlash(name)), dst); err != nil {
			return err
		}
	}
	for _, name := range []string{"lib/data", "lib/logs", "lib/world/etc", "lib/etc", "lib/plrobjs"} {
		if err := os.MkdirAll(filepath.Join(instance, name), 0o700); err != nil {
			return err
		}
	}
	cfg := Config{Format: Format, HTTPPort: port, JWTSecret: hex.EncodeToString(secret), TelnetPort: telnetPort, TelnetBind: "127.0.0.1"}
	f, err := os.OpenFile(filepath.Join(instance, "instance.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	encodeErr := json.NewEncoder(f).Encode(cfg)
	syncErr := f.Sync()
	return errors.Join(encodeErr, syncErr, f.Close())
}

func Load(instance string) (Config, error) {
	var c Config
	if err := readJSON(filepath.Join(instance, "instance.json"), &c); err != nil {
		return c, fmt.Errorf("read instance configuration: %w", err)
	}
	secret, err := hex.DecodeString(c.JWTSecret)
	if c.Format != Format || c.HTTPPort < 1 || c.HTTPPort > 65535 || err != nil || len(secret) != 32 {
		return c, errors.New("invalid instance configuration; restore the original instance.json")
	}
	if err := c.validateHosting(); err != nil {
		return c, err
	}
	return c, nil
}

func readJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("unexpected trailing JSON data")
	}
	return nil
}

func copyNew(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	return errors.Join(copyErr, out.Close())
}
