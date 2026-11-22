package game

// room_order_regression_test.go — the PR-review repro that the decay pass
// ticks each object once even when a rotted container spills (R3b).

import "testing"

func TestReviewCorpseSpillDoesNotRetickEarlierObject(t *testing.T) {
	w := newDecayTestWorld(t, 0)
	puddle := newTransferItem(20, "puddle", "puddle", 1)
	puddle.SetTimer(10)
	corpse := makeCorpseObject(1)
	other := newTransferItem(7100, "other", "other", 1)
	for _, obj := range []*ObjectInstance{puddle, corpse, other} {
		registerTransferObject(w, obj)
		w.AddItemToRoom(obj, 2001)
	}
	for i := range 2 {
		child := newTransferItem(7200+i, "child", "child", 1)
		registerTransferObject(w, child)
		if err := w.MoveObjectToContainer(child, corpse); err != nil {
			t.Fatal(err)
		}
	}
	items := make([]*ObjectInstance, 3, 8)
	copy(items, w.roomItems[2001])
	w.roomItems[2001] = items
	w.decayObjectsInRoom(2001)
	if got := puddle.GetTimer(); got != 9 {
		t.Fatalf("puddle timer = %d, want 9 after one tick", got)
	}
}
