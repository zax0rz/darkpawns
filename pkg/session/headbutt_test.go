package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestHeadbuttRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["headbutt"]
	if !ok {
		t.Fatal("headbutt command has no C gate")
	}
	if gate.MinLevel != 1 || gate.MinPosition != combat.PosFighting {
		t.Fatalf("headbutt gate = level %d position %d, want level 1 position %d", gate.MinLevel, gate.MinPosition, combat.PosFighting)
	}
	entry, ok := cmdRegistry.Lookup("headbutt")
	if !ok {
		t.Fatal("headbutt command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("headbutt registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
