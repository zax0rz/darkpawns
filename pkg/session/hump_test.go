package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHumpRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["hump"]
	if !ok {
		t.Fatal("hump command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("hump gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["hump"]
	if !ok {
		t.Fatal("hump social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("hump social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You hump around.",
		"$n humps around.",
		"You hump $N's leg.",
		"$n humps $N's leg.",
		"$n humps your leg.",
		"Keep it in your pants, that person isn't here.",
		"You hump your own leg, how talented you are.",
		"$n humps all over $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("hump social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("hump social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
