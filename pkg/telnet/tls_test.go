package telnet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
)

// writeTestCert writes a self-signed certificate for names to dir and
// returns the cert and key paths plus a pool that trusts it.
func writeTestCert(t *testing.T, dir, prefix string, names ...string) (certFile, keyFile string, pool *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: names[0]},
		DNSNames:              names,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, prefix+"cert.pem")
	keyFile = filepath.Join(dir, prefix+"key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool = x509.NewCertPool()
	pool.AddCert(leaf)
	return certFile, keyFile, pool
}

func tlsListenerAddr(t *testing.T) string {
	t.Helper()
	connMu.Lock()
	defer connMu.Unlock()
	if tlsListener == nil {
		t.Fatal("TLS listener not running")
	}
	return tlsListener.Addr().String()
}

// TestTLSTelnetServesTheSameGame: the TLS port is transport only. The same
// session over TLS and over plain telnet produces byte-identical text, and a
// client that verifies the certificate by name connects.
func TestTLSTelnetServesTheSameGame(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	certFile, keyFile, pool := writeTestCert(t, t.TempDir(), "", "darkpawns.test")

	plainPort := listenerEntryPort(t)
	if err := Listen(plainPort, manager); err != nil {
		t.Fatal(err)
	}
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}

	const name = "guest_tls_twin"
	script := []string{"north", "say over the wire", "south", "quit"} // quit away from the start room asks for REALLYQUIT
	plain := visibleText(parseTelnetStream(t, runTelnetScript(t, plainPort, nil, name, script)))
	waitSessionGone(t, manager, name)

	conn, err := tls.Dial("tcp", tlsListenerAddr(t), &tls.Config{RootCAs: pool, ServerName: "darkpawns.test", MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatalf("verified TLS dial: %v", err)
	}
	secure := visibleText(parseTelnetStream(t, runTelnetScriptOn(t, conn, nil, name, script)))

	if !strings.Contains(plain, "You say 'over the wire'") {
		t.Fatalf("plain session did not run the script: %q", plain)
	}
	if secure != plain {
		t.Fatalf("TLS text differs from plain text:\n tls: %q\nplain: %q", secure, plain)
	}
}

func TestTLSRejectsOldProtocolVersions(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	certFile, keyFile, pool := writeTestCert(t, t.TempDir(), "", "darkpawns.test")
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	conn, err := tls.Dial("tcp", tlsListenerAddr(t), &tls.Config{
		RootCAs: pool, ServerName: "darkpawns.test",
		MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11, // #nosec G402 -- deliberately old, to prove it is refused
	})
	if err == nil {
		_ = conn.Close()
		t.Fatal("TLS 1.1 handshake succeeded; the listener must require TLS 1.2+")
	}
}

// TestTLSStalledHandshakeIsDropped: a client that connects and never speaks
// TLS is closed after the handshake deadline and gives its slot back.
func TestTLSStalledHandshakeIsDropped(t *testing.T) {
	origTimeout := tlsHandshakeTimeout
	tlsHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { tlsHandshakeTimeout = origTimeout })

	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	certFile, keyFile, _ := writeTestCert(t, t.TempDir(), "", "darkpawns.test")
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", tlsListenerAddr(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("stalled connection was not closed by the server: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		connMu.Lock()
		open := connCount
		connMu.Unlock()
		if open == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("connection slot not released: connCount=%d", open)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// msspFields asks the plain listener for MSSP and returns its variables.
func msspFields(t *testing.T, port int) map[string]string {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	readTelnetUntil(t, conn, "By what name do you wish to be known?")
	if _, err := conn.Write([]byte{IAC, DO, OPT_MSSP}); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte("\r\n")); err != nil { // blank name: the server says goodbye and closes
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	raw, _ := io.ReadAll(conn)
	start := bytes.Index(raw, []byte{IAC, SB, OPT_MSSP})
	if start < 0 {
		t.Fatalf("no MSSP subnegotiation in %q", raw)
	}
	body, _, ok := bytes.Cut(raw[start+3:], []byte{IAC, SE})
	if !ok {
		t.Fatalf("unterminated MSSP subnegotiation in %q", raw)
	}
	fields := map[string]string{}
	for _, pair := range bytes.Split(body, []byte{MSSP_VAR})[1:] {
		name, value, _ := bytes.Cut(pair, []byte{MSSP_VAL})
		fields[string(name)] = string(value)
	}
	return fields
}

