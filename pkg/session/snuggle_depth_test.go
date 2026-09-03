package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSnuggleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["snuggle"]
	if !ok {
		t.Fatal("snuggle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("snuggle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["snuggle"]
	if !ok {
		t.Fatal("snuggle social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("snuggle social metadata = hide %d, victim-position %d, override %d; want hide 1, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Who?",
		"#",
		"you snuggle $M.",
		"$n snuggles up to $N.",
		"$n snuggles up to you.",
		"They aren't here.",
		"Hmmm...",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("snuggle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("snuggle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
