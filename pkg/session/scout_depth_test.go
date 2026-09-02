package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestScoutRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["scout"]
	if !ok {
		t.Fatal("scout command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("scout gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
}
