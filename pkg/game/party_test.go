package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newXPTestWorld(t *testing.T) *World {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "XP Test Room", Zone: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func TestGainExpUsesCLimitsWithoutPerLevelCap(t *testing.T) {
	w := newXPTestWorld(t)
	p := NewPlayer(1, "Hero", 1001)
	p.SetLevel(10)
	p.SetExp(0)

	w.GainExp(p, 50000)

	if got := p.GetExp(); got != 50000 {
		t.Fatalf("GainExp = %d, want 50000 per src/limits.c gain_exp max_exp_gain cap only", got)
	}
}

func TestAwardMobKillXPUsesGainExpCap(t *testing.T) {
	w := newXPTestWorld(t)
	killer := NewPlayer(1, "Hero", 1001)
	killer.SetLevel(10)
	killer.SetExp(0)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(killer, 250000, 0, 10, 0)

	if got := killer.GetExp(); got != maxExpGain {
		t.Fatalf("kill XP = %d, want %d cap from src/limits.c gain_exp", got, maxExpGain)
	}
}

func TestAwardMobKillXPCanAdvanceOneLevel(t *testing.T) {
	w := newXPTestWorld(t)
	killer := NewPlayer(1, "Hero", 1001)
	killer.SetLevel(10)
	killer.SetExp(FindExp(killer.Class, killer.Level) - 1)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(killer, 200000, 0, 10, 0)

	if got := killer.GetLevel(); got != 11 {
		t.Fatalf("level after kill XP = %d, want 11 via src/limits.c gain_exp", got)
	}
}

func TestCalcKillXPShareMatchesCLevelDiff(t *testing.T) {
	tests := []struct {
		name        string
		chLevel     int
		victimLevel int
		base        int
		inGroup     bool
		want        int
	}{
		{
			name:        "no fight-up bonus",
			chLevel:     10,
			victimLevel: 20,
			base:        1000,
			want:        1000,
		},
		{
			name:        "solo higher-level slack prevents small penalty",
			chLevel:     16,
			victimLevel: 10,
			base:        1000,
			want:        1000,
		},
		{
			name:        "grouped higher-level member gets no solo slack",
			chLevel:     16,
			victimLevel: 10,
			base:        1000,
			inGroup:     true,
			want:        700,
		},
		{
			name:        "over level 20 penalty stacks after level-diff penalty",
			chLevel:     30,
			victimLevel: 10,
			base:        1000,
			want:        240,
		},
		{
			name:        "C truncates after percentage subtraction",
			chLevel:     30,
			victimLevel: 10,
			base:        99,
			want:        23,
		},
		{
			name:        "max_exp_gain cap applies before reductions",
			chLevel:     10,
			victimLevel: 10,
			base:        250000,
			want:        maxExpGain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := combat.CalcXPShare(tt.chLevel, tt.victimLevel, tt.base, tt.inGroup, maxExpGain)
			if got != tt.want {
				t.Fatalf("combat.CalcXPShare(%d, %d, %d, %t) = %d, want %d",
					tt.chLevel, tt.victimLevel, tt.base, tt.inGroup, got, tt.want)
			}
		})
	}
}

func TestAwardMobKillXPDoesNotGrantFightUpBonus(t *testing.T) {
	w := newXPTestWorld(t)
	killer := NewPlayer(1, "Hero", 1001)
	killer.SetLevel(10)
	killer.SetExp(0)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(killer, 1000, 0, 20, 0)

	if got := killer.GetExp(); got != 1000 {
		t.Fatalf("kill XP against higher-level victim = %d, want 1000 with no C fight-up bonus", got)
	}
}

func TestAwardMobKillXPSoloHigherLevelSlack(t *testing.T) {
	w := newXPTestWorld(t)
	killer := NewPlayer(1, "Hero", 1001)
	killer.SetLevel(16)
	killer.SetExp(0)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	w.AwardMobKillXP(killer, 1000, 0, 10, 0)

	if got := killer.GetExp(); got != 1000 {
		t.Fatalf("solo kill XP = %d, want 1000 because C applies two-level solo slack", got)
	}
}

func TestAwardMobKillXPGroupHigherLevelPenalty(t *testing.T) {
	w := newXPTestWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	leader.SetLevel(16)
	leader.SetExp(0)
	leader.SetInGroup(true)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer leader failed: %v", err)
	}

	follower := NewPlayer(2, "Follower", 1001)
	follower.SetLevel(16)
	follower.SetExp(0)
	follower.SetFollowing(leader.Name)
	follower.SetInGroup(true)
	if err := w.AddPlayer(follower); err != nil {
		t.Fatalf("AddPlayer follower failed: %v", err)
	}

	w.AwardMobKillXP(leader, 2000, 0, 10, 0)

	if got := leader.GetExp(); got != 693 {
		t.Fatalf("leader grouped kill XP = %d, want 693", got)
	}
	if got := follower.GetExp(); got != 693 {
		t.Fatalf("follower grouped kill XP = %d, want 693", got)
	}
}

func TestAwardMobKillXPGroupAlignmentShift(t *testing.T) {
	w := newXPTestWorld(t)

	leader := NewPlayer(1, "Leader", 1001)
	leader.SetAlignment(0)
	leader.SetInGroup(true)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer leader failed: %v", err)
	}

	member1 := NewPlayer(2, "Member1", 1001)
	member1.SetAlignment(0)
	member1.SetFollowing(leader.Name)
	member1.SetInGroup(true)
	if err := w.AddPlayer(member1); err != nil {
		t.Fatalf("AddPlayer member1 failed: %v", err)
	}

	member2 := NewPlayer(3, "Member2", 1001)
	member2.SetAlignment(0)
	member2.SetFollowing(leader.Name)
	member2.SetInGroup(true)
	if err := w.AddPlayer(member2); err != nil {
		t.Fatalf("AddPlayer member2 failed: %v", err)
	}

	// Strong evil victim should pull every group member toward positive.
	victimAlign := -1000
	w.AwardMobKillXP(leader, 1000, 0, 10, victimAlign)

	if got := leader.GetAlignment(); got <= 0 {
		t.Fatalf("leader alignment = %d, want > 0 after group kill of evil victim", got)
	}
	if got := member1.GetAlignment(); got <= 0 {
		t.Fatalf("member1 alignment = %d, want > 0 after group kill of evil victim", got)
	}
	if got := member2.GetAlignment(); got <= 0 {
		t.Fatalf("member2 alignment = %d, want > 0 after group kill of evil victim", got)
	}
}
