package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeLookTestManager builds a Manager with a richer test World:
// Room 1001 (lit) + Room 1002 (dark), one mob proto, one obj proto.
func makeLookTestManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        1000,
				Name:        "Entry Hall",
				Description: "You are in the entry hall of the test world.",
				Zone:        1,
				Flags:       []string{"0", "0", "0", "0"}, // lit
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1001},
				},
			},
			{
				VNum:        1001,
				Name:        "Room A",
				Description: "A plain room for testing.",
				Zone:        1,
				Flags:       []string{"0", "0", "0", "0"}, // lit
				Exits: map[string]parser.Exit{
					"south": {Direction: "south", ToRoom: 1000},
				},
			},
			{
				VNum:        1002,
				Name:        "Deep Cave",
				Description: "A pitch-black cave with no light.",
				Zone:        1,
				Flags:       []string{"1", "0", "0", "0"}, // ROOM_DARK (bit 0)
			},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1,
				ShortDesc: "a test goblin",
				LongDesc:  "A small test goblin stands here, looking confused.\n",
				Keywords:  "goblin test",
			},
		},
		Objs: []parser.Obj{
			{
				VNum:      10,
				ShortDesc: "a shiny bronze coin",
				LongDesc:  "A shiny bronze coin lies on the ground.",
				Keywords:  "coin bronze shiny",
			},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return NewManager(w, nil) // nil database is fine for tests
}

// readMsgState reads one JSON message from s.send and returns it as StateData.
// Fails the test if no message arrives within the timeout.
func readMsgState(t *testing.T, s *Session) StateData {
	t.Helper()
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case msg := <-s.send:
			var sm ServerMessage
			if err := json.Unmarshal(msg, &sm); err != nil {
				t.Fatalf("failed to unmarshal ServerMessage: %v", err)
			}
			if sm.Type != MsgState {
				continue
			}
			data, err := json.Marshal(sm.Data)
			if err != nil {
				t.Fatalf("failed to re-marshal Data: %v", err)
			}
			var state StateData
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatalf("failed to unmarshal StateData: %v", err)
			}
			return state
		case <-deadline:
			t.Fatal("timeout waiting for state message on s.send")
		}
	}
}

// readMsgText reads one JSON message from s.send and returns it as a text string.
// Fails the test if no message arrives within the timeout.
func readMsgText(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case msg := <-s.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatalf("failed to unmarshal ServerMessage: %v", err)
		}
		data, err := json.Marshal(sm.Data)
		if err != nil {
			t.Fatalf("failed to re-marshal Data: %v", err)
		}
		switch sm.Type {
		case MsgText:
			var td TextData
			if err := json.Unmarshal(data, &td); err != nil {
				t.Fatalf("failed to unmarshal TextData: %v", err)
			}
			return td.Text
		case MsgEvent:
			var event EventData
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatalf("failed to unmarshal EventData: %v", err)
			}
			return event.Text
		default:
			t.Fatalf("expected text/event message, got %q", sm.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for text message on s.send")
	}
	panic("unreachable")
}

// ---------------------------------------------------------------------------
// TestLook_RoomDescription
// ---------------------------------------------------------------------------

func TestLook_RoomDescription(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if state.Room.VNum != 1001 {
		t.Errorf("expected room 1001, got %d", state.Room.VNum)
	}
	if state.Room.Name != "Room A" {
		t.Errorf("expected room name 'Room A', got %q", state.Room.Name)
	}
	if state.Room.Description != "A plain room for testing." {
		t.Errorf("unexpected description: %q", state.Room.Description)
	}
}

func TestMovementLookHonorsBriefWhileExplicitLookDoesNot(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.SetPlrFlag(game.PrfBrief, true)

	if err := cmdMovementLook(s); err != nil {
		t.Fatalf("cmdMovementLook returned error: %v", err)
	}
	if state := readMsgState(t, s); state.Room.Description != "" {
		t.Fatalf("movement room description = %q, want hidden in brief mode", state.Room.Description)
	}

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}
	if state := readMsgState(t, s); state.Room.Description != "A plain room for testing." {
		t.Fatalf("explicit look description = %q, want full room text", state.Room.Description)
	}
}

