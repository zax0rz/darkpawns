package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFartRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["fart"]
	if !ok {
		t.Fatal("fart command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("fart gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
