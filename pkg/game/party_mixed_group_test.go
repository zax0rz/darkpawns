package game

// Mixed player/NPC group reward proof — C group_gain() counts every in-room
// AFF_GROUP actor in its divisor and recipient list, including mobiles
// (src/fight.c:708-848). Mobile recipients skip gain_exp and change_alignment
// (which itself returns for NPCs, src/fight.c:486-489) but receive autosplit
// gold (src/fight.c:798).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newMixedGroupWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Group Room", Zone: 1},
			{VNum: 1002, Name: "Far Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 1, ShortDesc: "a trainee", Keywords: "trainee", Level: 5, Exp: 1000},
			{VNum: 2, ShortDesc: "a bodyguard", Keywords: "bodyguard", Level: 5, Exp: 0},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// TestMixedGroupKillIncludesNPCInDivisor proves the divisor counts every
// in-room grouped actor. It fails if NPC members are accepted as recipients
// while the divisor stays player-only: exp 1000 over {leader, pet} shares a
// base of 500, not the 1000 a player-only divisor yields.
func TestMixedGroupKillIncludesNPCInDivisor(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	pet, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	leader.SetInGroup(true)
	pet.SetFollowingBody(leader)
	pet.SetAffected(affGroup)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(leader, nil, 1000, 0, 5, 0)

	if got, want := leader.GetExp(), 495; got != want {
		t.Fatalf("grouped leader XP = %d, want %d (1000/2 divisor, 1%% group penalty)", got, want)
	}
	if got := pet.GetExp(); got != 0 {
		t.Fatalf("grouped mobile XP = %d, want 0: C skips gain_exp for NPCs (src/fight.c:702)", got)
	}
}

// TestGroupedNPCKillerAwardsPlayerShares drives the kill through a grouped
// mobile body (a charmed pet landing the blow): C still runs group_gain with
// ch = the pet, so the grouped player leader receives a share.
func TestGroupedNPCKillerAwardsPlayerShares(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	leader.SetAlignment(0)
	pet, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	pet.SetAlignment(0)
	leader.SetInGroup(true)
	pet.SetFollowingBody(leader)
	pet.SetAffected(affGroup)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(pet, nil, 1000, 0, 5, -1000)

	if got, want := leader.GetExp(), 495; got != want {
		t.Fatalf("leader XP after mobile kill = %d, want %d", got, want)
	}
	if got := leader.GetAlignment(); got <= 0 {
		t.Fatalf("leader alignment = %d, want a positive shift from an evil victim (src/fight.c:704)", got)
	}
	if got := pet.GetAlignment(); got != 0 {
		t.Fatalf("mobile killer alignment = %d, want 0: change_alignment returns for NPCs (src/fight.c:486-489)", got)
	}
	if got := pet.GetExp(); got != 0 {
		t.Fatalf("mobile killer XP = %d, want 0", got)
	}
}

// TestUngroupedMobKillerAwardsNothing retains the ungrouped-killer boundary
// for mobile killers: a pet without AFF_GROUP takes the solo path, and a solo
// mobile kill produces nothing observable (src/fight.c:1639-1667).
func TestUngroupedMobKillerAwardsNothing(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetInGroup(true)
	pet, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	pet.SetFollowingBody(leader)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(pet, nil, 1000, 0, 5, 0)

	if got := leader.GetExp(); got != 0 {
		t.Fatalf("leader XP = %d, want 0 for an ungrouped mobile killer", got)
	}
}

// TestMixedGroupAutosplitPaysNPCMember proves the split divisor and the gold
// transfers both include grouped mobiles, and that the split conserves gold.
func TestMixedGroupAutosplitPaysNPCMember(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	leader.AutoGold = true
	leader.AutoSplit = true
	pet, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	leader.SetInGroup(true)
	pet.SetFollowingBody(leader)
	pet.SetAffected(affGroup)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	output := captureMovementOutput(w)

	w.AwardMobKillXP(leader, nil, 1000, 101, 5, 0)

	// 101 coins over 2 members: 50 each and the killer keeps the 1 remainder.
	if got, want := leader.GetGold(), 51; got != want {
		t.Fatalf("leader gold = %d, want %d (share 50 + remainder 1)", got, want)
	}
	if got, want := pet.GetGold(), 50; got != want {
		t.Fatalf("mobile member gold = %d, want %d: C pays NPC followers their share (src/fight.c:798)", got, want)
	}
	msg := output[leader.Name].String()
	for _, want := range []string{
		"You loot 101 coins from the corpse of .\r\n",
		"You share 50 gold with a bodyguard.\r\n",
		"You split the gold and keep 50 for yourself.\r\n",
		"You keep 1 gold leftover.\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("leader output missing %q; got %q", want, msg)
		}
	}
	if pet.GetGold()+leader.GetGold() != 101 {
		t.Fatalf("gold not conserved: %d + %d != 101", pet.GetGold(), leader.GetGold())
	}
}

