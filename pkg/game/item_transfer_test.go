package game

import (
	"strings"
	"testing"
)

// TestPerformDropNoDrop verifies performDrop's NODROP check uses ITEM_NODROP
// (bit 7), not bit 0 (ITEM_GLOW) — src/structs.h:468-496.
func TestPerformDropNoDrop(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	cursed := newDonatableItem(4000, "a cursed dagger", "dagger", 100)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := w.MoveObjectToPlayerInventory(cursed, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}

	w.performDrop(ch, cursed)
	if msg := lastMsg(); !strings.Contains(msg, "You can't drop") || !strings.Contains(msg, "CURSED") {
		t.Errorf("NODROP drop message: got %q", msg)
	}
	if _, ok := ch.Inventory.FindItem("dagger"); !ok {
		t.Errorf("NODROP item should remain in inventory")
	}

	// A merely glowing (bit 0) item is not NODROP and must still drop —
	// guards against the old bug that checked bit 0 instead of bit 7.
	glowing := newDonatableItem(4001, "a glowing orb", "orb", 100)
	glowing.SetExtraFlag(0, 0) // ITEM_GLOW
	if err := w.MoveObjectToPlayerInventory(glowing, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
	w.performDrop(ch, glowing)
	if msg := lastMsg(); !strings.Contains(msg, "You drop") {
		t.Errorf("glowing item should drop normally: got %q", msg)
	}
	if _, ok := ch.Inventory.FindItem("orb"); ok {
		t.Errorf("dropped item should leave inventory")
	}
}

// TestPerformGiveNoDrop verifies performGive's NODROP check uses ITEM_NODROP
// (bit 7), not bit 0 (ITEM_GLOW) — the live path reachable via the give command.
func TestPerformGiveNoDrop(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	vict := NewPlayer(2, "Victim", 1001)
	if err := w.AddPlayer(vict); err != nil {
		t.Fatalf("AddPlayer vict: %v", err)
	}

	cursed := newDonatableItem(4010, "a cursed ring", "ring", 100)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := w.MoveObjectToPlayerInventory(cursed, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}

	w.performGive(ch, vict, cursed)
	if msg := lastMsg(); !strings.Contains(msg, "You can't let go") {
		t.Errorf("NODROP give message: got %q", msg)
	}
	if _, ok := ch.Inventory.FindItem("ring"); !ok {
		t.Errorf("NODROP item should remain with the giver")
	}
	if _, ok := vict.Inventory.FindItem("ring"); ok {
		t.Errorf("NODROP item must never reach the recipient")
	}

	// A glowing (bit 0) item is not NODROP and must still transfer.
	glowing := newDonatableItem(4011, "a glowing pouch", "pouch", 100)
	glowing.SetExtraFlag(0, 0) // ITEM_GLOW
	if err := w.MoveObjectToPlayerInventory(glowing, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
	w.performGive(ch, vict, glowing)
	if _, ok := vict.Inventory.FindItem("pouch"); !ok {
		t.Errorf("glowing item should have been given to the recipient")
	}
	if _, ok := ch.Inventory.FindItem("pouch"); ok {
		t.Errorf("given item should have left the giver's inventory")
	}
}

// TestDoGiveAllSkipsUnseenItems verifies the give-all loop skips items the
// giver can't see (CAN_SEE_OBJ — act.item.c:824), not items with some
// unrelated extra-flag bit set.
func TestDoGiveAllSkipsUnseenItems(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	vict := NewPlayer(2, "Victim", 1001)
	if err := w.AddPlayer(vict); err != nil {
		t.Fatalf("AddPlayer vict: %v", err)
	}

	visible := newDonatableItem(4020, "a leather pouch", "pouch", 100)
	if err := w.MoveObjectToPlayerInventory(visible, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory visible: %v", err)
	}
	invisible := newDonatableItem(4021, "an invisible coin", "coin", 100)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if err := w.MoveObjectToPlayerInventory(invisible, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory invisible: %v", err)
	}

	w.doGive(ch, nil, "give", "all "+vict.Name)

	if _, ok := vict.Inventory.FindItem("pouch"); !ok {
		t.Errorf("visible item should have been given")
	}
	if _, ok := ch.Inventory.FindItem("pouch"); ok {
		t.Errorf("given item should have left the giver's inventory")
	}
	if _, ok := vict.Inventory.FindItem("coin"); ok {
		t.Errorf("invisible item the giver can't see should be skipped by give-all")
	}
	if _, ok := ch.Inventory.FindItem("coin"); !ok {
		t.Errorf("skipped invisible item should remain with the giver")
	}
}
