package game

// Regression tests for GLM Batch A (BRIEF-2026-07-11-glm-batch-a.md):
//   - DP-1040: counter_procs C fall-through (including the default clause)
//   - DP-1038: carry weight (CAN_CARRY_W) enforced in Inventory.addItem
//
// DP-1043 (damage tier boundaries) is tested in pkg/combat/damage_messages_golden_test.go.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// DP-1040: counter_procs fall-through
//
// C switch(number(1,3)) with fall-through AND a default clause:
//   roll 1: +2 hit, +1 mana, +1 move  (case 1→2→3→default)
//   roll 2: +1 hit, +1 mana, +1 move  (case 2→3→default)
//   roll 3: +1 hit, +1 move           (case 3→default, no mana)
// ---------------------------------------------------------------------------

func newCounterProcsTestWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Counter Procs Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func TestCounterProcs_BoostMilestone(t *testing.T) {
	// At a boost milestone (1000 kills), stats must increase. Over many runs
	// all three stats (hit, mana, move) should increase at least once because
	// every roll gives +move and +hit (via default), and roll 1 or 2 gives
	// +mana.
	w := newCounterProcsTestWorld(t)
	p := NewPlayer(1, "Hero", 1001)
	p.MaxHealth = 100
	p.MaxMana = 100
	p.MaxMove = 100
	p.Health = 50

	sawManaIncrease := false
	sawHitIncrease2 := false // roll 1 gives +2 hit (not just +1)
	for i := 0; i < 200; i++ {
		preHit := p.MaxHealth
		preMana := p.MaxMana
		preMove := p.MaxMove
		// Reset so we can measure each call independently.
		p.MaxHealth = 100
		p.MaxMana = 100
		p.MaxMove = 100

		w.counter_procs(p, 1000)

		hitGain := p.MaxHealth - 100
		manaGain := p.MaxMana - 100
		moveGain := p.MaxMove - 100

		// Every roll gives +1 move and +1 hit (default clause).
		if moveGain < 1 {
			t.Errorf("roll %d: expected +move, got move gain %d", i, moveGain)
		}
		if hitGain < 1 {
			t.Errorf("roll %d: expected +hit (default), got hit gain %d", i, hitGain)
		}
		if hitGain >= 2 {
			sawHitIncrease2 = true
		}
		if manaGain >= 1 {
			sawManaIncrease = true
		}
		_ = preHit
		_ = preMana
		_ = preMove
	}

	// Over 200 runs, roll 1 (1/3 chance) should appear, giving +2 hit.
	if !sawHitIncrease2 {
		t.Error("expected to see +2 hit (roll 1) at least once in 200 iterations")
	}
	// Over 200 runs, roll 1 or 2 (2/3 chance) should appear, giving +1 mana.
	if !sawManaIncrease {
		t.Error("expected to see +1 mana (roll 1 or 2) at least once in 200 iterations")
	}
}

func TestCounterProcs_Roll3NoMana(t *testing.T) {
	// Roll 3 gives +1 hit, +1 move, NO mana. We can't force roll 3 with the
	// production roller, but over 200 iterations we can verify that sometimes
	// mana does NOT increase (roll 3), proving the fall-through is correct.
	w := newCounterProcsTestWorld(t)
	p := NewPlayer(1, "Hero", 1001)
	p.MaxHealth = 100
	p.MaxMana = 100
	p.MaxMove = 100

	sawNoManaIncrease := false
	for i := 0; i < 200; i++ {
		p.MaxHealth = 100
		p.MaxMana = 100
		p.MaxMove = 100

		w.counter_procs(p, 1000)

		if p.MaxMana == 100 {
			sawNoManaIncrease = true
			break
		}
	}
	if !sawNoManaIncrease {
		t.Error("expected to see at least one roll with no mana increase (roll 3) in 200 iterations")
	}
}

func TestCounterProcs_HealsToFull(t *testing.T) {
	// C: GET_HIT(ch) = GET_MAX_HIT(ch) after the boost — player is healed.
	w := newCounterProcsTestWorld(t)
	p := NewPlayer(1, "Hero", 1001)
	p.MaxHealth = 100
	p.Health = 10

	w.counter_procs(p, 1000)

	if p.Health != p.MaxHealth {
		t.Errorf("expected health healed to max (%d), got %d", p.MaxHealth, p.Health)
	}
}

// ---------------------------------------------------------------------------
// DP-1038: Carry weight (CAN_CARRY_W) enforcement
//
// Inventory.addItem must reject items that exceed MaxWeight, matching C's
// CAN_GET_OBJ macro (utils.h:543-545). The check comes before the count check.
// ---------------------------------------------------------------------------

func makeWeightedObj(vnum, weight int) *ObjectInstance {
	return &ObjectInstance{
		VNum: vnum,
		Prototype: &parser.Obj{
			VNum:   vnum,
			Weight: weight,
		},
	}
}

func TestAddItem_RejectsOverWeight(t *testing.T) {
	inv := NewInventory()
	inv.Capacity = 100 // high count limit so weight is the binding constraint
	inv.MaxWeight = 10 // str 0 carry weight — very low

	// Add a 6-weight item (fits: 6 <= 10).
	item1 := makeWeightedObj(1, 6)
	if err := inv.AddItem(item1); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Add a 6-weight item (would be 12 > 10 → too heavy).
	item2 := makeWeightedObj(2, 6)
	err := inv.AddItem(item2)
	if err != ErrInventoryTooHeavy {
		t.Errorf("expected ErrInventoryTooHeavy, got %v", err)
	}
}

func TestAddItem_CountFullStillWorks(t *testing.T) {
	// Verify the count-based limit still works independently of weight.
	inv := NewInventory()
	inv.Capacity = 1
	inv.MaxWeight = 1000 // high so weight never binds

	item1 := makeWeightedObj(1, 1)
	if err := inv.AddItem(item1); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	item2 := makeWeightedObj(2, 1)
	if err := inv.AddItem(item2); err != ErrInventoryFull {
		t.Errorf("expected ErrInventoryFull, got %v", err)
	}
}

func TestAddItem_WeightCheckedBeforeCount(t *testing.T) {
	// If both weight and count would fail, weight error takes priority
	// (matching C CAN_GET_OBJ macro order: weight check is first).
	inv := NewInventory()
	inv.Capacity = 0  // count is full
	inv.MaxWeight = 1 // weight is also full
	item := makeWeightedObj(1, 5)
	err := inv.AddItem(item)
	if err != ErrInventoryTooHeavy {
		t.Errorf("expected ErrInventoryTooHeavy (weight checked first), got %v", err)
	}
}

func TestSetCapacity_SetsMaxWeight(t *testing.T) {
	// SetCapacity(str, strAdd, dex, level) must set MaxWeight from str_app.
	inv := NewInventory()
	inv.SetCapacity(18, 0, 18, 10) // str 18 → carry_w 255

	if inv.MaxWeight != 255 {
		t.Errorf("str 18 MaxWeight = %d, want 255 (constants.c str_app[])", inv.MaxWeight)
	}
	if inv.Capacity != 5+9+5 { // 5 + (18>>1) + (10>>1)
		t.Errorf("dex 18 level 10 capacity = %d, want %d", inv.Capacity, 5+9+5)
	}
}

func TestSetCapacity_Str0Weight(t *testing.T) {
	inv := NewInventory()
	inv.SetCapacity(0, 0, 10, 1) // str 0 → carry_w 0

	if inv.MaxWeight != 0 {
		t.Errorf("str 0 MaxWeight = %d, want 0", inv.MaxWeight)
	}
}

// combat import guard for constants used above.
var _ = combat.PosStanding
