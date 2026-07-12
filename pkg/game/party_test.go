package game

import (
	"testing"

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

	w.AwardMobKillXP(killer, 250000, 0, 10)

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

	w.AwardMobKillXP(killer, 200000, 0, 10)

	if got := killer.GetLevel(); got != 11 {
		t.Fatalf("level after kill XP = %d, want 11 via src/limits.c gain_exp", got)
	}
}
