package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newConsumableTestWorld(t *testing.T, protos ...*parser.Obj) (*World, *Player, func() string) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
	}
	for _, proto := range protos {
		parsed.Objs = append(parsed.Objs, *proto)
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	lastMsg := func() string { s := out.String(); out.Reset(); return s }
	return w, ch, lastMsg
}

func makeFoodProto(vnum int, bites, poison int) *parser.Obj {
	return &parser.Obj{
		VNum:      vnum,
		ShortDesc: "a loaf of bread",
		Keywords:  "bread loaf",
		TypeFlag:  ITEM_FOOD,
		Weight:    1,
		Values:    [4]int{bites, 0, 0, poison},
	}
}

func makeDrinkconProto(vnum, capacity, amount, liquid, poison int) *parser.Obj {
	return &parser.Obj{
		VNum:      vnum,
		ShortDesc: "a waterskin",
		Keywords:  "waterskin skin",
		TypeFlag:  ITEM_DRINKCON,
		Weight:    capacity,
		Values:    [4]int{capacity, amount, liquid, poison},
	}
}

func makeFountainProto(vnum, capacity, amount, liquid, poison int) *parser.Obj {
	return &parser.Obj{
		VNum:      vnum,
		ShortDesc: "a bubbling fountain",
		Keywords:  "fountain",
		TypeFlag:  ITEM_FOUNTAIN,
		Weight:    capacity,
		Values:    [4]int{capacity, amount, liquid, poison},
	}
}

func makePuddleProto(vnum int) *parser.Obj {
	return &parser.Obj{
		VNum:      vnum,
		ShortDesc: "a puddle",
		Keywords:  "puddle",
		TypeFlag:  ITEM_OTHER,
		Weight:    0,
		Values:    [4]int{0, 0, 0, 0},
	}
}

// TestConsumables_InstanceIsolation is the headline regression guard: consuming,
// drinking, or pouring one instance must not mutate the shared prototype or any
// other instance of the same vnum.
func TestConsumables_InstanceIsolation(t *testing.T) {
	foodProto := makeFoodProto(7001, 5, 0)
	drinkProto := makeDrinkconProto(7002, 10, 10, LiqBeer, 0)
	w, ch, _ := newConsumableTestWorld(t, foodProto, drinkProto)

	// --- eat isolation ---
	eaten := NewObjectInstance(foodProto, -1)
	otherFood := NewObjectInstance(foodProto, -1)
	if err := ch.Inventory.AddItem(eaten); err != nil {
		t.Fatalf("AddItem eaten: %v", err)
	}
	if err := ch.Inventory.AddItem(otherFood); err != nil {
		t.Fatalf("AddItem otherFood: %v", err)
	}

	w.DoEat(ch, nil, "eat", "bread", scmdEat)

	if otherFood.GetValue(0) != 5 {
		t.Errorf("eat isolation: other instance value[0] = %d, want 5", otherFood.GetValue(0))
	}
	if foodProto.Values[0] != 5 {
		t.Errorf("eat isolation: prototype value[0] = %d, want 5", foodProto.Values[0])
	}

	// --- drink isolation ---
	drunk := NewObjectInstance(drinkProto, -1)
	otherDrink := NewObjectInstance(drinkProto, -1)
	if err := ch.Inventory.AddItem(drunk); err != nil {
		t.Fatalf("AddItem drunk: %v", err)
	}
	if err := ch.Inventory.AddItem(otherDrink); err != nil {
		t.Fatalf("AddItem otherDrink: %v", err)
	}

	w.DoDrink(ch, nil, "drink", "skin", scmdDrink)

	if otherDrink.GetValue(1) != 10 {
		t.Errorf("drink isolation: other instance value[1] = %d, want 10", otherDrink.GetValue(1))
	}
	if drinkProto.Values[1] != 10 {
		t.Errorf("drink isolation: prototype value[1] = %d, want 10", drinkProto.Values[1])
	}

	// --- pour isolation ---
	fromProto := makeDrinkconProto(7003, 10, 10, LiqBeer, 0)
	toProto := makeDrinkconProto(7004, 10, 0, 0, 0)
	fromInst := NewObjectInstance(fromProto, -1)
	otherFrom := NewObjectInstance(fromProto, -1)
	toInst := NewObjectInstance(toProto, -1)
	if err := ch.Inventory.AddItem(fromInst); err != nil {
		t.Fatalf("AddItem fromInst: %v", err)
	}
	if err := ch.Inventory.AddItem(otherFrom); err != nil {
		t.Fatalf("AddItem otherFrom: %v", err)
	}
	if err := ch.Inventory.AddItem(toInst); err != nil {
		t.Fatalf("AddItem toInst: %v", err)
	}

	w.DoPour(ch, nil, "pour", "skin skin2", scmdPour)

	if otherFrom.GetValue(1) != 10 {
		t.Errorf("pour isolation: other instance value[1] = %d, want 10", otherFrom.GetValue(1))
	}
	if fromProto.Values[1] != 10 {
		t.Errorf("pour isolation: prototype value[1] = %d, want 10", fromProto.Values[1])
	}
}

