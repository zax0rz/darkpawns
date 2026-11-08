package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestDoLookInHeaderUsesCFName proves the look-in container header renders
// fname(obj->name) (src/act.informative.c:942) with C's letters-only rule: a
// container keyed "climber's pack" prints "climber", not "climber's". The old
// space-splitting fname() yielded "climber's", so this test fails on revert.
func TestDoLookInHeaderUsesCFName(t *testing.T) {
	w, p, _ := newViewTestWorld(t)

	pack := NewObjectInstance(&parser.Obj{
		VNum: 16005, ShortDesc: "a climber's pack", Keywords: "climber's pack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{100, 0, 0, 0},
	}, -1)
	if err := p.Inventory.AddItem(pack); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	got := observationMessageText(w.DoLookIn(p, "pack"))
	if !strings.Contains(got, "climber (carried):") {
		t.Fatalf("look-in header = %q, want it to contain %q", got, "climber (carried):")
	}
	if strings.Contains(got, "climber's (carried):") {
		t.Fatalf("look-in header used the first space-word instead of fname(): %q", got)
	}
}

// TestDoorFarSideNameUsesCFName proves the open/close far-side door name renders
// fname(back->keyword) (src/act.movement.c:522) with C's letters-only rule: a
// far door keyed "iron-gate door" prints "The iron is opened from the other
// side.", not "iron-gate". The keyword keeps a "door" token so the door command
// still resolves; the old firstWord() yielded "iron-gate", so this fails on
// revert.
func TestDoorFarSideNameUsesCFName(t *testing.T) {
	w, p, messages := newDoorHandlerTestWorld(t)
	observer := NewPlayer(2, "FarObserver", 101)
	observer.SetPosition(8)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.mu.Lock()
	near := w.rooms[100].Exits["north"]
	near.Keywords = "iron-gate door"
	w.rooms[100].Exits["north"] = near
	far := w.rooms[101].Exits["south"]
	far.Keywords = "iron-gate door"
	w.rooms[101].Exits["south"] = far
	w.mu.Unlock()

	giveDoorTestObject(t, w, p, 900) // iron key
	w.DoUnlock(p, "door north")
	w.DoOpen(p, "door north")

	const want = "The iron is opened from the other side.\r\n"
	if got := joinedDoorMessages(messages, observer.Name); !strings.Contains(got, want) {
		t.Fatalf("far observer = %q, want substring %q", got, want)
	}
}
