package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestJinRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["jin"]
	if !ok {
		t.Fatal("jin command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosStanding {
		t.Fatalf("jin gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosStanding)
	}

	entry, ok := cmdRegistry.Lookup("jin")
	if !ok {
		t.Fatal("jin command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("jin registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
