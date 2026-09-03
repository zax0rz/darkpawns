package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestRetsuRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["retsu"]
	if !ok {
		t.Fatal("retsu command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosStanding {
		t.Fatalf("retsu gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosStanding)
	}

	entry, ok := cmdRegistry.Lookup("retsu")
	if !ok {
		t.Fatal("retsu command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("retsu registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
