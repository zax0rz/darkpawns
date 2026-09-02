package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestShaRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["sha"]
	if !ok {
		t.Fatal("sha command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("sha gate = level %d position %d, want level 0 position %d",
			entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
	registered, ok := cmdRegistry.Lookup("sha")
	if !ok {
		t.Fatal("sha command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("sha registry gate = level %d position %d, want level %d position %d",
			registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}
