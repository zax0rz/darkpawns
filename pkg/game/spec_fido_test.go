package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecFido_EntryGates(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*MobInstance)
	}{
		{name: "command", setup: func(*MobInstance) {}},
		{name: "fighting", setup: func(mob *MobInstance) { mob.SetFighting("Tester") }},
		{name: "sleeping", setup: func(mob *MobInstance) { mob.SetPosition(combat.PosSleeping) }},
		{name: "dead", setup: func(mob *MobInstance) { mob.CurrentHP = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			tt.setup(mob)
			lastMsg() // discard the spawn announcement

			cmd := ""
			if tt.name == "command" {
				cmd = "look"
			}
			if specFido(w, nil, mob, cmd, "") {
				t.Fatalf("%s gate should return false", tt.name)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("%s gate output = %q, want empty", tt.name, got)
			}
		})
	}
}

func TestSpecFido_CorpsePredicateAndTransfer(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	containerProto := &parser.Obj{
		VNum:      3101,
		Keywords:  "body remains",
		ShortDesc: "the body remains",
		TypeFlag:  ITEM_CONTAINER,
		Values:    [4]int{0, 0, 0, 1},
	}
	contentProto := &parser.Obj{
		VNum:      3102,
		Keywords:  "silver ring",
		ShortDesc: "a silver ring",
	}
	secondContentProto := &parser.Obj{
		VNum:      3103,
		Keywords:  "gold coin",
		ShortDesc: "a gold coin",
	}
	w.mu.Lock()
	w.objs[containerProto.VNum] = containerProto
	w.objs[contentProto.VNum] = contentProto
	w.objs[secondContentProto.VNum] = secondContentProto
	w.mu.Unlock()

	corpse, err := w.SpawnObject(containerProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject corpse: %v", err)
	}
	w.AddItemToRoom(corpse, 1001)
	content, err := w.SpawnObject(contentProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject content: %v", err)
	}
	if err := w.MoveObjectToContainer(content, corpse); err != nil {
		t.Fatalf("MoveObjectToContainer: %v", err)
	}
	secondContent, err := w.SpawnObject(secondContentProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject second content: %v", err)
	}
	if err := w.MoveObjectToContainer(secondContent, corpse); err != nil {
		t.Fatalf("MoveObjectToContainer second content: %v", err)
	}
	lastMsg()

	if !specFido(w, nil, mob, "", "") {
		t.Fatal("C corpse predicate should be handled")
	}
	if got := lastMsg(); got != "A test mob savagely devours a corpse.\r\n" {
		t.Fatalf("fido output = %q, want exact C room Act", got)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 2 || got[0] != content || got[1] != secondContent {
		t.Fatalf("room contents after fido = %#v, want C prepend order", got)
	}
	if content.Location != LocRoom(1001) {
		t.Fatalf("content location = %#v, want room", content.Location)
	}
	if secondContent.Location != LocRoom(1001) {
		t.Fatalf("second content location = %#v, want room", secondContent.Location)
	}
	if len(corpse.GetContents()) != 0 {
		t.Fatalf("corpse contents = %#v, want empty", corpse.GetContents())
	}
}

func TestSpecFido_UsesContainerValueNotKeyword(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	keywordProto := &parser.Obj{
		VNum:      3201,
		Keywords:  "corpse-looking thing",
		ShortDesc: "a corpse-looking thing",
		TypeFlag:  ITEM_CONTAINER,
		Values:    [4]int{0, 0, 0, 0},
	}
	valueProto := &parser.Obj{
		VNum:      3202,
		Keywords:  "ordinary box",
		ShortDesc: "an ordinary box",
		TypeFlag:  ITEM_OTHER,
		Values:    [4]int{0, 0, 0, 1},
	}
	w.mu.Lock()
	w.objs[keywordProto.VNum] = keywordProto
	w.objs[valueProto.VNum] = valueProto
	w.mu.Unlock()
	for _, vnum := range []int{keywordProto.VNum, valueProto.VNum} {
		obj, err := w.SpawnObject(vnum, -1)
		if err != nil {
			t.Fatalf("SpawnObject %d: %v", vnum, err)
		}
		w.AddItemToRoom(obj, 1001)
	}
	lastMsg()

	if specFido(w, nil, mob, "", "") {
		t.Fatal("non-corpse containers should not be handled")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("non-corpse predicate output = %q, want empty", got)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 2 {
		t.Fatalf("room contents after non-corpse scan = %d, want 2", len(got))
	}
}
