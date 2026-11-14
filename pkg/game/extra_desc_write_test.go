package game

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// extraDescWorld builds a one-room world with one object prototype carrying two
// extra descriptions, plus one live instance of it.
func extraDescWorld(t *testing.T) (*World, *parser.Obj, *ObjectInstance) {
	t.Helper()
	proto := &parser.Obj{
		VNum:      3001,
		Keywords:  "bread",
		ShortDesc: "a loaf of bread",
		ExtraDescs: []parser.ExtraDesc{
			{Keywords: "crust", Description: "A golden crust."},
			{Keywords: "crumb", Description: "A single crumb."},
		},
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Bakery", Zone: 1}},
		Objs:  []parser.Obj{*proto},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	live, ok := w.GetObjPrototype(3001)
	if !ok {
		t.Fatal("prototype 3001 missing")
	}
	inst := w.NewObjectFromProto(live, 1001)
	if inst == nil {
		t.Fatal("NewObjectFromProto returned nil")
	}
	return w, live, inst
}

// TestSetLiveObjExtraDescPublishesCloneAndPatchesInstances covers the world half
// of DP-1292's option A: the prototype entry changes, escaped prototype pointers
// stay immutable (C's edit is visible to every instance because it mutates the
// shared node; the port publishes a clone instead), and the live instance's
// published list carries the same text while its overrides survive.
func TestSetLiveObjExtraDescPublishesCloneAndPatchesInstances(t *testing.T) {
	w, old, inst := extraDescWorld(t)
	weight := 7
	values := [4]int{1, 2, 3, 4}
	instant := 9
	inst.WeightOverride = &weight
	inst.ValuesOverride = &values
	inst.Timer = instant
	inst.AffectsOverride = []parser.ObjAffect{{Location: 1, Modifier: 2}}

	if !w.SetLiveObjExtraDesc(3001, "crust", "A crust of ash and salt.") {
		t.Fatal("SetLiveObjExtraDesc refused an existing prototype keyword")
	}

	current, ok := w.GetObjPrototype(3001)
	if !ok {
		t.Fatal("prototype 3001 vanished")
	}
	if current == old {
		t.Fatal("prototype pointer was reused; escaped readers would see the edit in place")
	}
	if got := current.ExtraDescs[0].Description; got != "A crust of ash and salt." {
		t.Fatalf("prototype description = %q", got)
	}
	if got := current.ExtraDescs[1].Description; got != "A single crumb." {
		t.Fatalf("sibling entry changed: %q", got)
	}
	if got := old.ExtraDescs[0].Description; got != "A golden crust." {
		t.Fatalf("the pointer readers already hold changed to %q", got)
	}
	descs := inst.GetExtraDescs()
	if len(descs) != 2 || descs[0].Description != "A crust of ash and salt." || descs[1].Description != "A single crumb." {
		t.Fatalf("instance list = %+v", descs)
	}
	if inst.WeightOverride != &weight || *inst.WeightOverride != 7 {
		t.Fatal("WeightOverride was replaced by the extra-description write")
	}
	if inst.ValuesOverride != &values {
		t.Fatal("ValuesOverride was replaced by the extra-description write")
	}
	if len(inst.AffectsOverride) != 1 {
		t.Fatalf("AffectsOverride = %+v", inst.AffectsOverride)
	}
	if inst.Timer != instant {
		t.Fatalf("Timer = %d", inst.Timer)
	}

	if w.SetLiveObjExtraDesc(3001, "no-such-keyword", "x") {
		t.Fatal("unknown keyword accepted")
	}
	if w.SetLiveObjExtraDesc(9999, "crust", "x") {
		t.Fatal("missing prototype accepted")
	}
}

// TestClearLiveObjExtraDescMarksNull covers the editor-entry half: C frees the
// node's description and leaves the pointer NULL (src/modify.c:702-704), so the
// keyword stops matching until the editor writes.
func TestClearLiveObjExtraDescMarksNull(t *testing.T) {
	w, _, inst := extraDescWorld(t)
	if !w.ClearLiveObjExtraDesc(3001, "CRUST") {
		t.Fatal("ClearLiveObjExtraDesc refused a case-insensitive match")
	}
	current, _ := w.GetObjPrototype(3001)
	entry := current.ExtraDescs[0]
	if entry.Description != "" || !entry.NullDescription {
		t.Fatalf("prototype entry = %+v, want an empty NULL description", entry)
	}
	got := inst.GetExtraDescs()[0]
	if got.Description != "" || !got.NullDescription {
		t.Fatalf("instance entry = %+v, want an empty NULL description", got)
	}
}

// TestSetLiveObjExtraDescPatchesAPreexistingSnapshot is the case the issue is
// about: an instance that already published its own list (it created a keyword)
// still sees a prototype keyword edited later, and keeps its own entry.
func TestSetLiveObjExtraDescPatchesAPreexistingSnapshot(t *testing.T) {
	w, _, inst := extraDescWorld(t)
	// do_string prepends instance-created nodes, so the own entry sits first.
	inst.SetLiveExtraDescs([]parser.ExtraDesc{
		{Keywords: "ribbon", Description: "A blue ribbon."},
		{Keywords: "crust", Description: "A golden crust."},
		{Keywords: "crumb", Description: "A single crumb."},
	})

	if !w.SetLiveObjExtraDesc(3001, "crust", "Edited later.") {
		t.Fatal("SetLiveObjExtraDesc refused")
	}
	descs := inst.GetExtraDescs()
	if len(descs) != 3 {
		t.Fatalf("instance list = %+v, want the instance entry preserved", descs)
	}
	if descs[0].Keywords != "ribbon" || descs[0].Description != "A blue ribbon." {
		t.Fatalf("instance-owned entry changed: %+v", descs[0])
	}
	if descs[1].Description != "Edited later." {
		t.Fatalf("prototype-owned entry = %+v, want the write-through text", descs[1])
	}
}

// TestRefreshLiveObjInstancesDiscardsLiveExtraDescs mirrors C's
// oedit_save_internally, whose whole-struct copy (*obj = obj_proto[nr]) throws
// away an instance's own extra-description edits.
func TestRefreshLiveObjInstancesDiscardsLiveExtraDescs(t *testing.T) {
	w, _, inst := extraDescWorld(t)
	inst.SetLiveExtraDescs([]parser.ExtraDesc{{Keywords: "ribbon", Description: "A blue ribbon."}})
	current, _ := w.GetObjPrototype(3001)
	w.RefreshLiveObjInstances(3001, *current)

	descs := inst.GetExtraDescs()
	if len(descs) != 2 || descs[0].Keywords != "crust" {
		t.Fatalf("instance list after the OLC replace = %+v, want the prototype's", descs)
	}
	if inst.Runtime.ExtraDescsLive {
		t.Fatal("published list still marked live after the whole-struct replace")
	}
}

// TestSetLiveObjExtraDescConcurrentReaders is the race witness: a write-through
// in one player's goroutine must not race looks in another's. Run under -race.
func TestSetLiveObjExtraDescConcurrentReaders(t *testing.T) {
	w, _, inst := extraDescWorld(t)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if descs := inst.GetExtraDescs(); len(descs) != 2 {
					t.Errorf("reader saw %d extra descriptions", len(descs))
					return
				}
				if _, ok := w.GetObjPrototype(3001); !ok {
					t.Error("prototype lookup failed while a write was in flight")
					return
				}
			}
		}()
	}
	for i := 0; i < 2000; i++ {
		if !w.SetLiveObjExtraDesc(3001, "crust", fmt.Sprintf("crust %d", i)) {
			t.Fatalf("write %d refused", i)
		}
	}
	close(stop)
	wg.Wait()
}
