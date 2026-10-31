package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// group is a POS_SLEEPING command (src/interpreter.c:471), and do_group answers
// two of its cases with send_to_char rather than act(): NOPERSON when the name
// resolves to nobody, and "Everyone following you here is already in your group."
// when "all" finds nothing to enrol (src/act.other.c:714-720). send_to_char
// writes straight to the descriptor (src/comm.c:2335-2339), so a sleeping leader
// reads both; Act(...TO_CHAR) would drop them through SENDOK's position gate.
func TestGroupSleepingLeaderReadsTheSendToCharAnswers(t *testing.T) {
	m := makeTestManager(t)
	leader := makeCommandTestSession(t, m, "Sleeper", 5, 1001)
	leader.player.SetPosition(combat.PosSleeping)
	registerInWorld(t, leader)

	t.Run("nobody", func(t *testing.T) {
		if err := cmdGroup(leader, []string{"Nobody"}); err != nil {
			t.Fatal(err)
		}
		if got := drainSessionOutput(t, leader); !strings.Contains(got, game.NoPersonHere) {
			t.Fatalf("sleeping leader read %q, want C's NOPERSON", got)
		}
	})

	t.Run("all with nothing to enrol", func(t *testing.T) {
		// Already grouped, and no followers in the room: C's found count stays 0.
		leader.player.SetInGroup(true)
		leader.player.SetAffect(game.AffGroup, true)
		if err := cmdGroup(leader, []string{"all"}); err != nil {
			t.Fatal(err)
		}
		got := drainSessionOutput(t, leader)
		if !strings.Contains(got, "Everyone following you here is already in your group.") {
			t.Fatalf("sleeping leader read %q, want C's already-in-your-group line", got)
		}
	})
}

// do_group's single-target gate is
//
//	((vict->master != ch) && (vict != ch)) || IS_SHADOWING(vict)
//
// (src/act.other.c:721). A shadowing follower still has the master link, so only
// the IS_SHADOWING arm refuses it — AFF_DODGE, which do_follow's quiet path sets.
// The session-era implementation checked the master link alone and enrolled the
// shadowing follower anyway.
func TestGroupRefusesAShadowingFollowerByName(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)

	leader := makeCommandTestSession(t, m, "Leader", 5, 1001)
	registerInWorld(t, leader)

	follower := makeCommandTestSession(t, m, "Shadower", 5, 1001)
	follower.player.SetFollowingBody(leader.player)
	follower.player.SetAffect(game.AffDodge, true) // IS_SHADOWING
	registerInWorld(t, follower)

	if err := cmdGroup(leader, []string{"Shadower"}); err != nil {
		t.Fatal(err)
	}
	if got := drainSessionOutput(t, leader); !strings.Contains(got, "Shadower must follow you to enter your group.") {
		t.Fatalf("leader read %q, want C's must-follow refusal", got)
	}
	if game.GroupedActor(follower.player) {
		t.Fatal("a shadowing follower was enrolled; C's IS_SHADOWING arm refuses it")
	}
}
