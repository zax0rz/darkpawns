package telnet

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// comm.c:1510-1514: C writes
// "Sorry, Dark Pawns is full right now... please try again later!\r\n"
// to the descriptor when sockets_connected >= max_players, then closes it.
func TestTotalConnectionLimitRefusalBytes(t *testing.T) {
	origMaxTotal := maxTotalConns.Load()
	origMaxPerIP := maxConnsPerIP.Load()
	defer func() {
		maxTotalConns.Store(origMaxTotal)
		maxConnsPerIP.Store(origMaxPerIP)
	}()

	maxTotalConns.Store(1)
	maxConnsPerIP.Store(10)

	waitForNoConnections(t)
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return listener.Addr() })
	first, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial first: %v", err)
	}
	defer func() { _ = first.Close() }()

	waitUntilAccepted(t, first)

	second, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial second: %v", err)
	}
	defer func() { _ = second.Close() }()
	_ = second.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := io.ReadAll(second)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("read refusal: %v", err)
	}
	want := "Sorry, Dark Pawns is full right now... please try again later!\r\n"
	if string(got) != want {
		t.Fatalf("refusal bytes = %q, want %q", got, want)
	}
}

// Per-IP limit is a Go-only operations guard that C does not have (R4):
// it closes silently without inventing refusal bytes.
func TestPerIPConnectionLimitClosesSilently(t *testing.T) {
	origMaxTotal := maxTotalConns.Load()
	origMaxPerIP := maxConnsPerIP.Load()
	defer func() {
		maxTotalConns.Store(origMaxTotal)
		maxConnsPerIP.Store(origMaxPerIP)
	}()

	maxTotalConns.Store(10)
	maxConnsPerIP.Store(1)

	waitForNoConnections(t)
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return listener.Addr() })
	first, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial first: %v", err)
	}
	defer func() { _ = first.Close() }()

	waitUntilAccepted(t, first)

	second, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial second: %v", err)
	}
	defer func() { _ = second.Close() }()
	_ = second.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := io.ReadAll(second)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("read per-IP close: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("per-IP close bytes = %q, want empty", got)
	}
}

// The TLS port shares the accept loop and drops an over-limit connection before
// the handshake, so C's plaintext refusal is never written into an encrypted stream.
func TestTLSTotalConnectionLimitClosesWithoutPlaintext(t *testing.T) {
	origMaxTotal := maxTotalConns.Load()
	origMaxPerIP := maxConnsPerIP.Load()
	defer func() {
		maxTotalConns.Store(origMaxTotal)
		maxConnsPerIP.Store(origMaxPerIP)
	}()

	maxTotalConns.Store(1)
	maxConnsPerIP.Store(10)

	waitForNoConnections(t)
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	certFile, keyFile, pool := writeTestCert(t, t.TempDir(), "full-", "darkpawns.test")
	if err := ListenTLS(0, manager, certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return tlsListener.Addr() })
	first, err := tls.Dial("tcp", addr, &tls.Config{RootCAs: pool, ServerName: "darkpawns.test"})
	if err != nil {
		t.Fatalf("tls dial first: %v", err)
	}
	defer func() { _ = first.Close() }()

	second, err := tls.Dial("tcp", addr, &tls.Config{RootCAs: pool, ServerName: "darkpawns.test"})
	if err == nil {
		_ = second.SetReadDeadline(time.Now().Add(time.Second))
		buf, _ := io.ReadAll(second)
		_ = second.Close()
		if bytes.Contains(buf, []byte("Sorry, Dark Pawns is full right now")) {
			t.Fatalf("TLS client received plaintext refusal: %q", buf)
		}
	}
}

// waitUntilAccepted blocks until the server has sent the connection its first
// byte. The accept loop counts a connection (connCount, connPerIP) before its
// handler starts writing, so any received byte proves the count happened.
// Without it, a slow accept loop (race detector, loaded host) could take the
// second dial first, refuse the first instead, and leave the test reading a
// live session until its deadline.
func waitUntilAccepted(t *testing.T, conn net.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("first connection was not accepted: %v", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
}

// waitForNoConnections blocks until connections left by an earlier test have
// been released. The handler decrements connCount and connPerIP only after its
// connection closes, asynchronously, so a following test could otherwise start
// with the loopback address already at its per-IP limit.
func waitForNoConnections(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		connMu.Lock()
		n := connCount
		connMu.Unlock()
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d connections from an earlier test were never released", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
