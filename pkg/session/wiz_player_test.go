package session

import (
	"strings"
	"testing"
)

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

// C's promotion act preserves each embedded CRLF and both blank paragraphs
// (src/act.wizard.c:1556-1569).
func TestCmdAdvancePromotionUsesCActLineEndings(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Advanceactor", 1001, true)
	actor.player.Level = LVL_GRGOD
	target := makeTestSession(t, m, "Advancetarget", 1001, true)
	target.player.Level = 1
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdAdvance(actor, []string{"Advancetarget", "2"}); err != nil {
		t.Fatalf("cmdAdvance returned error: %v", err)
	}
	const wantPrefix = "Advanceactor makes some strange gestures.\r\n" +
		"A strange feeling comes upon you,\r\n" +
		"Like a giant hand, light comes down\r\n" +
		"from above, grabbing your body, that\r\n" +
		"begins to pulse with colored lights\r\n" +
		"from inside.\r\n\r\n" +
		"Your head seems to be filled with demons\r\n" +
		"from another plane as your body dissolves\r\n" +
		"to the elements of time and space itself.\r\n" +
		"Suddenly a silent explosion of light\r\n" +
		"snaps you back to reality.\r\n\r\n" +
		"You feel slightly different.\r\n"
	if got := readSessionText(t, target); !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("promotion output prefix = %q, want %q", got, wantPrefix)
	}
}
