package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestHideFamilyRegistrationUsesCEntryGates(t *testing.T) {
	for _, name := range []string{"hide", "kabuki"} {
		entry, ok := commandGates[name]
		if !ok {
			t.Fatalf("%s command has no C gate", name)
		}
		if entry.MinLevel != 1 || entry.MinPosition != combat.PosResting {
			t.Fatalf("%s gate = level %d position %d, want level 1 position %d", name, entry.MinLevel, entry.MinPosition, combat.PosResting)
		}
	}
}
