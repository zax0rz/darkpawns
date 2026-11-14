package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// objectTimerFixture builds a one-player world with a takeable timed object
// prototype, mirroring the oedit-driven timer the oracle scenario drives.
func objectTimerFixture(t *testing.T, protos ...*parser.Obj) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}}}
	for _, proto := range protos {
		parsed.Objs = append(parsed.Objs, *proto)
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Carrier", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, p
}

// TestCarriedTimerDecrements pins update_char_objects' carried walk
// (src/handler.c:1081): a carried item's timer decrements by 1 per call even
// when no light is held — the pre-fix Go returned before the walk whenever
// the held-light block found nothing to burn (DP-1447).
func TestCarriedTimerDecrements(t *testing.T) {
	proto := &parser.Obj{VNum: 7001, Keywords: "skin", Timer: 5}
	w, p := objectTimerFixture(t, proto)
	item := NewObjectInstance(proto, -1)
	if item.GetTimer() != 5 {
		t.Fatalf("instance timer = %d, want the prototype's 5 (read_object copy)", item.GetTimer())
	}
	if err := p.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	w.updateCharObjects(p)

	if got := item.GetTimer(); got != 4 {
		t.Fatalf("carried timer after one call = %d, want 4 (handler.c:1081, use 1)", got)
	}
}

// TestWornTimerDecrementsByTwo pins the worn walk (src/handler.c:1076-1079):
// an equipped item's timer decrements by 2 per call.
func TestWornTimerDecrementsByTwo(t *testing.T) {
	proto := &parser.Obj{VNum: 7002, Keywords: "ring", WearFlags: [4]int{1 << 11}, Timer: 5}
	w, p := objectTimerFixture(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := p.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	// The wear flow removes the item from inventory before equipping: a worn
	// object is not in ch->carrying (C equip_char calls obj_from_char).
	if !p.Inventory.RemoveItem(item) {
		t.Fatal("RemoveItem: not in inventory")
	}
	if err := p.Equipment.Equip(item, p.Inventory); err != nil {
		t.Fatalf("Equip: %v", err)
	}

	w.updateCharObjects(p)

	if got := item.GetTimer(); got != 3 {
		t.Fatalf("worn timer after one call = %d, want 3 (handler.c:1076-1079, use 2)", got)
	}
}

// TestUntimedCarriedObjectTimerStaysZero pins update_object's guard
// (src/handler.c:1041-1042): timers at or below zero are untouched, so the
// walk never invents decay for ordinary carried inventory.
func TestUntimedCarriedObjectTimerStaysZero(t *testing.T) {
	proto := &parser.Obj{VNum: 7003, Keywords: "bread"}
	w, p := objectTimerFixture(t, proto)
	item := NewObjectInstance(proto, -1)
	if err := p.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	for range 3 {
		w.updateCharObjects(p)
	}

	if got := item.GetTimer(); got != 0 {
		t.Fatalf("untimed carried object timer = %d, want 0", got)
	}
}
