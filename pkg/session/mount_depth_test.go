package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestMountAndRideRegistrationUseCEntryGates(t *testing.T) {
	for _, name := range []string{"mount", "ride"} {
		gate, ok := commandGates[name]
		if !ok {
			t.Fatalf("%s command has no C gate", name)
		}
		if gate.MinLevel != 0 || gate.MinPosition != combat.PosStanding {
			t.Fatalf("%s gate = level %d position %d, want level 0 position %d", name, gate.MinLevel, gate.MinPosition, combat.PosStanding)
		}

		entry, ok := cmdRegistry.Lookup(name)
		if !ok {
			t.Fatalf("%s command is not registered", name)
		}
		if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
			t.Fatalf("%s registry gate = level %d position %d, want level %d position %d", name, entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
		}
	}
}
