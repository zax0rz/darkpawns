package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPissRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["piss"]
	if !ok {
		t.Fatal("piss command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("piss gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["piss"]
	if !ok {
		t.Fatal("piss social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("piss social metadata = level %d, hide %d, victim-position %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You take a piss in the corner.",
		"$n takes a piss.",
		"You piss on $M.",
		"$n lifts up $s hind leg and pisses on $N.",
		"$n lifts up $s hind leg and pisses on you.",
		"There's no one by that name around.",
		"You piss on yourself.",
		"$n pisses all over $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("piss social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("piss social message %d = %q, want %q", i, message, want[i])
		}
	}
}
