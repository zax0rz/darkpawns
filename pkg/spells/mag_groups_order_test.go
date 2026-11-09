package spells_test

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// C mag_groups walks the master's follower list — newest first (add_follower
// prepends, src/utils.c:474-475,491-492), then k, then the caster last
// (src/magic.c:1466-1501) — not the room. The fixtures deliberately make
// room-entry order differ from follower-list order so a room walk fails.
func groupMember(t *testing.T, w *game.World, outputs map[string]*strings.Builder, id int, name string, grouped bool) *game.Player {
	t.Helper()
	p := addGroupRecallPlayer(t, w, outputs, id, name, 1001)
	if grouped {
		p.SetInGroup(true)
		p.SetAffect(game.AffGroup, true)
	}
	return p
}

func TestMagGroupsMemberOrder(t *testing.T) {
	t.Run("root caster: followers newest-first, caster last", func(t *testing.T) {
		w, outputs := newGroupRecallWorld(t, false)
		// Entry order F1, F2, Leader puts the caster newest in the room; the
		// follower list is F2 (newest) then F1. C's walk is the list.
		f1 := groupMember(t, w, outputs, 1, "F1", true)
		f2 := groupMember(t, w, outputs, 2, "F2", true)
		leader := groupMember(t, w, outputs, 3, "Leader", true)
		w.AddFollowerQuiet(f1, leader)
		w.AddFollowerQuiet(f2, leader)
		// The scenario's reciter had detect invisibility; a room line about a
		// now-invisible body only reaches listeners who can see them
		// (act() hideInvisible, src/comm.c act()).
		leader.SetAffect(game.AffDetectInvisible, true)

		spells.MagGroups(30, leader, spells.SpellGroupInvis, int(spells.SaveSpell), w)

		want := "F2 slowly fades out of existence.\r\nF1 slowly fades out of existence.\r\nYou vanish.\r\n"
		if got := outputs["Leader"].String(); got != want {
			t.Fatalf("leader stream = %q, want %q", got, want)
		}
	})

	t.Run("caster with master: siblings, master, caster last", func(t *testing.T) {
		w, outputs := newGroupRecallWorld(t, false)
		f1 := groupMember(t, w, outputs, 1, "F1", true)
		f2 := groupMember(t, w, outputs, 2, "F2", true)
		leader := groupMember(t, w, outputs, 3, "Leader", true)
		w.AddFollowerQuiet(f1, leader)
		w.AddFollowerQuiet(f2, leader)
		f1.SetAffect(game.AffDetectInvisible, true)

		// F1 casts: k = Leader. C does F2 (the other follower of k), then k,
		// then F1 (src/magic.c:1481-1498).
		spells.MagGroups(30, f1, spells.SpellGroupInvis, int(spells.SaveSpell), w)

		want := "F2 slowly fades out of existence.\r\nLeader slowly fades out of existence.\r\nYou vanish.\r\n"
		if got := outputs["F1"].String(); got != want {
			t.Fatalf("caster stream = %q, want %q", got, want)
		}
	})
}

// C perform_mag_groups has no break after SPELL_MASS_DOMINATE: it falls
// through into SPELL_GROUP_INVIS (src/magic.c:1438-1441). Unreachable in play
// (MASS_DOMINATE is MAG_AREAS-only, src/spell_parser.c:1480-1481) but the
// function twin keeps C's fall-through bytes.
func TestMagGroupsMassDominateFallsThroughToGroupInvis(t *testing.T) {
	w, outputs := newGroupRecallWorld(t, false)
	leader := groupMember(t, w, outputs, 1, "Leader", true)
	member := groupMember(t, w, outputs, 2, "Member", true)
	w.AddFollowerQuiet(member, leader)

	spells.MagGroups(30, leader, spells.SpellMassDominate, int(spells.SaveSpell), w)

	// The charm arm's outcome varies (save roll); the fall-through's invis
	// bytes are unconditional.
	if got := outputs["Member"].String(); !strings.Contains(got, "You vanish.\r\n") {
		t.Fatalf("member stream = %q, want the GROUP_INVIS fall-through line %q", got, "You vanish.")
	}
}
