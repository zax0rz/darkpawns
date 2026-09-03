package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSneerRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["sneer"]
	if !ok {
		t.Fatal("sneer command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("sneer gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["sneer"]
	if !ok {
		t.Fatal("sneer social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("sneer social metadata = level %d, hide %d, victim-position %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You curl your lip in a sneer.",
		"$n sneers.",
		"You sneer at $N.",
		"$n sneers at $N.",
		"$n sneers at you.",
		"Sneer at whom?",
		"You curl your lip in a sneer.",
		"$n sneers.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("sneer social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("sneer social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
