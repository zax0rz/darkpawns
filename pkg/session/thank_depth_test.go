package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThankRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["thank"]
	if !ok {
		t.Fatal("thank command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("thank gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["thank"]
	if !ok {
		t.Fatal("thank social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("thank social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Thank you too.",
		"#",
		"You thank $N heartily.",
		"$n thanks $N heartily.",
		"$n thanks you heartily.",
		"No one answers to that name here.",
		"You thank yourself since nobody else wants to!",
		"$n thanks $mself since you won't.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("thank social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("thank social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
