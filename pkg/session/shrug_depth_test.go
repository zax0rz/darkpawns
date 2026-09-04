package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShrugRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["shrug"]
	if !ok {
		t.Fatal("shrug command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shrug gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shrug"]
	if !ok {
		t.Fatal("shrug social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("shrug social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You shrug.",
		"$n shrugs.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("shrug social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("shrug social message %d = %q, want %q", i, message, want[i])
		}
	}
}
