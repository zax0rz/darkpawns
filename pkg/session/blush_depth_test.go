package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBlushRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["blush"]
	if !ok {
		t.Fatal("blush command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("blush gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["blush"]
	if !ok {
		t.Fatal("blush social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("blush social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	wantMessages := []string{
		"Your cheeks are burning.",
		"$n blushes.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("blush social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("blush social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
