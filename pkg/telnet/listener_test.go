package telnet

import (
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
