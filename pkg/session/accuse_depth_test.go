package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestAccuseRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["accuse"]
	if !ok {
		t.Fatal("accuse command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosSitting {
		t.Fatalf("accuse gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosSitting)
	}

	social, ok := game.Socials["accuse"]
	if !ok {
		t.Fatal("accuse social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("accuse social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position %d, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("accuse social has %d messages, want 8", len(social.Messages))
	}
}
