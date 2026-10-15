package game

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCOrderedCharactersMergesPlayerAndMobileInsertions(t *testing.T) {
	w := newMobactDrawTestWorld()
	w.mobs[9401] = &parser.Mob{VNum: 9401, Keywords: "older", ShortDesc: "an older mob", Position: 8, DefaultPos: 8}
	w.mobs[9402] = &parser.Mob{VNum: 9402, Keywords: "newer", ShortDesc: "a newer mob", Position: 8, DefaultPos: 8}
	older, err := w.SpawnMob(9401, 1001)
	if err != nil {
		t.Fatalf("SpawnMob older: %v", err)
	}
	player := NewPlayer(1, "Middle", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	newer, err := w.SpawnMob(9402, 1001)
	if err != nil {
		t.Fatalf("SpawnMob newer: %v", err)
	}

	got := w.cOrderedCharacters()
	if len(got) != 3 || got[0].mob != newer || got[1].player != player || got[2].mob != older {
		t.Fatalf("C character order = %#v, want newer mob, player, older mob", got)
	}
}

func TestLiveObjectsInCOrder(t *testing.T) {
	older := &ObjectInstance{ID: 1}
	newer := &ObjectInstance{ID: 2}
	w := &World{objectInstances: map[int]*ObjectInstance{older.ID: older, newer.ID: newer}}

	got := w.liveObjectsInCOrder()
	if len(got) != 2 || !reflect.DeepEqual(got, []*ObjectInstance{newer, older}) {
		gotIDs := make([]int, len(got))
		for i, object := range got {
			gotIDs[i] = object.ID
		}
		t.Fatalf("C object order IDs = %v, want [2 1]", gotIDs)
	}
}
