package game

import (
	"slices"
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetMobileCapRetainsLastMob(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "G", IfFlag: 1, Arg1: 201, Arg2: 1},
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "G", IfFlag: 1, Arg1: 204, Arg2: 1},
		{Command: "G", Arg1: 200, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	mobs := w.GetMobsInRoom(100)
	if len(mobs) != 1 {
		t.Fatalf("mob count=%d, want 1", len(mobs))
	}
	if got := mobInventoryVNums(mobs[0]); !slices.Equal(got, []int{200, 201}) {
		t.Fatalf("retained mob inventory=%v, want [200 201]", got)
	}
}

func mobInventoryVNums(mob *MobInstance) []int {
	var out []int
	for _, obj := range mob.Inventory {
		out = append(out, obj.GetVNum())
	}
	return out
}

func TestZoneResetMobileCapCountsOutsideSpawner(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	if _, err := w.SpawnMob(300, 100); err != nil {
		t.Fatal(err)
	}
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "G", IfFlag: 1, Arg1: 200, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if w.countMobInstances(300) != 1 || w.countObjectInstances(200) != 0 {
		t.Fatal("M ignored global cap or enabled failed conditional")
	}
}

func TestZoneResetMobileBothRandomPlacements(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 100, Zone: 79, Flags: []string{strconv.Itoa(RoomPrivate)}},
			{VNum: 200, Zone: 79, Sector: sectCity},
			{VNum: 300, Zone: 163},
			{VNum: 400, Zone: 79},
		},
		Mobs: []parser.Mob{{VNum: 7901, ShortDesc: "a wanderer", Position: 8, DefaultPos: 8, ActionFlags: []string{"RANDZON"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	draws := installZoneRoomNumbers(t, 0, 1, 2, 3, 2, 1)
	s := NewSpawner(w)
	s.world.zoneResetMu.Lock()
	err = s.executeZoneResetLocked(&parser.Zone{Number: 79, Commands: []parser.ZoneCommand{{Command: "M", Arg1: 7901, Arg2: 1, Arg3: 100}}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if *draws != 6 || len(w.GetMobsInRoom(200)) != 1 {
		t.Fatalf("both placements: draws=%d city occupants=%d, want 6 and 1", *draws, len(w.GetMobsInRoom(200)))
	}
}

func TestZoneResetMobilePlacementFlagMatrix(t *testing.T) {
	for _, flag := range []int{RoomPrivate, RoomGodRoom, RoomDeath, RoomNoMob, RoomHouse, RoomAtrium} {
		room := &parser.Room{Zone: 79, Flags: []string{strconv.Itoa(flag)}}
		if isRoomValidForSpawn(room) || isRoomValidForRandZon(room, 79) {
			t.Fatalf("random placement accepted restricted flag %d", flag)
		}
	}
	if isRoomValidForRandZon(&parser.Room{Zone: 78}, 79) {
		t.Fatal("RANDZON accepted another zone")
	}
	if isRoomValidForSpawn(&parser.Room{Zone: 163}) {
		t.Fatal("zone79 accepted excluded zone 163")
	}
	if !isRoomValidForRandZon(&parser.Room{Zone: 163}, 163) {
		t.Fatal("RANDZON incorrectly excludes its own zone 163")
	}
}

func TestZoneResetRandZonUsesResetZone(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 100, Zone: 1}, {VNum: 200, Zone: 2}},
		Mobs:  []parser.Mob{{VNum: 300, ShortDesc: "a wanderer", Position: 8, DefaultPos: 8, ActionFlags: []string{"RANDZON"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	draws := installZoneRoomNumbers(t, 1, 0)
	s := NewSpawner(w)
	s.world.zoneResetMu.Lock()
	err = s.executeZoneResetLocked(&parser.Zone{Number: 2, Commands: []parser.ZoneCommand{{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100}}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if *draws != 1 || len(w.GetMobsInRoom(200)) != 1 {
		t.Fatalf("RANDZON used spawn room's zone: draws=%d zone2 mobs=%d", *draws, len(w.GetMobsInRoom(200)))
	}
}

func TestZoneResetObjectRoomOrderAndOwnership(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "O", Arg1: 200, Arg2: 2, Arg3: 100},
		{Command: "O", Arg1: 201, Arg2: 1, Arg3: 100},
		{Command: "O", Arg1: 200, Arg2: 2, Arg3: 100},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	items := w.GetItemsInRoom(100)
	if len(items) != 3 {
		t.Fatalf("room contents=%d, want 3", len(items))
	}
	if items[0].GetVNum() != 200 || items[1].GetVNum() != 201 || items[2].GetVNum() != 200 || items[0].ID < items[2].ID {
		t.Fatal("O did not prepend in C room-list order")
	}
	for _, item := range items {
		if item.Location != LocRoom(100) {
			t.Fatal("O did not establish canonical room ownership")
		}
	}
}

func TestZoneResetGiveWithoutLastMob(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	mob, err := w.SpawnMob(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	s.world.zoneResetMu.Lock()
	err = s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{{Command: "G", Arg1: 200, Arg2: 1}}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if w.countObjectInstances(200) != 0 || len(mob.Inventory) != 0 {
		t.Fatal("G invented a last mob from the world registry")
	}
}

func TestZoneResetGiveGlobalCapAndConditionalFailure(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	if _, err := w.SpawnObject(200, -1); err != nil {
		t.Fatal(err)
	}
	calls := installZoneObjectOrderHooks(t, true)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "G", Arg1: 200, Arg2: 1},
		{Command: "G", Arg1: 201, Arg2: 1},
		{Command: "G", IfFlag: 1, Arg1: 201, Arg2: 1},
		{Command: "G", IfFlag: 1, Arg1: 204, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	mobs := w.GetMobsInRoom(100)
	if len(mobs) != 1 {
		t.Fatalf("mob count=%d, want 1", len(mobs))
	}
	if got := mobInventoryVNums(mobs[0]); !slices.Equal(got, []int{204, 201}) {
		t.Fatalf("G inventory=%v, want [204 201] after global cap and failed conditional", got)
	}
	for _, obj := range mobs[0].Inventory {
		if obj.Location != LocInventoryMob(mobs[0].ID) {
			t.Fatal("G did not establish canonical mobile ownership")
		}
	}
	assertZoneObjectCalls(t, calls, "percent", "percent")
}
