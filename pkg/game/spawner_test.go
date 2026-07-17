package game

import (
	"slices"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newZoneResetTestSpawner(t *testing.T) (*World, *Spawner) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Name: "Reset test room"}},
		Objs: []parser.Obj{
			{VNum: 200, Keywords: "target", LoadPercent: 100},
			{VNum: 201, Keywords: "dependent", LoadPercent: 100},
			{
				VNum:        202,
				Keywords:    "rare",
				LoadPercent: 50,
				ExtraFlags:  [4]int{FlagItemRare},
				Affects:     []parser.ObjAffect{{Location: 19, Modifier: 1}},
			},
			{
				VNum:        203,
				Keywords:    "rare child",
				LoadPercent: 50,
				ExtraFlags:  [4]int{FlagItemRare},
				Affects:     []parser.ObjAffect{{Location: 19, Modifier: 1}},
			},
			{VNum: 204, Keywords: "container", LoadPercent: 100},
		},
		Mobs: []parser.Mob{{VNum: 300, Keywords: "reset mob", ShortDesc: "a reset mob", Position: 8, DefaultPos: 8}},
	}
	world, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	return world, NewSpawner(world)
}

func TestZoneResetFailedRemoveDoesNotEnableDependentCommand(t *testing.T) {
	for _, removeType := range []int{0, 1} {
		t.Run(map[int]string{0: "mob", 1: "object"}[removeType], func(t *testing.T) {
			_, spawner := newZoneResetTestSpawner(t)
			targetVNum := 300
			if removeType == 1 {
				targetVNum = 200
			}
			zone := &parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "R", Arg1: 100, Arg2: targetVNum, Arg3: removeType},
				{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
			}}

			if err := spawner.ExecuteZoneReset(zone); err != nil {
				t.Fatal(err)
			}
			if got := len(spawner.objInstances[201]); got != 0 {
				t.Fatalf("dependent object instances = %d, want 0 after failed R", got)
			}
		})
	}
}

func TestZoneResetSuccessfulRemoveEnablesDependentCommand(t *testing.T) {
	_, spawner := newZoneResetTestSpawner(t)
	if _, err := spawner.SpawnObject(200, 100); err != nil {
		t.Fatal(err)
	}
	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "R", Arg1: 100, Arg2: 200, Arg3: 1},
		{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
	}}

	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	if got := len(spawner.objInstances[200]); got != 0 {
		t.Fatalf("removed object instances = %d, want 0", got)
	}
	if got := len(spawner.objInstances[201]); got != 1 {
		t.Fatalf("dependent object instances = %d, want 1 after successful R", got)
	}
}

func TestZoneResetObjectLoadIndexesObjectInRoom(t *testing.T) {
	world, spawner := newZoneResetTestSpawner(t)
	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100},
	}}

	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	items := world.GetItemsInRoom(100)
	if len(items) != 1 || items[0].GetVNum() != 200 {
		t.Fatalf("room 100 zone objects = %v, want one object vnum 200", items)
	}
}

func TestZoneResetObjectMaxCountsCharacterCreationObjects(t *testing.T) {
	world, spawner := newZoneResetTestSpawner(t)
	proto, ok := world.GetObjPrototype(200)
	if !ok {
		t.Fatal("missing object prototype 200")
	}
	if obj := world.newObjectInstance(proto, -1); obj == nil {
		t.Fatal("failed to create non-spawner runtime object")
	}

	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100},
	}}
	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	if got := len(world.GetItemsInRoom(100)); got != 0 {
		t.Fatalf("zone reset spawned %d room objects despite global max already being reached", got)
	}
}

func installZoneObjectOrderHooks(t *testing.T, percentResult bool) *[]string {
	t.Helper()
	calls := &[]string{}
	previousInit := zoneObjectInitRare
	previousPercent := zoneObjectPercentLoad
	zoneObjectInitRare = func(*ObjectInstance) {
		*calls = append(*calls, "create")
	}
	zoneObjectPercentLoad = func(*parser.Obj) bool {
		*calls = append(*calls, "percent")
		return percentResult
	}
	t.Cleanup(func() {
		zoneObjectInitRare = previousInit
		zoneObjectPercentLoad = previousPercent
	})
	return calls
}

func assertZoneObjectCalls(t *testing.T, got *[]string, want ...string) {
	t.Helper()
	if !slices.Equal(*got, want) {
		t.Fatalf("object load calls = %v, want %v", *got, want)
	}
}

