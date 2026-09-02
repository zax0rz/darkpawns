package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestRecallRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["recall"]
	if !ok {
		t.Fatal("recall command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("recall gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	entry, ok := cmdRegistry.Lookup("recall")
	if !ok {
		t.Fatal("recall command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("recall registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
