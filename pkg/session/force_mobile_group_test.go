package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C's do_force resolves its target with get_char_vis — mobiles included — and
// hands the command to command_interpreter (src/act.wizard.c:1869-1880), so
// `force <mob> group <player>` runs do_group as that mobile. do_group has no
// IS_NPC gate (src/act.other.c:685-740) and perform_group is the only thing in
// the tree that sets AFF_GROUP (line 629), so a mobile is the only body that can
// enrol a player whose master is a mobile.
//
// The port accepted the command — telling the forcer "Okay." — and then dropped
// it as unsupported, so the follower stayed ungrouped and its own `group`
// answered "But you are not the member of a group!" where C answers with its
// listing (F9).
func TestForceMobileRunsGroupAndEnrolsTheFollower(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{{
			VNum: 99100, Keywords: "guard", ShortDesc: "a guard",
			Level: 3, Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)

	actor := makeCommandTestSession(t, m, "Forcegod", LVL_IMPL, 1001)
	registerInWorld(t, actor)

	guard, err := w.SpawnMob(99100, 1001)
	if err != nil {
		t.Fatal(err)
	}

	// The follower's master is the mobile, so only the mobile can enrol it.
	follower := makeCommandTestSession(t, m, "Follower", 5, 1001)
	follower.player.SetFollowingBody(guard)
	registerInWorld(t, follower)

	if err := cmdForce(actor, []string{"guard", "group Follower"}); err != nil {
		t.Fatalf("cmdForce returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("forcer acknowledgement = %q, want C's Okay", got)
	}
	if !game.GroupedActor(follower.player) {
		t.Fatal("the mobile's group did not enrol the follower: the forced command was dropped")
	}
	// perform_group's TO_VICT line, which is the only one the follower sees: the
	// forcer is LVL_IMPL, so C sends no "has forced you" notice.
	if got := readMsgText(t, follower); !strings.Contains(got, "You are now a member of a guard's group.") {
		t.Fatalf("follower notification = %q, want C's perform_group TO_VICT line", got)
	}

	// The follower can now read its own group listing, which is what the oracle
	// scenario compares.
	if err := cmdGroup(follower, nil); err != nil {
		t.Fatal(err)
	}
	if got := drainSessionOutput(t, follower); !strings.Contains(got, "Your group consists of:") {
		t.Fatalf("follower group listing = %q, want C's listing", got)
	}
}