// TestDoDrink_EmptyContainer verifies the empty-drinkcon message.
func TestDoDrink_EmptyContainer(t *testing.T) {
	proto := makeDrinkconProto(7001, 10, 0, LiqWater, 0)
	w, ch, lastMsg := newConsumableTestWorld(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	ch.SetCondition(CondFull, 0)
	ch.SetCondition(CondThirst, 10)
	w.DoDrink(ch, nil, "drink", "skin", scmdDrink)

	if msg := lastMsg(); !strings.Contains(msg, "It's empty.") {
		t.Errorf("empty message: got %q", msg)
	}
}

// TestDoDrink_WaterSeededRNG verifies the RNG water-drink path with a seeded
// consumable RNG source. Beer/wine have deterministic amounts; water uses
// number(3,8), so we seed to make it deterministic.
func TestDoDrink_WaterSeededRNG(t *testing.T) {
	proto := makeDrinkconProto(7001, 10, 10, LiqWater, 0)
	w, ch, _ := newConsumableTestWorld(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	oldNumber := consumableNumber
	defer func() { consumableNumber = oldNumber }()
	const wantAmount = 5
	consumableNumber = func(from, to int) int { return wantAmount }

	ch.SetCondition(CondFull, 0)
	ch.SetCondition(CondThirst, 10)
	w.DoDrink(ch, nil, "drink", "skin", scmdDrink)

	if item.GetValue(1) != 10-wantAmount {
		t.Errorf("water amount: got remaining %d, want %d", item.GetValue(1), 10-wantAmount)
	}
	// Water has thirst affect 1; full/thirst affect 0.
	if ch.GetCondition(CondThirst) != 10+wantAmount/4 {
		t.Errorf("water thirst: got %d, want %d", ch.GetCondition(CondThirst), 10+wantAmount/4)
	}
}

// TestDoDrink_Poison applies a poison affect with the correct duration.
func TestDoDrink_Poison(t *testing.T) {
	proto := makeDrinkconProto(7001, 10, 4, LiqWater, 1)
	w, ch, _ := newConsumableTestWorld(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	// Seed so amount is deterministic.
	oldNumber := consumableNumber
	defer func() { consumableNumber = oldNumber }()
	consumableNumber = func(from, to int) int { return 3 }

	ch.SetCondition(CondFull, 0)
	ch.SetCondition(CondThirst, 0)
	w.DoDrink(ch, nil, "drink", "skin", scmdDrink)

	amount := 4 - item.GetValue(1)
	if amount <= 0 {
		t.Fatalf("poison test: amount consumed was %d", amount)
	}

	if !ch.IsAffected(affPoison) {
		t.Errorf("poison affect bit not set")
	}
	var found bool
	for _, aff := range ch.ActiveAffects {
		if aff.Flags&engine.AFFPoison != 0 {
			found = true
			if aff.Duration != amount*3 {
				t.Errorf("poison duration: got %d, want %d", aff.Duration, amount*3)
			}
		}
	}
	if !found {
		t.Errorf("poison affect not found in ActiveAffects")
	}
}

// TestDoEat_TasteDecrements verifies taste decrements the bite counter without
// consuming the item until it reaches zero.
func TestDoEat_TasteDecrements(t *testing.T) {
	proto := makeFoodProto(7001, 2, 0)
	w, ch, _ := newConsumableTestWorld(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	w.DoEat(ch, nil, "taste", "bread", scmdTaste)
	if item.GetValue(0) != 1 {
		t.Errorf("first taste: value[0] = %d, want 1", item.GetValue(0))
	}
	if _, ok := ch.Inventory.FindItem("bread"); !ok {
		t.Errorf("first taste: item should still be in inventory")
	}

	w.DoEat(ch, nil, "taste", "bread", scmdTaste)
	if _, ok := ch.Inventory.FindItem("bread"); ok {
		t.Errorf("second taste: item should be consumed")
	}
}

// TestDoPour_OutCreatesPuddle verifies pour-out creates a puddle with the
// correct liquid, poison flag, and decay timer.
func TestDoPour_OutCreatesPuddle(t *testing.T) {
	proto := makeDrinkconProto(7001, 10, 5, LiqBeer, 1)
	puddleProto := makePuddleProto(20)
	w, ch, _ := newConsumableTestWorld(t, proto, puddleProto)
	item := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	w.DoPour(ch, nil, "pour", "skin out", scmdPour)

	if item.GetValue(1) != 0 || item.GetValue(2) != 0 || item.GetValue(3) != 0 {
		t.Errorf("pour-out did not empty from-con: values=%d,%d,%d", item.GetValue(1), item.GetValue(2), item.GetValue(3))
	}

	puddles := w.GetItemsInRoom(1001)
	var found bool
	for _, p := range puddles {
		if p.VNum == 20 {
			found = true
			if p.GetValue(2) != LiqBeer {
				t.Errorf("puddle liquid = %d, want %d", p.GetValue(2), LiqBeer)
			}
			if p.GetValue(3) != 1 {
				t.Errorf("puddle poison = %d, want 1", p.GetValue(3))
			}
			if p.GetTimer() != 2 {
				t.Errorf("puddle timer = %d, want 2", p.GetTimer())
			}
		}
	}
	if !found {
		t.Errorf("pour-out did not create puddle in room")
	}
}

// TestDoPour_FillFromFountain verifies fill transfers liquid and poison from a
// room fountain into an inventory drink container.
func TestDoPour_FillFromFountain(t *testing.T) {
	fountainProto := makeFountainProto(7001, 100, 100, LiqWater, 1)
	skinProto := makeDrinkconProto(7002, 10, 0, 0, 0)
	w, ch, _ := newConsumableTestWorld(t, fountainProto, skinProto)

	fountain := NewObjectInstance(fountainProto, 1001)
	w.objectInstances[fountain.ID] = fountain
	w.roomItems[1001] = append(w.roomItems[1001], fountain)

	skin := NewObjectInstance(skinProto, -1)
	if err := ch.Inventory.AddItem(skin); err != nil {
		t.Fatalf("AddItem skin: %v", err)
	}

	w.DoPour(ch, nil, "fill", "skin fountain", scmdFill)

	if skin.GetValue(1) != 10 {
		t.Errorf("fill amount: got %d, want 10", skin.GetValue(1))
	}
	if skin.GetValue(2) != LiqWater {
		t.Errorf("fill liquid: got %d, want %d", skin.GetValue(2), LiqWater)
	}
	if skin.GetValue(3) != 1 {
		t.Errorf("fill poison: got %d, want 1", skin.GetValue(3))
	}
}
