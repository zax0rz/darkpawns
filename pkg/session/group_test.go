package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGroupRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["group"]
	if !ok {
		t.Fatal("group command has no C gate")
	}
	if entry.MinLevel != 1 || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("group gate = level %d position %d, want level 1 position %d", entry.MinLevel, entry.MinPosition, combat.PosSleeping)
	}
}
