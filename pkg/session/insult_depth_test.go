package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestInsultRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["insult"]
	if !ok {
		t.Fatal("insult command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("insult gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	entry, ok := cmdRegistry.Lookup("insult")
	if !ok {
		t.Fatal("insult command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("insult registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
