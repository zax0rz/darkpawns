package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGaspRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["gasp"]
	if !ok {
		t.Fatal("gasp command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("gasp gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
