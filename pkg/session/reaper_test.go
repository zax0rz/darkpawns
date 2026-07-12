package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeTestManagerWithVoidRooms builds a Manager whose world contains the
// linkdead void room (vnum 1), disconnect room (vnum 3), and the mortal
// start room so WebSocket end-to-end tests can complete character creation.
func makeTestManagerWithVoidRooms(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1, Name: "Limbo", Zone: 0},
			{VNum: 3, Name: "A Totally Empty Room", Zone: 0},
			{VNum: 1001, Name: "Room A", Zone: 1},
			{VNum: game.MortalStartRoom, Name: "The Adventurers Guild", Zone: 80},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return NewManager(w, nil)
}

// registerTestSession adds a test session to the manager and its player to the
// world, marking the IP connection count as already decremented so cleanup
// paths do not touch the empty IP counter.
func registerTestSession(t *testing.T, m *Manager, s *Session, key string) {
	t.Helper()
	s.connCountDecremented = true
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()
}

// TestReapMovesToVoid: a session idle for 90s is moved to void room 1 and
// remembers its original room.
func TestReapMovesToVoid(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Voidie", 1001, true)
	s.lastActive.Store(time.Now().Add(-90 * time.Second).UnixNano())
	registerTestSession(t, m, s, "Voidie")

	m.ReapLinkdeadSessions()

	if s.player.GetRoom() != 1 {
		t.Errorf("player room = %d, want 1 (void)", s.player.GetRoom())
	}
	if s.player.GetWasInRoom() != 1001 {
		t.Errorf("WasInRoom = %d, want 1001", s.player.GetWasInRoom())
	}
	if _, ok := m.GetSession("Voidie"); !ok {
		t.Error("session should still be registered after move-to-void")
	}
}

// TestReapExtractsLinkdeadSession: a session idle for 10 minutes is removed
// from the sessions map and from the world.
func TestReapExtractsLinkdeadSession(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Ghost", 1001, true)
	s.lastActive.Store(time.Now().Add(-10 * time.Minute).UnixNano())
	registerTestSession(t, m, s, "Ghost")

	m.ReapLinkdeadSessions()

	if _, ok := m.GetSession("ghost"); ok {
		t.Error("linkdead session should have been unregistered")
	}
	if _, ok := m.world.GetPlayer("Ghost"); ok {
		t.Error("linkdead player should have been removed from world")
	}
}

// TestReapDoesNotKillActiveSession: a recently-active session is untouched.
func TestReapDoesNotKillActiveSession(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Alive", 1001, true)
	s.lastActive.Store(time.Now().UnixNano())
	registerTestSession(t, m, s, "Alive")

	m.ReapLinkdeadSessions()

	if _, ok := m.GetSession("Alive"); !ok {
		t.Error("active session should not have been reaped")
	}
	if s.player.GetRoom() != 1001 {
		t.Errorf("active player room = %d, want 1001", s.player.GetRoom())
	}
}

// TestReapReturnFromVoid: an authenticated player in the void room with
// WasInRoom set returns to the original room on the next inbound message.
func TestReapReturnFromVoid(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Returner", 1, true)
	s.player.SetWasInRoom(1001)
	registerTestSession(t, m, s, "Returner")

	s.maybeReturnFromVoid()

	if s.player.GetRoom() != 1001 {
		t.Errorf("player room = %d, want 1001", s.player.GetRoom())
	}
	if s.player.GetWasInRoom() != 0 {
		t.Errorf("WasInRoom = %d, want 0", s.player.GetWasInRoom())
	}
}

// TestWritePumpExitTriggersCleanup: closing the client socket without sending
// quit causes writePump (and readPump) to exit and Unregister the session.
func TestWritePumpExitTriggersCleanup(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)

	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.labz0rz.com")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Create a new character through the WebSocket so the session is registered.
	wsWrite(t, client, MsgLogin, map[string]interface{}{
		"player_name": "WritePumpGhost",
		"password":    "hunter2",
		"new_char":    true,
	})
	charStages := []string{"Y", "hunter2", "hunter2", "N", "M", "H", "W", "K"}
	for _, choice := range charStages {
		wsReadUntilType(t, client, MsgCharCreate)
		wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": choice})
	}
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": "Y"})
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": ""})
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": "1"})
	wsReadUntilType(t, client, MsgState)

	if _, ok := m.GetSession("WritePumpGhost"); !ok {
		t.Fatal("session should be registered after char creation")
	}

	// Abruptly close the client socket without sending quit.
	_ = client.Close()

	// Wait for writePump/readPump to notice and clean up.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := m.GetSession("WritePumpGhost"); !ok {
			return // cleaned up
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("session should have been unregistered after client disconnect")
}
