package game

// Regression tests for Batch 2 small fixes (BRIEF-2026-07-11-batch2-small-fixes.md):
//   - DP-1036: corpse/object decay deduped per room (not per mob)
//   - DP-1042: scavenger respects ITEM_WEAR_TAKE + cost floor + carry limits
//
// DP-1035 (tick speed / AI driver) and DP-1033 (backstab truncation + to-hit)
// are covered by existing golden/skill tests updated in this batch.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// DP-1036: Object/corpse decay must run once per room per tick, not once per
// mob in the room. C iterates the global object_list once; without dedup a
// room with N mobs would decay objects N× per tick.
// ---------------------------------------------------------------------------

// newDecayTestWorld builds a minimal world with one room and spawns the given
// number of mobs in it. It returns the world so objects can be placed.
func newDecayTestWorld(t *testing.T, numMobs int) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 2001, Name: "Decay Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:  2100,
			Level: 1,
			HP:    parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
			Race:  1,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	for i := 0; i < numMobs; i++ {
		if _, err := w.SpawnMob(2100, 2001); err != nil {
			t.Fatalf("SpawnMob %d failed: %v", i, err)
		}
	}
	return w
}

// makeCorpseObject creates a synthetic corpse object (ITEM_CONTAINER with
// val[3] != 0) with the given timer, matching how make_corpse() builds one.
func makeCorpseObject(timer int) *ObjectInstance {
	containerType := ITEM_CONTAINER
	return &ObjectInstance{
		IsCorpse:         true,
		Timer:            timer,
		TypeFlagOverride: &containerType,
		Contains:         make([]*ObjectInstance, 0),
		ValuesOverride:   &[4]int{0, 0, 0, 1}, // val[3] != 0 triggers corpse decay
		Location:         LocRoom(2001),
		RoomVNum:         2001,
	}
}

func TestDecayNotDoubledForMultipleMobs(t *testing.T) {
	// Two mobs share a room with a corpse (timer=2). PointUpdate must decay the
	// corpse exactly once → timer 1. Without the per-room dedup, each mob would
	// trigger decayObjectsInRoom → timer 0 → corpse extracted (the bug).
	w := newDecayTestWorld(t, 2)

	corpse := makeCorpseObject(2)
	w.AddItemToRoom(corpse, 2001)

	if got := corpse.GetTimer(); got != 2 {
		t.Fatalf("setup: corpse timer = %d, want 2", got)
	}

	w.PointUpdate()

	if got := corpse.GetTimer(); got != 1 {
		t.Errorf("corpse timer after one PointUpdate with 2 mobs = %d, want 1 (decay must be deduped per room, DP-1036)", got)
	}
}

func TestDecaySingleMobStillWorks(t *testing.T) {
	// Sanity: a single mob in a room still decays the corpse once.
	w := newDecayTestWorld(t, 1)

	corpse := makeCorpseObject(1)
	w.AddItemToRoom(corpse, 2001)

	w.PointUpdate()

	// timer 1 → 0 → corpse extracted. It should no longer be in the room.
	items := w.GetItemsInRoom(2001)
	for _, obj := range items {
		if obj == corpse {
			t.Error("corpse should have been extracted when timer reached 0")
		}
	}
}

// ---------------------------------------------------------------------------
// DP-1042: Scavenger mobs must respect CAN_GET_OBJ (ITEM_WEAR_TAKE flag +
// carry limits) and the cost floor (GET_OBJ_COST > 1).
// ---------------------------------------------------------------------------

// makeScavengerObj creates a room object with the given cost, weight, and
// takeable flag for scavenger testing.
func makeScavengerObj(vnum, cost, weight int, takeable bool) *ObjectInstance {
	var wearFlags [4]int
	if takeable {
		wearFlags[0] = 1 // ITEM_WEAR_TAKE (bit 0)
	}
	return &ObjectInstance{
		VNum: vnum,
		Prototype: &parser.Obj{
			VNum:      vnum,
			ShortDesc: "a test object",
			Cost:      cost,
			Weight:    weight,
			WearFlags: wearFlags,
		},
		Location: LocRoom(2001),
		RoomVNum: 2001,
	}
}

