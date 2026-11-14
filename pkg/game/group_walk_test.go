package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// groupWalkFixture builds a one-room world with a leader, two player
// followers and a charmed mobile follower, capturing every player message.
func groupWalkFixture(t *testing.T) (*World, *Player, *Player, *Player, *MobInstance, func() []string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 9001, Keywords: "rat", ShortDesc: "a giant rat", Level: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	leader := NewPlayer(1, "Leader", 1001)
	alpha := NewPlayer(2, "Alpha", 1001)
	beta := NewPlayer(3, "Beta", 1001)
	for _, p := range []*Player{leader, alpha, beta} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}
	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	// C's follower list PREPENDS (utils.c:472-476): alpha follows first,
	// beta second, the charmed rat last, so the walk order is rat, beta,
	// alpha.
	alpha.SetFollowing("Leader")
	beta.SetFollowing("Leader")
	mob.SetFollowing("Leader")
	mob.SetAffected(AffCharm)

	leader.SetInGroup(true)
	leader.SetAffect(AffGroup, true)
	for _, p := range []*Player{alpha, beta} {
		p.SetInGroup(true)
		p.SetAffect(AffGroup, true)
	}
	mob.SetAffected(AffGroup)

	drain := func() []string {
		var lines []string
		for l := range strings.SplitSeq(strings.TrimRight(out.String(), "\r\n"), "\r\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
		out.Reset()
		return lines
	}
	return w, leader, alpha, beta, mob, drain
}

// TestOrderedGroupFollowersWalksNewestFirst pins add_follower's prepend
// order (src/utils.c:472-476): the disband walks followers newest-first with
// mobiles included (act.other.c:758). The pre-fix walk ranged over the player
// map — random order, no mobiles (DP-1495).
func TestOrderedGroupFollowersWalksNewestFirst(t *testing.T) {
	w, leader, _, _, mob, _ := groupWalkFixture(t)

	followers := w.OrderedGroupFollowers(leader.Name)
	if len(followers) != 3 {
		t.Fatalf("followers = %d, want 3 (two players and the mobile)", len(followers))
	}
	if got := followers[0].GetName(); got != mob.GetName() {
		t.Fatalf("newest follower = %q, want the charmed mobile %q", got, mob.GetName())
	}
	if followers[1].GetName() != "Beta" || followers[2].GetName() != "Alpha" {
		t.Fatalf("player order = %q then %q, want Beta (newer) before Alpha", followers[1].GetName(), followers[2].GetName())
	}
}

// TestDisbandWalkIncludesMobAndKeepsCharmedFollow pins the disband arm
// (act.other.c:757-765) over the ordered walk: every grouped follower loses
// AFF_GROUP and is told, and stop_follower runs only for the non-charmed —
// the charmed mobile stays following. The pre-fix walk never reached the
// mobile, so its AFF_GROUP survived the disband (DP-1495).
func TestDisbandWalkIncludesMobAndKeepsCharmedFollow(t *testing.T) {
	w, _, alpha, beta, mob, _ := groupWalkFixture(t)

	for _, member := range w.OrderedGroupFollowers("Leader") {
		if !GroupedActor(member) {
			continue
		}
		SetGroupedActor(member, false)
		switch body := member.(type) {
		case *Player:
			body.SendMessage("Leader has disbanded the group.\r\n")
			if !body.IsAffected(AffCharm) {
				StopFollower(w, body)
			}
		case *MobInstance:
			body.SendMessage("Leader has disbanded the group.\r\n")
			if !body.IsAffected(AffCharm) {
				StopFollowerMob(w, body)
			}
		}
	}

	if mob.IsAffected(AffGroup) {
		t.Fatal("charmed mobile still grouped after the disband walk")
	}
	if mob.GetFollowing() != "Leader" {
		t.Fatalf("charmed mobile follow = %q, want Leader (charm keeps the follow, act.other.c:761-762)", mob.GetFollowing())
	}
	for _, p := range []*Player{alpha, beta} {
		if p.InGroup || p.IsAffected(AffGroup) {
			t.Fatalf("%s still grouped after disband", p.Name)
		}
		if p.GetFollowing() != "" {
			t.Fatalf("%s still following after disband (stop_follower ran for the non-charmed)", p.Name)
		}
	}
}
