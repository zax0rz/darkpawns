package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestInactiveRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["inactive"]
	if !ok {
		t.Fatal("inactive command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosSleeping {
		t.Fatalf("inactive gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosSleeping)
	}

	entry, ok := cmdRegistry.Lookup("inactive")
	if !ok {
		t.Fatal("inactive command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("inactive registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
