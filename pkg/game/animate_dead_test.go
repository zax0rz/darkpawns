package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newAnimateTestWorld creates a minimal world with one room.
func newAnimateTestWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 10, ShortDesc: "a zombie", Keywords: "zombie"}},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// newAnimatePlayer creates a player with the given charisma and adds them to w.
func newAnimatePlayer(t *testing.T, w *World, name string, cha int) *Player {
	t.Helper()
	p := NewPlayer(1, name, 1001)
	p.Stats.Cha = cha
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s) failed: %v", name, err)
	}
	return p
}

func TestCanRaiseUndeadI_CharmedCasterBlocked(t *testing.T) {
	w := newAnimateTestWorld(t)
	p := newAnimatePlayer(t, w, "Necro", 18)
	p.SetAffect(affCharm, true)

	ok, msg := w.CanRaiseUndeadI(p)
	if ok {
		t.Error("expected charmed caster to be blocked")
	}
	if !strings.Contains(msg, "too giddy") {
		t.Errorf("expected giddy message, got %q", msg)
	}
}

func TestCanRaiseUndeadI_FollowerCapBlocked(t *testing.T) {
	w := newAnimateTestWorld(t)
	// CHA 10 -> cap 5 followers.
	leader := newAnimatePlayer(t, w, "Necro", 10)

	// Add 3 player followers.
	for i := 0; i < 3; i++ {
		f := NewPlayer(i+10, "follower-"+string(rune('a'+i)), 1001)
		if err := w.AddPlayer(f); err != nil {
			t.Fatalf("AddPlayer failed: %v", err)
		}
		AddFollowerQuiet(f, leader)
	}

	// Add 2 charmed mob followers (still under cap).
	for i := 0; i < 2; i++ {
		m, err := w.SpawnMob(10, 1001)
		if err != nil {
			t.Fatalf("SpawnMob failed: %v", err)
		}
		m.SetAffected(affCharm)
		m.SetFollowing(leader.Name)
	}

	// At exactly CHA/2 == 5 followers, should be blocked.
	ok, msg := w.CanRaiseUndeadI(leader)
	if ok {
		t.Error("expected caster at follower cap to be blocked")
	}
	if !strings.Contains(msg, "can't have any more followers") {
		t.Errorf("expected cap message, got %q", msg)
	}
}

func TestCanRaiseUndeadI_UnderCap(t *testing.T) {
	w := newAnimateTestWorld(t)
	p := newAnimatePlayer(t, w, "Necro", 18)

	ok, msg := w.CanRaiseUndeadI(p)
	if !ok {
		t.Errorf("expected caster to be allowed, got message %q", msg)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestCharmAndFollowI(t *testing.T) {
	w := newAnimateTestWorld(t)
	leader := newAnimatePlayer(t, w, "Necro", 18)

	mob, err := w.SpawnMob(10, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	w.CharmAndFollowI(mob, leader)

	if !mob.IsAffected(affCharm) {
		t.Error("expected zombie to be charmed")
	}
	if mob.GetFollowing() != leader.Name {
		t.Errorf("expected zombie to follow %q, got %q", leader.Name, mob.GetFollowing())
	}
}

func TestCharmAndFollowI_MobLeaderCharmsButDoesNotFollow(t *testing.T) {
	w := newAnimateTestWorld(t)

	mob, err := w.SpawnMob(10, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	leaderMob, err := w.SpawnMob(10, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	w.CharmAndFollowI(mob, leaderMob)

	if !mob.IsAffected(affCharm) {
		t.Error("expected zombie to be charmed even with mob leader")
	}
	// Mob leaders are not yet supported for the follow relationship.
	if mob.GetFollowing() != "" {
		t.Errorf("expected no following with mob leader, got %q", mob.GetFollowing())
	}
}
