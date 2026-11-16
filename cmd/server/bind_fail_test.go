package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func bindOccupiedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind test port: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := ln.Close(); closeErr != nil {
			t.Errorf("close bound port: %v", closeErr)
		}
	})
	return ln.Addr().(*net.TCPAddr).Port
}

func writeTestCertFiles(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test cert: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal test key: %v", err)
	}
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certPath, keyPath
}

// DP-1367: a server whose configured listener port is already bound must exit
// non-zero rather than logging and keeping the other listeners running.
func TestServerExitsNonZeroWhenListenerPortIsBound(t *testing.T) {
	world := parseableWorld(t)
	certFile, keyFile := writeTestCertFiles(t)

	cases := []struct {
		name    string
		args    func(busyPort int) []string
		env     []string
		wantLog string
	}{
		{
			name: "telnet",
			args: func(busyPort int) []string {
				return []string{
					"-world", world,
					"-port", "0",
					"-telnet-bind", "127.0.0.1",
					"-telnet-port", fmt.Sprintf("%d", busyPort),
					"-whod-port", "0",
				}
			},
			wantLog: "Telnet listener failed",
		},
		{
			name: "telnet-tls",
			args: func(busyPort int) []string {
				return []string{
					"-world", world,
					"-port", "0",
					"-telnet-port", "0",
					"-whod-port", "0",
					"-telnet-tls-bind", "127.0.0.1",
					"-telnet-tls-port", fmt.Sprintf("%d", busyPort),
				}
			},
			env: []string{
				"TELNET_TLS_CERT_FILE=" + certFile,
				"TELNET_TLS_KEY_FILE=" + keyFile,
			},
			wantLog: "Telnet TLS listener failed",
		},
		{
			name: "whod",
			args: func(busyPort int) []string {
				return []string{
					"-world", world,
					"-port", "0",
					"-telnet-port", "0",
					"-whod-bind", "127.0.0.1",
					"-whod-port", fmt.Sprintf("%d", busyPort),
				}
			},
			wantLog: "WHOD listener failed",
		},
		{
			name: "http",
			args: func(busyPort int) []string {
				return []string{
					"-world", world,
					"-http-bind", "127.0.0.1",
					"-port", fmt.Sprintf("%d", busyPort),
					"-telnet-port", "0",
					"-whod-port", "0",
				}
			},
			wantLog: "Server error, shutting down gracefully",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			busyPort := bindOccupiedPort(t)
			env := append([]string{"ENVIRONMENT=development", "DATABASE_URL="}, tc.env...)
			code, out := bootServerContext(t, tc.args(busyPort), 10*time.Second, env...)
			if code != 1 {
				t.Fatalf("exit code = %d, want 1 (output:\n%s)", code, out)
			}
			if !strings.Contains(out, tc.wantLog) {
				t.Fatalf("output missing %q:\n%s", tc.wantLog, out)
			}
		})
	}
}
