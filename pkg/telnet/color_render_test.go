package telnet

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWriteLoopRecipientColors(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()
	s := manager.NewSession()
	if !s.TerminalLine("guest") || s.GetPlayer() == nil {
		t.Fatal("guest fixture login failed")
	}
	for {
		select {
		case <-s.SendChannel():
			continue
		default:
			goto drained
		}
	}
drained:
	s.GetPlayer().SetPlrFlag(game.PrfColor1, true)
	s.Send("&RRed&n &&r")
	s.CloseSend()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	tc := &telnetConn{Conn: server, wmu: make(chan struct{}, 1), manager: manager, sess: s}
	done := make(chan struct{})
	go func() { writeLoop(tc, s); server.Close(); close(done) }()
	data, err := io.ReadAll(client)
	if err != nil {
		t.Fatal(err)
	}
	<-done
	if string(data) != "\x1b[1;31mRed\x1b[0m &r\r\n" {
		t.Fatalf("telnet bytes = %q", data)
	}
}
