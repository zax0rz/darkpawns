package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestGiggleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["giggle"]
	if !ok {
		t.Fatal("giggle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("giggle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Giggler", 1, 1001)
	if err := m.Register("Giggler", s); err != nil {
		t.Fatalf("register Giggler: %v", err)
	}
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if err := ExecuteCommand(s, "giggle", nil); err != nil {
		t.Fatalf("ExecuteCommand(giggle): %v", err)
	}
	got := readMsgText(t, s)
	if got == "Huh?!?\r\n" || got == "" {
		t.Fatalf("giggle dispatch output = %q, want social output instead of Huh?!?", got)
	}
	if got != "You giggle.\r\n" {
		t.Fatalf("giggle dispatch output = %q, want %q", got, "You giggle.\r\n")
	}
}
