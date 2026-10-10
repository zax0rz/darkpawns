package telnet

import (
	"io"
	"net"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

// Use the real telnet writeLoop on a pipe, with the writer ready before output.

func TestHeartbeatOutputTelnetOrderlyBoundary(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	defer world.StopAITicker()
	m := session.NewManager(world, nil)
	defer m.Stop()
	s := m.NewSession()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	s.SetCloseFunc(func() { _ = server.Close() })
	tc := &telnetConn{Conn: server, sess: s, wmu: make(chan struct{}, 1), manager: m}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer server.Close()
		writeLoop(tc, s)
	}()
	transcript := make(chan []byte, 1)
	go func() { out, _ := io.ReadAll(client); transcript <- out }()
	m.BeginHeartbeatOutput()
	s.Send("Goodbye.\r\n")
	s.CloseSend()
	s.Close()
	m.EndHeartbeatOutput()
	<-done
	if got := string(<-transcript); got != "Goodbye.\r\n" {
		t.Fatalf("telnet lost goodbye: %q", got)
	}
}
