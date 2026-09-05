package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestReportRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["report"]
	if !ok {
		t.Fatal("report command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("report gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}
