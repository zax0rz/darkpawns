package game

import (
	"strings"
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

func TestGetFollowerActorsUsesCPrependOrder(t *testing.T) {
	w := newXPTestWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	first := NewPlayer(2, "First", 1001)
	second := NewPlayer(3, "Second", 1001)
	for _, player := range []*Player{leader, first, second} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}

	first.SetFollowing(leader.Name)
	second.SetFollowing(leader.Name)
	followers := w.GetFollowerActors(leader.Name)
	if len(followers) != 2 {
		t.Fatalf("follower count = %d, want 2", len(followers))
	}
	if got := followers[0].GetName(); got != second.Name {
		t.Errorf("first follower = %q, want latest follower %q", got, second.Name)
	}
	if got := followers[1].GetName(); got != first.Name {
		t.Errorf("second follower = %q, want earlier follower %q", got, first.Name)
	}
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

	w.AwardMobKillXP(killer, nil, 250000, 0, 10, 0)

	if got := killer.GetExp(); got != maxExpGain {
		t.Fatalf("kill XP = %d, want %d cap from src/limits.c gain_exp", got, maxExpGain)
	}
}

func TestUngroupedFollowerKillUsesSoloXP(t *testing.T) {
	w := newXPTestWorld(t)
	leader := NewPlayer(1, "Leader", 1001)
	killer := NewPlayer(2, "Killer", 1001)
	leader.SetInGroup(true)
	killer.SetFollowingBody(leader)
	for _, player := range []*Player{leader, killer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}
	output := captureMovementOutput(w)

	w.AwardMobKillXP(killer, nil, 1000, 0, killer.GetLevel(), 0)

	if got := killer.GetExp(); got != 1000 {
		t.Fatalf("ungrouped killer XP = %d, want solo share 1000", got)
	}
	if got := leader.GetExp(); got != 0 {
		t.Fatalf("grouped leader XP = %d, want 0 for an ungrouped killer", got)
	}
	if got := output[killer.Name].String(); !strings.Contains(got, "You receive 1000 experience points.\r\n") {
		t.Fatalf("killer output = %q, want solo XP message", got)
	}
	if got := output[leader.Name]; got != nil && got.String() != "" {
		t.Fatalf("leader output = %q, want no XP message", got.String())
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

	w.AwardMobKillXP(killer, nil, 200000, 0, 10, 0)

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

	w.AwardMobKillXP(killer, nil, 1000, 0, 20, 0)

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

	w.AwardMobKillXP(killer, nil, 1000, 0, 10, 0)

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
	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }

	w.AwardMobKillXP(leader, nil, 2000, 0, 10, 0)

	if got := leader.GetExp(); got != 693 {
		t.Fatalf("leader grouped kill XP = %d, want 693", got)
	}
	if got := follower.GetExp(); got != 693 {
		t.Fatalf("follower grouped kill XP = %d, want 693", got)
	}
	for _, name := range []string{leader.Name, follower.Name} {
		if !strings.Contains(output[name], "You receive your share of experience -- 693 points.\r\n") {
			t.Errorf("%s output = %q, missing exact numeric group XP line", name, output[name])
		}
	}
}

