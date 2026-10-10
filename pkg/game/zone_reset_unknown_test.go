package game

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetUnknownCommandDisabledOnce(t *testing.T) {
	var logs bytes.Buffer
	oldWriter := getLogWriter()
	SetLogWriter(&logs)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	zone := parser.Zone{Number: 1, TopRoom: 199, Commands: []parser.ZoneCommand{
		{Command: "L", Arg1: 100, Arg3: 2},
		{Command: "Q", IfFlag: 1, Arg1: 3, Arg2: 4},
		{Command: "O", IfFlag: 1, Arg1: 200, Arg2: 10, Arg3: 100},
		{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1},
	}}
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 100, Zone: 1}}, Objs: []parser.Obj{{VNum: 200, LoadPercent: 100}}, Zones: []parser.Zone{zone}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := NewSpawner(w)
	w.spawner = s
	before, _ := w.GetZone(1)
	for i := 0; i < 2; i++ {
		if err := w.ResetZone(1); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := w.GetZone(1)
	if after.Commands[1].Command != "*" {
		t.Fatal("unknown reset command was not permanently disabled")
	}
	if before.Commands[1].Command != "Q" {
		t.Fatal("unknown disable mutated a published zone snapshot")
	}
	if w.countObjectInstances(200) != 4 {
		t.Fatal("conditional unknown command cleared prior last_cmd")
	}
	if got := strings.Count(logs.String(), "SYSERR: error in zone file: unknown cmd in reset table; cmd disabled"); got != 1 {
		t.Fatalf("unknown command logs=%d want 1", got)
	}
}

func TestZoneResetUnknownCommandConditionalSkip(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	zone := parser.Zone{Number: 1, Commands: []parser.ZoneCommand{{Command: "Q", IfFlag: 1}, {Command: "O", IfFlag: 1, Arg1: 200, Arg2: 1, Arg3: 100}}}
	w.parsedData = nil
	w.zones[1] = &zone
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&zone)
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	after, _ := w.GetZone(1)
	if after.Commands[0].Command != "Q" || w.GetObjNum(200) != nil {
		t.Fatal("skipped unknown reset command was disabled or enabled successor")
	}
}

func TestZoneResetUnknownDisablePreservesEditedTable(t *testing.T) {
	w, _ := newZoneResetTestSpawner(t)
	original := parser.Zone{Number: 1, Commands: []parser.ZoneCommand{{Command: "Q"}}}
	w.parsedData = nil
	w.zones[1] = &original
	updated := parser.Zone{Number: 1, Commands: []parser.ZoneCommand{{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100}}}
	if !w.CommitEditedZone(1, 0, updated) {
		t.Fatal("edit failed")
	}
	w.disableResetCommand(1, 0, original.Commands[0])
	current, _ := w.GetZone(1)
	if current.Commands[0].Command != "O" {
		t.Fatal("stale reset disabled a concurrent editor replacement")
	}
}

func TestZoneResetUnknownConcurrentEditorAndReaders(t *testing.T) {
	definition := parser.Zone{Number: 1, TopRoom: 199, Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "E", IfFlag: 1, Arg1: 200, Arg2: 1, Arg3: 3},
		{Command: "Q", IfFlag: 1},
	}}
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 100, Zone: 1}}, Mobs: []parser.Mob{{VNum: 300, Position: 8, DefaultPos: 8}}, Objs: []parser.Obj{{VNum: 200, LoadPercent: 100}}, Zones: []parser.Zone{definition}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.StartZoneResets(); err != nil {
		t.Fatal(err)
	}
	mob := w.GetMobsInRoom(100)[0]
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			if err := w.ResetZone(1); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Go(func() {
		for range 100 {
			if !w.CommitEditedZone(1, 0, definition) {
				t.Error("editor failed")
			}
		}
	})
	wg.Go(func() {
		for range 200 {
			before, ok := w.GetZone(1)
			if !ok {
				t.Error("zone missing")
				return
			}
			command := before.Commands[2]
			_ = mob.GetDex()
			_, _ = w.SnapshotZone(1)
			if before.Commands[2] != command {
				t.Error("published zone snapshot changed")
			}
		}
	})
	wg.Wait()
	if w.countObjectInstances(200) != 1 || mob.Equipment[3].Location != LocEquippedMob(mob.ID, EquipmentSlot(3)) {
		t.Fatal("concurrent resets lost equipped ownership or cap")
	}
}
