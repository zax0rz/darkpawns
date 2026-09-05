package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestScroungeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["scrounge"]
	if !ok {
		t.Fatal("scrounge command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("scrounge gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
}
