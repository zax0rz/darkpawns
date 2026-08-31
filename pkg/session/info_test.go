package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestInfoRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["info"]
	if !ok {
		t.Fatal("info command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosSleeping {
		t.Fatalf("info gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosSleeping)
	}

	entry, ok := cmdRegistry.Lookup("info")
	if !ok {
		t.Fatal("info command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("info registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
