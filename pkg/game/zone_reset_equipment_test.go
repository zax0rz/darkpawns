package game

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetEquipmentOwnershipAndOccupiedSlot(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	calls := installZoneObjectOrderHooks(t, true)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
		{Command: "E", Arg1: 200, Arg2: 1, Arg3: 3},
		{Command: "E", Arg1: 201, Arg2: 1, Arg3: 3},
		{Command: "G", IfFlag: 1, Arg1: 204, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	mob := w.GetMobsInRoom(100)[0]
	equipped := mob.Equipment[3]
	if equipped == nil || equipped.GetVNum() != 200 {
		t.Fatal("occupied E replaced the original equipment")
	}
	if equipped.Location != LocEquippedMob(mob.ID, EquipmentSlot(3)) {
		t.Fatal("E did not establish canonical equipped ownership")
	}
	refused := w.GetObjNum(201)
	if refused == nil || refused.Location != LocNowhere() || len(mob.Inventory) != 1 || mob.Inventory[0].GetVNum() != 204 {
		t.Fatal("occupied E did not leave new object floating and enable conditional G")
	}
	assertZoneObjectCalls(t, calls, "percent", "percent", "percent")
	w.ExtractObject(equipped, -1)
	if mob.Equipment[3] != nil || w.GetObjNum(200) != nil {
		t.Fatal("canonical extraction did not detach E equipment")
	}
}

func TestZoneResetEquipmentAttributeBoundary(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 100}}, Mobs: []parser.Mob{{VNum: 300, Position: 8, DefaultPos: 8}}, Objs: []parser.Obj{{VNum: 200, LoadPercent: 100, Affects: []parser.ObjAffect{{Location: 2, Modifier: 30}}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := NewSpawner(w)
	s.world.zoneResetMu.Lock()
	err = s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100}, {Command: "E", Arg1: 200, Arg2: 1, Arg3: 3}}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	mob := w.GetMobsInRoom(100)[0]
	if mob.GetDex() != 25 {
		t.Fatalf("E effective dex=%d, want NPC cap 25", mob.GetDex())
	}
	obj := mob.Equipment[3]
	w.ExtractObject(obj, -1)
	if mob.GetDex() != mob.Dex {
		t.Fatal("E removal did not restore base attributes")
	}
}

func TestZoneResetEquipmentLoadGates(t *testing.T) {
	for _, tc := range []struct {
		name          string
		slot, cap     int
		load, withMob bool
		wantPercent   int
	}{
		{"no-mob", 3, 1, true, false, 0}, {"negative-slot", -1, 1, true, true, 0}, {"past-last-slot", 22, 1, true, true, 0}, {"global-cap", 3, 1, true, true, 0}, {"percent-failure", 3, 2, false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			if tc.name == "global-cap" {
				if _, err := w.SpawnObject(200, -1); err != nil {
					t.Fatal(err)
				}
			}
			calls := installZoneObjectOrderHooks(t, tc.load)
			var commands []parser.ZoneCommand
			if tc.withMob {
				commands = append(commands, parser.ZoneCommand{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100})
			}
			commands = append(commands, parser.ZoneCommand{Command: "E", Arg1: 200, Arg2: tc.cap, Arg3: tc.slot}, parser.ZoneCommand{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100})
			s.world.zoneResetMu.Lock()
			err := s.executeZoneResetLocked(&parser.Zone{Commands: commands})
			s.world.zoneResetMu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if len(*calls) != tc.wantPercent {
				t.Fatalf("E gate percent calls=%d, want %d", len(*calls), tc.wantPercent)
			}
			if w.GetObjNum(201) != nil {
				t.Fatal("failed E enabled conditional O")
			}
			want := 0
			if tc.name == "global-cap" {
				want = 1
			}
			if got := w.countObjectInstances(200); got != want {
				t.Fatalf("E instance count=%d want %s", got, strconv.Itoa(want))
			}
		})
	}
}
