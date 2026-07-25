package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func registerTransferObject(w *World, obj *ObjectInstance) {
	obj.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[obj.ID] = obj
}

func newTransferItem(vnum int, shortDesc, keywords string, wearMask int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		WearFlags: [4]int{wearMask},
		Weight:    1,
	}, -1)
}

func TestCanTakeObjUsesTakeBit(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	obj := newTransferItem(3990, "a short sword", "sword", (1<<0)|(1<<13))
	obj.SetWeight(0)
	if !w.canTakeObj(ch, obj) {
		t.Fatal("TAKE|WIELD object should be takeable; TAKE is a bit test")
	}
}

func TestDoGetChecksFullArmsBeforeEmptyArgument(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	for i := 0; i < ch.MaxCarryItems(); i++ {
		obj := newTransferItem(5000+i, "a pebble", "pebble", 1)
		obj.SetWeight(0)
		registerTransferObject(w, obj)
		if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
			t.Fatalf("fill inventory: %v", err)
		}
	}

	w.DoGet(ch, "")
	if got := lastMsg(); !strings.Contains(got, "Your arms are already full!") {
		t.Fatalf("full-arms precheck: got %q", got)
	}
}

func TestDoGetAllFromContainerUsesSecondaryObject(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	sack := NewObjectInstance(&parser.Obj{
		VNum: 5100, ShortDesc: "a leather sack", Keywords: "sack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{100, 0, -1, 0}, WearFlags: [4]int{1},
	}, -1)
	registerTransferObject(w, sack)
	if err := w.MoveObjectToPlayerInventory(sack, ch); err != nil {
		t.Fatalf("move sack: %v", err)
	}

	bread := newTransferItem(5101, "a loaf of bread", "loaf bread", 1)
	registerTransferObject(w, bread)
	if err := w.MoveObjectToContainer(bread, sack); err != nil {
		t.Fatalf("move bread: %v", err)
	}

	w.DoGet(ch, "all sack")
	if _, ok := ch.Inventory.FindItem("bread"); !ok {
		t.Fatal("get all sack should move the bread into inventory")
	}
	if got := lastMsg(); !strings.Contains(got, "You get a loaf of bread from a leather sack.") {
		t.Fatalf("two-object Act rendering: got %q", got)
	}
}

func TestDoDropAllDotAndCoins(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	ch.Stats.Str = 10
	ch.Stats.Dex = 10
	for i := 0; i < 2; i++ {
		obj := newTransferItem(5200+i, "a short sword", "short sword", 1|(1<<13))
		registerTransferObject(w, obj)
		if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
			t.Fatalf("move sword: %v", err)
		}
	}

	w.DoDrop(ch, "all.sword")
	if _, ok := ch.Inventory.FindItem("sword"); ok {
		t.Fatal("drop all.sword should move every matching item")
	}
	if got := lastMsg(); strings.Count(got, "You drop a short sword.") != 2 {
		t.Fatalf("drop all.sword output: got %q", got)
	}

	ch.SetGold(25)
	w.DoDrop(ch, "10 coins")
	if got := ch.GetGold(); got != 15 {
		t.Fatalf("gold after drop = %d, want 15", got)
	}
	if got := lastMsg(); !strings.Contains(got, "You drop some gold.") {
		t.Fatalf("drop coins output: got %q", got)
	}
	w.DoGet(ch, "coins")
	if got := ch.GetGold(); got != 25 {
		t.Fatalf("gold after re-getting coins = %d, want 25; output %q", got, lastMsg())
	}
}

func TestDoGiveMovesObjectToMobBeforeHook(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5300, Keywords: "dolkvin drunkard", ShortDesc: "Dolkvin the Drunkard",
			ActionFlags: []string{"OKGIVE"}, Level: 10, Str: 10, Dex: 10,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := w.SpawnMob(5300, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	obj := newTransferItem(5301, "a short sword", "sword", 1|(1<<13))
	registerTransferObject(w, obj)
	if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
		t.Fatalf("move sword: %v", err)
	}

	w.DoGive(ch, "sword dolkvin")
	if len(mob.Inventory) != 1 || mob.Inventory[0] != obj {
		t.Fatalf("mob inventory = %#v, want transferred sword", mob.Inventory)
	}
	if _, ok := ch.Inventory.FindItem("sword"); ok {
		t.Fatal("given object should leave player inventory")
	}
}

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

// TestGiveFindVictNoperson verifies giveFindVict uses C's canonical NOPERSON
// string when the target person is not in the room.
func TestGiveFindVictNoperson(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	// give to "nobody" — no player or mob with that name in room
	vict := w.giveFindVict(ch, "nobody")
	if vict != nil {
		t.Fatal("giveFindVict should return nil for missing target")
	}
	if got := lastMsg(); got != NoPersonHere {
		t.Fatalf("giveFindVict not-found = %q, want exact C NOPERSON %q", got, NoPersonHere)
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
