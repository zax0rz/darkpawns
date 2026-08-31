package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFwapRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["fwap"]
	if !ok {
		t.Fatal("fwap command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("fwap gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["fwap"]
	if !ok {
		t.Fatal("fwap social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("fwap social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
}
