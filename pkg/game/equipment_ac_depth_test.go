package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// equipment_ac_depth_test.go — DP-1430. do_score's clothing line reads GET_AC
// and its weight line reads IS_CARRYING_W; both diverged from C once gear was
// worn. C's equip path is src/handler.c:679-750 (apply_ac at :610-635) and its
// score reader is src/act.informative.c:1240-1315.

// equipmentACWorld is a room with a body piece (val0 5), a head piece (val0 4)
// and an about piece whose APPLY_AC is +10 and APPLY_CON is -5.
func equipmentACWorld(t *testing.T) (*World, func(vnum int) *ObjectInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "AC"}},
		Objs: []parser.Obj{
			{VNum: 4101, Keywords: "cuirass", ShortDesc: "a cuirass", TypeFlag: ITEM_ARMOR, Weight: 6, Values: [4]int{5, 0, 0, 0}},
			{VNum: 4102, Keywords: "helm", ShortDesc: "a helm", TypeFlag: ITEM_ARMOR, Weight: 3, Values: [4]int{4, 0, 0, 0}},
			{
				VNum: 4103, Keywords: "rags", ShortDesc: "tattered rags", TypeFlag: 11, // ITEM_WORN
				Weight: 2,
				Affects: []parser.ObjAffect{
					{Location: ApplyAC, Modifier: 10},
					{Location: ApplyCon, Modifier: -5},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	spawn := func(vnum int) *ObjectInstance {
		t.Helper()
		obj, err := w.SpawnObject(vnum, 1001)
		if err != nil {
			t.Fatalf("SpawnObject %d: %v", vnum, err)
		}
		return obj
	}
	return w, spawn
}

// TestCarriedWeightExcludesWornGear pins C's IS_CARRYING_W to the carrying
// list: do_wear takes the object off that list before equip_char, so worn gear
// never reaches do_score's pack ratio (src/utils.h:322; src/handler.c:559-601).
func TestCarriedWeightExcludesWornGear(t *testing.T) {
	w, spawn := equipmentACWorld(t)
	p := NewPlayer(1, "Packrat", 1001)
	p.AC = 100
	if err := p.Inventory.AddItem(spawn(4101)); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if got := p.CarriedWeight(); got != 6 {
		t.Fatalf("CarriedWeight with one 6-weight item = %d, want 6", got)
	}
	if err := w.EquipItem(p, spawn(4103), 12); err != nil { // WEAR_ABOUT
		t.Fatalf("EquipItem: %v", err)
	}
	if got := p.CarriedWeight(); got != 6 {
		t.Fatalf("CarriedWeight after wearing a 2-weight item = %d, want 6 (worn gear leaves the carrying list)", got)
	}
}

// TestGetArmorClassAppliesCSlotFactors pins apply_ac's factor table: body 3,
// head and legs 2, everything else 1 (src/handler.c:618-632).
func TestGetArmorClassAppliesCSlotFactors(t *testing.T) {
	w, spawn := equipmentACWorld(t)
	p := NewPlayer(1, "Armored", 1001)
	p.AC = 100
	if err := w.EquipItem(p, spawn(4101), 5); err != nil { // WEAR_BODY, val0 5
		t.Fatalf("EquipItem body: %v", err)
	}
	if err := w.EquipItem(p, spawn(4102), 6); err != nil { // WEAR_HEAD, val0 4
		t.Fatalf("EquipItem head: %v", err)
	}
	if got, want := p.Equipment.GetArmorClass(), 3*5+2*4; got != want {
		t.Fatalf("GetArmorClass = %d, want %d (body x3, head x2)", got, want)
	}
	if got, want := p.GetAC(), 100-(3*5+2*4); got != want {
		t.Fatalf("GetAC = %d, want %d", got, want)
	}
}

// TestGetACIncludesWornApplyAC pins the second half of equip_char's AC path:
// every worn object's APPLY_AC modifier folds into GET_AC, and a WEAR_ABOUT
// ITEM_WORN piece contributes no apply_ac value at all
// (src/handler.c:182, :734-746; src/scripts.c has no part in this).
func TestGetACIncludesWornApplyAC(t *testing.T) {
	w, spawn := equipmentACWorld(t)
	p := NewPlayer(1, "Robed", 1001)
	p.AC = 100
	if err := w.EquipItem(p, spawn(4103), 12); err != nil { // WEAR_ABOUT
		t.Fatalf("EquipItem: %v", err)
	}
	if got := p.Equipment.GetArmorClass(); got != 0 {
		t.Fatalf("apply_ac for an ITEM_WORN about the body = %d, want 0", got)
	}
	if got := p.Equipment.GetWornAffectAC(); got != 10 {
		t.Fatalf("worn APPLY_AC = %d, want 10", got)
	}
	if got := p.GetAC(); got != 110 {
		t.Fatalf("GetAC = %d, want 110 (100 + the rags' APPLY_AC +10)", got)
	}
}
