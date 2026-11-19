package oraclediff

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestTCPReadUntilPromptCapturesMultiPulseEditorAlias(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		_, _ = server.Read(buf)
		// Pulse 1: enters string editor and emits "] ".
		_, _ = server.Write([]byte("Write your message.\r\nInstructions: /s or @ to save, /h for more options.\r\n\r\n] "))
		// Gap exceeds quiescence (30ms), simulating CPU load between pulses.
		time.Sleep(100 * time.Millisecond)
		// Pulse 2: body line consumed, emits "] ".
		_, _ = server.Write([]byte("] "))
		time.Sleep(100 * time.Millisecond)
		// Pulse 3: "@" saves and exits editor, emits command prompt "> ".
		_, _ = server.Write([]byte("\x1b[32m> \x1b[0m"))
	}()

	conn := NewTCPConn(client)
	if err := conn.Send("post"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, err := conn.ReadUntilPrompt(30*time.Millisecond, 2*time.Second, "> ", 1)
	<-done
	if err != nil {
		t.Fatalf("ReadUntilPrompt: %v", err)
	}
	want := "Write your message.\r\nInstructions: /s or @ to save, /h for more options.\r\n\r\n] ] \x1b[32m> \x1b[0m"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTCPReadUntilPromptMinCount(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		_, _ = server.Read(buf)
		_, _ = server.Write([]byte("Currently defined aliases:\r\nc alias;alias\r\n> "))
		time.Sleep(100 * time.Millisecond)
		_, _ = server.Write([]byte("Currently defined aliases:\r\nc alias;alias\r\n> "))
	}()

	conn := NewTCPConn(client)
	if err := conn.Send("c"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, err := conn.ReadUntilPrompt(30*time.Millisecond, 2*time.Second, "> ", 2)
	<-done
	if err != nil {
		t.Fatalf("ReadUntilPrompt: %v", err)
	}
	if strings.Count(got, "Currently defined aliases:") != 2 {
		t.Fatalf("expected 2 alias listings, got %q", got)
	}
}

func TestTCPReadUntilPromptEditorSuffix(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		_, _ = server.Read(buf)
		_, _ = server.Write([]byte("Write your message.\r\n\r\n"))
		time.Sleep(90 * time.Millisecond)
		_, _ = server.Write([]byte("] "))
	}()

	conn := NewTCPConn(client)
	if err := conn.Send("write test"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, err := conn.ReadUntilPrompt(30*time.Millisecond, 2*time.Second, "] ", 1)
	<-done
	if err != nil {
		t.Fatalf("ReadUntilPrompt: %v", err)
	}
	if got != "Write your message.\r\n\r\n] " {
		t.Fatalf("got %q, want complete editor banner + prompt", got)
	}
}

func TestTCPReadUntilPromptTimeoutReturnsPartialVerbatim(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		_, _ = server.Read(buf)
		_, _ = server.Write([]byte("Write your message.\r\n\r\n] "))
	}()

	conn := NewTCPConn(client)
	if err := conn.Send("post"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	start := time.Now()
	got, err := conn.ReadUntilPrompt(30*time.Millisecond, 150*time.Millisecond, "> ", 1)
	elapsed := time.Since(start)
	<-done
	if err != nil {
		t.Fatalf("ReadUntilPrompt: %v", err)
	}
	if got != "Write your message.\r\n\r\n] " {
		t.Fatalf("got %q, want partial output verbatim", got)
	}
	if elapsed < 120*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("elapsed=%v, want bounded near 150ms", elapsed)
	}
}

func TestParsePromptWait(t *testing.T) {
	cases := []struct {
		in      string
		suffix  string
		count   int
		inner   string
		wantErr bool
	}{
		{in: "post", suffix: "", count: 0, inner: "post"},
		{in: "wait-prompt:command post", suffix: "> ", count: 1, inner: "post"},
		{in: "wait-prompt:editor write note", suffix: "] ", count: 1, inner: "write note"},
		{in: "wait-prompt:2:command c", suffix: "> ", count: 2, inner: "c"},
		{in: "wait-prompt:command send:peer post", suffix: "> ", count: 1, inner: "send:peer post"},
		{in: "wait-prompt:bogus post", wantErr: true},
		{in: "wait-prompt:0:command post", wantErr: true},
		{in: "wait-prompt:command", wantErr: true},
	}
	for _, tc := range cases {
		suffix, count, inner, err := parsePromptWait(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parsePromptWait(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePromptWait(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if suffix != tc.suffix || count != tc.count || inner != tc.inner {
			t.Errorf("parsePromptWait(%q) = (%q, %d, %q), want (%q, %d, %q)",
				tc.in, suffix, count, inner, tc.suffix, tc.count, tc.inner)
		}
	}
}
