package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFingerRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["finger"]
	if !ok {
		t.Fatal("finger command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("finger gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}
