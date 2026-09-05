package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestInvisRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["invis"]
	if !ok {
		t.Fatal("invis command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("invis gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("invis")
	if !ok {
		t.Fatal("invis command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("invis registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestInvisPromptIncludesCLevel(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Invprompt", LVL_IMMORT, 1001)
	s.player.SetInvisLevel(31)

	if got := s.promptText(); got != "\r\ni31 > " {
		t.Fatalf("invis prompt = %q, want %q", got, "\r\ni31 > ")
	}
}
