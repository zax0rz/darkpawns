package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestLeaveRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["leave"]
	if !ok {
		t.Fatal("leave command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosStanding {
		t.Fatalf("leave gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosStanding)
	}

	entry, ok := cmdRegistry.Lookup("leave")
	if !ok {
		t.Fatal("leave command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("leave registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
