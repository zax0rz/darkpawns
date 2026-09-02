package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestAnguishRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["anguish"]
	if !ok {
		t.Fatal("anguish command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("anguish gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["anguish"]
	if !ok {
		t.Fatal("anguish social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("anguish social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 3 {
		t.Fatalf("anguish social has %d messages, want 3", len(social.Messages))
	}
}
