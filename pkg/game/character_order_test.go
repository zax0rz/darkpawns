package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Mob snapshots follow the links C builds: char_to_room prepends to people
// (src/handler.c:532-550), obj_to_char prepends to carrying
// (src/handler.c:559-571), and obj_to_room prepends to contents
// (src/handler.c:897-910).
func TestMobSpecialSnapshotsUseCListOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "First"}, {VNum: 1002, Name: "Second"}},
		Mobs:  []parser.Mob{{VNum: 99001, Keywords: "probe", ShortDesc: "a probe"}},
		Objs: []parser.Obj{
			{VNum: 99002, Keywords: "one", ShortDesc: "one"},
			{VNum: 99003, Keywords: "two", ShortDesc: "two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	older, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	assertMobOrder := func(want ...*MobInstance) {
		t.Helper()
		got := w.COrderedRoomMobs(1001)
		if len(got) != len(want) {
			t.Fatalf("room mob count = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("room mob at %d = %p, want %p", i, got[i], want[i])
			}
		}
	}
	assertMobOrder(newer, older)
	older.SetRoom(1002)
	older.SetRoom(1001)
	assertMobOrder(older, newer)

	first, err := w.SpawnObject(99002, -1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnObject(99003, -1)
	if err != nil {
		t.Fatal(err)
	}
	older.AddToInventory(first)
	older.AddToInventory(second)
	if got := older.InventorySnapshot(); len(got) != 2 || got[0] != second || got[1] != first {
		t.Fatalf("mob inventory = %v, want [second, first] in obj_to_char prepend order", got)
	}

	if err := w.MoveObjectToRoom(first, 1001); err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(second, 1001); err != nil {
		t.Fatal(err)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 2 || got[0] != second || got[1] != first {
		t.Fatalf("room objects = %v, want [second, first] in obj_to_room prepend order", got)
	}

	rawFirst, err := w.SpawnObject(99002, -1)
	if err != nil {
		t.Fatal(err)
	}
	rawSecond, err := w.SpawnObject(99003, -1)
	if err != nil {
		t.Fatal(err)
	}
	w.AddItemToRoom(rawFirst, 1001)
	w.AddItemToRoom(rawSecond, 1001)
	if got := w.GetItemsInRoom(1001); len(got) != 4 || got[0] != rawSecond || got[1] != rawFirst || got[2] != second || got[3] != first {
		t.Fatalf("raw room placements = %v, want new objects prepended in call order", got)
	}
}

func TestGetAllCharsInRoomUsesCPrependOrderAcrossBodyTypes(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Order proof"}},
		Mobs:  []parser.Mob{{VNum: 99001, Keywords: "probe", ShortDesc: "a probe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	first := NewPlayer(1, "First", 1001)
	if err := w.AddPlayer(first); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMobQuiet(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	last := NewPlayer(2, "Last", 1001)
	if err := w.AddPlayer(last); err != nil {
		t.Fatal(err)
	}
	got := w.GetAllCharsInRoom(1001)
	if len(got) != 3 || got[0] != last || got[1] != mob || got[2] != first {
		t.Fatalf("room body order = %#v, want newest-first [last player, mob, first player]", got)
	}
}
