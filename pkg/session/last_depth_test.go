package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestLastRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["last"]
	if !ok {
		t.Fatal("last command has no C gate")
	}
	if gate.MinLevel != LVL_GOD-1 || gate.MinPosition != combat.PosDead {
		t.Fatalf("last gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GOD-1, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("last")
	if !ok {
		t.Fatal("last command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("last registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestLastUsesCOneArgumentAndMissingPlayerText(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Lastgod", LVL_IMPL, 1001)

	if err := cmdLast(s, []string{"the", "Nobody", "trailing"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	if got, want := readSessionText(t, s), "There is no such player.\r\n"; got != want {
		t.Fatalf("cmdLast missing target = %q, want %q", got, want)
	}
}
