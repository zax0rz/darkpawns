package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSmileRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["smile"]
	if !ok {
		t.Fatal("smile command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("smile gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["smile"]
	if !ok {
		t.Fatal("smile social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("smile social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You smile happily.",
		"$n smiles happily.",
		"You smile at $M.",
		"$n beams a smile at $N.",
		"$n smiles at you.",
		"There's no one by that name around.",
		"You smile at yourself.",
		"$n smiles at $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("smile social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("smile social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
