package telnet

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"testing"
)

func TestEscapeIAC(t *testing.T) {
	cases := []struct{ in, want []byte }{
		{[]byte("plain text\r\n"), []byte("plain text\r\n")},
		{[]byte{'a', IAC, 'b'}, []byte{'a', IAC, IAC, 'b'}},
		{[]byte{IAC, IAC}, []byte{IAC, IAC, IAC, IAC}},
		{[]byte{}, []byte{}},
	}
	for _, tc := range cases {
		if got := escapeIAC(tc.in); !bytes.Equal(got, tc.want) {
			t.Errorf("escapeIAC(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// A 0xFF in game text must reach the wire doubled; unescaped, the client reads
// the following game bytes as a telnet command (REF-1). The EOR prompt marker
// stays a single IAC because it is a command, not text.
func TestTextWritesEscapeIAC(t *testing.T) {
	server, client := net.Pipe()
	defer func() { _ = server.Close(); _ = client.Close() }()
	tc := &telnetConn{Conn: server, br: bufio.NewReader(server), wmu: make(chan struct{}, 1)}
	tc.hasEOR.Store(true)

	got := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(client)
		got <- buf
	}()

	tc.writeLine("a\xffb\r\n")
	tc.writePrompt("p\xff> ")
	_ = server.Close()

	want := []byte{'a', IAC, IAC, 'b', '\r', '\n', 'p', IAC, IAC, '>', ' ', IAC, EOR}
	if out := <-got; !bytes.Equal(out, want) {
		t.Fatalf("wire bytes = %v, want %v", out, want)
	}
}
