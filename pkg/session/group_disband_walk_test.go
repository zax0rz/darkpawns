package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestDisbandWalkIncludesMobAndKeepsCharmedFollow drives the real player
// command body. C walks grouped followers newest first, clears each group bit,
// sends the disband line, and only calls stop_follower for non-charmed bodies
// (src/act.other.c:757-765; src/utils.c:419-426,472-476).
func TestCmdUngroupDisbandIncludesMobileAndPreservesCharmedFollow(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 9001, Keywords: "rat", ShortDesc: "a giant rat", Level: 1,
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)

	leader := makeCommandTestSession(t, m, "Leader", 10, 1001)
	leader.player.SetInGroup(true)
	leader.player.SetAffect(game.AffGroup, true)
	registerInWorld(t, leader)

	alpha := makeCommandTestSession(t, m, "Alpha", 5, 1001)
	alpha.player.SetFollowingBody(leader.player)
	alpha.player.SetInGroup(true)
	alpha.player.SetAffect(game.AffGroup, true)
	registerInWorld(t, alpha)

	beta := makeCommandTestSession(t, m, "Beta", 5, 1001)
	beta.player.SetFollowingBody(leader.player)
	beta.player.SetInGroup(true)
	beta.player.SetAffect(game.AffGroup, true)
	registerInWorld(t, beta)

	pet, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	pet.SetFollowingBody(leader.player)
	pet.SetAffected(game.AffCharm)
	pet.SetAffected(game.AffGroup)

	output := make(map[string]string)
	w.MessageSink = func(name string, message []byte) {
		output[name] += string(message)
	}
	if err := cmdUngroup(leader, nil); err != nil {
		t.Fatalf("cmdUngroup: %v", err)
	}
	if got, want := readMsgText(t, leader), "You disband the group."; got != want {
		t.Fatalf("leader command output = %q, want %q", got, want)
	}
	if got, want := output["Alpha"], "Beta stops following Leader.\r\nLeader has disbanded the group.\r\nYou stop following Leader.\r\n"; got != want {
		t.Fatalf("Alpha output = %q, want %q", got, want)
	}
	if got, want := output["Beta"], "Leader has disbanded the group.\r\nYou stop following Leader.\r\nAlpha stops following Leader.\r\n"; got != want {
		t.Fatalf("Beta output = %q, want %q", got, want)
	}
	if got, want := output["Leader"], "Beta stops following you.\r\nAlpha stops following you.\r\n"; got != want {
		t.Fatalf("leader room output = %q, want %q", got, want)
	}
	if leader.player.InGroup || leader.player.IsAffected(game.AffGroup) {
		t.Fatalf("leader remains grouped after disband: mirror=%v affect=%v", leader.player.InGroup, leader.player.IsAffected(game.AffGroup))
	}
	for _, follower := range []*game.Player{alpha.player, beta.player} {
		if follower.InGroup || follower.IsAffected(game.AffGroup) || follower.GetFollowing() != "" {
			t.Fatalf("%s state after disband: grouped=%v following=%q", follower.Name, follower.IsAffected(game.AffGroup), follower.GetFollowing())
		}
	}
	if pet.IsAffected(game.AffGroup) || pet.GetFollowing() != leader.player.Name {
		t.Fatalf("charmed mobile state after disband: grouped=%v following=%q", pet.IsAffected(game.AffGroup), pet.GetFollowing())
	}
}