// TestGroupSplitNoGoldLeaderSilent pins C's asymmetric no-gold notices: the
// leader-share block exists only while gold_per_member > 0, so with a zero
// share only follower-loop members see the none-to-split lines
// (src/fight.c:768-823).
func TestGroupSplitNoGoldLeaderSilent(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	follower := NewPlayer(2, "Follower", 1001)
	follower.SetLevel(5)
	follower.AutoGold = true
	follower.AutoSplit = true
	follower.SetFollowingBody(leader)
	follower.SetInGroup(true)
	leader.SetInGroup(true)
	for _, p := range []*Player{leader, follower} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) failed: %v", p.Name, err)
		}
	}
	output := captureMovementOutput(w)

	w.AwardMobKillXP(follower, nil, 1000, 1, 5, 0)

	// The follower is the killer; the leader is k != ch with share 0: C's
	// leader block is skipped entirely and only the killer's own
	// no-gold line plus the leftover keep appear.
	kmsg := output[follower.Name].String()
	for _, want := range []string{
		"You loot 1 coins from the corpse of .\r\n",
		"When you split no gold, you got none.\r\n",
		"You keep 1 gold leftover.\r\n",
	} {
		if !strings.Contains(kmsg, want) {
			t.Errorf("killer output missing %q; got %q", want, kmsg)
		}
	}
	if strings.Contains(kmsg, "none to split") {
		t.Errorf("killer output has a none-to-split notice for the leader; C's leader arm is gpm>0 only: %q", kmsg)
	}
	if lmsg := output[leader.Name].String(); strings.Contains(lmsg, "gold") || strings.Contains(lmsg, "split") {
		t.Errorf("leader output = %q, want the XP share only (leader gold arm is gpm>0 only)", lmsg)
	}
	if got, want := follower.GetGold(), 1; got != want {
		t.Fatalf("killer gold = %d, want %d leftover", got, want)
	}
	if got := leader.GetGold(); got != 0 {
		t.Fatalf("leader gold = %d, want 0", got)
	}
}

// TestGroupExcludesOutOfRoomMembers keeps C's room scoping: an in-group
// member in another room is neither counted in the divisor nor paid.
func TestGroupExcludesOutOfRoomMembers(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	away := NewPlayer(2, "Away", 1002)
	away.SetLevel(5)
	away.SetFollowingBody(leader)
	away.SetInGroup(true)
	leader.SetInGroup(true)
	for _, p := range []*Player{leader, away} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) failed: %v", p.Name, err)
		}
	}

	w.AwardMobKillXP(leader, nil, 1000, 0, 5, 0)

	// Solo is_in_group(): the only other member is out of the room.
	if got, want := leader.GetExp(), 1000; got != want {
		t.Fatalf("leader XP = %d, want %d (out-of-room member leaves the killer solo)", got, want)
	}
	if got := away.GetExp(); got != 0 {
		t.Fatalf("out-of-room member XP = %d, want 0", got)
	}
}

// TestGroupShareMessageOrderFollowsCFollowerList pins the killer's byte
// order: the leader share is announced before follower shares, and followers
// appear in C follower-list order (newest first).
func TestGroupShareMessageOrderFollowsCFollowerList(t *testing.T) {
	w := newMixedGroupWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(5)
	first := NewPlayer(2, "First", 1001)
	first.SetLevel(5)
	second := NewPlayer(3, "Second", 1001)
	second.SetLevel(5)
	first.AutoGold = true
	first.AutoSplit = true
	first.SetFollowingBody(leader)
	second.SetFollowingBody(leader)
	first.SetInGroup(true)
	second.SetInGroup(true)
	leader.SetInGroup(true)
	for _, p := range []*Player{leader, first, second} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) failed: %v", p.Name, err)
		}
	}
	// SetFollowingBody alone does not stamp the sequence; use the world API
	// the follow command uses so prepend order is recorded.
	first.setFollowingSequenceForTest(10)
	second.setFollowingSequenceForTest(20)
	output := captureMovementOutput(w)

	w.AwardMobKillXP(first, nil, 9000, 300, 5, 0)

	// 300 over 3 members = 100 each; killer is `first`. The leader (k != ch)
	// is paid first, then the newest follower `second`, then `first`'s own
	// share is kept (src/fight.c:768-832).
	msg := output[first.Name].String()
	wantOrder := []string{
		"You share 100 gold with Leader.\r\n",
		"You share 100 gold with Second.\r\n",
		"You split the gold and keep 100 for yourself.\r\n",
	}
	last := -1
	for _, want := range wantOrder {
		idx := strings.Index(msg, want)
		if idx < 0 {
			t.Fatalf("killer output missing %q; got %q", want, msg)
		}
		if idx < last {
			t.Fatalf("killer output order wrong: %q at %d after %d; got %q", want, idx, last, msg)
		}
		last = idx
	}
	if total := leader.GetGold() + first.GetGold() + second.GetGold(); total != 300 {
		t.Fatalf("distributed gold = %d, want 300", total)
	}
}

func (p *Player) setFollowingSequenceForTest(seq uint64) {
	p.mu.Lock()
	p.followingSequence = seq
	p.mu.Unlock()
}
