package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobFlagComparisonCaseInsensitive(t *testing.T) {
	// Simulate parser output: flags stored as UPPERCASE
	mob := &MobInstance{}
	mob.SetProto(&parser.Mob{ActionFlags: []string{"SENTINEL", "STAY_ZONE"}})

	// This should find the flag — if it doesn't, the lowercase comparison is wrong
	found := false
	for _, f := range mob.Proto().ActionFlags {
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
	mob := &MobInstance{}
	setProtoWithFlags(mob, &parser.Mob{ActionFlags: []string{"STAY_ZONE"}})
	if !hasMobFlag(mob, "stay_zone") {
		t.Fatal("STAY_ZONE flag not found with case-insensitive hasMobFlag")
	}
}

func TestHasMobFlagCaseInsensitive(t *testing.T) {
	mob := &MobInstance{}
	setProtoWithFlags(mob, &parser.Mob{ActionFlags: []string{"AGGRESSIVE", "SENTINEL", "STAY_ZONE"}})

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

	mob := NewMob(&parser.Mob{ActionFlags: []string{"STAY_ZONE"}}, 10)
	for i := 0; i < 500; i++ {
		w.wanderMobWithDoor(mob, 0)
		if got := mob.GetRoom(); got != 10 {
			t.Fatalf("STAY_ZONE mob left its zone: now in room %d after %d ticks", got, i+1)
		}
	}
}

// TestWanderMobMovesWithinConstraints confirms a non-sentinel, non-stay-zone
// mob only ever lands in a connected room (1 or 2). The caller supplies each
// door because mobileActivityForMob now owns the unconditional draw (DP-1170).
func TestWanderMobMovesWithinConstraints(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMob(&parser.Mob{Keywords: "rat", ShortDesc: "a rat"}, 1)

	valid := map[int]bool{1: true, 2: true}
	for i := 0; i < 20; i++ {
		door := 0 // north from room 1
		if mob.GetRoom() == 2 {
			door = 2 // south from room 2
		}
		w.wanderMobWithDoor(mob, door)
		got := mob.GetRoom()
		if !valid[got] {
			t.Fatalf("mob wandered to invalid room %d", got)
		}
	}
}

// TestRunMobAISentinelNeverWanders confirms SENTINEL mobs are skipped before
// wandering (the guard lives in mobileActivityForMob, the single wander entry
// point after DP-908). A sentinel mob must stay put across many AI ticks.
func TestRunMobAISentinelNeverWanders(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMob(&parser.Mob{Keywords: "guard", ShortDesc: "a cityguard", ActionFlags: []string{"SENTINEL"}}, 1)
	mob.SetAlive(true)
	mob.SetStatus("standing")

	for i := 0; i < 500; i++ {
		w.runMobAI(mob)
		if got := mob.GetRoom(); got != 1 {
			t.Fatalf("SENTINEL mob moved to room %d after %d ticks", got, i+1)
		}
	}
}

// TestWanderMobStopsAtClosedDoor proves DP-1428's door gate: C's CAN_GO
// refuses an EX_CLOSED exit (src/utils.h:565-567; src/mobact.c:124), so a
// wandering mob stays put behind a closed door and moves again once it opens.
func TestWanderMobStopsAtClosedDoor(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMob(&parser.Mob{Keywords: "rat", ShortDesc: "a rat"}, 1)

	room := w.GetRoomInWorld(1)
	north := room.Exits["north"]
	north.ExitInfo |= parser.ExitIsDoor | parser.ExitClosed
	room.Exits["north"] = north

	if w.wanderMobWithDoor(mob, 0) {
		t.Fatal("a closed door must not count as a sector-gate continue")
	}
	if got := mob.GetRoom(); got != 1 {
		t.Fatalf("mob walked through a closed door into room %d", got)
	}

	north.ExitInfo &^= parser.ExitClosed
	room.Exits["north"] = north
	w.wanderMobWithDoor(mob, 0)
	if got := mob.GetRoom(); got != 2 {
		t.Fatalf("mob did not wander through the opened door: room %d", got)
	}
}

// newNoswimWanderWorld is newWanderTestWorld with the source room in
// SECT_WATER_NOSWIM and the destination a FIELD, mirroring room 4611 (the
// Swamp of Sadness) --north--> room 4601 in lib/world/wld/46.wld.
func newNoswimWanderWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 10, Name: "Swamp", Zone: 46, Sector: SECT_WATER_NOSWIM,
				Exits: map[string]parser.Exit{"north": {ToRoom: 20}},
			},
			{
				VNum: 20, Name: "Field", Zone: 46, Sector: 2,
				Exits: map[string]parser.Exit{"south": {ToRoom: 10}},
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

// TestWanderMobNeedsBoatInNoswimRoom proves perform_move's first do_simple_move
// gate (src/act.movement.c:126-132): "if this room or the one we're going to
// needs a boat, check for one". mobact's own gate list (src/mobact.c:121-138)
// only inspects the *destination* sector, so a port that stops there walks a
// boatless mob out of SECT_WATER_NOSWIM. C refuses the step, which is why the
// oracle's Spandaramet (mob 4600) stays in room 4611 forever and takes
// room_activity's 25 drowning damage every pulse while the port moved it to the
// field room 4621 (R1, R3a).
func TestWanderMobNeedsBoatInNoswimRoom(t *testing.T) {
	w := newNoswimWanderWorld(t)

	boatless := NewMob(&parser.Mob{Keywords: "naga", ShortDesc: "a naga"}, 10)
	if w.wanderMobWithDoor(boatless, 0) {
		t.Fatal("a boatless noswim step must not report a sector-gate continue")
	}
	if got := boatless.GetRoom(); got != 10 {
		t.Fatalf("boatless mob left the WATER_NOSWIM room for %d", got)
	}

	// has_boat returns 1 for AFF_WATERWALK (src/act.movement.c:62-63), so the
	// same step is legal for a waterwalking body.
	swimmer := NewMob(&parser.Mob{Keywords: "naga", ShortDesc: "a naga"}, 10)
	swimmer.SetAffected(affWaterWalk)
	w.wanderMobWithDoor(swimmer, 0)
	if got := swimmer.GetRoom(); got != 20 {
		t.Fatalf("a waterwalking mob did not make the legal step: room %d", got)
	}
}

// TestWanderMobMovesFromDryRoom is the revert control for
// TestWanderMobNeedsBoatInNoswimRoom: from a FIELD room the identical step must
// still land, so the boat gate is a water gate and not a blanket move refusal.
func TestWanderMobMovesFromDryRoom(t *testing.T) {
	w := newWanderTestWorld(t)
	mob := NewMob(&parser.Mob{Keywords: "rat", ShortDesc: "a rat"}, 1)

	w.wanderMobWithDoor(mob, 0) // north out of the field room 1
	if got := mob.GetRoom(); got != 2 {
		t.Fatalf("mob did not wander from a dry room: room %d", got)
	}
}
