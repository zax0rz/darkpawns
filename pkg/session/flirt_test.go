package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFlirtRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["flirt"]
	if !ok {
		t.Fatal("flirt command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("flirt gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
