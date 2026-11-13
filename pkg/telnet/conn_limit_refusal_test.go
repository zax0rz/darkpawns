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

	// Wait until the accept loop has registered the first connection.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		connMu.Lock()
		n := connCount
		connMu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		connMu.Lock()
		n := connCount
		connMu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

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
