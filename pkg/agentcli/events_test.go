package agentcli

import (
	"encoding/json"
	"strings"
	"testing"
)

// newTestEventBuffer returns an EventBuffer backed by a temp HOME so it does
// not touch the real ~/.dp-goat directory.
func newTestEventBuffer(t *testing.T, name string) *EventBuffer {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	eb, err := NewEventBuffer(name)
	if err != nil {
		t.Fatalf("NewEventBuffer: %v", err)
	}
	return eb
}

// TestCompactionWindowStateEventRoom verifies CompactionWindow reads the room
// name from a `state` event using the nested room.name path (matching the
// GameState shape and context.go), not the flat top-level ROOM_NAME path that
// room_enter uses. Previously both types shared the flat path, so state events
// never updated lastRoom.
func TestCompactionWindowStateEventRoom(t *testing.T) {
	eb := newTestEventBuffer(t, "state-room")

	// A real state-event payload: the server serializes a GameState whose room
	// name is nested under data.room.name.
	statePayload := mustJSON(t, map[string]any{
		"room": map[string]any{
			"vnum": 3001,
			"name": "The Crossroads",
		},
		"player": map[string]any{"name": "tester", "health": 100},
	})
	if _, err := eb.Append("state", json.RawMessage(statePayload)); err != nil {
		t.Fatalf("Append state: %v", err)
	}

	summary := eb.CompactionWindow(0)

	// The room name must surface in the summary. With the old flat ROOM_NAME
	// path this never matched, so the summary omitted the location.
	if want := "The Crossroads"; !strings.Contains(summary, want) {
		t.Fatalf("CompactionWindow summary %q does not mention room %q", summary, want)
	}
}

// TestCompactionWindowRoomEnterEventRoom confirms the room_enter path (flat
// ROOM_NAME) still works after splitting the two cases.
func TestCompactionWindowRoomEnterEventRoom(t *testing.T) {
	eb := newTestEventBuffer(t, "room-enter")

	enterPayload := mustJSON(t, map[string]any{
		"ROOM_NAME": "The Sewers",
		"ROOM_VNUM": 5042,
	})
	if _, err := eb.Append("room_enter", json.RawMessage(enterPayload)); err != nil {
		t.Fatalf("Append room_enter: %v", err)
	}

	summary := eb.CompactionWindow(0)
	if want := "The Sewers"; !strings.Contains(summary, want) {
		t.Fatalf("CompactionWindow summary %q does not mention room %q", summary, want)
	}
}

// TestCompactionWindowPrefersLatestRoom checks that when both a room_enter and
// a later state event arrive, the summary reflects the most recent location.
func TestCompactionWindowPrefersLatestRoom(t *testing.T) {
	eb := newTestEventBuffer(t, "latest-room")

	enterPayload := mustJSON(t, map[string]any{"ROOM_NAME": "Old Town"})
	if _, err := eb.Append("room_enter", json.RawMessage(enterPayload)); err != nil {
		t.Fatalf("Append room_enter: %v", err)
	}

	statePayload := mustJSON(t, map[string]any{
		"room": map[string]any{"vnum": 3002, "name": "New Town"},
	})
	if _, err := eb.Append("state", json.RawMessage(statePayload)); err != nil {
		t.Fatalf("Append state: %v", err)
	}

	summary := eb.CompactionWindow(0)
	if want := "New Town"; !strings.Contains(summary, want) {
		t.Fatalf("CompactionWindow summary %q does not mention latest room %q", summary, want)
	}
}

// TestCompactionWindowEmpty confirms the no-events message.
func TestCompactionWindowEmpty(t *testing.T) {
	eb := newTestEventBuffer(t, "empty")
	summary := eb.CompactionWindow(0)
	if summary == "" {
		t.Fatalf("expected a non-empty summary for zero events")
	}
}

// TestHandleEventPreservesTypeWhenDataUnexpected verifies that handleEvent
// falls back to the envelope's type when the event data payload has
// unexpected structure (plain string, array, or malformed JSON) or lacks an
// inner type field, so events are never dropped or stored with an empty type.
func TestHandleEventPreservesTypeWhenDataUnexpected(t *testing.T) {
	d := &Daemon{events: newTestEventBuffer(t, "handle-event")}

	cases := []struct {
		name     string
		data     json.RawMessage
		wantType string
	}{
		{name: "structured with type", data: json.RawMessage(`{"type":"tell","text":"hi"}`), wantType: "tell"},
		{name: "structured without type", data: json.RawMessage(`{"message":"hi"}`), wantType: "event"},
		{name: "plain string", data: json.RawMessage(`"hello world"`), wantType: "event"},
		{name: "array payload", data: json.RawMessage(`[1,2,3]`), wantType: "event"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := d.events.Since(0)
			if err := d.handleEvent("event", tc.data); err != nil {
				t.Fatalf("handleEvent: %v", err)
			}
			after := d.events.Since(0)
			if len(after) != len(before)+1 {
				t.Fatalf("expected event to be appended, before=%d after=%d", len(before), len(after))
			}
			ev := after[len(after)-1]
			if ev.Type != tc.wantType {
				t.Fatalf("event type = %q, want %q", ev.Type, tc.wantType)
			}
		})
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
