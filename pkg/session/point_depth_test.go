package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPointRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["point"]
	if !ok {
		t.Fatal("point command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("point gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
