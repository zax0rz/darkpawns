package agentcli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func makeTestGameState() *GameState {
	s := &GameState{}
	s.Player.Name = "TestBot"
	s.Player.Health = 100
	s.Player.MaxHealth = 100
	s.Player.Mana = 50
	s.Player.Level = 5
	s.Player.Exp = 1200
	s.Player.Gold = 250
	s.Room.Vnum = 1001
	s.Room.Name = "The Temple"
	s.Room.Exits = []string{"north", "south"}
	s.Room.Mobs = []Mob{{Name: "A blind monk", TargetString: "monk", Fighting: true}}
	return s
}

func makeTestEventBuffer(t *testing.T, name string) *EventBuffer {
	t.Helper()
	eb, err := NewEventBuffer(name)
	if err != nil {
		t.Fatalf("NewEventBuffer: %v", err)
	}
	t.Cleanup(func() { eb.Truncate() })
	return eb
}

func TestCompactNarrative_NoEvents(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-empty")
	cb := NewContextBuilder()
	state := makeTestGameState()

	summary := cb.CompactNarrative(eb, 0, state)
	if summary != "Nothing happened while you were away." {
		t.Errorf("expected no-events message, got: %s", summary)
	}
}

func TestCompactNarrative_CombatStart(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-combat")
	cb := NewContextBuilder()
	state := makeTestGameState()
	// state.Fighting is empty initially — combat_start event sets the narrative

	seq := eb.NextSeq() // baseline: 1
	eb.Append("combat_start", map[string]string{"target": "a goblin"})
	state.Fighting = "a goblin" // state after event

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "You are fighting a goblin." {
		t.Errorf("expected combat message, got: %s", summary)
	}
}

func TestCompactNarrative_TellReceived(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-tell")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("tell", map[string]string{"from": "Daeron", "message": "hey there"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != `Daeron told you: "hey there"` {
		t.Errorf("expected tell message, got: %s", summary)
	}
}

func TestCompactNarrative_SayReceived(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-say")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("say", map[string]string{"from": "Brenda", "message": "hello everyone"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != `Brenda said: "hello everyone"` {
		t.Errorf("expected say message, got: %s", summary)
	}
}

