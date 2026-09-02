package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSerpentRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["serpent"]
	if !ok {
		t.Fatal("serpent command has no C gate")
	}
	if entry.MinLevel != 1 || entry.MinPosition != combat.PosFighting {
		t.Fatalf("serpent gate = level %d position %d, want level 1 position %d",
			entry.MinLevel, entry.MinPosition, combat.PosFighting)
	}
	registered, ok := cmdRegistry.Lookup("serpent")
	if !ok {
		t.Fatal("serpent command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("serpent registry gate = level %d position %d, want level %d position %d",
			registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}
