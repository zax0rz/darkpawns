package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSnickerRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["snicker"]
	if !ok {
		t.Fatal("snicker command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("snicker gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["snicker"]
	if !ok {
		t.Fatal("snicker social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("snicker social metadata = level %d, hide %d, victim-position %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You snicker softly.",
		"$n snickers softly.",
		"You snicker at $N.",
		"$n snickers at $N.",
		"$n snickers at you.",
		"Who?",
		"#",
		"",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("snicker social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("snicker social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
