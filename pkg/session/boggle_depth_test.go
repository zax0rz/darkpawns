package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBoggleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["boggle"]
	if !ok {
		t.Fatal("boggle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("boggle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["boggle"]
	if !ok {
		t.Fatal("boggle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("boggle social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You boggle at the concept.",
		"$n boggles at the concept.",
		"You boggle at $M in disbelief.",
		"$n boggles at $N in disbelief.",
		"$n boggles at you in disbelief.",
		"Boggle away, they aren't here.",
		"You boggle in disbelief at yourself.",
		"$n seems to be having an inner struggle.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("boggle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("boggle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
