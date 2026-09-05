package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSaveRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["save"]
	if !ok {
		t.Fatal("save command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("save gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosSleeping)
	}
}
