package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func registerBeheadPrototypes(t *testing.T, w *World) {
	t.Helper()
	w.mu.Lock()
	w.objs[16] = &parser.Obj{
		VNum: 16, Keywords: "head", ShortDesc: "the head of someone",
		LongDesc: "The head of someone lies here.", TypeFlag: ITEM_DRINKCON,
		WearFlags: [4]int{16385}, Values: [4]int{1, 1, 13, 0}, Weight: 1,
	}
	w.objs[17] = &parser.Obj{
		VNum: 17, Keywords: "corpse headless beheaded", ShortDesc: "a beheaded corpse",
		LongDesc: "A beheaded corpse is here, spilling fresh blood.", TypeFlag: ITEM_CONTAINER,
		Values: [4]int{1, 0, -1, 0}, Weight: 1,
	}
	w.mu.Unlock()
}

func registerBeheadObject(t *testing.T, w *World, proto *parser.Obj) *ObjectInstance {
	t.Helper()
	w.mu.Lock()
	w.objs[proto.VNum] = proto
	w.mu.Unlock()
	obj, err := w.SpawnObject(proto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", proto.VNum, err)
	}
	if err := w.MoveObjectToRoom(obj, 1001); err != nil {
		t.Fatalf("MoveObjectToRoom(%d): %v", proto.VNum, err)
	}
	return obj
}

func TestDoBehead_ContainerValueSuccessAndTransformation(t *testing.T) {
	w, ch := newTestWorld(t)
	ch.Stats.Str = 10
	ch.Stats.Dex = 10
	registerBeheadPrototypes(t, w)

	original := registerBeheadObject(t, w, &parser.Obj{
		VNum: 4010, Keywords: "fractal thing", ShortDesc: "a fractal thing",
		LongDesc: "A fractal thing lies here.", TypeFlag: ITEM_CONTAINER,
		Values: [4]int{1, 0, -1, 1},
	})
	content := registerBeheadObject(t, w, &parser.Obj{
		VNum: 4011, Keywords: "silver ring", ShortDesc: "a silver ring",
		LongDesc: "A silver ring lies here.", TypeFlag: ITEM_OTHER,
		WearFlags: [4]int{1},
	})
	if err := w.MoveObjectToContainer(content, original); err != nil {
		t.Fatalf("MoveObjectToContainer: %v", err)
	}

	result := DoBehead(ch, "fractal trailing words", w)
	if !result.Success {
		t.Fatalf("DoBehead failed: %#v", result)
	}
	if result.MessageToCh != "You rip the head off a fractal thing with your bare hands!" {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "TestPlayer rips the head off a fractal thing with his bare hands!" {
		t.Errorf("room message = %q", result.MessageToRoom)
	}

	var head, corpse *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if obj.GetVNum() == 16 {
			head = obj
		}
	}
	for _, obj := range w.GetItemsInRoom(1001) {
		if obj.GetVNum() == 17 {
			corpse = obj
		}
	}
	if head == nil {
		t.Fatal("behead did not place the head in inventory")
	}
	if got := head.GetKeywords(); got != "head" {
		t.Errorf("head keywords = %q, want head", got)
	}
	if got := head.GetShortDesc(); got != "a bloody head ripped from a fractal thing" {
		t.Errorf("head short description = %q", got)
	}
	if got := head.GetLongDesc(); got != "A bloody head ripped from a fractal thing has been left here." {
		t.Errorf("head long description = %q", got)
	}
	if corpse == nil {
		t.Fatal("behead did not place a headless corpse in the room")
	}
	if got := corpse.GetKeywords(); got != "fractal thing headless beheaded" {
		t.Errorf("headless corpse keywords = %q", got)
	}
	if got := corpse.GetShortDesc(); got != "a beheaded corpse" {
		t.Errorf("headless corpse short description = %q", got)
	}
	if got := corpse.GetLongDesc(); got != "A beheaded corpse is here, spilling fresh blood." {
		t.Errorf("headless corpse long description = %q", got)
	}
	if corpse.GetTypeFlag() != ITEM_CONTAINER || corpse.GetValue(0) != 0 || corpse.GetValue(3) != 1 {
		t.Errorf("headless corpse values/type = type %d values (%d,%d), want container, 0/1", corpse.GetTypeFlag(), corpse.GetValue(0), corpse.GetValue(3))
	}
	if corpse.Timer != MaxNPCCorpseTime {
		t.Errorf("headless corpse timer = %d, want %d", corpse.Timer, MaxNPCCorpseTime)
	}
	if len(corpse.Contains) != 1 || corpse.Contains[0] != content {
		t.Fatalf("headless corpse contents = %#v, want transferred content", corpse.Contains)
	}
	if content.Location != LocContainer(corpse.ID) {
		t.Errorf("content location = %#v, want headless corpse", content.Location)
	}
	if original.Location != LocNowhere() {
		t.Errorf("original location = %#v, want nowhere", original.Location)
	}
	for _, obj := range w.GetAllObjects() {
		if obj == original {
			t.Fatal("original object remains registered after behead")
		}
	}
}

