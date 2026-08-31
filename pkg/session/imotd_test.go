package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestImotdRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["imotd"]
	if !ok {
		t.Fatal("imotd command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("imotd gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("imotd")
	if !ok {
		t.Fatal("imotd command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("imotd registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
