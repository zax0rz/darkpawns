package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGagRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["gag"]
	if !ok {
		t.Fatal("gag command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("gag gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["gag"]
	if !ok {
		t.Fatal("gag social is not registered")
	}
	if social.HideFlag != combat.PosResting || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("gag social metadata = hide %d, min-level %d, min-victim %d; want C hide 0, victim position %d, and no override", social.HideFlag, social.MinLevel, social.MinVictimPosition, combat.PosResting)
	}
}
