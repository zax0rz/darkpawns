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
// order (src/utils.c:472-476): C's disband walks that list in order and
// includes mobile followers (src/act.other.c:758-765). The pre-fix walk ranged
// over the player map — random order, no mobiles (DP-1495).
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

// TestCharacterUngroupDisbandsMobileFollowers exercises the production mobile
// command body (the player command is covered by the session-level
// TestDisbandWalkIncludesMobAndKeepsCharmedFollow). A charmed mobile follower
// loses AFF_GROUP but keeps its master, while an ordinary player follower is
// detached (src/act.other.c:757-765; src/utils.c:419-426).
func TestCharacterUngroupDisbandsMobileFollowers(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 9001, Keywords: "leader", ShortDesc: "the leader", Level: 10},
			{VNum: 9002, Keywords: "rat", ShortDesc: "a giant rat", Level: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	leader, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	leader.SetAffected(AffGroup)
	follower := NewPlayer(1, "Follower", 1001)
	if err := w.AddPlayer(follower); err != nil {
		t.Fatal(err)
	}
	follower.SetFollowingBody(leader)
	follower.SetInGroup(true)
	follower.SetAffect(AffGroup, true)

	pet, err := w.SpawnMob(9002, 1001)
	if err != nil {
		t.Fatal(err)
	}
	pet.SetFollowingBody(leader)
	pet.SetAffected(AffCharm)
	pet.SetAffected(AffGroup)

	if got := CharacterUngroup(CharacterCommandContext{Actor: leader, World: w}); got != CharacterCommandHandled {
		t.Fatalf("CharacterUngroup result = %v, want handled", got)
	}
	if leader.IsAffected(AffGroup) {
		t.Fatal("mobile leader remained grouped after disband")
	}
	if follower.IsAffected(AffGroup) || follower.GetFollowing() != "" {
		t.Fatalf("ordinary player follower remains grouped/following: group=%v following=%q", follower.IsAffected(AffGroup), follower.GetFollowing())
	}
	if pet.IsAffected(AffGroup) || pet.GetFollowing() != leader.GetName() {
		t.Fatalf("charmed mobile state after disband: group=%v following=%q", pet.IsAffected(AffGroup), pet.GetFollowing())
	}
}
