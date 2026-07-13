package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newRoomyContainer(vnum int, shortDesc, keywords string) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		TypeFlag:  ITEM_CONTAINER,
		Values:    [4]int{1000, 0, 0, 0}, // contCapacity = 1000, open, no key
	}, -1)
}

// TestDoPutAllSkipsUnseenItems verifies the put-all loop skips items the
// player can't see (CAN_SEE_OBJ — act.item.c:132), not items with some
// unrelated extra-flag bit set.
func TestDoPutAllSkipsUnseenItems(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)

	// The container must be registered in w.objectInstances — performPut's
	// MoveObjectToContainer looks up the destination container by ID.
	sack := newRoomyContainer(4200, "a leather sack", "sack")
	sack.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[sack.ID] = sack
	if err := w.MoveObjectToPlayerInventory(sack, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory sack: %v", err)
	}

	visible := newDonatableItem(4201, "a wooden block", "block", 10)
	if err := w.MoveObjectToPlayerInventory(visible, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory visible: %v", err)
	}
	invisible := newDonatableItem(4202, "an invisible marble", "marble", 10)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if err := w.MoveObjectToPlayerInventory(invisible, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory invisible: %v", err)
	}

	w.doPut(ch, nil, "put", "all sack")

	foundVisible, foundInvisible := false, false
	for _, item := range sack.Contains {
		if item == visible {
			foundVisible = true
		}
		if item == invisible {
			foundInvisible = true
		}
	}
	if !foundVisible {
		t.Errorf("visible item should have been put in the sack")
	}
	if foundInvisible {
		t.Errorf("invisible item the player can't see should be skipped by put-all")
	}
	if _, ok := ch.Inventory.FindItem("marble"); !ok {
		t.Errorf("skipped invisible item should remain in inventory")
	}
	if _, ok := ch.Inventory.FindItem("block"); ok {
		t.Errorf("put item should have left the player's inventory")
	}
}

func TestPerformPutUsesInstanceCapacity(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	sack := NewObjectInstance(&parser.Obj{
		VNum: 4300, ShortDesc: "a leather sack", Keywords: "sack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{1, 0, -1, 0},
	}, -1)
	sack.SetValue(contCapacity, 100)
	sack.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[sack.ID] = sack
	if err := w.MoveObjectToPlayerInventory(sack, ch); err != nil {
		t.Fatalf("move sack: %v", err)
	}

	stone := newDonatableItem(4301, "a small stone", "stone", 0)
	stone.SetWeight(5)
	stone.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[stone.ID] = stone
	if err := w.MoveObjectToPlayerInventory(stone, ch); err != nil {
		t.Fatalf("move stone: %v", err)
	}

	w.performPut(ch, stone, sack)
	if len(sack.Contains) != 1 || sack.Contains[0] != stone {
		t.Fatal("performPut should honor the instance capacity override")
	}
}
