package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// makeSetTestSession creates a wizard-actor session registered in the manager
// (so findSessionByName resolves the target) plus a separate mortal target.
func makeSetTestSession(t *testing.T) (*Session, *Session) {
	t.Helper()
	m := makeTestManager(t)
	wiz := makeTestSession(t, m, "God", 1001, true)
	wiz.player.Level = LVL_GRGOD

	target := makeTestSession(t, m, "Hero", 1001, true)
	m.mu.Lock()
	m.sessions["god"] = wiz
	m.sessions["hero"] = target
	m.mu.Unlock()
	return wiz, target
}

// TestCmdSetConditions covers do_set cases 29-31 (act.wizard.c:2977-2993):
// drunk/hunger/thirst map to GET_COND slots 0/1/2, accept "off" (→ -1) or a
// number clamped to [0,48], and echo C's ack bytes ("Hero's drunk set to
// 24.\r\n" / "Hero's drunk now off.\r\n" / "Must be 'off' or a value from 0
// to 48.\r\n").
func TestCmdSetConditions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
		cond int
		val  int
	}{
		{"drunk set", []string{"Hero", "drunk", "24"}, "Hero's drunk set to 24.\r\n", game.CondDrunk, 24},
		{"hunger set", []string{"Hero", "hunger", "36"}, "Hero's hunger set to 36.\r\n", game.CondFull, 36},
		{"thirst set", []string{"Hero", "thirst", "48"}, "Hero's thirst set to 48.\r\n", game.CondThirst, 48},
		{"drunk clamps high", []string{"Hero", "drunk", "49"}, "Hero's drunk set to 48.\r\n", game.CondDrunk, 48},
		{"drunk clamps low", []string{"Hero", "drunk", "-5"}, "Hero's drunk set to 0.\r\n", game.CondDrunk, 0},
		{"drunk off", []string{"Hero", "drunk", "off"}, "Hero's drunk now off.\r\n", game.CondDrunk, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wiz, target := makeSetTestSession(t)
			if err := cmdSet(wiz, tc.args); err != nil {
				t.Fatalf("cmdSet: %v", err)
			}
			if got := readSessionText(t, wiz); got != tc.want {
				t.Errorf("ack = %q, want %q", got, tc.want)
			}
			if got := target.player.GetCondition(tc.cond); got != tc.val {
				t.Errorf("condition = %d, want %d", got, tc.val)
			}
		})
	}

	t.Run("drunk rejects words", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		if err := cmdSet(wiz, []string{"Hero", "drunk", "tipsy"}); err != nil {
			t.Fatalf("cmdSet: %v", err)
		}
		if got := readSessionText(t, wiz); got != "Must be 'off' or a value from 0 to 48.\r\n" {
			t.Errorf("ack = %q, want usage error", got)
		}
		if got := target.player.GetCondition(game.CondDrunk); got != 0 {
			t.Errorf("condition = %d, want unchanged 0", got)
		}
	})
}

func TestCmdSetCStatFieldsAndUnsupportedPosition(t *testing.T) {
	t.Run("wis zero enables stupid gate vehicle", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		if err := cmdSet(wiz, []string{"Hero", "wis", "0"}); err != nil {
			t.Fatalf("cmdSet: %v", err)
		}
		if got := readSessionText(t, wiz); got != "Hero's wis set to 0.\r\n" {
			t.Fatalf("ack = %q, want C stat ack", got)
		}
		if target.player.Stats.Wis != 0 {
			t.Fatalf("wis = %d, want 0", target.player.Stats.Wis)
		}
	})

	t.Run("mortal stat range is zero through eighteen", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		if err := cmdSet(wiz, []string{"Hero", "wis", "99"}); err != nil {
			t.Fatalf("cmdSet: %v", err)
		}
		if got := readSessionText(t, wiz); got != "Hero's wis set to 18.\r\n" {
			t.Fatalf("ack = %q, want clamped C stat ack", got)
		}
		if target.player.Stats.Wis != 18 {
			t.Fatalf("wis = %d, want 18", target.player.Stats.Wis)
		}
	})

	t.Run("position is not a C field", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		if err := cmdSet(wiz, []string{"Hero", "position", "3"}); err != nil {
			t.Fatalf("cmdSet: %v", err)
		}
		if got := readSessionText(t, wiz); got != "Can't set that!\r\n" {
			t.Fatalf("ack = %q, want unsupported-field response", got)
		}
		if target.player.GetPosition() != combat.PosStanding {
			t.Fatalf("position = %d, want unchanged standing", target.player.GetPosition())
		}
	})
}
