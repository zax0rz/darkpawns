package telnet

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
)

// readUntil reads from conn until needle appears in the stream, so a test can
// drop the transport at a known point of the login dialogue instead of
// sleeping.
func readUntil(t *testing.T, conn net.Conn, needle string) string {
	t.Helper()
	var buf []byte
	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		chunk := make([]byte, 512)
		n, err := conn.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			if bytes.Contains(buf, []byte(needle)) {
				return string(buf)
			}
		}
		if err != nil {
			t.Fatalf("read until %q: %v (got %q)", needle, err, buf)
		}
	}
}

// dialPlainListener starts a plain telnet listener the test owns and returns a
// connection to it over IPv4 (so d->host is the padded quad under test).
func dialPlainListener(t *testing.T, manager *session.Manager) net.Conn {
	t.Helper()
	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)
	conn, err := net.Dial("tcp", ipv4ListenerAddr(t, func() net.Addr { return listener.Addr() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// comm.c:2140-2143: a client that drops before its first line at the name
// prompt leaves no character to name (C creates d->character only on that
// input, src/interpreter.c:1743-1750).
func TestDropBeforeNameLogsDescriptorLoss(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	provider := captureMudlog(t)

	conn := dialPlainListener(t, manager)
	readUntil(t, conn, "By what name do you wish to be known?")
	_ = conn.Close()

	got := waitForMessages(t, provider, 1)
	want := "[ Losing descriptor without char. ]\r\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("producer = %q, want exactly %q", got, want)
	}
}

// comm.c:2136-2138: an empty line at the name prompt created the character
// first, so close_socket finds one with a NULL name and prints C's "<null>".
func TestEmptyNameLineLogsNullPlayer(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	provider := captureMudlog(t)

	conn := dialPlainListener(t, manager)
	readUntil(t, conn, "By what name do you wish to be known?")
	if _, err := conn.Write([]byte("\r\n")); err != nil {
		t.Fatal(err)
	}

	got := waitForMessages(t, provider, 1)
	want := "[ Losing player: <null>. ]\r\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("producer = %q, want exactly %q", got, want)
	}
}

// comm.c:2136-2138 during character creation: the descriptor holds the named
// character, and C's nanny close names it.
func TestDropDuringCreationLogsPlayer(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	provider := captureMudlog(t)

	conn := dialPlainListener(t, manager)
	readUntil(t, conn, "By what name do you wish to be known?")
	if _, err := conn.Write([]byte("Ghost\r\n")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, conn, "Did I get that right, Ghost (Y/N)? ")
	_ = conn.Close()

	got := waitForMessages(t, provider, 1)
	want := "[ Losing player: Ghost. ]\r\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("producer = %q, want exactly %q", got, want)
	}
}
