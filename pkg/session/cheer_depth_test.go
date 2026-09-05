package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCheerRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["cheer"]
	if !ok {
		t.Fatal("cheer command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("cheer gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["cheer"]
	if !ok {
		t.Fatal("cheer social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("cheer social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You cheer wildly!",
		"$n cheers wildly for the home team!",
		"You cheer $N on.. GO $N! GO $N! GO $N!!",
		"$n cheers $N on.. GO $N! GO $N! GO $N!",
		"$n cheers you on. You RULE in $s eyes!",
		"You cheer wildly for someone who has already left.",
		"You cheer yourself on to greater conquests!",
		"$n thinks $e is hot shit!",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("cheer social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("cheer social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
