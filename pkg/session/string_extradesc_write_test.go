package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newExtraDescSessionWorld builds the world do_string's prototype write-through
// needs: object 3001 has one extra description ("crust"), the actor is an
// implementor, and the instance in their inventory is world-registered so the
// write-through can reach it.
func newExtraDescSessionWorld(t *testing.T) (*Session, *game.World, *game.ObjectInstance) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Bakery", Zone: 1}},
		Objs: []parser.Obj{{
			VNum:      3001,
			Keywords:  "bread",
			ShortDesc: "a loaf of bread",
			LongDesc:  "A loaf of bread lies here.\n",
			ExtraDescs: []parser.ExtraDesc{
				{Keywords: "crust", Description: "A golden crust.\n"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Stringtest", LVL_IMPL, 1001)
	registerInWorld(t, s)
	proto, ok := w.GetObjPrototype(3001)
	if !ok {
		t.Fatal("prototype 3001 missing")
	}
	obj := w.NewObjectFromProto(proto, -1)
	if s.player.Inventory == nil {
		s.player.Inventory = game.NewInventory()
	}
	if err := s.player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	return s, w, obj
}

// TestStringExistingExtraDescWritesThroughToPrototype is DP-1292's option A.
// C's read_object shares the prototype's ex_description list (src/db.c:1947-1951),
// so do_string's field-4 write for a keyword the prototype defines lands on the
// prototype (src/modify.c:701-707) and every instance — and every later
// read_object — sees it until the process restarts.
func TestStringExistingExtraDescWritesThroughToPrototype(t *testing.T) {
	s, w, obj := newExtraDescSessionWorld(t)

	runString(t, s, "obj bread description crust")
	if got := readMsgText(t, s); got != stringModifyingDescription {
		t.Fatalf("existing-keyword entry = %q, want %q", got, stringModifyingDescription)
	}
	if !s.IsTextEditing() {
		t.Fatal("editor is not active")
	}
	s.handleTextEditInput("A crust of ash and salt.")
	s.handleTextEditInput("@")
	if s.IsTextEditing() {
		t.Fatal("editor still active after @")
	}

	current, ok := w.GetObjPrototype(3001)
	if !ok {
		t.Fatal("prototype 3001 missing after the edit")
	}
	if got := current.ExtraDescs[0].Description; got != "A crust of ash and salt.\r\n" {
		t.Fatalf("prototype description = %q, want the write-through text", got)
	}
	if got := obj.GetExtraDescs()[0].Description; got != "A crust of ash and salt.\r\n" {
		t.Fatalf("the editing instance sees %q", got)
	}
	// A later read_object copies the edited prototype (src/db.c:1947-1951).
	second := w.NewObjectFromProto(current, -1)
	if got := second.GetExtraDescs()[0].Description; got != "A crust of ash and salt.\r\n" {
		t.Fatalf("a new instance sees %q, want the write-through text", got)
	}
}

// TestStringNewExtraDescStaysInstanceLocal is the other half of C's field 4: a
// keyword the object introduced is a new node at the head of the instance's own
// list (src/modify.c:690-700) and never reaches the prototype.
func TestStringNewExtraDescStaysInstanceLocal(t *testing.T) {
	s, w, obj := newExtraDescSessionWorld(t)

	runString(t, s, "obj bread description ribbon")
	if got := readMsgText(t, s); got != stringNewField {
		t.Fatalf("new-keyword entry = %q, want %q", got, stringNewField)
	}
	s.handleTextEditInput("A blue ribbon.")
	s.handleTextEditInput("@")

	current, _ := w.GetObjPrototype(3001)
	if len(current.ExtraDescs) != 1 || current.ExtraDescs[0].Keywords != "crust" {
		t.Fatalf("prototype gained an instance-local keyword: %+v", current.ExtraDescs)
	}
	descs := obj.GetExtraDescs()
	if len(descs) != 2 || descs[0].Keywords != "ribbon" || descs[1].Keywords != "crust" {
		t.Fatalf("instance list = %+v, want the new keyword at the head", descs)
	}
	if descs[0].Description != "A blue ribbon.\r\n" {
		t.Fatalf("instance keyword text = %q", descs[0].Description)
	}
	// The prototype keyword is untouched, and a fresh instance does not see the
	// instance-local one.
	fresh := w.NewObjectFromProto(current, -1)
	if got := fresh.GetExtraDescs(); len(got) != 1 || got[0].Keywords != "crust" {
		t.Fatalf("fresh instance list = %+v", got)
	}
}

// TestStringExistingExtraDescSurvivesAPriorSnapshot covers the mixed case: the
// instance already published its own list (it created "ribbon"), and the
// prototype keyword it shares is edited afterwards. C keeps both, because the
// instance's chain still points at the prototype's node.
func TestStringExistingExtraDescSurvivesAPriorSnapshot(t *testing.T) {
	s, w, obj := newExtraDescSessionWorld(t)

	// Instance-local keyword first, so the instance owns a published snapshot.
	runString(t, s, "obj bread description ribbon")
	_ = readMsgText(t, s)
	s.handleTextEditInput("A blue ribbon.")
	s.handleTextEditInput("@")

	// Now the shared prototype keyword.
	runString(t, s, "obj bread description crust")
	_ = readMsgText(t, s)
	s.handleTextEditInput("A crust of ash.")
	s.handleTextEditInput("@")

	current, _ := w.GetObjPrototype(3001)
	if got := current.ExtraDescs[0].Description; got != "A crust of ash.\r\n" {
		t.Fatalf("prototype description = %q", got)
	}
	descs := obj.GetExtraDescs()
	if len(descs) != 2 || descs[0].Keywords != "ribbon" || descs[1].Keywords != "crust" {
		t.Fatalf("instance list = %+v", descs)
	}
	if descs[0].Description != "A blue ribbon.\r\n" {
		t.Fatalf("instance-owned entry changed to %q", descs[0].Description)
	}
	if descs[1].Description != "A crust of ash.\r\n" {
		t.Fatalf("prototype-owned entry = %q, want the write-through text", descs[1].Description)
	}
}
