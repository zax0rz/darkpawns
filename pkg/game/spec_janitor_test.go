package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecJanitor_EntryGates(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*MobInstance)
	}{
		{name: "command", setup: func(*MobInstance) {}},
		{name: "sleeping", setup: func(mob *MobInstance) { mob.SetPosition(combat.PosSleeping) }},
		{name: "dead", setup: func(mob *MobInstance) { mob.CurrentHP = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			tt.setup(mob)
			lastMsg()

			cmd := ""
			if tt.name == "command" {
				cmd = "look"
			}
			if specJanitor(w, nil, mob, cmd, "") {
				t.Fatalf("%s gate should return false", tt.name)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("%s gate output = %q, want empty", tt.name, got)
			}
		})
	}
}

func TestSpecJanitor_PredicateAndTransfer(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	corpseLikeProto := &parser.Obj{
		VNum:      3301,
		Keywords:  "corpse",
		ShortDesc: "a corpse-shaped prop",
		WearFlags: [4]int{1, 0, 0, 0},
	}
	nontakeableProto := &parser.Obj{
		VNum:      3302,
		Keywords:  "broken stone",
		ShortDesc: "a broken stone",
	}
	trashProto := &parser.Obj{
		VNum:      3303,
		Keywords:  "scrap",
		ShortDesc: "a scrap of paper",
		WearFlags: [4]int{1, 0, 0, 0},
	}
	existingProto := &parser.Obj{
		VNum:      3304,
		Keywords:  "old coin",
		ShortDesc: "an old coin",
		WearFlags: [4]int{1, 0, 0, 0},
	}
	w.mu.Lock()
	for _, proto := range []*parser.Obj{corpseLikeProto, nontakeableProto, trashProto, existingProto} {
		w.objs[proto.VNum] = proto
	}
	w.mu.Unlock()

	existing, err := w.SpawnObject(existingProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject existing: %v", err)
	}
	if err := w.MoveObjectToMobInventory(existing, mob); err != nil {
		t.Fatalf("MoveObjectToMobInventory existing: %v", err)
	}
	for _, vnum := range []int{corpseLikeProto.VNum, nontakeableProto.VNum, trashProto.VNum} {
		obj, spawnErr := w.SpawnObject(vnum, -1)
		if spawnErr != nil {
			t.Fatalf("SpawnObject %d: %v", vnum, spawnErr)
		}
		w.AddItemToRoom(obj, 1001)
	}
	lastMsg()

	if !specJanitor(w, nil, mob, "", "") {
		t.Fatal("takeable non-corpse object should be handled")
	}
	if got := lastMsg(); got != "A test mob picks up some trash.\r\n" {
		t.Fatalf("janitor output = %q, want exact C room Act", got)
	}
	items := w.GetItemsInRoom(1001)
	if len(items) != 2 {
		t.Fatalf("room contents after janitor = %d, want 2 skipped objects", len(items))
	}
	if len(mob.Inventory) != 2 || mob.Inventory[0].GetVNum() != trashProto.VNum || mob.Inventory[1] != existing {
		t.Fatalf("mob inventory after janitor = %#v, want prepended trash before existing item", mob.Inventory)
	}
	if mob.Inventory[0].Location != LocInventoryMob(mob.GetID()) {
		t.Fatalf("picked object location = %#v, want mob inventory", mob.Inventory[0].Location)
	}
}

func TestSpecJanitor_NoEligibleObject(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	proto := &parser.Obj{
		VNum:      3401,
		Keywords:  "dust",
		ShortDesc: "some dust",
	}
	w.mu.Lock()
	w.objs[proto.VNum] = proto
	w.mu.Unlock()
	obj, err := w.SpawnObject(proto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.AddItemToRoom(obj, 1001)
	lastMsg()

	if specJanitor(w, nil, mob, "", "") {
		t.Fatal("non-takeable object should not be handled")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("no-eligible output = %q, want empty", got)
	}
}
