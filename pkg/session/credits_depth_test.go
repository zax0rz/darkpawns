package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestCreditsRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["credits"]
	if !ok {
		t.Fatal("credits command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("credits gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("credits")
	if !ok {
		t.Fatal("credits command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("credits registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
