package session

import "testing"

func TestCmdAdvanceAuthorityBranches(t *testing.T) {
	t.Run("equal-level target", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeTestSession(t, m, "Advanceactor", 1001, true)
		actor.player.Level = LVL_GRGOD
		target := makeTestSession(t, m, "Advancetarget", 1001, true)
		target.player.Level = LVL_GRGOD
		registerInWorld(t, actor)
		registerInWorld(t, target)

		if err := cmdAdvance(actor, []string{"Advancetarget", "39"}); err != nil {
			t.Fatalf("cmdAdvance returned error: %v", err)
		}
		if got := readMsgText(t, actor); got != "Maybe that's not such a great idea.\r\n" {
			t.Fatalf("message = %q, want C authority rejection", got)
		}
		if target.player.Level != LVL_GRGOD {
			t.Fatalf("target level changed to %d after authority rejection", target.player.Level)
		}
	})

	t.Run("promotion above actor", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeTestSession(t, m, "Advanceactor", 1001, true)
		actor.player.Level = LVL_GRGOD
		target := makeTestSession(t, m, "Advancetarget", 1001, true)
		target.player.Level = 1
		registerInWorld(t, actor)
		registerInWorld(t, target)

		if err := cmdAdvance(actor, []string{"Advancetarget", "39"}); err != nil {
			t.Fatalf("cmdAdvance returned error: %v", err)
		}
		if got := readMsgText(t, actor); got != "Yeah, right.\r\n" {
			t.Fatalf("message = %q, want C authority rejection", got)
		}
		if target.player.Level != 1 {
			t.Fatalf("target level changed to %d after authority rejection", target.player.Level)
		}
	})
}
