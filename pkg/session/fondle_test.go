package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFondleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["fondle"]
	if !ok {
		t.Fatal("fondle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("fondle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
