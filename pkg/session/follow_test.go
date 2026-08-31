package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFollowRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["follow"]
	if !ok {
		t.Fatal("follow command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("follow gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
