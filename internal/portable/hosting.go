package portable

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"unicode"
)

// Hosting is an explicit instance choice. Fresh local setup binds telnet to
// loopback and never inherits another installation's TLS or Mudlet settings.
func (c Config) validateHosting() error {
	ports := map[int]string{c.HTTPPort: "http_port"}
	for name, port := range map[string]int{"telnet_port": c.TelnetPort, "telnet_tls_port": c.TelnetTLSPort, "whod_port": c.WhodPort} {
		if port < 0 || port > 65535 {
			return fmt.Errorf("%s must be 0 (disabled) or 1-65535", name)
		}
		if port != 0 {
			if previous, exists := ports[port]; exists {
				return fmt.Errorf("%s collides with %s", name, previous)
			}
			ports[port] = name
		}
	}
	for name, bind := range map[string]string{"telnet_bind": c.TelnetBind, "telnet_tls_bind": c.TelnetTLSBind, "whod_bind": c.WhodBind} {
		if bind != "" && net.ParseIP(bind) == nil {
			return fmt.Errorf("%s must be an IP address, or empty for all interfaces", name)
		}
	}
	for key, value := range c.ServerEnv {
		if value == "" || containsControl(value) {
			return fmt.Errorf("server_env %s must be nonempty and contain no control characters", key)
		}
		switch key {
		case "DP_MUDLET_PACKAGE_URL", "DP_MUDLET_MAP_URL":
			u, err := url.Parse(value)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				return fmt.Errorf("server_env %s requires an HTTPS URL without credentials", key)
			}
		case "TELNET_TLS_CERT_FILE", "TELNET_TLS_KEY_FILE":
			if !filepath.IsAbs(value) {
				return fmt.Errorf("server_env %s must be an absolute path", key)
			}
		default:
			return fmt.Errorf("server_env %s is not an allowed hosting setting", key)
		}
	}
	if c.TelnetTLSPort != 0 && (c.ServerEnv["TELNET_TLS_CERT_FILE"] == "" || c.ServerEnv["TELNET_TLS_KEY_FILE"] == "") {
		return errors.New("telnet_tls_port requires explicit certificate and key paths in server_env")
	}
	if c.MudletDirectory != "" && (!filepath.IsAbs(c.MudletDirectory) || containsControl(c.MudletDirectory)) {
		return errors.New("mudlet_directory must be an absolute path without control characters")
	}
	return nil
}

func containsControl(value string) bool {
	for _, c := range value {
		if unicode.IsControl(c) {
			return true
		}
	}
	return false
}

// Publish only the versioned package, before the server can advertise its
// compiled version. No directory sync or deletion touches other public files.
func publishMudlet(bundle string, c Config) (result error) {
	if c.MudletDirectory == "" {
		return nil
	}
	source, err := os.Open(filepath.Join(bundle, "release-assets", "mudlet", "darkpawns.xml"))
	if err != nil {
		return fmt.Errorf("open bundled Mudlet package: %w", err)
	}
	defer func() { result = errors.Join(result, source.Close()) }()
	root, err := os.OpenRoot(c.MudletDirectory)
	if err != nil {
		return fmt.Errorf("open existing public Mudlet directory: %w", err)
	}
	defer func() { result = errors.Join(result, root.Close()) }()
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	name := ".darkpawns-" + hex.EncodeToString(random[:]) + ".xml"
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if err := root.Remove(name); err != nil && !os.IsNotExist(err) {
			result = errors.Join(result, err)
		}
	}()
	_, copyErr := io.Copy(file, source)
	copyErr = errors.Join(copyErr, file.Chmod(0o644))
	if err := errors.Join(copyErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := root.Rename(name, "darkpawns.xml"); err != nil {
		return err
	}
	return syncUpdateDirectory(c.MudletDirectory)
}

func preflightHosting(c Config) error {
	if c.TelnetTLSPort != 0 {
		if _, err := tls.LoadX509KeyPair(c.ServerEnv["TELNET_TLS_CERT_FILE"], c.ServerEnv["TELNET_TLS_KEY_FILE"]); err != nil {
			return fmt.Errorf("hosted telnet TLS certificate: %w", err)
		}
	}
	for _, endpoint := range []struct {
		bind string
		port int
	}{{c.TelnetBind, c.TelnetPort}, {c.TelnetTLSBind, c.TelnetTLSPort}, {c.WhodBind, c.WhodPort}} {
		port := endpoint.port
		if port == 0 {
			continue
		}
		listener, err := net.Listen("tcp", net.JoinHostPort(endpoint.bind, strconv.Itoa(port)))
		if err != nil {
			return fmt.Errorf("hosted listener port %d is unavailable: %w", port, err)
		}
		if err := listener.Close(); err != nil {
			return err
		}
	}
	return nil
}
