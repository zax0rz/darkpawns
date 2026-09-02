package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestShowRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["show"]
	if !ok {
		t.Fatal("show command has no C gate")
	}
	if gate.MinLevel != 31 || gate.MinPosition != combat.PosDead {
		t.Fatalf("show gate = level %d position %d, want level 31 position %d",
			gate.MinLevel, gate.MinPosition, combat.PosDead)
	}
	entry, ok := cmdRegistry.Lookup("show")
	if !ok {
		t.Fatal("show command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("show registry gate = level %d position %d, want level %d position %d",
			entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
