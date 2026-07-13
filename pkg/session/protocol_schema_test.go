package session

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestStateDataSchemaGolden freezes the WebSocket state payload before the
// observation Result-DTO migration. Existing fields may only change additively:
// renaming, removing, or retyping any representative field changes this JSON.
func TestStateDataSchemaGolden(t *testing.T) {
	t.Parallel()

	state := StateData{
		Player: PlayerState{
			Name:      "Schema Keeper",
			Health:    101,
			MaxHealth: 202,
			Mana:      303,
			MaxMana:   404,
			Move:      505,
			MaxMove:   606,
			Gold:      707,
			Level:     8,
			Class:     "Warrior",
			Race:      "Human",
			Str:       11,
			Int:       12,
			Wis:       13,
			Dex:       14,
			Con:       15,
			Cha:       16,
		},
		Room: RoomState{
			VNum:        8004,
			Name:        "At the Temple Altar",
			Description: "A representative room.",
			Exits:       []string{"north", "east"},
			Doors: []DoorInfo{{
				Direction: "east",
				Closed:    true,
				Locked:    false,
			}},
			Players: []string{"Another Player"},
			Mobs:    []string{"a temple guardian is here."},
			Items:   []string{"A brass lantern rests here."},
		},
		Token: "schema-token",
	}

	got, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal StateData: %v", err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile(filepath.Join("testdata", "state_data.golden.json"))
	if err != nil {
		t.Fatalf("read StateData golden: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("StateData WebSocket schema changed\n--- want\n%s--- got\n%s", want, got)
	}
}
