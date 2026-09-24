package oraclediff

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A stand-in driver: it echoes each typed line the way the browser client
// does (line, then CRLF) and answers with one line of "server" text.
const fakeDriver = `while IFS= read -r l; do printf '%s\r\nyou typed %s\r\n' "$l" "$l"; done`

func TestWSConnDropsLocalEchoAndReadsToQuiescence(t *testing.T) {
	script := filepath.Join(t.TempDir(), "driver.sh")
	if err := os.WriteFile(script, []byte(fakeDriver), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := NewWSConn("/bin/sh", script, "client.js", "ws://unused")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if err := c.Send("look"); err != nil {
		t.Fatal(err)
	}
	got, err := c.ReadUntilQuiescent(200 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if want := "you typed look\r\n"; got != want {
		t.Fatalf("read %q, want %q", got, want)
	}
}

func TestWSConnDropsClientConnectionChrome(t *testing.T) {
	c := &WSConn{chrome: clientChrome("ws://h/ws")}
	c.buf.WriteString("\x1b[2mConnecting to ws://h/ws...\x1b[0m\r\n" +
		"\x1b[32mConnected.\x1b[0m\r\n\r\n" +
		"Connected. is game text here\r\n" +
		"\x1b[31m\r\n--- Connection lost ---\x1b[0m\r\n")
	if got, want := c.take(), "Connected. is game text here\r\n"; got != want {
		t.Fatalf("take() = %q, want %q", got, want)
	}
}