func TestZoneResetObjectLoadCreatesBeforePercentAndExtractsOnFailure(t *testing.T) {
	tests := []struct {
		name     string
		commands []parser.ZoneCommand
	}{
		{
			name: "O",
			commands: []parser.ZoneCommand{
				{Command: "O", Arg1: 202, Arg2: 1, Arg3: 100},
			},
		},
		{
			name: "G",
			commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
				{Command: "G", IfFlag: 1, Arg1: 202, Arg2: 1},
			},
		},
		{
			name: "E",
			commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
				{Command: "E", IfFlag: 1, Arg1: 202, Arg2: 1, Arg3: 0},
			},
		},
		{
			name: "P",
			commands: []parser.ZoneCommand{
				{Command: "O", Arg1: 204, Arg2: 1, Arg3: -1},
				{Command: "P", IfFlag: 1, Arg1: 202, Arg2: 1, Arg3: 204},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			world, spawner := newZoneResetTestSpawner(t)
			calls := installZoneObjectOrderHooks(t, false)
			if err := spawner.ExecuteZoneReset(&parser.Zone{Commands: test.commands}); err != nil {
				t.Fatal(err)
			}
			assertZoneObjectCalls(t, calls, "create", "percent")
			if got := len(spawner.objInstances[202]); got != 0 {
				t.Fatalf("failed-load object instances = %d, want 0", got)
			}
			if !spawner.canSpawnObject(202, 1) {
				t.Fatal("failed percent_load still counts against max-in-world")
			}
			for _, obj := range world.objectInstances {
				if obj.VNum == 202 {
					t.Fatal("failed percent_load object remains in world registry")
				}
			}
		})
	}
}

func TestZoneResetFloatingOSkipsPercentLoad(t *testing.T) {
	_, spawner := newZoneResetTestSpawner(t)
	calls := installZoneObjectOrderHooks(t, false)
	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "O", Arg1: 202, Arg2: 1, Arg3: -1},
		{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
	}}

	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	assertZoneObjectCalls(t, calls, "create", "percent")
	if got := len(spawner.objInstances[202]); got != 1 {
		t.Fatalf("floating object instances = %d, want 1", got)
	}
	if obj := spawner.objInstances[202][0]; obj.Location.Kind != ObjNowhere {
		t.Fatalf("floating object location = %v, want nowhere", obj.Location)
	}
}

func TestZoneResetMissingPContainerCreatesThenSkipsPercent(t *testing.T) {
	_, spawner := newZoneResetTestSpawner(t)
	calls := installZoneObjectOrderHooks(t, false)
	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "P", Arg1: 202, Arg2: 1, Arg3: 999},
		{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
	}}

	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	assertZoneObjectCalls(t, calls, "create")
	if got := len(spawner.objInstances[202]); got != 1 {
		t.Fatalf("missing-container object instances = %d, want C-faithful floating count 1", got)
	}
	if got := len(spawner.objInstances[201]); got != 0 {
		t.Fatalf("dependent object instances = %d, want 0 after failed P", got)
	}
}

func TestZoneResetInvalidEquipPositionSkipsCreationAndPercent(t *testing.T) {
	_, spawner := newZoneResetTestSpawner(t)
	calls := installZoneObjectOrderHooks(t, true)
	zone := &parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "E", IfFlag: 1, Arg1: 202, Arg2: 1, Arg3: numWears},
	}}

	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	assertZoneObjectCalls(t, calls)
	if got := len(spawner.objInstances[202]); got != 0 {
		t.Fatalf("invalid-position object instances = %d, want 0", got)
	}
}

func TestInitRareMatchesCDrawRangesAndUnsupportedApplyBurn(t *testing.T) {
	previous := zoneRareNumber
	var ranges [][2]int
	values := []int{20, 0, 20, 1}
	zoneRareNumber = func(from, to int) int {
		ranges = append(ranges, [2]int{from, to})
		value := values[0]
		values = values[1:]
		return value
	}
	t.Cleanup(func() { zoneRareNumber = previous })

	proto := &parser.Obj{
		VNum: 202,
		Affects: []parser.ObjAffect{
			{Location: 5, Modifier: 7},  // unsupported: modifier stays put, sign draw still burns
			{Location: 19, Modifier: 1}, // damroll: positive when number(0,1) is 1
		},
	}
	obj := NewObjectInstance(proto, -1)
	initRare(obj)

	wantRanges := [][2]int{{1, 100}, {0, 1}, {1, 100}, {0, 1}}
	if !slices.Equal(ranges, wantRanges) {
		t.Fatalf("initRare ranges = %v, want %v", ranges, wantRanges)
	}
	got := obj.GetAffects()
	if got[0].Modifier != 7 {
		t.Fatalf("unsupported modifier = %d, want 7", got[0].Modifier)
	}
	if got[1].Modifier != 2 {
		t.Fatalf("damroll modifier = %d, want 2", got[1].Modifier)
	}
}
