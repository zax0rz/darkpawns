package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newRenumWorld boots a world whose one zone holds the given commands, so
// NewWorld's renum_zone_table pass runs on them as it does at server boot.
func newRenumWorld(t *testing.T, commands []parser.ZoneCommand) (*World, *parser.Zone) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Name: "Reset test room"}},
		Objs: []parser.Obj{
			{VNum: 200, Keywords: "target", LoadPercent: 100},
			{VNum: 201, Keywords: "dependent", LoadPercent: 100},
		},
		Mobs:  []parser.Mob{{VNum: 300, Keywords: "reset mob", ShortDesc: "a reset mob", Position: 8, DefaultPos: 8}},
		Zones: []parser.Zone{{Number: 1, Commands: commands}},
	}
	world, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	zone, ok := world.zones[1]
	if !ok {
		t.Fatal("zone 1 not indexed")
	}
	return world, zone
}

// renum_zone_table (db.c:1001-1015) turns the legacy "R <if> <room> <vnum> -1"
// into C's form, an object removal with the vnum in arg3. Every R line in the
// shipped world is legacy; the port used to read them as mobile removals.
func TestRenumConvertsLegacyRemoveToObject(t *testing.T) {
	_, zone := newRenumWorld(t, []parser.ZoneCommand{{Command: "R", Arg1: 100, Arg2: 200, Arg3: -1}})
	got := zone.Commands[0]
	if got.Command != "R" || got.Arg2 != 1 || got.Arg3 != 200 {
		t.Fatalf("legacy R = %+v, want R kind 1 (object) vnum 200", got)
	}
}

// A command naming a missing mobile, object or room is disabled once at boot
// ("Invalid vnum, cmd disabled"), and C's if-flag chain then skips the
// commands that depend on it.
func TestRenumDisablesInvalidCommandsAndTheirDependents(t *testing.T) {
	world, zone := newRenumWorld(t, []parser.ZoneCommand{
		{Command: "M", Arg1: 399, Arg2: 1, Arg3: 100},            // no such mobile
		{Command: "G", IfFlag: 1, Arg1: 201, Arg2: 1},            // depends on it
		{Command: "O", Arg1: 200, Arg2: 1, Arg3: 999},            // no such room
		{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100},            // valid
		{Command: "P", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 250}, // no such container
		{Command: "R", Arg1: 100, Arg2: 0, Arg3: 399},            // no such mobile
	})
	want := []string{"*", "G", "*", "O", "*", "*"}
	for i, cmd := range zone.Commands {
		if cmd.Command != want[i] {
			t.Fatalf("command %d = %q, want %q", i, cmd.Command, want[i])
		}
	}
	spawner := NewSpawner(world)
	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	if got := len(spawner.objInstances[201]); got != 0 {
		t.Fatalf("dependent G loaded %d objects after a disabled M; C skips it", got)
	}
	if got := len(spawner.objInstances[200]); got != 1 {
		t.Fatalf("valid O loaded %d objects, want 1", got)
	}
}

// reset_zone's R searches the whole room (get_obj_in_list_num over
// world[room].contents), so it removes an object a player dropped, not only
// one the spawner placed (db.c:2220-2226).
func TestZoneResetRemoveFindsAnyObjectInRoom(t *testing.T) {
	world, zone := newRenumWorld(t, []parser.ZoneCommand{{Command: "R", Arg1: 100, Arg2: 200, Arg3: -1}})
	dropped, err := world.SpawnObject(200, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := world.MoveObjectToRoom(dropped, 100); err != nil {
		t.Fatal(err)
	}
	spawner := NewSpawner(world)
	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatal(err)
	}
	for _, obj := range world.GetItemsInRoom(100) {
		if obj.VNum == 200 {
			t.Fatal("R left the dropped object in the room")
		}
	}
}
