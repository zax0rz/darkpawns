package game

// room_order_regression_test.go — the two PR-review repros for the room
// list's C-order invariant: the decay pass ticks each object once even when
// a rotted container spills (R3b), and the butler takes the newest four
// zone-loaded items (C walks contents head-first, db.c:2155 prepends).

import "testing"

func TestReviewButlerZoneLoadedOrder(t *testing.T) {
	w, actor, _, mob, _ := newButlerTestWorld(t)
	setupButlerContainers(t, w)
	s := NewSpawner(w)
	var objects []*ObjectInstance
	for i := 0; i < 5; i++ {
		obj, err := s.SpawnObject(3005, butlerTestRoom)
		if err != nil {
			t.Fatal(err)
		}
		objects = append(objects, obj)
	}
	specButler(w, actor, mob, "", "")
	if objects[0].Location.Kind != ObjInRoom {
		t.Fatal("oldest zone-loaded item was taken; C takes the newest four")
	}
}

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
	for i := 0; i < 2; i++ {
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
