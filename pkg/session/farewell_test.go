package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFarewellRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["farewell"]
	if !ok {
		t.Fatal("farewell command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("farewell gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
