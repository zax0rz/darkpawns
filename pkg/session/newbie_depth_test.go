package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestNewbieRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["newbie"]
	if !ok {
		t.Fatal("newbie command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosSleeping {
		t.Fatalf("newbie gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosSleeping)
	}

	entry, ok := cmdRegistry.Lookup("newbie")
	if !ok {
		t.Fatal("newbie command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("newbie registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
