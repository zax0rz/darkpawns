package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHugRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["hug"]
	if !ok {
		t.Fatal("hug command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("hug gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["hug"]
	if !ok {
		t.Fatal("hug social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("hug social metadata = hide %d, victim-position %d, override %d; want hide 1, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Hug who?",
		"#",
		"You hug $M.",
		"$n hugs $N.",
		"$n hugs you.",
		"Sorry, friend, I can't see that person here.",
		"You hug yourself.",
		"$n hugs $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("hug social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("hug social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
