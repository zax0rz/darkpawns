package game

import "testing"

func TestRecruiterDirectTellUsesTellDeliveryGates(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	me := newSpecProcTestMob(t, w, 1001, 10)
	player.SetPlrFlag(PrfNotell, true)
	lastMsg()

	if !specRecruiter(w, player, me, "kill", "") {
		t.Fatal("recruiter should handle the kill command")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("recruiter direct do_tell bypassed NOTELL: %q", got)
	}
}
