package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestHandbookRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["handbook"]
	if !ok {
		t.Fatal("handbook command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("handbook gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("handbook")
	if !ok {
		t.Fatal("handbook command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("handbook registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
