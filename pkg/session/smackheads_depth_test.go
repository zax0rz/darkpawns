package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSmackheadsRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["smackheads"]
	if !ok {
		t.Fatal("smackheads command has no C gate")
	}
	if gate.MinLevel != 1 || gate.MinPosition != combat.PosFighting {
		t.Fatalf("smackheads gate = level %d position %d, want level 1 position %d", gate.MinLevel, gate.MinPosition, combat.PosFighting)
	}

	entry, ok := cmdRegistry.Lookup("smackheads")
	if !ok {
		t.Fatal("smackheads command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("smackheads registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
