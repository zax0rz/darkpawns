package session

import "testing"

func TestCmdHitUsesCOneArgumentBoundary(t *testing.T) {
	m := makeGateTestManager(t, false)
	if _, err := m.world.SpawnMob(5000, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)

	// C one_argument skips the leading fill word and ignores everything after
	// the first non-fill token before get_char_room_vis (R1/R2/R5e).
	if err := cmdHit(s, []string{"the", "target", "ignored"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if !m.combatEngine.IsFighting("Hero") {
		t.Fatal("hit should resolve the first non-fill token and start combat")
	}
}

func TestCmdHitFillWordsWithoutTargetUseNoArgumentMessage(t *testing.T) {
	m := makeGateTestManager(t, false)
	s := makeGateSession(t, m, 1, "Hero", 20)

	if err := cmdHit(s, []string{"the", "with"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "Hit who?\r\n"; got != want {
		t.Fatalf("fill-only hit message = %q, want %q", got, want)
	}
}
