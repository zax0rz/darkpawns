package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPerformRemoveGates proves perform_remove's two failure branches
// (act.item.c:1717): a cursed (NODROP) item cannot be removed, and a
// full-inventory actor cannot remove an item either.
func TestPerformRemoveGates(t *testing.T) {
	t.Run("nodrop-cursed", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		cursed := NewObjectInstance(&parser.Obj{
			VNum: 9100, ShortDesc: "a cursed helm", Keywords: "helm",
			WearFlags: [4]int{1 << 6}, ExtraFlags: [4]int{1 << 7}, // WEAR_HEAD | NODROP
		}, -1)
		registerTransferObject(w, cursed)
		if err := w.EquipItem(ch, cursed, eqWearHead); err != nil {
			t.Fatalf("equip cursed helm: %v", err)
		}
		w.performRemove(ch, eqWearHead)
		if got := lastMsg(); !strings.Contains(got, "can't remove") || !strings.Contains(got, "CURSED") {
			t.Fatalf("nodrop-cursed byte: got %q", got)
		}
		if _, ok := ch.Inventory.FindItem("helm"); ok {
			t.Fatal("cursed item must not move to inventory")
		}
	})

	t.Run("carry-full", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		helm := NewObjectInstance(&parser.Obj{
			VNum: 9200, ShortDesc: "a plain helm", Keywords: "helm",
			WearFlags: [4]int{1 << 6},
		}, -1)
		registerTransferObject(w, helm)
		if err := w.EquipItem(ch, helm, eqWearHead); err != nil {
			t.Fatalf("equip helm: %v", err)
		}
		for i := 0; i < ch.Inventory.GetCapacity(); i++ {
			p := newTransferItem(9300+i, "a pebble", "pebble", 1)
			p.SetWeight(0)
			registerTransferObject(w, p)
			if err := w.MoveObjectToPlayerInventory(p, ch); err != nil {
				t.Fatalf("fill inventory: %v", err)
			}
		}
		w.performRemove(ch, eqWearHead)
		if got := lastMsg(); !strings.Contains(got, "you can't carry that many items!") {
			t.Fatalf("carry-full byte: got %q", got)
		}
	})
}

// TestPerformWearGates proves two perform_wear branches (act.item.c:1636):
// wearing onto an occupied slot, and wielding while flesh-altered.
func TestPerformWearGates(t *testing.T) {
	t.Run("slot-occupied", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		worn := NewObjectInstance(&parser.Obj{VNum: 9400, ShortDesc: "a helm", Keywords: "helm", WearFlags: [4]int{1 << 4}}, -1)
		registerTransferObject(w, worn)
		if err := w.EquipItem(ch, worn, eqWearHead); err != nil {
			t.Fatalf("equip first helm: %v", err)
		}
		second := NewObjectInstance(&parser.Obj{VNum: 9401, ShortDesc: "a cap", Keywords: "cap", WearFlags: [4]int{1 << 4}}, -1)
		registerTransferObject(w, second)
		if err := w.MoveObjectToPlayerInventory(second, ch); err != nil {
			t.Fatalf("seat cap: %v", err)
		}
		w.performWear(ch, second, eqWearHead)
		if got := lastMsg(); !strings.Contains(got, "already wearing something on your head") {
			t.Fatalf("slot-occupied byte: got %q", got)
		}
	})

	t.Run("wield-flesh-altered", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		blade := NewObjectInstance(&parser.Obj{VNum: 9500, ShortDesc: "a blade", Keywords: "blade", WearFlags: [4]int{1 << 13}}, -1)
		blade.SetWeight(1)
		registerTransferObject(w, blade)
		if err := w.MoveObjectToPlayerInventory(blade, ch); err != nil {
			t.Fatalf("seat blade: %v", err)
		}
		ch.SetAffect(affFleshAlter, true)
		w.performWear(ch, blade, eqWearWield)
		if got := lastMsg(); !strings.Contains(got, "flesh is altered") {
			t.Fatalf("flesh-alter byte: got %q", got)
		}
	})
}
