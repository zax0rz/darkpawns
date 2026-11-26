package session

import (
	"testing"
)

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
