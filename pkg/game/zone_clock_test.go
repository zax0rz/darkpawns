package game

import (
	"reflect"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func zoneClockFixture(t *testing.T) *World {
	t.Helper()
	t.Setenv("DP_CLOCK", "1")
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 100, Zone: 1}, {VNum: 200, Zone: 2}, {VNum: 300, Zone: 3, Exits: map[string]parser.Exit{"north": {ToRoom: 200}}}},
		Zones: []parser.Zone{{Number: 3, Lifespan: 1, ResetMode: 2, Commands: []parser.ZoneCommand{{Command: "D", Arg1: 300, Arg2: 0, Arg3: 2}}}, {Number: 2, Lifespan: 1, ResetMode: 1}, {Number: 1, Lifespan: 1, ResetMode: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.StartZoneResets(); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestZoneClockMinuteModesAndQueue(t *testing.T) {
	w := zoneClockFixture(t)
	w.SetExitInfo(300, "north", 0)
	occupied := []int{100, 200, 300}
	w.OccupiedZoneRooms = func() []int { return occupied }
	for range 5 {
		w.ZoneUpdate()
	}
	if got := w.ZoneClockSnapshot(); got.Ages[2] != 0 || len(got.Queue) != 0 {
		t.Fatalf("aged before minute: %+v", got)
	}
	if room, _ := w.GetRoom(300); room.Exits["north"].ExitInfo != 0 {
		t.Fatal("door reset before minute")
	}
	w.ZoneUpdate()
	if room, _ := w.GetRoom(300); room.Exits["north"].ExitInfo&parser.ExitLocked == 0 {
		t.Fatal("automatic reset did not execute door command")
	}
	got := w.ZoneClockSnapshot()
	if got.Ages[1] != 0 || got.Ages[2] != 999 || got.Ages[3] != 0 || !reflect.DeepEqual(got.Queue, []int{2}) {
		t.Fatalf("mode0/blocked mode1/bypass mode2: %+v", got)
	}
	// Aging does not enqueue an additional copy of the sentinel zone.
	for range 6 {
		w.ZoneUpdate()
	}
	if got := w.ZoneClockSnapshot(); !reflect.DeepEqual(got.Queue, []int{2}) {
		t.Fatalf("sentinel enqueued again: %+v", got)
	}
	occupied = nil
	w.ZoneUpdate()
	if got := w.ZoneClockSnapshot(); len(got.Queue) != 0 || got.Ages[2] != 0 {
		t.Fatalf("unblocked reset: %+v", got)
	}
}

func TestZoneClockManualResetPreservesQueue(t *testing.T) {
	w := zoneClockFixture(t)
	w.OccupiedZoneRooms = func() []int { return []int{200} }
	for range 6 {
		w.ZoneUpdate()
	}
	if err := w.ResetZone(2); err != nil {
		t.Fatal(err)
	}
	if got := w.ZoneClockSnapshot(); got.Ages[2] != 0 || !reflect.DeepEqual(got.Queue, []int{2}) {
		t.Fatalf("manual reset: %+v", got)
	}
	// A manual reset does not deduplicate C's queue. A still occupied zone can
	// reach its lifespan again while its original entry remains queued.
	for range 6 {
		w.ZoneUpdate()
	}
	if got := w.ZoneClockSnapshot(); !reflect.DeepEqual(got.Queue, []int{2, 2}) {
		t.Fatalf("C duplicate queue entries: %+v", got)
	}
	w.OccupiedZoneRooms = nil
	w.ZoneUpdate()
	if got := w.ZoneClockSnapshot(); !reflect.DeepEqual(got.Queue, []int{2}) || got.Ages[2] != 0 {
		t.Fatalf("one reset per heartbeat: %+v", got)
	}
}

func TestZoneClockConcurrentManualAndHeartbeat(t *testing.T) {
	w := zoneClockFixture(t)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 60 {
				w.ZoneUpdate()
				if err := w.ResetZone(2); err != nil {
					t.Error(err)
				}
				_ = w.ZoneClockSnapshot()
			}
		})
	}
	wg.Wait()
}

func TestZoneClockPumpedHeartbeatExecutesReset(t *testing.T) {
	w := zoneClockFixture(t)
	w.SetExitInfo(300, "north", 0)
	gl := engine.NewGameLoop(engine.GameLoopCallbacks{OnZoneUpdate: w.ZoneUpdate})
	if err := gl.PumpPulses(599); err != nil {
		t.Fatal(err)
	}
	if got := w.ZoneClockSnapshot(); got.MinuteTicks != 5 || got.Ages[3] != 0 {
		t.Fatalf("before pulse 600: %+v", got)
	}
	if room, _ := w.GetRoom(300); room.Exits["north"].ExitInfo != 0 {
		t.Fatal("door reset before pulse 600")
	}
	if err := gl.PumpPulses(1); err != nil {
		t.Fatal(err)
	}
	if got := w.ZoneClockSnapshot(); got.Ages[3] != 999 || !reflect.DeepEqual(got.Queue, []int{3}) {
		t.Fatalf("second due zone reset on same pulse: %+v", got)
	}
	if err := gl.PumpPulses(100); err != nil {
		t.Fatal(err)
	}
	if room, _ := w.GetRoom(300); room.Exits["north"].ExitInfo&parser.ExitLocked == 0 {
		t.Fatal("queued door reset did not run at pulse 700")
	}
	if got := w.ZoneClockSnapshot(); len(got.Queue) != 0 || got.Ages[3] != 0 {
		t.Fatalf("after pulse 700: %+v", got)
	}
}

func TestZoneClockTrackingConcurrentRestore(t *testing.T) {
	w := zoneClockFixture(t)
	s := w.GetSpawner()
	proto := &parser.Obj{VNum: 10}
	first := &ObjectInstance{VNum: 10, Prototype: proto}
	s.RegisterObjectInstance(first)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			obj := &ObjectInstance{VNum: 10, Prototype: proto}
			s.RegisterObjectInstance(obj)
			s.forgetObjectInstance(-1, obj)
		}
	})
	wg.Go(func() {
		for range 100 {
			if got := s.findObjectInstance(10); got != first {
				t.Errorf("tracking changed first retained object: %p", got)
			}
		}
	})
	wg.Wait()
}
