package telnet

import (
	"io"
	"net"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

// Use the real telnet writeLoop on a pipe, with the writer ready before output.
func TestHeartbeatOutputTelnetIdleBoundary(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 0}, {VNum: 1}, {VNum: 3}, {VNum: 4}, {VNum: game.MortalStartRoom}}})
	if err != nil {
		t.Fatal(err)
	}
	defer world.StopAITicker()
	m := session.NewManager(world, nil)
	defer m.Stop()
	s := m.NewSession()
	if err := s.HandleMessage([]byte(`{"type":"login","data":{"player_name":"guest_pipeidler"}}`)); err != nil {
		t.Fatal(err)
	}
	p := s.GetPlayer()
	if p == nil {
		t.Fatal("guest did not enter world")
	}
	if err := world.PlayerTransfer(p, 1); err != nil {
		t.Fatal(err)
	}
	p.SetLevel(1)
	p.SetIdleTimer(30)
	p.WasInRoom = 3
	for len(s.SendChannel()) > 0 {
		<-s.SendChannel()
	}
	var peers []*session.Session
	for _, name := range []string{"Pipelistener", "Pipespy"} {
		peer := m.NewSession()
		for _, line := range []string{name, "y", "oraclepass", "oraclepass", "N", "M", "H", "W", "K", "Y", "", "1"} {
			if !peer.TerminalLine(line) {
				t.Fatalf("peer entry closed at %q", line)
			}
		}
		if peer.GetPlayer() == nil {
			t.Fatal("peer did not enter world")
		}
		if err := world.PlayerTransfer(peer.GetPlayer(), 4); err != nil {
			t.Fatal(err)
		}
		peer.GetPlayer().SetLevel(game.LVL_IMPL)
		peers = append(peers, peer)
	}
	if err := peers[1].HandleMessage([]byte(`{"type":"command","data":{"command":"snoop","args":["guest_pipeidler"]}}`)); err != nil {
		t.Fatal(err)
	}
	var peerTranscripts []chan []byte
	for _, peer := range peers {
		for len(peer.SendChannel()) > 0 {
			<-peer.SendChannel()
		}
		pc, ps := net.Pipe()
		t.Cleanup(func() { _ = pc.Close(); _ = ps.Close(); peer.CloseSend() })
		ptc := &telnetConn{Conn: ps, sess: peer, wmu: make(chan struct{}, 1), manager: m}
		out := make(chan []byte, 1)
		go func() { data, _ := io.ReadAll(pc); out <- data }()
		go func() { defer ps.Close(); writeLoop(ptc, peer) }()
		peerTranscripts = append(peerTranscripts, out)
	}
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	s.SetCloseFunc(func() { _ = server.Close() })
	tc := &telnetConn{Conn: server, sess: s, wmu: make(chan struct{}, 1), manager: m}
	done := make(chan struct{})
	go func() { defer close(done); writeLoop(tc, s) }()
	transcript := make(chan []byte, 1)
	go func() { out, _ := io.ReadAll(client); transcript <- out }()
	m.BeginHeartbeatOutput()
	m.SendToOutdoor("The suns slowly disappear in the west and south.\r\n")
	world.CheckIdling(p)
	if !s.SendClosed() {
		s.CloseSend()
		_ = server.Close()
		m.EndHeartbeatOutput()
		t.Fatal("idle descriptor still open before commit")
	}
	m.EndHeartbeatOutput()
	<-done
	for i, peer := range peers {
		peer.CloseSend()
		got := string(<-peerTranscripts[i])
		if strings.Count(got, "Guest_pipeidler has lost") != 1 || strings.Count(got, "> ") != 1 || strings.Count(got, "suns slowly disappear") != 1 {
			t.Fatalf("telnet room listener %d received %q", i, got)
		}
		if i == 1 && strings.Count(got, "Your victim is no longer among us.") != 1 {
			t.Fatalf("telnet snooper lost close notification: %q", got)
		}
	}
	if got := string(<-transcript); strings.Contains(got, "suns") || got != "" {
		t.Fatalf("telnet received discarded output: %q", got)
	}
}

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