func TestAwardMobKillXPSplitsOddGoldRemainderSeparately(t *testing.T) {
	w, leader, victim := newDeathOrderWorld(t, 10, 100)
	leader.SetInGroup(true)
	leader.AutoGold = true
	leader.AutoSplit = true
	follower := NewPlayer(2, "Follower", 1001)
	follower.SetFollowing(leader.Name)
	follower.SetInGroup(true)
	if err := w.AddPlayer(follower); err != nil {
		t.Fatalf("AddPlayer follower: %v", err)
	}
	victim.SetGold(101)

	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }
	w.AwardMobKillXP(leader, victim, 0, 101, victim.GetLevel(), 0)
	for _, name := range []string{leader.Name, follower.Name} {
		if !strings.Contains(output[name], "You receive your share of experience -- one measly little point!\r\n") {
			t.Errorf("%s output = %q, missing exact one-point group XP line", name, output[name])
		}
	}

	if got := leader.GetGold(); got != 51 {
		t.Errorf("leader gold = %d, want 51", got)
	}
	if got := follower.GetGold(); got != 50 {
		t.Errorf("follower gold = %d, want 50", got)
	}
	for _, want := range []string{
		"You loot 101 coins from the corpse of a victim.\r\n",
		"You split the gold and keep 50 for yourself.\r\n",
		"You keep 1 gold leftover.\r\n",
	} {
		if !strings.Contains(output[leader.Name], want) {
			t.Errorf("leader output %q does not contain %q", output[leader.Name], want)
		}
	}
	if strings.Contains(output[leader.Name], "You split the gold and keep 51 for yourself.") {
		t.Errorf("leader output combined the equal share and remainder: %q", output[leader.Name])
	}
	roomLoot := "Killer loots some gold from the corpse of a victim.\r\n"
	followerShare := "Killer splits some gold with you, you get 50.\r\n"
	if !strings.Contains(output[follower.Name], roomLoot) || !strings.Contains(output[follower.Name], followerShare) {
		t.Errorf("follower output = %q, want room loot and 50-gold receipt", output[follower.Name])
	}
	if strings.Index(output[follower.Name], roomLoot) > strings.Index(output[follower.Name], followerShare) {
		t.Errorf("follower received gold before the corpse-loot room act: %q", output[follower.Name])
	}
}

func TestAwardMobKillXPGroupGoldZeroShareMessages(t *testing.T) {
	w, leader, victim := newDeathOrderWorld(t, 10, 100)
	leader.SetInGroup(true)
	leader.AutoGold = true
	leader.AutoSplit = true
	followers := []*Player{NewPlayer(2, "FollowerOne", 1001), NewPlayer(3, "FollowerTwo", 1001)}
	for _, follower := range followers {
		follower.SetFollowing(leader.Name)
		follower.SetInGroup(true)
		if err := w.AddPlayer(follower); err != nil {
			t.Fatalf("AddPlayer follower: %v", err)
		}
	}
	victim.SetGold(1)
	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }

	w.AwardMobKillXP(leader, victim, 0, 1, victim.GetLevel(), 0)

	if !strings.Contains(output[leader.Name], "When you split no gold, you got none.\r\n") {
		t.Errorf("leader output = %q, missing zero-share self line", output[leader.Name])
	}
	for _, follower := range followers {
		if !strings.Contains(output[leader.Name], "You would share gold with "+follower.Name+", but there was none to split!\r\n") {
			t.Errorf("leader output = %q, missing warning for %s", output[leader.Name], follower.Name)
		}
		if !strings.Contains(output[follower.Name], leader.Name+" would have shared some gold with you but there was none to split!\r\n") {
			t.Errorf("%s output = %q, missing exact zero-share receipt", follower.Name, output[follower.Name])
		}
	}
}

func TestAwardMobKillXPGroupGoldWithoutAutosplitNotifiesRoom(t *testing.T) {
	w, leader, victim := newDeathOrderWorld(t, 10, 100)
	leader.SetInGroup(true)
	leader.AutoGold = true
	follower := NewPlayer(2, "Follower", 1001)
	follower.SetFollowing(leader.Name)
	follower.SetInGroup(true)
	if err := w.AddPlayer(follower); err != nil {
		t.Fatalf("AddPlayer follower: %v", err)
	}
	victim.SetGold(7)
	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }

	w.AwardMobKillXP(leader, victim, 0, 7, victim.GetLevel(), 0)

	if want := "You loot 7 coins from the corpse of a victim.\r\n"; !strings.Contains(output[leader.Name], want) {
		t.Errorf("leader output = %q, missing %q", output[leader.Name], want)
	}
	if want := "Killer loots some gold from the corpse of a victim.\r\n"; !strings.Contains(output[follower.Name], want) {
		t.Errorf("follower output = %q, missing room announcement %q", output[follower.Name], want)
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
	w.AwardMobKillXP(leader, nil, 1000, 0, 10, victimAlign)

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
