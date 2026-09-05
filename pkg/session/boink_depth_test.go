package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBoinkRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["boink"]
	if !ok {
		t.Fatal("boink command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("boink gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["boink"]
	if !ok {
		t.Fatal("boink social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("boink social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	wantMessages := []string{
		"Who do you want to boink?",
		"$n looks around, puzzled.",
		"You boink $M.",
		"$n boinks $N with ultimate passion.",
		"$n boinks you passionately.",
		"Who's that?",
		"You boink yourself on the head.",
		"$n boinks $mself on the head.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("boink social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("boink social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
