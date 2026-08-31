package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHighfiveRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["highfive"]
	if !ok {
		t.Fatal("highfive command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("highfive gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["highfive"]
	if !ok {
		t.Fatal("highfive social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("highfive social metadata = hide %d, victim-position %d, override %d; want hide 1, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You give the air some skin.",
		"$n gives everyone a highfive.",
		"You whoop in joy and give $M a highfive.",
		"$n whoops in joy and gives $N a highfive.",
		"$n whoops in joy and gives you a highfive.",
		"Sorry, friend, I can't see that person here.",
		"You highfive yourself in satisfaction.",
		"$n highfives $mself.  How strange.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("highfive social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("highfive social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
