package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestMoldRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["mold"]
	if !ok {
		t.Fatal("mold command has no C gate")
	}
	if entry.MinLevel != LVL_IMMORT || entry.MinPosition != combat.PosResting {
		t.Fatalf("mold gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_IMMORT, combat.PosResting)
	}

	registered, ok := cmdRegistry.Lookup("mold")
	if !ok {
		t.Fatal("mold command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("mold registry gate = level %d position %d, want level %d position %d", registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}