func TestCompactNarrative_RoomTransition(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-move")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("vars", map[string]string{"ROOM_NAME": "The Tavern"})
	eb.Append("vars", map[string]string{"ROOM_NAME": "The Square"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "You traveled: The Tavern → The Square." {
		t.Errorf("expected travel message, got: %s", summary)
	}
}

func TestCompactNarrative_SingleRoomTransition(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-single-move")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("vars", map[string]string{"ROOM_NAME": "The Tavern"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "You moved to The Tavern." {
		t.Errorf("expected move message, got: %s", summary)
	}
}

func TestCompactNarrative_Death(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-death")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("vars", map[string]int{"HEALTH": 0})
	// Note: state.Player.Health stays 100 (the event data doesn't auto-update state)
	// But the event itself carries HEALTH=0 which CompactNarrative reads.

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "You died." {
		t.Errorf("expected death message, got: %s", summary)
	}
}

func TestCompactNarrative_MultipleErrors(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-errors")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("error", map[string]string{"message": "err1"})
	eb.Append("error", map[string]string{"message": "err2"})
	eb.Append("error", map[string]string{"message": "err3"})
	eb.Append("error", map[string]string{"message": "err4"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "[4 errors occurred]" {
		t.Errorf("expected error summary, got: %s", summary)
	}
}

func TestCompactNarrative_FewErrors(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-few-errors")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("error", map[string]string{"message": "disk full"})
	eb.Append("error", map[string]string{"message": "timeout"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	expected := "Error: disk full Error: timeout"
	if summary != expected {
		t.Errorf("expected %q, got %q", expected, summary)
	}
}

func TestCompactNarrative_CombatRounds(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-combat-rounds")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("combat_tick", map[string]int{"round": 1})
	eb.Append("combat_tick", map[string]int{"round": 2})
	eb.Append("combat_tick", map[string]int{"round": 3})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "Combat lasted 3 rounds." {
		t.Errorf("expected combat rounds message, got: %s", summary)
	}
}

func TestCompactNarrative_PlayersEntered(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-players")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("entity_enter", map[string]string{"name": "Zach", "entity_type": "player"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "Players nearby: Zach." {
		t.Errorf("expected players message, got: %s", summary)
	}
}

func TestCompactNarrative_ItemsGotten(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-items")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("item_get", map[string]string{"name": "sword"})
	eb.Append("item_get", map[string]string{"name": "shield"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary != "You picked up: sword, shield." {
		t.Errorf("expected items message, got: %s", summary)
	}
}

func TestCompactNarrative_MixedEvents(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-narrative-mixed")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("vars", map[string]string{"ROOM_NAME": "The Tavern"})
	eb.Append("tell", map[string]string{"from": "Daeron", "message": "welcome"})
	eb.Append("item_get", map[string]string{"name": "ale"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if summary == "" {
		t.Error("expected non-empty mixed narrative")
	}
	if !strings.Contains(summary, "The Tavern") {
		t.Errorf("narrative should mention the tavern, got: %s", summary)
	}
	if !strings.Contains(summary, "Daeron") {
		t.Errorf("narrative should mention Daeron, got: %s", summary)
	}
	if !strings.Contains(summary, "ale") {
		t.Errorf("narrative should mention ale, got: %s", summary)
	}
}

func TestBuildContextPacket(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-context-packet")
	cb := NewContextBuilder()
	state := makeTestGameState()

	eb.Append("tell", map[string]string{"from": "Zach", "message": "hey"})

	pkt := cb.Build(state, eb, 0)
	if pkt == nil {
		t.Fatal("expected non-nil context packet")
	}
	if pkt.State != state {
		t.Error("state mismatch")
	}
	if pkt.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if len(pkt.Events) == 0 {
		t.Error("expected non-empty events")
	}
	if pkt.GeneratedAt.IsZero() {
		t.Error("expected non-zero generated_at")
	}
}

func TestFormatContext(t *testing.T) {
	state := makeTestGameState()
	pkt := &ContextPacket{
		State:   state,
		Summary: "You moved to The Tavern. Daeron told you: \"hello\"",
	}

	formatted := FormatContext(pkt)
	if formatted == "" {
		t.Fatal("expected non-empty formatted context")
	}
	if !strings.Contains(formatted, "The Temple") {
		t.Error("formatted context should mention room name")
	}
	if !strings.Contains(formatted, "What do you do?") {
		t.Error("formatted context should end with prompt")
	}
}

func TestFormatContext_WithFighting(t *testing.T) {
	state := makeTestGameState()
	state.Fighting = "a dragon"
	pkt := &ContextPacket{
		State:   state,
		Summary: "A dragon attacked you.",
	}

	formatted := FormatContext(pkt)
	if !strings.Contains(formatted, "Fighting: a dragon") {
		t.Error("formatted context should show fighting status")
	}
}

func TestCompactNarrative_EventTypes(t *testing.T) {
	eb := makeTestEventBuffer(t, "test-event-types")
	cb := NewContextBuilder()
	state := makeTestGameState()

	seq := eb.NextSeq()
	eb.Append("item_drop", map[string]string{"name": "broken shield"})

	summary := cb.CompactNarrative(eb, seq-1, state)
	if !strings.Contains(summary, "broken shield") {
		t.Errorf("expected drop in narrative, got: %s", summary)
	}
}

func TestNewContextBuilder(t *testing.T) {
	cb := NewContextBuilder()
	if cb.maxRecentEvents != 20 {
		t.Errorf("expected maxRecentEvents=20, got %d", cb.maxRecentEvents)
	}
}

func TestContextPacketJSON(t *testing.T) {
	pkt := &ContextPacket{
		State:       makeTestGameState(),
		Summary:     "test summary",
		Events:      []AgentEvent{{Seq: 1, Type: "test"}},
		GeneratedAt: time.Now().UTC(),
		SeqFloor:    0,
	}

	data, err := json.Marshal(pkt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ContextPacket
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Summary != "test summary" {
		t.Errorf("summary mismatch: %s", decoded.Summary)
	}
}
