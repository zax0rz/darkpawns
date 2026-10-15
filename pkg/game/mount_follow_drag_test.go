package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestRiderWalkMovesMountOnceNotTwice pins C perform_move's follower-drag
// rider exclusion (src/act.movement.c:343-352): a mover's follower list in C
// includes the mount mobile the mover rides, and get_rider(k->follower) == ch
// skips that follower so do_simple_move's mount arm (which carries the mount
// and deducts its movement exactly once, act.movement.c:153-162) is its only
// move. Go's player-path follower enumeration is players-only, so the mount
// is structurally excluded from the drag — this test guards that the mount
// still arrives and pays movement exactly once, the observable C behavior
// (R1/R5e/R5h).
func TestRiderWalkMovesMountOnceNotTwice(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Yard", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Hall", Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
		},
		Mobs: []parser.Mob{{
			VNum: 99071, Keywords: "horse", ShortDesc: "a horse",
			LongDesc: "A horse stands here.", Level: 10,
			HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 40},
			Alignment: 0, Race: 1,
			Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	rider := NewPlayer(1, "Rider", 1001)
	rider.SetLevel(10)
	rider.SetPosition(combat.PosStanding)
	rider.SetMove(100)
	rider.SetMaxMove(100)
	if err := w.AddPlayer(rider); err != nil {
		t.Fatal(err)
	}
	mount, err := w.SpawnMob(99071, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mount.SetPosition(combat.PosStanding)
	mount.SetMove(100)
	// Ride: both bodies carry AFF_MOUNT and the mount follows its rider
	// through the body registry (C mounts follow their riders, so the
	// rider's follower list contains the mount in C).
	rider.SetAffect(affMounted, true)
	rider.MountName = mount.GetName()
	mount.SetAffected(affMounted)
	mount.SetFollowingBody(rider)

	before := mount.GetMove()
	if !performMoveResult(w, rider, 0, false, nil) {
		t.Fatal("mounted rider's move failed")
	}
	if got := mount.GetRoom(); got != 1002 {
		t.Fatalf("mount room = %d, want carried to 1002", got)
	}
	if got := rider.GetRoom(); got != 1002 {
		t.Fatalf("rider room = %d, want 1002", got)
	}
	// do_simple_move deducts the mount's movement exactly once
	// (act.movement.c:159-162); a follower-drag double move would deduct
	// twice. City-sector loss for both rooms is movementLoss[SECT_CITY]
	// averaged with itself.
	once := (movementLoss[0] + movementLoss[0]) >> 1
	if before-once < 0 {
		once = before
	}
	if got := mount.GetMove(); got != before-once {
		t.Fatalf("mount move points = %d, want exactly one deduction to %d", got, before-once)
	}
}
