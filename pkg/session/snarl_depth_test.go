package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSnarlRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["snarl"]
	if !ok {
		t.Fatal("snarl command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("snarl gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["snarl"]
	if !ok {
		t.Fatal("snarl social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("snarl social metadata = level %d, hide %d, victim-position %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You snarl like a vicious animal.",
		"$n snarls like a cornered, vicious animal.",
		"You snarl at $M angrily.  Control yourself!",
		"$n snarls angrily at $N.  $e seems incapable of controlling $mself.",
		"$n snarls viciously at you.  $s self-control seems to have gone bananas.",
		"Eh?  Who?  Not here, my friend.",
		"You snarl at yourself, obviously suffering from schizophrenia.",
		"$n snarls at $mself, and suddenly looks very frightened.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("snarl social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("snarl social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
