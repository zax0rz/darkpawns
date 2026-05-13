package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newMessageTestWorld builds a minimal World with two rooms for message tests.
func newMessageTestWorld(t *testing.T) (*World, []*Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1},
			{VNum: 1002, Name: "Room B", Zone: 1},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	p1 := NewPlayer(1, "Player1", 1001)
	p2 := NewPlayer(2, "Player2", 1001)
	p3 := NewPlayer(3, "Player3", 1002) // different room

	if err := w.AddPlayer(p1); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := w.AddPlayer(p2); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}
	if err := w.AddPlayer(p3); err != nil {
		t.Fatalf("AddPlayer p3 failed: %v", err)
	}

	t.Cleanup(func() {
		w.StopAITicker()
	})

	return w, []*Player{p1, p2, p3}
}

// ---------------------------------------------------------------------------
// TestSendMessage_WithSink — MessageSink wired, verify message delivered
// ---------------------------------------------------------------------------

func TestSendMessage_WithSink(t *testing.T) {
	w, players := newMessageTestWorld(t)

	var captured []struct {
		player string
		msg    string
	}

	w.MessageSink = func(playerName string, msg []byte) {
		captured = append(captured, struct {
			player string
			msg    string
		}{playerName, string(msg)})
	}

	players[0].SendMessage("hello there")

	if len(captured) != 1 {
		t.Fatalf("expected 1 message, got %d", len(captured))
	}
	if captured[0].player != "Player1" {
		t.Errorf("message sent to %q, want %q", captured[0].player, "Player1")
	}
	if captured[0].msg != "hello there" {
		t.Errorf("message content = %q, want %q", captured[0].msg, "hello there")
	}
}

// ---------------------------------------------------------------------------
// TestSendMessage_NoSink — MessageSink nil, verify no panic
// ---------------------------------------------------------------------------

func TestSendMessage_NoSink(t *testing.T) {
	_, players := newMessageTestWorld(t)

	// MessageSink is nil (default)
	// Should not panic
	players[0].SendMessage("this should be silently dropped")
	// If we got here without panic, the test passes
}

// ---------------------------------------------------------------------------
// TestRoomMessage — message sent to all players in room
// ---------------------------------------------------------------------------

func TestRoomMessage(t *testing.T) {
	w, _ := newMessageTestWorld(t)

	var received []string
	w.MessageSink = func(playerName string, msg []byte) {
		received = append(received, playerName)
	}

	// Send room message to room 1001 (Player1 and Player2)
	w.roomMessage(1001, "A loud noise echoes through the room.")

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received))
	}

	// Both Player1 and Player2 should receive it
	found := make(map[string]bool)
	for _, name := range received {
		found[name] = true
	}
	if !found["Player1"] {
		t.Error("Player1 did not receive room message")
	}
	if !found["Player2"] {
		t.Error("Player2 did not receive room message")
	}
}

// ---------------------------------------------------------------------------
// TestRoomMessageExclude — excluded player doesn't receive message
// ---------------------------------------------------------------------------

func TestRoomMessageExclude(t *testing.T) {
	w, _ := newMessageTestWorld(t)

	var received []string
	w.MessageSink = func(playerName string, msg []byte) {
		received = append(received, playerName)
	}

	// Send room message excluding Player1 (use the two-exclude variant with empty second)
	w.roomMessageExcludeTwo(1001, "You feel a chill.", "Player1", "")

	if len(received) != 1 {
		t.Fatalf("expected 1 message (Player2 only), got %d", len(received))
	}
	if received[0] != "Player2" {
		t.Errorf("message sent to %q, want %q", received[0], "Player2")
	}
}

// ---------------------------------------------------------------------------
// TestRoomMessageExcludeTwo — two excluded players don't receive message
// ---------------------------------------------------------------------------

func TestRoomMessageExcludeTwo(t *testing.T) {
	w, _ := newMessageTestWorld(t)

	var received []string
	w.MessageSink = func(playerName string, msg []byte) {
		received = append(received, playerName)
	}

	// Send room message excluding Player1 and Player2
	w.roomMessageExcludeTwo(1001, "You whisper something.", "Player1", "Player2")

	// No one in room 1001 should receive it (both excluded)
	if len(received) != 0 {
		t.Errorf("expected 0 messages, got %d: %v", len(received), received)
	}
}

// ---------------------------------------------------------------------------
// TestRoomMessage_WrongRoom — message not sent to players in other rooms
// ---------------------------------------------------------------------------

func TestRoomMessage_WrongRoom(t *testing.T) {
	w, players := newMessageTestWorld(t)

	var received []string
	w.MessageSink = func(playerName string, msg []byte) {
		received = append(received, playerName)
	}

	// Send room message to room 1002 (only Player3)
	w.roomMessage(1002, "Something stirs in the darkness.")

	if len(received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received))
	}
	if received[0] != "Player3" {
		t.Errorf("message sent to %q, want %q", received[0], "Player3")
	}

	// Verify Player1 and Player2 did NOT receive it
	for _, name := range received {
		if name == "Player1" || name == "Player2" {
			t.Errorf("player in wrong room received message: %s", name)
		}
	}

	_ = players // suppress unused
}

// ---------------------------------------------------------------------------
// TestSendToChar — sendToChar delivers message via SendMessage
// ---------------------------------------------------------------------------

func TestSendToChar(t *testing.T) {
	w, players := newMessageTestWorld(t)

	var capturedMsg string
	w.MessageSink = func(playerName string, msg []byte) {
		if playerName == "Player1" {
			capturedMsg = string(msg)
		}
	}

	sendToChar(players[0], "You feel refreshed.")

	if capturedMsg != "You feel refreshed.\r\n" {
		t.Errorf("captured message = %q, want %q", capturedMsg, "You feel refreshed.\r\n")
	}
}
