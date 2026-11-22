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
