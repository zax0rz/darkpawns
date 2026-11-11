package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestGenericFindEquipFirstLookInObj pins C generic_find (src/handler.c:1513-1564)
// searching equipment before inventory for look_in_obj (src/act.informative.c:924).
// A worn backpack and a carried backpack share "pack"; lookup of "pack" must
// return the worn one with observationEquipment location.
// R1/R5c/R5h: C chooses (used), old Go chose (carried).
func TestGenericFindEquipFirstLookInObj(t *testing.T) {
	w, p, _ := newViewTestWorld(t)

	worn := NewObjectInstance(&parser.Obj{
		VNum: 8038, Keywords: "backpack pack", ShortDesc: "a backpack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{20, 5, -1, 0},
	}, -1)
	carried := NewObjectInstance(&parser.Obj{
		VNum: 8038, Keywords: "backpack pack", ShortDesc: "a backpack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{20, 5, -1, 0},
	}, -1)
	if err := p.Inventory.AddItem(worn); err != nil {
		t.Fatalf("AddItem worn: %v", err)
	}
	if err := w.EquipItem(p, worn, eqWearAbout); err != nil {
		t.Fatalf("EquipItem worn: %v", err)
	}
	if err := p.Inventory.AddItem(carried); err != nil {
		t.Fatalf("AddItem carried: %v", err)
	}

	gotObj, gotLoc := w.findObservationObjectWithLocation(p, "pack")
	if gotObj != worn {
		t.Fatalf("findObservationObjectWithLocation(pack) = %v location %v, want worn backpack (equip-first)", gotObj, gotLoc)
	}
	if gotLoc != observationEquipment {
		t.Fatalf("location = %v, want observationEquipment", gotLoc)
	}
}

// TestGenericFindEquipFirstLookAtTarget pins the same order for look_at_target
// (src/act.informative.c:1017). Worn short sword and carried crude sword both
// match "sword"; the worn one must win.
func TestGenericFindEquipFirstLookAtTarget(t *testing.T) {
	w, p, _ := newViewTestWorld(t)

	worn := NewObjectInstance(&parser.Obj{
		VNum: 18214, Keywords: "short sword", ShortDesc: "a short sword",
		TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
	}, -1)
	carried := NewObjectInstance(&parser.Obj{
		VNum: 19901, Keywords: "sword crude", ShortDesc: "a crude sword",
		TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
	}, -1)
	if err := p.Inventory.AddItem(worn); err != nil {
		t.Fatalf("AddItem worn: %v", err)
	}
	if err := w.EquipItem(p, worn, eqWearWield); err != nil {
		t.Fatalf("EquipItem worn: %v", err)
	}
	if err := p.Inventory.AddItem(carried); err != nil {
		t.Fatalf("AddItem carried: %v", err)
	}

	if got := w.findObservationObject(p, "sword"); got != worn {
		t.Fatalf("findObservationObject(sword) = %v, want worn short sword (equip-first)", got)
	}
}

// TestGenericFindEquipFirstMagObjectMagic pins mag_objectmagic's generic_find
// (src/spell_parser.c:566) for wand use: resolveMagicItemTarget and
// resolveUseActionObjectTarget must search equipment before inventory.
// Worn short sword vs carried crude sword, arg "sword" must pick the worn one.
func TestGenericFindEquipFirstMagObjectMagic(t *testing.T) {
	newSwords := func() (*ObjectInstance, *ObjectInstance) {
		worn := NewObjectInstance(&parser.Obj{
			VNum: 18214, Keywords: "short sword", ShortDesc: "a short sword",
			TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
		}, -1)
		carried := NewObjectInstance(&parser.Obj{
			VNum: 19901, Keywords: "sword crude", ShortDesc: "a crude sword",
			TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
		}, -1)
		return worn, carried
	}

	t.Run("resolveMagicItemTarget", func(t *testing.T) {
		w, p, _ := newViewTestWorld(t)
		worn, carried := newSwords()
		if err := p.Inventory.AddItem(worn); err != nil {
			t.Fatalf("AddItem worn: %v", err)
		}
		if err := w.EquipItem(p, worn, eqWearWield); err != nil {
			t.Fatalf("EquipItem worn: %v", err)
		}
		if err := p.Inventory.AddItem(carried); err != nil {
			t.Fatalf("AddItem carried: %v", err)
		}
		_, obj, found := w.resolveMagicItemTarget(p, "sword", spells.SpellCreateWater)
		if !found || obj != worn {
			t.Fatalf("resolveMagicItemTarget(sword) found=%v obj=%v, want worn short sword (equip-first)", found, obj)
		}
	})

	t.Run("resolveUseActionObjectTarget", func(t *testing.T) {
		w, p, _ := newViewTestWorld(t)
		worn, carried := newSwords()
		if err := p.Inventory.AddItem(worn); err != nil {
			t.Fatalf("AddItem worn: %v", err)
		}
		if err := w.EquipItem(p, worn, eqWearWield); err != nil {
			t.Fatalf("EquipItem worn: %v", err)
		}
		if err := p.Inventory.AddItem(carried); err != nil {
			t.Fatalf("AddItem carried: %v", err)
		}
		if got := w.resolveUseActionObjectTarget(p, "sword"); got != worn {
			t.Fatalf("resolveUseActionObjectTarget(sword) = %v, want worn short sword (equip-first)", got)
		}
	})
}

// TestGenericFindEquipFirstInventoryBeforeRoom pins that inventory still beats
// room when no equipment matches (C order is equip, inv, room).
func TestGenericFindEquipFirstInventoryBeforeRoom(t *testing.T) {
	w, p, _ := newViewTestWorld(t)

	carried := NewObjectInstance(&parser.Obj{
		VNum: 19901, Keywords: "sword crude", ShortDesc: "a crude sword",
		TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
	}, -1)
	room := NewObjectInstance(&parser.Obj{
		VNum: 18214, Keywords: "short sword", ShortDesc: "a short sword",
		TypeFlag: ITEM_WEAPON, WearFlags: [4]int{8193, 0, 0, 0},
	}, -1)
	if err := p.Inventory.AddItem(carried); err != nil {
		t.Fatalf("AddItem carried: %v", err)
	}
	w.AddItemToRoom(room, 100)

	gotObj, gotLoc := w.findObservationObjectWithLocation(p, "sword")
	if gotObj != carried || gotLoc != observationInventory {
		t.Fatalf("findObservationObjectWithLocation(sword) = %v/%v, want carried (inv before room)", gotObj, gotLoc)
	}
}
