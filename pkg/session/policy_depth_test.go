package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPolicyRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["policy"]
	if !ok {
		t.Fatal("policy command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("policy gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("policy")
	if !ok {
		t.Fatal("policy command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("policy registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
