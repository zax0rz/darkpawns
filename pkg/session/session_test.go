package session

import (
	"net/http"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeTestManager builds a minimal Manager with a test World (2 rooms).
func makeTestManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1},
			{VNum: 1002, Name: "Room B", Zone: 1},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return NewManager(w, nil) // nil database is fine for tests
}

// makeTestSession creates a minimal Session with a mock WebSocket connection.
func makeTestSession(t *testing.T, m *Manager, name string, roomVNum int, authenticated bool) *Session {
	t.Helper()

	s := &Session{
		conn:           nil, // nil conn is fine for manager-level tests
		request:        &http.Request{},
		manager:        m,
		send:           make(chan []byte, 256),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
	}
	player := game.NewPlayer(1, name, roomVNum)
	s.player = player
	s.playerName = name
	s.authenticated = authenticated
	return s
}

// ---------------------------------------------------------------------------
// TestManager_GetSession — create session, lookup by name, verify found
// ---------------------------------------------------------------------------

func TestManager_GetSession(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Register the session
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	found, ok := m.GetSession("alice")
	if !ok {
		t.Fatal("GetSession should find alice")
	}
	if found != s {
		t.Error("GetSession returned wrong session")
	}
}

// ---------------------------------------------------------------------------
// TestManager_GetSession_NotFound — lookup nonexistent name
// ---------------------------------------------------------------------------

func TestManager_GetSession_NotFound(t *testing.T) {
	m := makeTestManager(t)

	found, ok := m.GetSession("nobody")
	if ok {
		t.Error("GetSession should not find nonexistent player")
	}
	if found != nil {
		t.Error("GetSession should return nil for nonexistent player")
	}
}

// ---------------------------------------------------------------------------
// TestManager_SendToAll — broadcast to all authenticated sessions
// ---------------------------------------------------------------------------

func TestManager_SendToAll(t *testing.T) {
	m := makeTestManager(t)

	s1 := makeTestSession(t, m, "Alice", 1001, true)
	s2 := makeTestSession(t, m, "Bob", 1001, true)
	s3 := makeTestSession(t, m, "Unauth", 1001, false) // not authenticated

	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.sessions["unauth"] = s3
	m.mu.Unlock()

	m.SendToAll("Hello everyone!")

	// Both authenticated sessions should receive a message
	for _, name := range []string{"alice", "bob"} {
		select {
		case <-m.sessions[name].send:
			// received
		case <-time.After(100 * time.Millisecond):
			t.Errorf("session %s did not receive broadcast", name)
		}
	}

	// Unauthenticated session should NOT receive
	select {
	case <-m.sessions["unauth"].send:
		t.Error("unauthenticated session should not receive broadcast")
	default:
		// expected — no message
	}
}

// ---------------------------------------------------------------------------
// TestManager_BroadcastToRoom — only room occupants get the message
// ---------------------------------------------------------------------------

func TestManager_BroadcastToRoom(t *testing.T) {
	m := makeTestManager(t)

	s1 := makeTestSession(t, m, "Alice", 1001, true) // room 1001
	s2 := makeTestSession(t, m, "Bob", 1001, true)   // room 1001
	s3 := makeTestSession(t, m, "Carol", 1002, true)  // room 1002

	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.sessions["carol"] = s3
	m.mu.Unlock()

	msg := []byte(`{"type":"event","data":{"type":"test","text":"room message"}}`)
	m.BroadcastToRoom(1001, msg, "")

	// Alice and Bob should receive
	for _, name := range []string{"alice", "bob"} {
		select {
		case <-m.sessions[name].send:
			// received
		case <-time.After(100 * time.Millisecond):
			t.Errorf("session %s in room 1001 did not receive message", name)
		}
	}

	// Carol (room 1002) should NOT receive
	select {
	case <-m.sessions["carol"].send:
		t.Error("session in different room should not receive message")
	default:
		// expected
	}
}

// ---------------------------------------------------------------------------
// TestManager_BroadcastToRoom_Exclude — excluded player doesn't get message
// ---------------------------------------------------------------------------

func TestManager_BroadcastToRoom_Exclude(t *testing.T) {
	m := makeTestManager(t)

	s1 := makeTestSession(t, m, "Alice", 1001, true)
	s2 := makeTestSession(t, m, "Bob", 1001, true)

	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.mu.Unlock()

	msg := []byte(`{"type":"event","data":{"type":"test","text":"whisper"}}`)
	m.BroadcastToRoom(1001, msg, "alice")

	// Bob should receive, Alice should not
	select {
	case <-m.sessions["bob"].send:
		// received
	case <-time.After(100 * time.Millisecond):
		t.Error("Bob should have received the message")
	}

	select {
	case <-m.sessions["alice"].send:
		t.Error("Alice was excluded but still received message")
	default:
		// expected
	}
}

// ---------------------------------------------------------------------------
// TestSetCommandExecFunc — wire callback, verify World.CommandExecFunc is set
// ---------------------------------------------------------------------------

func TestSetCommandExecFunc(t *testing.T) {
	m := makeTestManager(t)

	// Before wiring, CommandExecFunc should be nil
	if m.world.CommandExecFunc != nil {
		t.Error("CommandExecFunc should be nil before wiring")
	}

	m.SetCommandExecFunc()

	if m.world.CommandExecFunc == nil {
		t.Fatal("CommandExecFunc should be set after SetCommandExecFunc")
	}

	// Test that the callback delegates to ExecuteCommand
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	// Wire the callback — it should find Alice's session and dispatch
	called := false
	m.world.CommandExecFunc = func(ch *game.Player, command string) bool {
		called = true
		// The callback should be called with the correct player
		if ch.GetName() != "Alice" {
			t.Errorf("callback received player %q, want %q", ch.GetName(), "Alice")
		}
		return true
	}

	result := m.world.CommandExecFunc(game.NewPlayer(1, "Alice", 1001), "look")
	if !result {
		t.Error("callback should return true")
	}
	if !called {
		t.Error("callback was not called")
	}
}

// ---------------------------------------------------------------------------
// TestManager_SendToAll_EmptyMessage — empty message should be ignored
// ---------------------------------------------------------------------------

func TestManager_SendToAll_EmptyMessage(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	m.SendToAll("")

	// Should not receive anything
	select {
	case <-m.sessions["alice"].send:
		t.Error("empty message should not be sent")
	default:
		// expected
	}
}

// ---------------------------------------------------------------------------
// TestManager_GetSession_Lookup — find by lowercase key
// ---------------------------------------------------------------------------

func TestManager_GetSession_Lookup(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	// GetSession is case-sensitive (keys are lowercased on registration)
	found, ok := m.GetSession("alice")
	if !ok {
		t.Error("GetSession should find alice")
	}
	if found != s {
		t.Error("GetSession returned wrong session")
	}
}

// ---------------------------------------------------------------------------
// TestManager_BroadcastToRoom_EmptyRoom — no one in the room, no crash
// ---------------------------------------------------------------------------

func TestManager_BroadcastToRoom_EmptyRoom(t *testing.T) {
	m := makeTestManager(t)

	// No sessions at all
	msg := []byte(`{"type":"event","data":{"type":"test","text":"hello"}}`)
	m.BroadcastToRoom(9999, msg, "") // room with no one
	// Should not panic
}
