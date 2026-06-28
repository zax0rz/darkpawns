package telnet

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

func newTestManager(t *testing.T) (*session.Manager, *game.World) {
	t.Helper()
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	manager := session.NewManager(world, nil)
	return manager, world
}

// drain reads and discards everything from c until it is closed.
func drain(c net.Conn) {
	buf := make([]byte, 4096)
	for {
		_, err := c.Read(buf)
		if err != nil {
			return
		}
	}
}

// TestHandleConnDisconnectDuringPasswordPrompt verifies that handleConn returns
// when the client disconnects instead of proceeding with an empty password.
func TestHandleConnDisconnectDuringPasswordPrompt(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	go drain(client)

	// Provide a name, then close the connection while the server is waiting
	// for a password/confirmation response.
	_, _ = client.Write([]byte("testplayer\r\n"))
	time.Sleep(50 * time.Millisecond)
	_ = client.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after disconnect")
	}
}

// TestHandleConnRejectsEmptyPassword verifies that empty new-character passwords
// are rejected and the connection handler returns instead of continuing.
func TestHandleConnRejectsEmptyPassword(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	go drain(client)

	// No DB path: name -> yes -> empty password -> empty confirmation.
	_, _ = client.Write([]byte("newplayer\r\ny\r\n\r\n\r\n"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after empty password")
	}
}

func TestEffectiveBanLevel_IPAndHostname(t *testing.T) {
	bm := game.NewBanManager()
	bm.AddBan("192.0.2.10", game.BanNew, "test")
	bm.AddBan("evil.example.com", game.BanAll, "test")

	origLookup := lookupAddr
	defer func() { lookupAddr = origLookup }()
	lookupAddr = func(addr string) ([]string, error) {
		if addr == "192.0.2.10" {
			return []string{"evil.example.com."}, nil
		}
		return nil, nil
	}

	// IP-only check would return BanNew; hostname check elevates to BanAll.
	if got := effectiveBanLevel("192.0.2.10", bm); got != game.BanAll {
		t.Fatalf("effectiveBanLevel = %d, want BanAll (%d)", got, game.BanAll)
	}
}

func TestEffectiveBanLevel_TimeoutFallsBackToIP(t *testing.T) {
	bm := game.NewBanManager()
	bm.AddBan("198.51.100.5", game.BanAll, "test")

	origLookup := lookupAddr
	origTimeout := dnsLookupTimeout
	defer func() {
		lookupAddr = origLookup
		dnsLookupTimeout = origTimeout
	}()

	lookupAddr = func(addr string) ([]string, error) {
		time.Sleep(100 * time.Millisecond)
		return []string{"slow.example.com."}, nil
	}
	dnsLookupTimeout = 10 * time.Millisecond

	// Even though DNS times out, the raw IP is still banned.
	if got := effectiveBanLevel("198.51.100.5", bm); got != game.BanAll {
		t.Fatalf("effectiveBanLevel = %d, want BanAll (%d)", got, game.BanAll)
	}
}

func TestEffectiveBanLevel_NoMatch(t *testing.T) {
	bm := game.NewBanManager()

	origLookup := lookupAddr
	defer func() { lookupAddr = origLookup }()
	lookupAddr = func(addr string) ([]string, error) {
		return []string{"clean.example.com."}, nil
	}

	if got := effectiveBanLevel("203.0.113.7", bm); got != game.BanNot {
		t.Fatalf("effectiveBanLevel = %d, want BanNot (%d)", got, game.BanNot)
	}
}

// TestReadLineCapsBufferWithoutNewline verifies that a client sending more
// than maxInputLen bytes without a newline cannot grow the input buffer
// unboundedly (DP-622).
func TestReadLineCapsBufferWithoutNewline(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	data := make([]byte, maxInputLen+10)
	for i := range data {
		data[i] = 'a'
	}
	data = append(data, '\n')
	go func() {
		_, _ = client.Write(data)
		go drain(client)
	}()

	line, ok := tc.readLine()
	if !ok {
		t.Fatal("readLine returned false, want true")
	}
	if len(line) != maxInputLen {
		t.Errorf("readLine length = %d, want %d", len(line), maxInputLen)
	}
}

// TestReadLineCapsIACEscapedBytes verifies that a stream of IAC IAC escape
// sequences (each appending a literal 0xFF byte) is also capped at
// maxInputLen (DP-622).
func TestReadLineCapsIACEscapedBytes(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	data := make([]byte, 0, (maxInputLen+10)*2+1)
	for i := 0; i < maxInputLen+10; i++ {
		data = append(data, IAC, IAC)
	}
	data = append(data, '\n')

	go func() {
		_, _ = client.Write(data)
		go drain(client)
	}()

	line, ok := tc.readLine()
	if !ok {
		t.Fatal("readLine returned false, want true")
	}
	if len(line) != maxInputLen {
		t.Errorf("readLine length = %d, want %d", len(line), maxInputLen)
	}
}
