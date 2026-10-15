package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newCommandTestWorld builds a minimal World with one room for command-exec tests.
// The World's MessageSink captures all sent messages for assertion.
func newCommandTestWorld(t *testing.T) (*World, *Player, *[]string) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Capture messages via MessageSink
	var captured []string
	w.MessageSink = func(playerName string, msg []byte) {
		captured = append(captured, string(msg))
	}

	t.Cleanup(func() {
		w.StopAITicker()
	})

	return w, player, &captured
}

// ---------------------------------------------------------------------------
// TestExecuteCommand_WithCallback — set CommandExecFunc, verify invocation
// ---------------------------------------------------------------------------

func TestExecuteCommand_WithCallback(t *testing.T) {
	w, player, _ := newCommandTestWorld(t)

	var calledPlayer *Player
	var calledCommand string

	w.CommandExecFunc = func(ch *Player, command string) bool {
		calledPlayer = ch
		calledCommand = command
		return true
	}

	result := w.executeCommand(player, "look")
	if !result {
		t.Error("executeCommand should return true when callback returns true")
	}
	if calledPlayer != player {
		t.Error("callback did not receive the correct player")
	}
	if calledCommand != "look" {
		t.Errorf("callback received command %q, want %q", calledCommand, "look")
	}
}

// ---------------------------------------------------------------------------
// TestExecuteCommand_NoCallback — CommandExecFunc is nil, should return false
// ---------------------------------------------------------------------------

func TestExecuteCommand_NoCallback(t *testing.T) {
	w, player, _ := newCommandTestWorld(t)

	// CommandExecFunc is nil (default)
	result := w.executeCommand(player, "look")
	if result {
		t.Error("executeCommand should return false when CommandExecFunc is nil")
	}
}

// ---------------------------------------------------------------------------
// TestExecuteCommand_CallbackReturnsFalse — propagate false from callback
// ---------------------------------------------------------------------------

func TestExecuteCommand_CallbackReturnsFalse(t *testing.T) {
	w, player, _ := newCommandTestWorld(t)

	w.CommandExecFunc = func(ch *Player, command string) bool {
		return false
	}

	result := w.executeCommand(player, "look")
	if result {
		t.Error("executeCommand should return false when callback returns false")
	}
}

// ---------------------------------------------------------------------------
// TestDoForced — doForced delegates to executeCommand
// ---------------------------------------------------------------------------

func TestDoForced(t *testing.T) {
	w, player, _ := newCommandTestWorld(t)

	var receivedCmd string
	w.CommandExecFunc = func(ch *Player, command string) bool {
		receivedCmd = command
		return true
	}

	result := w.doForced(player, "say hello")
	if !result {
		t.Error("doForced should return true when callback succeeds")
	}
	if receivedCmd != "say hello" {
		t.Errorf("doForced received command %q, want %q", receivedCmd, "say hello")
	}
}

// ---------------------------------------------------------------------------
// TestExecuteCommand_DifferentPlayers — verify correct player is passed
// ---------------------------------------------------------------------------

func TestExecuteCommand_DifferentPlayers(t *testing.T) {
	w, player1, _ := newCommandTestWorld(t)

	player2 := NewPlayer(2, "OtherPlayer", 1001)
	if err := w.AddPlayer(player2); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	var receivedNames []string
	w.CommandExecFunc = func(ch *Player, command string) bool {
		receivedNames = append(receivedNames, ch.GetName())
		return true
	}

	w.executeCommand(player1, "look")
	w.executeCommand(player2, "north")

	if len(receivedNames) != 2 {
		t.Fatalf("expected 2 callbacks, got %d", len(receivedNames))
	}
	if receivedNames[0] != "TestPlayer" {
		t.Errorf("first callback got player %q, want %q", receivedNames[0], "TestPlayer")
	}
	if receivedNames[1] != "OtherPlayer" {
		t.Errorf("second callback got player %q, want %q", receivedNames[1], "OtherPlayer")
	}
}

// TestMobCommandMovement_NoDeadlock verifies that a dispatched mobile command
// (a northward perform_move, whose transfer takes the world write lock) runs
// without deadlocking because of nested World locks. The retired
// executeMobCommand adapter's "open north" case is gone with the adapter: an
// unported command must leave the world untouched rather than run an
// imitation, so the door case now asserts no-op.
func TestMobCommandMovement_NoDeadlock(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 5001, Keywords: "test mob", ShortDesc: "a test mob"},
		},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	defer w.StopAITicker()

	// Load the mob into room 1001
	mobProto, ok := w.GetMobPrototype(5001)
	if !ok {
		t.Fatal("mob prototype not found")
	}
	mobInst := NewMob(mobProto, 1001)

	// Register active mob
	w.mu.Lock()
	mobInst.ID = w.nextMobID
	w.nextMobID++
	w.activeMobs[mobInst.ID] = mobInst
	w.mu.Unlock()

	// Capture messages to verify it doesn't hang
	w.MessageSink = func(playerName string, msg []byte) {}

	// Setup a closed door exit in room 1001 to a dummy north room
	w.mu.Lock()
	room := w.rooms[1001]
	room.Exits = map[string]parser.Exit{
		"north": {
			ToRoom:    1002,
			DoorState: 1,
			ExitInfo:  parser.ExitIsDoor | parser.ExitClosed,
			Keywords:  "door",
		},
	}
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "North Room", Exits: map[string]parser.Exit{
		"south": {
			ToRoom:    1001,
			DoorState: 1,
			ExitInfo:  parser.ExitIsDoor | parser.ExitClosed,
		},
	}}
	w.mu.Unlock()

	// An openable east exit gives the movement handler a lock-taking success
	// leg in the same dispatch: do_simple_move's transfer locks w.mu.
	w.mu.Lock()
	room.Exits["east"] = parser.Exit{ToRoom: 1003}
	w.rooms[1003] = &parser.Room{VNum: 1003, Name: "East Room"}
	w.mu.Unlock()

	// performMoveMob walks the closed-door refusal (no transfer) and the
	// open-exit transfer (w.mu.Lock inside MobTransfer) without holding a
	// reader across either.
	w.performMoveMob(mobInst, 0, false)
	if got := mobInst.GetRoom(); got != 1001 {
		t.Fatalf("closed-door leg moved the mob to %d, want 1001", got)
	}

	w.performMoveMob(mobInst, 1, false)
	if got := mobInst.GetRoom(); got != 1003 {
		t.Fatalf("east leg left the mob in %d, want 1003", got)
	}

	// The unported "open north" command no longer runs an imitation door
	// handler: the dispatcher has no actor handler for open, so the door
	// stays closed.
	if doorState := w.rooms[1001].Exits["north"].ExitInfo; doorState&parser.ExitClosed == 0 {
		t.Errorf("unported open command opened the door; want it untouched")
	}
}
