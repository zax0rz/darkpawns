package telnet

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ipv4ListenerAddr returns the listener's port with an explicit IPv4 host. The
// ban spellings under test are d->host's padded IPv4 quad, and a dial to the
// listener's own IPv6 address would present ::1 as the remote address.
func ipv4ListenerAddr(t *testing.T, addr func() net.Addr) string {
	t.Helper()
	connMu.Lock()
	listenAddr := addr().String()
	connMu.Unlock()
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		t.Fatalf("listener address %q: %v", listenAddr, err)
	}
	return net.JoinHostPort("127.0.0.1", port)
}

// comm.c:1571-1575: C writes "Sorry, your site is banned.\r\n" to the
// descriptor, closes it, and only then logs "Connection attempt denied from
// [%s]" at CMP / LVL_GOD / file TRUE. The ban is written in the spelling C
// stores in d->host, or in the wildcard C derives from it.
func TestBannedConnectionRefusalBytesAndLog(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	// C's wildhost string for 127.0.0.1 (src/comm.c:1541-1544).
	if err := manager.GetBanManager().AddBan("127.000.000.*", game.BanAll, "God"); err != nil {
		t.Fatal(err)
	}
	provider := captureMudlog(t)

	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return listener.Addr() })
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := io.ReadAll(conn)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("read refusal: %v", err)
	}
	if string(got) != "Sorry, your site is banned.\r\n" {
		t.Fatalf("refusal bytes = %q", got)
	}

	messages := waitForMessages(t, provider, 1)
	want := "[ Connection attempt denied from [127.000.000.001] ]\r\n"
	if len(messages) != 1 || messages[0] != want {
		t.Fatalf("producer = %q, want exactly %q", messages, want)
	}
	if filtered := provider.filtered(); len(filtered) != 0 {
		t.Fatalf("below-threshold or brief observer received %q", filtered)
	}
}

// The TLS port shares the accept loop and drops a banned address before the
// handshake, so C's plaintext refusal is never written into an encrypted
// stream: TLS keeps its bare close (a Go-only transport) and still logs.
func TestTLSBannedConnectionClosesWithoutPlaintextButLogs(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	if err := manager.GetBanManager().AddBan("127.000.000.*", game.BanAll, "God"); err != nil {
		t.Fatal(err)
	}
	provider := captureMudlog(t)

	certFile, keyFile, pool := writeTestCert(t, t.TempDir(), "banned-", "darkpawns.test")
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return tlsListener.Addr() })
	conn, err := tls.Dial("tcp", addr, &tls.Config{RootCAs: pool, ServerName: "darkpawns.test"})
	if err == nil {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		buf, _ := io.ReadAll(conn)
		_ = conn.Close()
		if bytes.Contains(buf, []byte("Sorry, your site is banned.")) {
			t.Fatalf("TLS client received the plaintext refusal: %q", buf)
		}
	}

	messages := waitForMessages(t, provider, 1)
	want := "[ Connection attempt denied from [127.000.000.001] ]\r\n"
	if len(messages) != 1 || messages[0] != want {
		t.Fatalf("producer = %q, want exactly %q", messages, want)
	}
}
