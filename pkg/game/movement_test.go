package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newMovementTestWorld builds a minimal World with two connected rooms for
// movement tests: room 1001 (north) ↔ room 1002 (south).
func newMovementTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Room North", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: 0},
				},
			},
			{
				VNum: 1002, Name: "Room South", Zone: 1,
				Exits: map[string]parser.Exit{
					"south": {Direction: "south", ToRoom: 1001, DoorState: 0},
				},
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	player.SetMove(100) // ensure enough movement points
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	t.Cleanup(func() {
		w.StopAITicker()
	})

	return w, player
}

// ---------------------------------------------------------------------------
// TestDoSimpleMove_ValidExit — player moves north to adjacent room
// ---------------------------------------------------------------------------

func TestDoSimpleMove_ValidExit(t *testing.T) {
	w, player := newMovementTestWorld(t)

	if player.GetRoom() != 1001 {
		t.Fatalf("player should start in room 1001, got %d", player.GetRoom())
	}

	ok := doSimpleMove(w, player, 0, false) // 0 = north
	if !ok {
		t.Fatal("doSimpleMove should succeed for valid exit")
	}

	if player.GetRoom() != 1002 {
		t.Errorf("player should be in room 1002 after moving north, got %d", player.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestDoSimpleMove_NoExit — player tries to move in direction with no exit
// ---------------------------------------------------------------------------

func TestDoSimpleMove_NoExit(t *testing.T) {
	w, player := newMovementTestWorld(t)

	// Room 1001 has no south exit
	ok := doSimpleMove(w, player, 2, false) // 2 = south
	if ok {
		t.Error("doSimpleMove should fail when no exit exists")
	}

	if player.GetRoom() != 1001 {
		t.Errorf("player should still be in room 1001, got %d", player.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestDoSimpleMove_ClosedDoor — player tries to move through closed door
// ---------------------------------------------------------------------------

func TestDoSimpleMove_ClosedDoor(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Room A", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: doorClosed, Keywords: "wooden door"},
				},
			},
			{
				VNum:  1002, Name: "Room B", Zone: 1,
				Exits: map[string]parser.Exit{},
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	player.SetMove(100)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// trySimpleMove checks door state first
	ok := performMove(w, player, 0, false) // north
	if ok {
		t.Error("performMove should fail when door is closed")
	}

	if player.GetRoom() != 1001 {
		t.Errorf("player should still be in room 1001, got %d", player.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestDoSimpleMove_LockedDoor — locked door also blocks movement
// ---------------------------------------------------------------------------

func TestDoSimpleMove_LockedDoor(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Room A", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: doorLocked, Keywords: "iron gate"},
				},
			},
			{
				VNum:  1002, Name: "Room B", Zone: 1,
				Exits: map[string]parser.Exit{},
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	player.SetMove(100)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ok := performMove(w, player, 0, false) // north
	if ok {
		t.Error("performMove should fail when door is locked")
	}

	if player.GetRoom() != 1001 {
		t.Errorf("player should still be in room 1001, got %d", player.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestPerformMove_FollowersFollow — player with followers moves, followers follow
// ---------------------------------------------------------------------------

func TestPerformMove_FollowersFollow(t *testing.T) {
	w, leader := newMovementTestWorld(t)

	// Create a follower
	follower := NewPlayer(2, "Follower", 1001)
	follower.SetMove(100)
	follower.Following = leader.Name
	if err := w.AddPlayer(follower); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Both in room 1001
	if leader.GetRoom() != 1001 || follower.GetRoom() != 1001 {
		t.Fatal("both players should start in room 1001")
	}

	ok := performMove(w, leader, 0, false) // north
	if !ok {
		t.Fatal("performMove should succeed")
	}

	if leader.GetRoom() != 1002 {
		t.Errorf("leader should be in room 1002, got %d", leader.GetRoom())
	}
	if follower.GetRoom() != 1002 {
		t.Errorf("follower should also be in room 1002, got %d", follower.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestPerformMove_Exhausted — player with no move points can't move
// ---------------------------------------------------------------------------

func TestPerformMove_Exhausted(t *testing.T) {
	w, player := newMovementTestWorld(t)

	player.SetMove(0) // no movement points

	ok := performMove(w, player, 0, false) // north
	if ok {
		t.Error("performMove should fail when player has no movement points")
	}

	if player.GetRoom() != 1001 {
		t.Errorf("player should still be in room 1001, got %d", player.GetRoom())
	}
}

// ---------------------------------------------------------------------------
// TestPerformMove_Sneak — sneaking player doesn't broadcast room message
// ---------------------------------------------------------------------------

func TestPerformMove_Sneak(t *testing.T) {
	w, player := newMovementTestWorld(t)

	// Set sneak affect
	player.SetAffect(affSneak, true)
	defer player.SetAffect(affSneak, false)

	// Capture messages
	var messages []string
	w.MessageSink = func(playerName string, msg []byte) {
		messages = append(messages, string(msg))
	}

	ok := performMove(w, player, 0, false) // north
	if !ok {
		t.Fatal("performMove should succeed with sneak")
	}

	if player.GetRoom() != 1002 {
		t.Errorf("player should be in room 1002, got %d", player.GetRoom())
	}

	// Sneaking should not produce leave/arrival room messages
	// (only the player's own movement messages are sent, not room broadcasts)
	for _, msg := range messages {
		if len(msg) > 0 && (contains(msg, "leaves") || contains(msg, "arrives")) {
			t.Errorf("sneaking produced room broadcast: %q", msg)
		}
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// TestPerformMove_TunnelFull — tunnel room with player already there blocks move
// ---------------------------------------------------------------------------

func TestPerformMove_TunnelFull(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Room A", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: 0},
				},
			},
			{
				VNum:        1002, Name: "Tunnel", Zone: 1,
				Description: "A narrow tunnel.",
				Flags:       []string{"tunnel"},
				Exits: map[string]parser.Exit{
					"south": {Direction: "south", ToRoom: 1001, DoorState: 0},
				},
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
		Zones: []parser.Zone{
			{Number: 1},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	// Room 1002 already has tunnel flag from parser.Room.Flags

	player1 := NewPlayer(1, "Player1", 1002) // already in tunnel
	player1.SetMove(100)
	if err := w.AddPlayer(player1); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player2 := NewPlayer(2, "Player2", 1001)
	player2.SetMove(100)
	if err := w.AddPlayer(player2); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	t.Cleanup(func() { w.StopAITicker() })

	// Player2 tries to move north into tunnel — should be blocked
	ok := performMove(w, player2, 0, false)
	if ok {
		t.Error("performMove should fail when tunnel is full")
	}

	if player2.GetRoom() != 1001 {
		t.Errorf("player2 should still be in room 1001, got %d", player2.GetRoom())
	}
}