// TestMSSPAdvertisesTLSPort: with TLS on, MSSP carries TLS=<port> and the
// certificate's hostname, which is what Mudlet reads to offer a plaintext
// player the encrypted port. With TLS off, neither field is sent.
func TestMSSPAdvertisesTLSPort(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	plainPort := listenerEntryPort(t)
	if err := Listen(plainPort, manager); err != nil {
		t.Fatal(err)
	}

	fields := msspFields(t, plainPort)
	if _, ok := fields["TLS"]; ok {
		t.Fatalf("MSSP advertised TLS with no TLS listener: %v", fields)
	}
	if fields["GMCP"] != "1" || fields["NAME"] != "Dark Pawns" {
		t.Fatalf("MSSP fields = %v", fields)
	}

	certFile, keyFile, _ := writeTestCert(t, t.TempDir(), "", "*.darkpawns.test", "darkpawns.test")
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(tlsListenerAddr(t))
	if err != nil {
		t.Fatal(err)
	}
	fields = msspFields(t, plainPort)
	if fields["TLS"] != port || fields["HOSTNAME"] != "darkpawns.test" {
		t.Fatalf("MSSP TLS=%q HOSTNAME=%q, want %s and the first non-wildcard name", fields["TLS"], fields["HOSTNAME"], port)
	}
}

func TestCertReloaderFollowsRenewals(t *testing.T) {
	origInterval := certCheckInterval
	certCheckInterval = 0
	t.Cleanup(func() { certCheckInterval = origInterval })

	dir := t.TempDir()
	certFile, keyFile, _ := writeTestCert(t, dir, "", "old.darkpawns.test")
	reloader, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	served := func() string {
		cert, err := reloader.getCertificate(nil)
		if err != nil {
			t.Fatal(err)
		}
		return cert.Leaf.DNSNames[0]
	}
	if got := served(); got != "old.darkpawns.test" {
		t.Fatalf("serving %s", got)
	}

	// A renewal rewrites both files in place.
	newCert, newKey, _ := writeTestCert(t, dir, "next-", "new.darkpawns.test")
	for from, to := range map[string]string{newCert: certFile, newKey: keyFile} {
		if err := os.Rename(from, to); err != nil {
			t.Fatal(err)
		}
	}
	future := time.Now().Add(time.Minute)
	for _, path := range []string{certFile, keyFile} {
		if err := os.Chtimes(path, future, future); err != nil {
			t.Fatal(err)
		}
	}
	if got := served(); got != "new.darkpawns.test" {
		t.Fatalf("renewed certificate not picked up; serving %s", got)
	}
	if reloader.hostname() != "new.darkpawns.test" {
		t.Fatalf("hostname = %s", reloader.hostname())
	}

	// A half-written renewal keeps the certificate already loaded.
	if err := os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\ntruncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	later := future.Add(time.Minute)
	if err := os.Chtimes(certFile, later, later); err != nil {
		t.Fatal(err)
	}
	if got := served(); got != "new.darkpawns.test" {
		t.Fatalf("broken renewal replaced the certificate; serving %s", got)
	}
}

func TestListenTLSRefusesMissingCertificate(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	dir := t.TempDir()
	if err := ListenTLS(0, manager, filepath.Join(dir, "missing.pem"), filepath.Join(dir, "missing-key.pem")); err == nil {
		t.Fatal("ListenTLS started without a certificate")
	}
	connMu.Lock()
	defer connMu.Unlock()
	if tlsListener != nil || tlsAdvert.Load() != nil {
		t.Fatal("a failed ListenTLS left a listener or an MSSP advertisement behind")
	}
}
