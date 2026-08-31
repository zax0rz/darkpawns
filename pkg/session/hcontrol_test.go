package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestHcontrolRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["hcontrol"]
	if !ok {
		t.Fatal("hcontrol command has no C gate")
	}
	if gate.MinLevel != LVL_GRGOD || gate.MinPosition != combat.PosDead {
		t.Fatalf("hcontrol gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GRGOD, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("hcontrol")
	if !ok {
		t.Fatal("hcontrol command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("hcontrol registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}
