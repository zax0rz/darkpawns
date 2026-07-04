package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobFlagComparisonCaseInsensitive(t *testing.T) {
	// Simulate parser output: flags stored as UPPERCASE
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"SENTINEL", "STAY_ZONE"}}}

	// This should find the flag — if it doesn't, the lowercase comparison is wrong
	found := false
	for _, f := range mob.Prototype.ActionFlags {
		if strings.EqualFold(f, "sentinel") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("SENTINEL flag not found with case-insensitive comparison")
	}
}

func TestMobStayZonePreventsWander(t *testing.T) {
	// A STAY_ZONE mob should be recognized by hasMobFlag regardless of case.
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"STAY_ZONE"}}}
	if !hasMobFlag(mob, "stay_zone") {
		t.Fatal("STAY_ZONE flag not found with case-insensitive hasMobFlag")
	}
}

func TestHasMobFlagCaseInsensitive(t *testing.T) {
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"AGGRESSIVE", "SENTINEL", "STAY_ZONE"}}}

	if !hasMobFlag(mob, "aggressive") {
		t.Error("hasMobFlag failed for lowercase 'aggressive' against uppercase 'AGGRESSIVE'")
	}
	if !hasMobFlag(mob, "SENTINEL") {
		t.Error("hasMobFlag failed for uppercase 'SENTINEL' against uppercase 'SENTINEL'")
	}
	if !hasMobFlag(mob, "Stay_Zone") {
		t.Error("hasMobFlag failed for mixed-case 'Stay_Zone' against uppercase 'STAY_ZONE'")
	}
	if hasMobFlag(mob, "wimpy") {
		t.Error("hasMobFlag falsely found 'wimpy'")
	}
}

func TestRoomHasFlagCaseInsensitive(t *testing.T) {
	room := &parser.Room{Flags: []string{"DEATH", "NO_MOB"}}
	if !roomHasFlag(room, "death") {
		t.Error("roomHasFlag failed for lowercase 'death' against uppercase 'DEATH'")
	}
	if !roomHasFlag(room, "NO_MOB") {
		t.Error("roomHasFlag failed for uppercase 'NO_MOB' against uppercase 'NO_MOB'")
	}
	if roomHasFlag(room, "peaceful") {
		t.Error("roomHasFlag falsely found 'peaceful'")
	}
}

// newWanderTestWorld builds a minimal world with two same-zone rooms linked
// north/south, for wander tests. Room 1 --north--> Room 2 (and south back).
func newWanderTestWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1, Name: "Room One", Zone: 80,
				Exits: map[string]parser.Exit{"north": {ToRoom: 2}},
			},
			{
				VNum: 2, Name: "Room Two", Zone: 80,
				Exits: map[string]parser.Exit{"south": {ToRoom: 1}},
			},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// TestWanderMobRespectsStayZone confirms a STAY_ZONE mob never crosses into a
// different zone (DP-908). Build two rooms in different zones; the mob must
// stay put even though an exit exists.
func TestWanderMobRespectsStayZone(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 10, Name: "Home", Zone: 80, Exits: map[string]parser.Exit{"north": {ToRoom: 20}}},
			{VNum: 20, Name: "Other Zone", Zone: 99, Exits: map[string]parser.Exit{"south": {ToRoom: 10}}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob := NewMobInstance(&parser.Mob{ActionFlags: []string{"STAY_ZONE"}}, 10)
	for i := 0; i < 500; i++ {
		w.wanderMob(mob)
		if got := mob.GetRoom(); got != 10 {
			t.Fatalf("STAY_ZONE mob left its zone: now in room %d after %d ticks", got, i+1)
		}
	}
}

// TestWanderMobMovesWithinConstraints confirms a non-sentinel, non-stay-zone
// mob only ever lands in a connected room (1 or 2), and that it moves at least
// once over many ticks — exercising the single-draw gate (DP-908).
func TestWanderMobMovesWithinConstraints(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMobInstance(&parser.Mob{Keywords: "rat", ShortDesc: "a rat"}, 1)

	moved := false
	valid := map[int]bool{1: true, 2: true}
	for i := 0; i < 2000; i++ {
		w.wanderMob(mob)
		got := mob.GetRoom()
		if !valid[got] {
			t.Fatalf("mob wandered to invalid room %d", got)
		}
		if got != 1 {
			moved = true
		}
	}
	if !moved {
		t.Fatal("mob never moved over 2000 ticks; single-draw gate appears too strict")
	}
}

// TestRunMobAISentinelNeverWanders confirms SENTINEL mobs are skipped before
// wandering (the guard lives in mobileActivityForMob, the single wander entry
// point after DP-908). A sentinel mob must stay put across many AI ticks.
func TestRunMobAISentinelNeverWanders(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMobInstance(&parser.Mob{Keywords: "guard", ShortDesc: "a cityguard", ActionFlags: []string{"SENTINEL"}}, 1)
	mob.SetAlive(true)
	mob.SetStatus("standing")

	for i := 0; i < 500; i++ {
		w.runMobAI(mob)
		if got := mob.GetRoom(); got != 1 {
			t.Fatalf("SENTINEL mob moved to room %d after %d ticks", got, i+1)
		}
	}
}
