package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestRollRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["roll"]
	if !ok {
		t.Fatal("roll command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("roll gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
	if _, ok := cmdRegistry.Lookup("roll"); !ok {
		t.Fatal("roll command is not registered")
	}
}
