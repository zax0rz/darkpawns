package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSneakRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["sneak"]
	if !ok {
		t.Fatal("sneak command has no C gate")
	}
	if gate.MinLevel != 1 || gate.MinPosition != combat.PosStanding {
		t.Fatalf("sneak gate = level %d position %d, want level 1 position %d", gate.MinLevel, gate.MinPosition, combat.PosStanding)
	}

	entry, ok := cmdRegistry.Lookup("sneak")
	if !ok {
		t.Fatal("sneak command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("sneak registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
