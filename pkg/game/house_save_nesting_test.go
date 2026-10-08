package game

// DP-1401, Option A: house saves record container nesting by file index
// (the C obj_file_elem "locate" idiom), and loading floors whatever cannot
// be nested — which is what C's own House_load does with contained items
// (house.c:70 obj_to_room's every record). Before this change the save wrote
// a runtime registry ID as container_id, which can never match the fresh
// objects a later boot builds, so contained items were silently dropped
// instead of landing on the floor. The round-trip and legacy tests fail with
// the fix reverted.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newHouseNestingWorld builds a world with one house room (4100), a sword
// (3001), a bag container (3002), and a chest container (3003).
func newHouseNestingWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 4100, Name: "House Room", Zone: 1, Exits: map[string]parser.Exit{}}},
		Objs: []parser.Obj{
			{VNum: 3001, Keywords: "sword", ShortDesc: "a sword", TypeFlag: 5},
			{VNum: 3002, Keywords: "bag", ShortDesc: "a bag", TypeFlag: 15},
			{VNum: 3003, Keywords: "chest", ShortDesc: "a chest", TypeFlag: 15},
		},
		Zones: []parser.Zone{{Number: 1, Name: "Test Zone", TopRoom: 4999}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// findInRoomByVNum returns the first room object with the given vnum.
func findInRoomByVNum(w *World, room, vnum int) *ObjectInstance {
	for _, obj := range w.GetItemsInRoom(room) {
		if obj.VNum == vnum {
			return obj
		}
	}
	return nil
}

// countObjectsInWorld counts registered objects (the registry is the
// authority on what exists).
func countObjectsInWorld(w *World) int {
	return len(w.GetAllObjects())
}

func TestHouseSaveLoadRoundTripsContainerNesting(t *testing.T) {
	t.Chdir(t.TempDir()) // house/4100.house is written relative to cwd
	w := newHouseNestingWorld(t)

	bag, err := w.SpawnObject(3002, 4100)
	if err != nil {
		t.Fatalf("spawn bag: %v", err)
	}
	sword, err := w.SpawnObject(3001, 4100)
	if err != nil {
		t.Fatalf("spawn sword: %v", err)
	}
	// SpawnObject registers the object; the room placement is a move.
	if err := w.MoveObjectToRoom(bag, 4100); err != nil {
		t.Fatalf("place bag in room: %v", err)
	}
	if err := w.MoveObjectToContainer(sword, bag); err != nil {
		t.Fatalf("nest sword in bag: %v", err)
	}

	w.houseCrashsave(4100)

	// The file records one item per object, contents first — C's House_save
	// order (house.c:112-120).
	var saved houseSaveData
	raw, err := os.ReadFile(filepath.Clean(HouseGetFilename(4100)))
	if err != nil {
		t.Fatalf("read save file: %v", err)
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("parse save file: %v", err)
	}
	if len(saved.Items) != 2 {
		t.Fatalf("saved items = %d, want 2 (sword + bag): %s", len(saved.Items), raw)
	}
	if saved.Items[0].VNum != 3001 || saved.Items[1].VNum != 3002 {
		t.Errorf("save order = [%d, %d], want contents-first [3001, 3002]",
			saved.Items[0].VNum, saved.Items[1].VNum)
	}
	if saved.Items[0].ContainerIndex == nil || *saved.Items[0].ContainerIndex != 1 {
		t.Errorf("sword container_index = %v, want pointer to 1 (the bag)", saved.Items[0].ContainerIndex)
	}
	if saved.Items[1].ContainerIndex != nil {
		t.Errorf("bag container_index = %v, want nil (room-level)", saved.Items[1].ContainerIndex)
	}

	// Remove the live objects, then load the file back — a fresh boot.
	live := countObjectsInWorld(w)
	w.ExtractObject(sword, 4100)
	w.ExtractObject(bag, 4100)
	if got := countObjectsInWorld(w); got != live-2 {
		t.Fatalf("extraction left objects behind: %d of %d", got, live)
	}
	if !w.houseLoad(4100) {
		t.Fatal("houseLoad failed")
	}

	bag2 := findInRoomByVNum(w, 4100, 3002)
	if bag2 == nil {
		t.Fatal("bag not restored to the room")
	}
	sword2 := findInRoomByVNum(w, 4100, 3001)
	if sword2 != nil {
		t.Fatal("sword restored to the room floor; it should be inside the bag")
	}
	if len(bag2.Contains) != 1 || bag2.Contains[0].VNum != 3001 {
		t.Fatalf("bag contains = %v, want the sword", bag2.Contains)
	}
	restored := bag2.Contains[0]
	if restored.Location.Kind != ObjInContainer || restored.Location.ContainerObjID != bag2.ID {
		t.Errorf("sword location = %+v, want container %d", restored.Location, bag2.ID)
	}
	if got := countObjectsInWorld(w); got != live {
		t.Errorf("registry count after load = %d, want %d", got, live)
	}
}

func TestHouseSaveLoadRoundTripsDepthTwo(t *testing.T) {
	t.Chdir(t.TempDir())
	w := newHouseNestingWorld(t)

	chest, _ := w.SpawnObject(3003, 4100)
	bag, _ := w.SpawnObject(3002, 4100)
	sword, _ := w.SpawnObject(3001, 4100)
	if err := w.MoveObjectToRoom(chest, 4100); err != nil {
		t.Fatalf("place chest in room: %v", err)
	}
	if err := w.MoveObjectToContainer(bag, chest); err != nil {
		t.Fatalf("nest bag: %v", err)
	}
	if err := w.MoveObjectToContainer(sword, bag); err != nil {
		t.Fatalf("nest sword: %v", err)
	}

	w.houseCrashsave(4100)
	for _, obj := range []*ObjectInstance{sword, bag, chest} {
		w.ExtractObject(obj, 4100)
	}
	if !w.houseLoad(4100) {
		t.Fatal("houseLoad failed")
	}

	chest2 := findInRoomByVNum(w, 4100, 3003)
	if chest2 == nil || len(chest2.Contains) != 1 || chest2.Contains[0].VNum != 3002 {
		t.Fatalf("chest nesting lost: chest=%v contains=%v", chest2, chest2.Contains)
	}
	bag2 := chest2.Contains[0]
	if len(bag2.Contains) != 1 || bag2.Contains[0].VNum != 3001 {
		t.Fatalf("bag nesting lost: %v", bag2.Contains)
	}
	if findInRoomByVNum(w, 4100, 3001) != nil || findInRoomByVNum(w, 4100, 3002) != nil {
		t.Error("nested items leaked to the room floor")
	}
}

// writeHouseFile stages a raw save file for load-path tests.
func writeHouseFile(t *testing.T, data houseSaveData) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("house", 0o750); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("house", "4100.house"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHouseLoadFloorsUnresolvableContainers(t *testing.T) {
	tests := []struct {
		name  string
		items []houseSaveItem
	}{
		{
			name: "container index out of range",
			items: []houseSaveItem{
				{VNum: 3001, ContainerIndex: intPtr(9)},
			},
		},
		{
			name: "container index is self",
			items: []houseSaveItem{
				{VNum: 3001, ContainerIndex: intPtr(0)},
			},
		},
		{
			name: "container failed to load (missing prototype)",
			items: []houseSaveItem{
				{VNum: 3001, ContainerIndex: intPtr(1)},
				{VNum: 9999},
			},
		},
		{
			name: "target is not a container",
			items: []houseSaveItem{
				{VNum: 3001, ContainerIndex: intPtr(1)},
				{VNum: 3001}, // a sword is not a container
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeHouseFile(t, houseSaveData{RoomVNum: 4100, Items: tt.items})
			w := newHouseNestingWorld(t)
			if !w.houseLoad(4100) {
				t.Fatal("houseLoad failed")
			}
			if findInRoomByVNum(w, 4100, 3001) == nil {
				t.Error("sword dropped entirely; unresolvable nesting must floor the item, not discard it")
			}
		})
	}
}

// Legacy files from before DP-1401 carry container_id (a runtime registry
// ID). The raw JSON below is exactly such a file: the unknown field is
// ignored by the new schema, so both items load onto the floor — C
// House_load's own behavior for contained items — never silently dropped.
func TestHouseLoadLegacyRuntimeIDFilesFloorEverything(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("house", 0o750); err != nil {
		t.Fatal(err)
	}
	legacy := `{"room_vnum":4100,"items":[` +
		`{"vnum":3001,"container_id":12345},` +
		`{"vnum":3002,"container_id":-1}]}`
	if err := os.WriteFile(filepath.Join("house", "4100.house"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	w := newHouseNestingWorld(t)
	if !w.houseLoad(4100) {
		t.Fatal("houseLoad failed")
	}
	if findInRoomByVNum(w, 4100, 3001) == nil {
		t.Error("legacy contained item dropped; it must land on the floor")
	}
	if findInRoomByVNum(w, 4100, 3002) == nil {
		t.Error("legacy container dropped")
	}
}

func intPtr(n int) *int { return &n }
