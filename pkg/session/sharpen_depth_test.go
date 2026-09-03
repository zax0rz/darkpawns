package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSharpenRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["sharpen"]
	if !ok {
		t.Fatal("sharpen command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("sharpen gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	entry, ok := cmdRegistry.Lookup("sharpen")
	if !ok {
		t.Fatal("sharpen command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("sharpen registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
