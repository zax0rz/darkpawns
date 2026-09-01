package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestLevelsRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["levels"]
	if !ok {
		t.Fatal("levels command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("levels gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}

func TestCmdLevelsSwitchedMobUsesCNpcEarlyReturn(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeCommandTestSession(t, m, "levels-wizard", LVL_IMPL, 1001)
	s.isSwitched = true
	s.switchedMob = registerMob(t, m, 2001, 1001)

	if err := cmdLevels(s); err != nil {
		t.Fatalf("cmdLevels returned error: %v", err)
	}
	if got := readSessionText(t, s); got != "You ain't nothin' but a hound-dog.\r\n" {
		t.Fatalf("switched-mob levels output = %q, want C NPC early return", got)
	}
}
