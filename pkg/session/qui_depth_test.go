package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestQuiRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["qui"]
	if !ok {
		t.Fatal("qui command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("qui gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}

func TestQuiMortalShortcutUsesCRefusal(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "QuiMortal", 1, 1001)

	if err := cmdQuiStub(s); err != nil {
		t.Fatalf("cmdQuiStub: %v", err)
	}
	if got := drainSendChannel(t, s); !strings.Contains(got, "You have to type quit--no less, to quit!") {
		t.Fatalf("qui output = %q, want C refusal", got)
	}
}
