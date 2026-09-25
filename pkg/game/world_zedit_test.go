package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCommitEditedZoneRebucketsRoomCommands(t *testing.T) {
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001},
			{VNum: 1002},
			{VNum: 1003},
		},
		// Every command names a real prototype: renum_zone_table disables
		// commands naming missing ones at boot.
		Mobs: []parser.Mob{{VNum: 2001}, {VNum: 2002}},
		Objs: []parser.Obj{{VNum: 3001}, {VNum: 3002}, {VNum: 3003}},
		Zones: []parser.Zone{{
			Number:  1,
			TopRoom: 1099,
			Commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 2001, Arg3: 1001},
				{Command: "G", Arg1: 3001},
				{Command: "D", Arg1: 1001},
				{Command: "M", Arg1: 2001, Arg3: 1002},
				{Command: "G", Arg1: 3002},
				{Command: "D", Arg1: 1003},
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	working := parser.Zone{
		Number:  1,
		TopRoom: 1099,
		Commands: []parser.ZoneCommand{
			{Command: "M", Arg1: 2002, Arg3: 1001},
			{Command: "G", Arg1: 3003},
		},
	}
	if !world.CommitEditedZone(1, 1001, working) {
		t.Fatal("CommitEditedZone returned false")
	}
	zone, ok := world.SnapshotZone(1)
	if !ok {
		t.Fatal("zone disappeared after commit")
	}
	if len(zone.Commands) != 5 {
		t.Fatalf("commands = %d, want 5: %+v", len(zone.Commands), zone.Commands)
	}
	wantCommands := []string{"M", "G", "M", "G", "D"}
	wantRooms := []int{1001, 1001, 1002, 1002, 1003}
	for index, command := range zone.Commands {
		if command.Command != wantCommands[index] {
			t.Fatalf("command %d = %q, want %q; commands: %+v", index, command.Command, wantCommands[index], zone.Commands)
		}
		if command.Command == "G" {
			continue
		}
		room, hasRoom := ZoneCommandRoom(command)
		if !hasRoom {
			t.Fatalf("command %d has no room: %+v", index, command)
		}
		if room != wantRooms[index] {
			t.Fatalf("command %d room = %d, want %d; commands: %+v", index, room, wantRooms[index], zone.Commands)
		}
	}
	if zone.Commands[0].Arg1 != 2002 || zone.Commands[1].Arg1 != 3003 {
		t.Fatalf("working bucket was not prepended: %+v", zone.Commands[:2])
	}
}
