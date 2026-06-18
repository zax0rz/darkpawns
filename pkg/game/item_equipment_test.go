package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newWieldableItem(vnum int, shortDesc, keywords string, twoHanded bool) *ObjectInstance {
	obj := NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		WearFlags: [4]int{1 << 13}, // ITEM_WEAR_WIELD
	}, -1)
	if twoHanded {
		obj.SetExtraFlag(0, extraFlagTwoHanded)
	}
	return obj
}

func newHoldableItem(vnum int, shortDesc, keywords string) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		WearFlags: [4]int{1 << 14}, // ITEM_WEAR_HOLD
	}, -1)
}

// TestPerformWearTwoHanded verifies the two-handed-weapon checks use
// ITEM_TWO_HANDED (bit 28), not bit 3 (ITEM_NODONATE's position) —
// src/structs.h:468-496.
//
// performWear's own two-handed checks (item_equipment.go) test occupancy via
// w.IsEquipped(ch, eqWear*), which casts the eqWear* position constant
// straight to EquipmentSlot. That space is unrelated to the SlotWield/SlotHold
// values Equipment.equip() actually assigns via getWearFlags() — a separate,
// pre-existing mismatch in this unwired ("not yet wired to command registry")
// code, out of scope here. So equipment state is seeded directly into
// ch.Equipment.Slots under the EquipmentSlot(eqWear*) keys performWear itself
// reads, to exercise the two-handed bit check in isolation.
func TestPerformWearTwoHanded(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	held := newHoldableItem(4100, "a wooden buckler", "buckler")
	twoHanded := newWieldableItem(4102, "a greatsword", "greatsword", true)

	// A two-handed weapon must be rejected while something is held.
	ch.Equipment.Slots[EquipmentSlot(eqWearHold)] = held
	if err := ch.Inventory.AddItem(twoHanded); err != nil {
		t.Fatalf("AddItem twoHanded: %v", err)
	}
	w.performWear(ch, twoHanded, eqWearWield)
	if w.IsEquipped(ch, eqWearWield) {
		t.Errorf("two-handed weapon should not wield while a hand is occupied")
	}
	if msg := lastMsg(); !strings.Contains(msg, "Both hands must be free") {
		t.Errorf("two-handed-wield-blocked message: got %q", msg)
	}

	// Symmetric case: a two-handed weapon already wielded blocks holding.
	delete(ch.Equipment.Slots, EquipmentSlot(eqWearHold))
	ch.Equipment.Slots[EquipmentSlot(eqWearWield)] = twoHanded
	if err := ch.Inventory.AddItem(held); err != nil {
		t.Fatalf("AddItem held: %v", err)
	}
	w.performWear(ch, held, eqWearHold)
	if w.IsEquipped(ch, eqWearHold) {
		t.Errorf("holding should be blocked while a two-handed weapon is wielded")
	}
	if msg := lastMsg(); !strings.Contains(msg, "Both your hands are occupied") {
		t.Errorf("two-handed-hold-blocked message: got %q", msg)
	}
}

// TestDoWearAllSkipsUnseenItems verifies the wear-all loop skips items the
// wearer can't see (CAN_SEE_OBJ — act.item.c:1611), not items with some
// unrelated extra-flag bit set.
func TestDoWearAllSkipsUnseenItems(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)

	visible := newWieldableItem(4110, "a steel sword", "sword", false)
	if err := ch.Inventory.AddItem(visible); err != nil {
		t.Fatalf("AddItem visible: %v", err)
	}
	invisible := newWieldableItem(4111, "an invisible dagger", "dagger", false)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if err := ch.Inventory.AddItem(invisible); err != nil {
		t.Fatalf("AddItem invisible: %v", err)
	}

	w.doWear(ch, nil, "wear", "all")

	// Equipment.equip() assigns the real storage slot via getWearFlags(),
	// which is SlotWield — not EquipmentSlot(eqWearWield) (see comment on
	// TestPerformWearTwoHanded) — so check occupancy through the real slot.
	if _, ok := ch.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Errorf("visible item should have been worn")
	}
	if _, ok := ch.Inventory.FindItem("dagger"); !ok {
		t.Errorf("invisible item the wearer can't see should be skipped by wear-all")
	}
}

// TestPerformRemoveNoDrop verifies performRemove's NODROP check uses
// ITEM_NODROP (bit 7), not bit 0 (ITEM_GLOW).
func TestPerformRemoveNoDrop(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	cursed := newWieldableItem(4120, "a cursed blade", "blade", false)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := w.EquipItem(ch, cursed, eqWearWield); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}

	// EquipItem's slot argument is ignored — Equipment.equip() places the
	// item via getWearFlags() under SlotWield (see TestPerformWearTwoHanded),
	// so performRemove must be called with that real slot.
	w.performRemove(ch, int(SlotWield))
	if msg := lastMsg(); !strings.Contains(msg, "You can't remove") || !strings.Contains(msg, "CURSED") {
		t.Errorf("NODROP remove message: got %q", msg)
	}
	if _, ok := ch.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Errorf("NODROP item should remain equipped")
	}
}