func TestDoBehead_EntryAndPredicateBranches(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T, *World, *Player)
		argument   string
		wantResult string
	}{
		{
			name:       "self",
			argument:   "self extra",
			wantResult: "This MUD doesn't support self-mutilation!",
		},
		{
			name:       "non-container",
			argument:   "stone",
			wantResult: "You can't behead that!",
			setup: func(t *testing.T, w *World, _ *Player) {
				registerBeheadObject(t, w, &parser.Obj{
					VNum: 4012, Keywords: "stone", ShortDesc: "a stone", TypeFlag: ITEM_OTHER,
				})
			},
		},
		{
			name:       "headless",
			argument:   "corpse",
			wantResult: "You can't behead something without a head!",
			setup: func(t *testing.T, w *World, _ *Player) {
				registerBeheadObject(t, w, &parser.Obj{
					VNum: 4013, Keywords: "corpse headless", ShortDesc: "a headless corpse",
					TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1},
				})
			},
		},
		{
			name:       "living target",
			argument:   "guard trailing words",
			wantResult: "You kill it first and THEN you behead it!",
			setup: func(t *testing.T, w *World, _ *Player) {
				w.mu.Lock()
				w.mobs[4016] = &parser.Mob{
					VNum: 4016, Keywords: "guard", ShortDesc: "a guard",
					LongDesc: "A guard is here.", Level: 5,
				}
				w.mu.Unlock()
				if _, err := w.SpawnMob(4016, 1001); err != nil {
					t.Fatalf("SpawnMob: %v", err)
				}
			},
		},
		{
			name:       "missing object",
			argument:   "missing trailing words",
			wantResult: "You can't seem to find a missing to behead!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ch := newTestWorld(t)
			registerBeheadPrototypes(t, w)
			if tt.setup != nil {
				tt.setup(t, w, ch)
			}
			result := DoBehead(ch, tt.argument, w)
			if result.Success {
				t.Fatal("DoBehead unexpectedly succeeded")
			}
			if result.MessageToCh != tt.wantResult {
				t.Errorf("message = %q, want %q", result.MessageToCh, tt.wantResult)
			}
		})
	}
}

func TestDoBehead_SlashWeaponMessages(t *testing.T) {
	w, ch := newTestWorld(t)
	ch.Stats.Str = 10
	ch.Stats.Dex = 10
	registerBeheadPrototypes(t, w)
	registerBeheadObject(t, w, &parser.Obj{
		VNum: 4014, Keywords: "box", ShortDesc: "a box", TypeFlag: ITEM_CONTAINER,
		Values: [4]int{1, 0, -1, 1},
	})
	w.mu.Lock()
	w.objs[4015] = &parser.Obj{
		VNum: 4015, Keywords: "sword", ShortDesc: "a sword", TypeFlag: ITEM_WEAPON,
		WearFlags: [4]int{1 << 13}, Values: [4]int{0, 1, 4, 3},
	}
	w.mu.Unlock()
	weapon, err := w.SpawnObject(4015, -1)
	if err != nil {
		t.Fatalf("SpawnObject weapon: %v", err)
	}
	if err := w.MoveObject(weapon, LocEquippedPlayer(ch.Name, SlotWield)); err != nil {
		t.Fatalf("equip weapon: %v", err)
	}

	result := DoBehead(ch, "box", w)
	if !result.Success {
		t.Fatalf("DoBehead failed: %#v", result)
	}
	if result.MessageToCh != "You behead a box!" || result.MessageToRoom != "TestPlayer beheads a box!" {
		t.Errorf("slash messages = (%q, %q)", result.MessageToCh, result.MessageToRoom)
	}
}
