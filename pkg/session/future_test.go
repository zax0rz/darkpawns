package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFutureRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["future"]
	if !ok {
		t.Fatal("future command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("future gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}
}
