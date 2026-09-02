package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestDiagnoseRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["diagnose"]
	if !ok {
		t.Fatal("diagnose command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("diagnose gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	for _, name := range []string{"diagnose", "diag", "glance"} {
		entry, ok := cmdRegistry.Lookup(name)
		if !ok {
			t.Fatalf("%s command alias is not registered", name)
		}
		if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
			t.Fatalf("%s registry gate = level %d position %d, want level %d position %d", name, entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
		}
	}
}
