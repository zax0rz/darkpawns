package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFumeRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["fume"]
	if !ok {
		t.Fatal("fume command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("fume gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["fume"]
	if !ok {
		t.Fatal("fume social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != combat.PosResting {
		t.Fatalf("fume social metadata = hide %d, min-level %d; want hide %d and min-level 1", social.HideFlag, social.MinLevel, combat.PosResting)
	}
}
