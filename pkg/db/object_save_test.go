package db

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestObjectSaveIdentity(t *testing.T) {
	for name, want := range map[string]string{"Alice": "plrobjs/A-E/alice.objs", "FROG": "plrobjs/F-J/frog.objs", "Knight": "plrobjs/K-O/knight.objs", "Saver": "plrobjs/P-T/saver.objs", "Zebra": "plrobjs/U-Z/zebra.objs"} {
		if got := ObjectSaveIdentity(name); got != want {
			t.Fatalf("%s: %s", name, got)
		}
	}
}

func TestObjectSavePostorderDepth(t *testing.T) {
	inv := []game.SaveItemData{{VNum: 10, Count: 1}, {VNum: 11, Count: 1, ContainerIndex: 1}, {VNum: 12, Count: 1, ContainerIndex: 2}, {VNum: 13, Count: 1, ContainerIndex: 1}, {VNum: 20, Count: 2}}
	eq := []game.SaveItemData{{VNum: 30, Count: 1, Locate: 17}, {VNum: 31, Count: 1, ContainerIndex: 1}, {VNum: 40, Count: 1, Locate: 6}}
	a, _ := json.Marshal(inv)
	b, _ := json.Marshal(eq)
	got, err := objectSaveRecords(a, b)
	if err != nil {
		t.Fatal(err)
	}
	var pairs [][2]int
	for _, item := range got {
		pairs = append(pairs, [2]int{item.VNum, item.Locate})
	}
	want := [][2]int{{40, 6}, {31, -1}, {30, 17}, {20, 0}, {20, 0}, {13, -1}, {12, -2}, {11, -1}, {10, 0}}
	if !reflect.DeepEqual(pairs, want) {
		t.Fatalf("record order/depth=%v want=%v", pairs, want)
	}
}

func TestObjectSavePresenceAndLoadRewrite(t *testing.T) {
	d := openGameStore(t, gameStoreBackends(t)[0].dsn)
	p := &PlayerRecord{Name: "Legacy", Inventory: []byte(`[{"vnum":10,"count":1}]`)}
	if err := d.CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	if err := ObjectSaveLoaded(d, "Legacy"); err != nil {
		t.Fatal(err)
	}
	s, err := d.GetObjectSave("Legacy")
	if err != nil || s != nil {
		t.Fatalf("legacy history invented: %v %v", s, err)
	}
	if err := SaveObjectSnapshot(d, "Legacy", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	s, err = d.GetObjectSave("LEGACY")
	if err != nil || s == nil || s.Kind != 2 || len(s.Objects) != 0 {
		t.Fatalf("empty save lost: %+v %v", s, err)
	}
	if err := d.SavePlayer(p); err != nil {
		t.Fatal(err)
	}
	if err := d.createTables(); err != nil {
		t.Fatal(err)
	}
	s, err = d.GetObjectSave("Legacy")
	if err != nil || s.Kind != 2 {
		t.Fatal("character save/restart changed object metadata")
	}
	if err := ObjectSaveLoaded(d, "Legacy"); err != nil {
		t.Fatal(err)
	}
	s, err = d.GetObjectSave("Legacy")
	if err != nil || s.Kind != 1 {
		t.Fatal("load did not rewrite header")
	}
}

func TestObjectSaveDelete(t *testing.T) {
	d := openGameStore(t, gameStoreBackends(t)[0].dsn)
	if err := SaveObjectSnapshot(d, "Deleted", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := DeleteObjectSave(d, "Deleted"); err != nil {
		t.Fatal(err)
	}
	saved, err := d.GetObjectSave("deleted")
	if err != nil || saved != nil {
		t.Fatalf("deleted identity remained: %+v %v", saved, err)
	}
}
