package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGrabRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["grab"]
	if !ok {
		t.Fatal("grab command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("grab gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