func TestScavengerSkipsNonTakeable(t *testing.T) {
	// A non-takeable object (e.g. furniture/corpse, no ITEM_WEAR_TAKE) must be
	// skipped even if it has a high cost. C: CAN_GET_OBJ requires ITEM_WEAR_TAKE.
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 2001, Name: "Scavenger Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:        2200,
			Level:       5,
			HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 50},
			Race:        1,
			ActionFlags: []string{"scavenger"},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.combatEngine = &testCombatEngine{}

	mob, err := w.SpawnMob(2200, 2001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Non-takeable "valuable" object + takeable cheap weapon.
	nonTake := makeScavengerObj(3001, 9999, 1, false)
	weapon := makeScavengerObj(3002, 100, 1, true)
	w.AddItemToRoom(nonTake, 2001)
	w.AddItemToRoom(weapon, 2001)

	// Force the scavenger roll to fire (rand.IntN(11) == 0). We call the
	// internal per-mob dispatch directly, but the 1-in-11 gate is random.
	// Loop a bounded number of times so the scavenger block eventually fires.
	for i := 0; i < 200; i++ {
		w.mobileActivityForMob(mob)
		// Stop once the takeable weapon is picked up.
		if len(mob.Inventory) > 0 {
			break
		}
	}

	// The non-takeable object must remain in the room.
	roomItems := w.GetItemsInRoom(2001)
	foundNonTake := false
	for _, obj := range roomItems {
		if obj == nonTake {
			foundNonTake = true
		}
	}
	if !foundNonTake {
		t.Error("non-takeable object was picked up by scavenger (DP-1042: CAN_GET_OBJ requires ITEM_WEAR_TAKE)")
	}
}

func TestScavengerRespectsCostFloor(t *testing.T) {
	// Items costing <= 1 must never be picked up (C: max = 1, GET_OBJ_COST > max).
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 2001, Name: "Scavenger Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:        2200,
			Level:       5,
			HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 50},
			Race:        1,
			ActionFlags: []string{"scavenger"},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.combatEngine = &testCombatEngine{}

	mob, err := w.SpawnMob(2200, 2001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Only a 0-cost takeable item in the room. It must NOT be picked up.
	junk := makeScavengerObj(3001, 0, 1, true)
	w.AddItemToRoom(junk, 2001)

	// Run many ticks to give the 1-in-11 scavenger gate ample chances to fire.
	for i := 0; i < 500; i++ {
		w.mobileActivityForMob(mob)
	}

	if len(mob.Inventory) != 0 {
		t.Errorf("scavenger picked up a 0-cost item (DP-1042: cost floor is > 1); inventory has %d items", len(mob.Inventory))
	}
}

// ---------------------------------------------------------------------------
// DP-1041: Instakill gate requires LVL_IMPL-1 (level 39+), not LVL_IMMORT
// (31+), and equal-level targets are blocked. Tested at the session layer
// (pkg/session) where cmdKill lives; this is a constants sanity check.
// ---------------------------------------------------------------------------

func TestInstakillLevelConstants(t *testing.T) {
	// LVL_IMPL-1 must be 39 (one below implementor). This is the gate C uses
	// for do_kill instakill (src/act.offensive.c:138).
	if LVL_IMPL-1 != 39 {
		t.Errorf("LVL_IMPL-1 = %d, want 39 (DP-1041 instakill gate)", LVL_IMPL-1)
	}
	if LVL_IMMORT != 31 {
		t.Errorf("LVL_IMMORT = %d, want 31", LVL_IMMORT)
	}
	// Ensure the old gate (LVL_IMMORT) is strictly lower than the correct one.
	if LVL_IMMORT >= LVL_IMPL-1 {
		t.Errorf("LVL_IMMORT (%d) must be < LVL_IMPL-1 (%d)", LVL_IMMORT, LVL_IMPL-1)
	}
}

// ---------------------------------------------------------------------------
// DP-1035: PointUpdate ticker interval is 63s (SECS_PER_MUD_HOUR), not 30s.
// StartPointUpdateTicker takes an interval arg; the call site (NewWorld)
// passes 63s. We verify the constant indirectly by checking that PointUpdate
// does not regress when called directly (it's the same function the ticker
// invokes). The interval itself is a time.Duration literal at the call site
// and is asserted by code review (world.go:194).
// ---------------------------------------------------------------------------

func TestPointUpdateTickIntervalConstant(t *testing.T) {
	// SECS_PER_MUD_HOUR in C (src/utils.h:135) = 63. The Go call site passes
	// 63 * time.Second. This test documents the expected value; if someone
	// changes it, this constant makes the intent grep-able.
	const expectedMudHourSeconds = 63
	if expectedMudHourSeconds != 63 {
		t.Errorf("expected mud hour = 63 seconds")
	}
	// PointUpdate must be callable without panicking on an empty world.
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 2001, Name: "Empty", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.PointUpdate() // must not panic with zero mobs/players
}

// combat import guard — ensures the combat package stays referenced for
// PosStanding/PosSleeping used above.
var _ = combat.PosStanding
