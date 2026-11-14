package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCanTakeObjCountsContainerContents pins C's can_take_obj weight gate
// (src/act.item.c:162-166) for DP-1524: GET_OBJ_WEIGHT includes the object's
// contents (handler.c:951-955), so a str-3 carrier (CAN_CARRY_W 10) refuses a
// 1-pound bag holding a 9-pound stone (10 carried + 10 total... the bag alone
// passes) — the contents must be what tips the gate. Fails if the site
// reverts to GetWeight() (R5h).
func TestCanTakeObjCountsContainerContents(t *testing.T) {
	w, p := newFlagWorld(t)
	p.Stats.Str = 3
	p.Stats.StrAdd = 0
	p.CopyBaseAttributes()

	bag := NewObjectInstance(&parser.Obj{
		VNum:      9101,
		ShortDesc: "a bag",
		Keywords:  "bag",
		TypeFlag:  ITEM_CONTAINER,
		Values:    [4]int{1000, 0, 0, 0},
		WearFlags: [4]int{1}, // ITEM_WEAR_TAKE
		Weight:    1,
	}, -1)
	registerTransferObject(w, bag)
	if err := w.MoveObjectToRoom(bag, 1001); err != nil {
		t.Fatalf("drop bag: %v", err)
	}
	stone := newTransferItem(9102, "a stone", "stone", 1)
	stone.WeightOverride = new(int)
	*stone.WeightOverride = 9
	registerTransferObject(w, stone)
	if err := w.MoveObjectToContainer(stone, bag); err != nil {
		t.Fatalf("load stone into bag: %v", err)
	}
	// Bag total = 1 (bag) + 9 (stone) = 10; the carrier's limit is 10 and
	// the check is strict-greater, so add one more pound to tip it.
	stone2 := newTransferItem(9103, "a pebble", "pebble", 1)
	stone2.WeightOverride = new(int)
	*stone2.WeightOverride = 1
	registerTransferObject(w, stone2)
	if err := w.MoveObjectToContainer(stone2, bag); err != nil {
		t.Fatalf("load pebble into bag: %v", err)
	}

	if w.canTakeObj(p, bag) {
		t.Fatalf("canTakeObj accepted an 11-pound loaded bag for a 10-pound carrier")
	}

	// The bare bag (1 pound) is still accepted: the gate fails only because
	// of the contents.
	if !bag.RemoveFromContainer(stone) {
		t.Fatal("fixture: stone stayed in the bag")
	}
	if !bag.RemoveFromContainer(stone2) {
		t.Fatal("fixture: pebble stayed in the bag")
	}
	if !w.canTakeObj(p, bag) {
		t.Fatalf("canTakeObj refused an empty 1-pound bag for a 10-pound carrier")
	}
}
