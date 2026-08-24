package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPerformPutWontFit proves perform_put's capacity gate (act.item.c:63):
// an object heavier than the container's remaining capacity is rejected with
// "$p won't fit in $P." and stays in the player's inventory.
func TestPerformPutWontFit(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	pouch := NewObjectInstance(&parser.Obj{
		VNum: 9000, ShortDesc: "a tiny pouch", Keywords: "pouch",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{5, 0, -1, 0}, WearFlags: [4]int{1},
	}, -1)
	registerTransferObject(w, pouch)
	if err := w.MoveObjectToPlayerInventory(pouch, ch); err != nil {
		t.Fatalf("seat pouch: %v", err)
	}
	anvil := newTransferItem(9001, "an anvil", "anvil", 1)
	anvil.SetWeight(50)
	registerTransferObject(w, anvil)
	if err := w.MoveObjectToPlayerInventory(anvil, ch); err != nil {
		t.Fatalf("seat anvil: %v", err)
	}
	w.performPut(ch, anvil, pouch)
	if got := lastMsg(); !strings.Contains(got, "won't fit in") {
		t.Fatalf("wont-fit byte: got %q", got)
	}
	if _, ok := ch.Inventory.FindItem("anvil"); !ok {
		t.Fatal("anvil should remain in inventory when it won't fit")
	}
}
