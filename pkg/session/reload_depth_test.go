package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestReloadRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["reload"]
	if !ok {
		t.Fatal("reload command has no C gate")
	}
	if gate.MinLevel != LVL_IMPL-1 || gate.MinPosition != combat.PosDead {
		t.Fatalf("reload gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMPL-1, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("reload")
	if !ok {
		t.Fatal("reload command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("reload registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
