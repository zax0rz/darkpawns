package game

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetDoorClearsSecretMark(t *testing.T) {
	for _, state := range []int{0, 1, 2, 99} {
		t.Run(strconv.Itoa(state), func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			initial := parser.ExitClosed | parser.ExitLocked | parser.ExitIsDoor
			w.updateRoom(100, func(r *parser.Room) {
				r.Flags = []string{strconv.Itoa(RoomSecretMark | RoomDark)}
				r.Exits = map[string]parser.Exit{"north": {ExitInfo: initial}}
			})
			before, _ := w.GetRoom(100)
			if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "D", Arg1: 100, Arg2: 0, Arg3: state},
				{Command: "O", IfFlag: 1, Arg1: 200, Arg2: 1, Arg3: 100},
			}}); err != nil {
				t.Fatal(err)
			}
			room, _ := w.GetRoom(100)
			if roomHasFlagBit(room.Flags, 20) {
				t.Fatal("valid D reset retained ROOM_SECRET_MARK")
			}
			if !roomHasFlagBit(room.Flags, 0) {
				t.Fatal("D reset removed unrelated room flag")
			}
			if got, want := room.Exits["north"].ExitInfo, parser.ApplyDoorReset(initial, state); got != want {
				t.Fatalf("door flags = %d, want %d", got, want)
			}
			if len(w.GetItemsInRoom(100)) != 1 {
				t.Fatal("valid D reset did not enable conditional O")
			}
			if !roomHasFlagBit(before.Flags, 20) || before.Exits["north"].ExitInfo != initial {
				t.Fatal("D reset mutated a previously published room snapshot")
			}
		})
	}
}

func TestZoneResetMissingDoorPreservesSecretMark(t *testing.T) {
	for _, direction := range []int{-1, 0, 6} {
		t.Run(strconv.Itoa(direction), func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			w.SetRoomFlagBit(100, 20)
			if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "D", Arg1: 100, Arg2: direction, Arg3: 2},
				{Command: "O", IfFlag: 1, Arg1: 200, Arg2: 1, Arg3: 100},
			}}); err != nil {
				t.Fatal(err)
			}
			room, _ := w.GetRoom(100)
			if !roomHasFlagBit(room.Flags, 20) {
				t.Fatal("invalid D reset cleared secret mark")
			}
			if len(w.GetItemsInRoom(100)) != 0 {
				t.Fatal("invalid D reset enabled conditional O")
			}
		})
	}
}

func TestZoneResetPUsesNewestGlobalObject(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		t.Run(strconv.FormatBool(tracked), func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			old, err := s.SpawnObject(204, 100)
			if err != nil {
				t.Fatal(err)
			}
			var newest *ObjectInstance
			if tracked {
				newest, err = s.SpawnObject(204, -1)
			} else {
				newest, err = w.SpawnObject(204, -1)
			}
			if err != nil {
				t.Fatal(err)
			}
			newest.Prototype.TypeFlag = 15
			if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "P", Arg1: 200, Arg2: 1, Arg3: 204},
				{Command: "P", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 204},
			}}); err != nil {
				t.Fatal(err)
			}
			if len(old.Contains) != 0 || len(newest.Contains) != 2 {
				t.Fatalf("P selected wrong global instance: old contents=%d newest contents=%d", len(old.Contains), len(newest.Contains))
			}
			for i, vnum := range []int{201, 200} {
				child := newest.Contains[i]
				if child.GetVNum() != vnum || child.Location != LocContainer(newest.ID) {
					t.Fatalf("P child %d = %d at %+v, want %d in newest container", i, child.GetVNum(), child.Location, vnum)
				}
			}
			w.ExtractObject(newest, -1)
			if got := w.GetObjNum(204); got != old {
				t.Fatalf("global lookup after extraction = %p, want retained %p", got, old)
			}
		})
	}
}

func TestZoneResetPAllowsNonContainerTarget(t *testing.T) {
	_, s := newZoneResetTestSpawner(t)
	target, err := s.SpawnObject(204, 100)
	if err != nil {
		t.Fatal(err)
	}
	if target.IsContainer() {
		t.Fatal("fixture must not have container type")
	}
	if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "P", Arg1: 200, Arg2: 1, Arg3: 204},
	}}); err != nil {
		t.Fatal(err)
	}
	if len(target.Contains) != 1 || target.Contains[0].Location != LocContainer(target.ID) {
		t.Fatal("P rejected non-container target accepted by C obj_to_obj")
	}
}
