package game

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetConcurrentClockManualAndReaders(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 100, Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 100}}}},
		Mobs:  []parser.Mob{{VNum: 300, ShortDesc: "a reset mobile", Position: 8, DefaultPos: 8}},
		Objs:  []parser.Obj{{VNum: 200, LoadPercent: 100}, {VNum: 201, LoadPercent: 100}, {VNum: 204, LoadPercent: 100}},
		Zones: []parser.Zone{{Number: 1, Lifespan: 1, ResetMode: 2, Commands: []parser.ZoneCommand{
			{Command: "R", Arg1: 100, Arg2: 0, Arg3: 300},
			{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
			{Command: "G", IfFlag: 1, Arg1: 201, Arg2: 1},
			{Command: "R", Arg1: 100, Arg2: 1, Arg3: 204},
			{Command: "O", Arg1: 204, Arg2: 1, Arg3: 100},
			{Command: "P", Arg1: 200, Arg2: 1, Arg3: 204},
			{Command: "D", Arg1: 100, Arg2: 0, Arg3: 2},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.StartZoneResets(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			if err := w.ResetZone(1); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Go(func() {
		for range 600 {
			w.ZoneUpdate()
		}
	})
	wg.Go(func() {
		for range 600 {
			_ = w.GetObjNum(204)
			_ = w.ZoneClockSnapshot()
			_ = w.HasPendingExtractions()
			room, ok := w.GetRoom(100)
			if !ok || room.Exits["north"].ExitInfo&parser.ExitLocked == 0 {
				t.Error("reader observed incomplete D publication")
			}
		}
	})
	wg.Wait()
	if w.countObjectInstances(204) != 1 || w.countObjectInstances(200) != 1 {
		t.Fatal("concurrent resets violated object caps or ownership cleanup")
	}
	if w.countMobInstances(300) != 1 || !w.HasPendingExtractions() {
		t.Fatal("concurrent reset lost deferred mobile count")
	}
	w.ExtractPendingChars()
	if w.countMobInstances(300) != 0 {
		t.Fatal("deferred mobile did not drain after concurrent resets")
	}
}
