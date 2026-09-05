package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestShadowRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["shadow"]
	if !ok {
		t.Fatal("shadow command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("shadow gate = level %d position %d, want level 0 position %d",
			entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
	registered, ok := cmdRegistry.Lookup("shadow")
	if !ok {
		t.Fatal("shadow command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("shadow registry gate = level %d position %d, want level %d position %d",
			registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}
