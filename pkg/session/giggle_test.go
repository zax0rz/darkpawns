package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGiggleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["giggle"]
	if !ok {
		t.Fatal("giggle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("giggle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
