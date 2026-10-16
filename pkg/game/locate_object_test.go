package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newLocateObjectTestWorld(t *testing.T) (*World, *Player, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Name: "Test Room"}},
		Objs: []parser.Obj{
			{VNum: 1, Keywords: "silver coin", ShortDesc: "a silver coin"},
			{VNum: 2, Keywords: "leather bag", ShortDesc: "a leather bag", TypeFlag: ITEM_CONTAINER, Values: [4]int{1000, 0, -1}},
			{VNum: 3, Keywords: "silver coin", ShortDesc: "a silver coin"},
			{VNum: 4, Keywords: "silver coin", ShortDesc: "a silver coin"},
			{VNum: 5, Keywords: "silver circlet", ShortDesc: "a silver circlet", WearFlags: [4]int{1 << 4}},
			{VNum: 6, Keywords: "silverfish", ShortDesc: "a silverfish"},
			{VNum: 7, Keywords: "silver charm", ShortDesc: "a silver charm", ExtraFlags: [4]int{1 << 25}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	viewer := NewPlayer(1, "Seeker", 100)
	viewer.Level = 30
	viewer.RoomVNum = 100
	w.AddPlayer(viewer)
	carrier := NewPlayer(2, "Carrier", 100)
	carrier.Level = 5
	carrier.RoomVNum = 100
	w.AddPlayer(carrier)
	return w, viewer, carrier
}

func TestFindObjectByNameUsesCGlobalObjectListOrderAndTokens(t *testing.T) {
	w, _, carrier := newLocateObjectTestWorld(t)

	roomCoin, err := w.SpawnObject(1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(roomCoin, 100); err != nil {
		t.Fatal(err)
	}
	bag, err := w.SpawnObject(2, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(bag, carrier); err != nil {
		t.Fatal(err)
	}
	containedCoin, err := w.SpawnObject(3, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToContainer(containedCoin, bag); err != nil {
		t.Fatal(err)
	}
	carriedCoin, err := w.SpawnObject(4, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(carriedCoin, carrier); err != nil {
		t.Fatal(err)
	}
	wornCirclet, err := w.SpawnObject(5, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(wornCirclet, carrier); err != nil {
		t.Fatal(err)
	}
	if err := w.EquipItem(carrier, wornCirclet, 6); err != nil { // C WEAR_HEAD.
		t.Fatal(err)
	}
	substringOnly, err := w.SpawnObject(6, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(substringOnly, 100); err != nil {
		t.Fatal(err)
	}

	got := w.FindObjectByName("silver coin")
	if len(got) != 4 {
		t.Fatalf("FindObjectByName matched %d objects, want four global exact-token matches", len(got))
	}
	if got[0] != wornCirclet || got[1] != carriedCoin || got[2] != containedCoin || got[3] != roomCoin {
		t.Fatalf("FindObjectByName order/types = %T %T %T %T, want newest-first global matches", got[0], got[1], got[2], got[3])
	}
	for _, candidate := range got {
		if candidate == substringOnly {
			t.Fatal("C isname matched silverfish as a substring of silver")
		}
	}
}

func TestLocateObjectResultsMatchCOutputAndNoLocateOrder(t *testing.T) {
	w, viewer, carrier := newLocateObjectTestWorld(t)

	roomCoin, err := w.SpawnObject(1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(roomCoin, 100); err != nil {
		t.Fatal(err)
	}
	bag, err := w.SpawnObject(2, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(bag, carrier); err != nil {
		t.Fatal(err)
	}
	containedCoin, err := w.SpawnObject(3, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToContainer(containedCoin, bag); err != nil {
		t.Fatal(err)
	}
	carriedCoin, err := w.SpawnObject(4, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(carriedCoin, carrier); err != nil {
		t.Fatal(err)
	}
	wornCirclet, err := w.SpawnObject(5, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(wornCirclet, carrier); err != nil {
		t.Fatal(err)
	}
	if err := w.EquipItem(carrier, wornCirclet, 6); err != nil {
		t.Fatal(err)
	}

	results, blocked := w.LocateObjectResults("silver coin", viewer, 10, "yocahu asido")
	if blocked {
		t.Fatal("ordinary silver objects unexpectedly triggered ITEM_NOLOCATE")
	}
	want := []string{
		"A silver circlet is being worn by Carrier.\n\r",
		"A silver coin is being carried by Carrier.\n\r",
		"A silver coin is in a leather bag.\n\r",
		"A silver coin is in Test Room.\n\r",
	}
	if strings.Join(results, "") != strings.Join(want, "") {
		t.Fatalf("locate output = %q, want %q", strings.Join(results, ""), strings.Join(want, ""))
	}

	noLocate, err := w.SpawnObject(7, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(noLocate, 100); err != nil {
		t.Fatal(err)
	}
	results, blocked = w.LocateObjectResults("silver", viewer, 10, "yocahu asido")
	if !blocked || len(results) != 0 {
		t.Fatalf("matching ITEM_NOLOCATE result = (%v, %v), want (nil, true)", results, blocked)
	}
	results, blocked = w.LocateObjectResults("silver", viewer, 0, "yocahu asido")
	if blocked || len(results) != 0 {
		t.Fatalf("zero result limit = (%v, %v), want no scan and no block", results, blocked)
	}
}

func TestLocateObjectInvisibleOwnerReusesSaySpellBuffer(t *testing.T) {
	w, viewer, carrier := newLocateObjectTestWorld(t)
	viewer.Level = 30
	carrier.SetInvisLevel(40)
	coin, err := w.SpawnObject(4, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(coin, carrier); err != nil {
		t.Fatal(err)
	}

	results, blocked := w.LocateObjectResults("silver", viewer, 15, "yocahu asido")
	if blocked || len(results) != 1 || results[0] != "Yocahu asido" {
		t.Fatalf("invisible-owner output = (%q, %v), want C's capitalized say_spell buffer without line ending", results, blocked)
	}
}
