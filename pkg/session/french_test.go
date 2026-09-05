package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFrenchRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["french"]
	if !ok {
		t.Fatal("french command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("french gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["french"]
	if !ok {
		t.Fatal("french social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("french social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
}
