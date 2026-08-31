package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGroinripRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["groinrip"]
	if !ok {
		t.Fatal("groinrip command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosFighting {
		t.Fatalf("groinrip gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosFighting)
	}
	if _, ok := cmdRegistry.Lookup("groinrip"); !ok {
		t.Fatal("groinrip command is not registered")
	}
}
