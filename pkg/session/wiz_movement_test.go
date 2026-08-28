package session

import "testing"

func TestCmdAtPreservesNestedMovementLocation(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = LVL_GRGOD

	if err := cmdAt(s, []string{"1002", "goto", "1001"}); err != nil {
		t.Fatalf("cmdAt failed: %v", err)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Fatalf("nested movement ended in room %d, want 1001", got)
	}
}
