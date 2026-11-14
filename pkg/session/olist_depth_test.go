package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// C's cmd_info row is { "olist", POS_DEAD, do_olist, LVL_BUILDER, 0 }
// (src/interpreter.c:591); LVL_BUILDER aliases LVL_IMMORT. The registered Go
// entry must use the same C-derived gate, not an ad hoc level.
func TestOlistRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["olist"]
	if !ok {
		t.Fatal("olist command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("olist gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("olist")
	if !ok {
		t.Fatal("olist command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("olist registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
