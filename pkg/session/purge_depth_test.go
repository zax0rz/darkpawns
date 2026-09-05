package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPurgeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["purge"]
	if !ok {
		t.Fatal("purge command has no C gate")
	}
	wantLevel := LVL_IMMORT + 1
	if entry.MinLevel != wantLevel || entry.MinPosition != combat.PosDead {
		t.Fatalf("purge gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, wantLevel, combat.PosDead)
	}
}
