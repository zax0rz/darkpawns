package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCreateRoomExitReplacesWithBareExit(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{
			VNum: 1001,
			Exits: map[string]parser.Exit{
				"north": {
					Direction:   "north",
					ToRoom:      9999,
					DoorState:   2,
					ExitInfo:    parser.ExitIsDoor | parser.ExitClosed,
					Key:         4242,
					Keywords:    "gate",
					Description: "an old gate",
				},
			},
		},
		{VNum: 1002},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	if !w.CreateRoomExit(1001, "north", 1002) {
		t.Fatal("CreateRoomExit returned false for an existing room")
	}
	exit := w.GetRoomInWorld(1001).Exits["north"]
	if exit.Direction != "north" || exit.ToRoom != 1002 {
		t.Fatalf("created exit = %+v, want north -> 1002", exit)
	}
	if exit.DoorState != 0 || exit.ExitInfo != 0 || exit.Key != 0 || exit.Keywords != "" || exit.Description != "" {
		t.Fatalf("created exit retained C-incompatible metadata: %+v", exit)
	}
	if w.CreateRoomExit(9999, "south", 1001) {
		t.Fatal("CreateRoomExit returned true for a missing room")
	}
}
