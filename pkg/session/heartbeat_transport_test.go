package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Exercise the actual WebSocket writer, rather than only inspecting send.
func TestHeartbeatOutputWebSocketIdleBoundary(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := m.NewSession()
	actor := makeTestSession(t, m, "Socketidler", 1, true)
	s.player = actor.player
	s.playerName = actor.playerName
	s.authenticated = true
	s.player.SetLevel(1)
	s.player.SetIdleTimer(30)
	s.player.WasInRoom = 1001
	s.writerDone = make(chan struct{})
	registerTestSession(t, m, s, s.playerName)
	listener := makeTestSession(t, m, "Socketlistener", 4, true)
	spy := makeTestSession(t, m, "Socketspy", 4, true)
	registerTestSession(t, m, listener, listener.playerName)
	registerTestSession(t, m, spy, spy.playerName)
	m.snoopMu.Lock()
	s.snoopBy = spy
	spy.snooping = s
	m.snoopMu.Unlock()
	var peerConns []*websocket.Conn
	for _, peer := range []*Session{listener, spy} {
		peer.writerDone = make(chan struct{})
		peerReady := make(chan struct{})
		peerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := m.upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			peer.conn = conn
			close(peerReady)
			peer.writePump()
		}))
		t.Cleanup(func() { peer.closeSendNow(); peer.Close(); peerServer.Close() })
		peerConn := entryWebSocketDial(t, peerServer.URL)
		t.Cleanup(func() { _ = peerConn.Close() })
		<-peerReady
		peerConns = append(peerConns, peerConn)
	}
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := m.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		s.conn = conn
		close(ready)
		s.writePump()
	}))
	defer server.Close()
	defer func() { s.closeSendNow(); s.Close() }()
	conn := entryWebSocketDial(t, server.URL)
	defer conn.Close()
	<-ready
	m.BeginHeartbeatOutput()
	m.SendToOutdoor("The suns slowly disappear in the west and south.\r\n")
	m.world.CheckIdling(s.player)
	if !s.SendClosed() {
		t.Fatal("idle descriptor still open before commit")
	}
	m.EndHeartbeatOutput()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				t.Fatalf("expected socket close, got %v", err)
			}
			break
		}
		t.Fatalf("WebSocket received discarded frame: %s", msg)
	}
	<-s.writerDone
	for i, peer := range []*Session{listener, spy} {
		peer.closeSendNow()
		pc := peerConns[i]
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		var text strings.Builder
		for {
			_, raw, err := pc.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
					t.Fatalf("expected peer socket close, got %v", err)
				}
				break
			}
			if frame, ok := RenderTerminalFrame(raw); ok {
				text.WriteString(frame.Text)
			}
		}
		<-peer.writerDone
		got := text.String()
		if strings.Count(got, "Socketidler has lost") != 1 || strings.Count(got, "> ") != 1 || strings.Count(got, "suns slowly disappear") != 1 {
			t.Fatalf("WebSocket room listener %d received %q", i, got)
		}
		if i == 1 && strings.Count(got, "Your victim is no longer among us.") != 1 {
			t.Fatalf("WebSocket snooper lost close notification: %q", got)
		}
	}
	if !s.SendClosed() {
		t.Fatal("WebSocket actor did not close")
	}
}

func TestHeartbeatOutputWebSocketOrderlyBoundary(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := m.NewSession()
	s.writerDone = make(chan struct{})
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := m.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		s.conn = conn
		close(ready)
		s.writePump()
	}))
	defer server.Close()
	defer func() { s.closeSendNow(); s.Close() }()
	conn := entryWebSocketDial(t, server.URL)
	defer conn.Close()
	<-ready
	m.BeginHeartbeatOutput()
	s.Send("Goodbye.\r\n")
	s.CloseSend()
	s.Close() // existing callers may request both; the writer must drain first
	m.EndHeartbeatOutput()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var text strings.Builder
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				t.Fatalf("expected socket close, got %v", err)
			}
			break
		}
		var msg struct {
			Data struct {
				Text string `json:"text"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatal(err)
		}
		text.WriteString(msg.Data.Text)
	}
	<-s.writerDone
	if text.String() != "Goodbye.\r\n" {
		t.Fatalf("WebSocket lost goodbye: %q", text.String())
	}
}