// ---------------------------------------------------------------------------
// TestLook_Exits
// ---------------------------------------------------------------------------

func TestLook_Exits(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1000, true) // room 1000 has north exit

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if len(state.Room.Exits) == 0 {
		t.Fatal("expected at least one exit")
	}
	found := false
	for _, exit := range state.Room.Exits {
		if exit == "north" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exit 'north' in %v", state.Room.Exits)
	}
}

// ---------------------------------------------------------------------------
// TestLook_EmptyRoom
// ---------------------------------------------------------------------------

func TestLook_EmptyRoom(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if len(state.Room.Players) != 0 {
		t.Errorf("expected 0 other players, got %d", len(state.Room.Players))
	}
	if len(state.Room.Mobs) != 0 {
		t.Errorf("expected 0 mobs, got %d", len(state.Room.Mobs))
	}
	if len(state.Room.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(state.Room.Items))
	}
}

// ---------------------------------------------------------------------------
// TestLook_DarkRoom
// ---------------------------------------------------------------------------

func TestLook_DarkRoom(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1002, true) // dark room
	m.mu.Lock()
	m.sessions["Alice"] = s
	m.mu.Unlock()
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	text := readMsgText(t, s)
	if text != "Darkness\r\n\r\nIt is too dark here to see much of anything...\r\n" {
		t.Errorf("unexpected C-faithful darkness text: %q", text)
	}
}

// ---------------------------------------------------------------------------
// TestLook_DarkRoomWithLight
// ---------------------------------------------------------------------------

func TestLook_DarkRoomWithLight(t *testing.T) {
	m := makeLookTestManager(t)
	s := makeTestSession(t, m, "Alice", 1002, true) // dark room
	s.player.SetLevel(31)                           // immortal → can see in dark

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if state.Room.VNum != 1002 {
		t.Errorf("expected room 1002, got %d", state.Room.VNum)
	}
	if state.Room.Name != "Deep Cave" {
		t.Errorf("expected room 'Deep Cave', got %q", state.Room.Name)
	}
}

// ---------------------------------------------------------------------------
// TestLook_PlayerInRoom
// ---------------------------------------------------------------------------

func TestLook_PlayerInRoom(t *testing.T) {
	m := makeLookTestManager(t)

	s1 := makeTestSession(t, m, "Alice", 1001, true)
	s2 := makeTestSession(t, m, "Bob", 1001, true)

	// Register both so the world can find them
	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.mu.Unlock()

	m.world.AddPlayer(s1.player)
	m.world.AddPlayer(s2.player)

	if err := cmdLook(s1, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s1)
	found := false
	for _, p := range state.Room.Players {
		if p == "Bob" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Bob' in room players, got %v", state.Room.Players)
	}
	// Alice should NOT be listed (self excluded)
	for _, p := range state.Room.Players {
		if p == "Alice" {
			t.Errorf("Alice should not appear in her own room player list")
		}
	}
}

// ---------------------------------------------------------------------------
// TestLook_MobInRoom
// ---------------------------------------------------------------------------

func TestLook_MobInRoom(t *testing.T) {
	m := makeLookTestManager(t)
	// Spawn the test goblin in room 1001
	mob, err := m.world.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	defer m.world.ExtractMob(mob)

	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if len(state.Room.Mobs) == 0 {
		t.Fatal("expected at least one mob in room")
	}
	found := false
	for _, m := range state.Room.Mobs {
		if len(m) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected non-empty mob description, got %v", state.Room.Mobs)
	}
}

// ---------------------------------------------------------------------------
// TestLook_ObjectInRoom
// ---------------------------------------------------------------------------

func TestLook_ObjectInRoom(t *testing.T) {
	m := makeLookTestManager(t)
	// Spawn the coin in room 1001
	obj, err := m.world.SpawnObject(10, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}
	m.world.AddItemToRoom(obj, 1001)

	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdLook(s, nil); err != nil {
		t.Fatalf("cmdLook returned error: %v", err)
	}

	state := readMsgState(t, s)
	if len(state.Room.Items) == 0 {
		t.Fatal("expected at least one item in room")
	}
	found := false
	for _, item := range state.Room.Items {
		if len(item) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected non-empty item description, got %v", state.Room.Items)
	}
}
