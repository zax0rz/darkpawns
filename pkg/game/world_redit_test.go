package game

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func routineMutationFixtureWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{
				VNum:  1001,
				Name:  "Room A",
				Zone:  1,
				Flags: []string{"0", "0", "0", "0"},
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, ExitInfo: parser.ExitIsDoor},
				},
			},
			{VNum: 1002, Name: "Room B", Zone: 1, Flags: []string{"0", "0", "0", "0"}},
		},
		Zones: []parser.Zone{{Number: 1, TopRoom: 1999}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func parsedRoom(t *testing.T, w *World, vnum int) *parser.Room {
	t.Helper()
	parsed := w.GetParsedWorld()
	if parsed == nil {
		t.Fatal("world has no parsed data")
	}
	for i := range parsed.Rooms {
		if parsed.Rooms[i].VNum == vnum {
			return &parsed.Rooms[i]
		}
	}
	t.Fatalf("room %d missing from parsed data", vnum)
	return nil
}

// Routine mutations (door EX_ bits, runtime room bits) must publish to
// lock-free readers without paying — or disturbing — the structural insertion
// state: no parsed-definition rewrite, no roomOrder/sortedRoomVNums churn, no
// admission of missing exits. Runtime door state must also survive a later
// structural commit of an unrelated room, which C never reverts.
func TestRoutineMutationSkipsStructuralRebuild(t *testing.T) {
	w := routineMutationFixtureWorld(t)
	sortedLen, orderLen := len(w.sortedRoomVNums), len(w.roomOrder)

	if !w.SetExitInfo(1001, "north", parser.ExitIsDoor|parser.ExitPickproof) {
		t.Fatal("SetExitInfo failed")
	}
	live, ok := w.GetRoom(1001)
	if !ok || live.Exits["north"].ExitInfo&parser.ExitPickproof == 0 {
		t.Fatalf("lock-free reader missed door state: %+v", live)
	}
	if locked := w.GetRoomInWorld(1001); locked.Exits["north"].ExitInfo&parser.ExitPickproof == 0 {
		t.Fatal("locked reader missed door state")
	}
	if len(w.sortedRoomVNums) != sortedLen || len(w.roomOrder) != orderLen {
		t.Fatalf("routine mutation changed topology indices: sorted %d→%d order %d→%d",
			sortedLen, len(w.sortedRoomVNums), orderLen, len(w.roomOrder))
	}
	if exit := parsedRoom(t, w, 1001).Exits["north"]; exit.ExitInfo != parser.ExitIsDoor {
		t.Fatalf("routine mutation leaked into parsed definition: %+v", exit)
	}
	if w.SetExitInfo(1001, "west", parser.ExitIsDoor) {
		t.Fatal("SetExitInfo admitted a missing exit")
	}
	if !w.SetRoomFlagBit(1001, 1) {
		t.Fatal("SetRoomFlagBit failed")
	}
	if live, _ := w.GetRoom(1001); live == nil {
		t.Fatal("room vanished after flag-bit routine write")
	}

	// A structural commit of another room must not revert room 1001's door
	// state by repointing the map into the fresh parsed array.
	roomB, _ := w.SnapshotRoom(1002)
	roomB.Name = "Edited B"
	if !w.CommitEditedRoom(roomB) {
		t.Fatal("CommitEditedRoom failed")
	}
	if live, _ := w.GetRoom(1001); live.Exits["north"].ExitInfo&parser.ExitPickproof == 0 {
		t.Fatal("structural commit reverted unrelated runtime door state")
	}

	// An editor commit of the same room publishes its definition (including
	// the runtime door state the working copy snapshotted) to parsed data.
	roomA, _ := w.SnapshotRoom(1001)
	roomA.Name = "Committed A"
	if !w.CommitEditedRoom(roomA) {
		t.Fatal("CommitEditedRoom failed")
	}
	if parsedRoom(t, w, 1001).Name != "Committed A" {
		t.Fatal("editor commit did not publish the parsed definition")
	}

	// New-room insertion is still structural.
	if !w.CommitEditedRoom(parser.Room{VNum: 1050, Name: "New", Zone: 1, Flags: []string{"0", "0", "0", "0"}}) {
		t.Fatal("new-room commit failed")
	}
	if len(w.sortedRoomVNums) != sortedLen+1 {
		t.Fatalf("new-room insertion sorted vnums = %d, want %d", len(w.sortedRoomVNums), sortedLen+1)
	}
}

// The routine path is the door-operation hot path (open/close/lock commands,
// zone-reset D commands, scripted doors); this is its race vehicle.
func TestRoutineMutationConcurrentDoorWritersAndReaders(t *testing.T) {
	w := routineMutationFixtureWorld(t)
	const iterations = 300
	var wg sync.WaitGroup
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				room, ok := w.GetRoom(1001)
				if !ok || room.VNum != 1001 {
					t.Errorf("lock-free room = (%+v, %v)", room, ok)
					return
				}
				if _, ok := w.SnapshotRoom(1001); !ok {
					t.Errorf("snapshot room missing")
					return
				}
			}
		}()
	}
	for writer := 0; writer < 2; writer++ {
		wg.Add(1)
		go func(writer int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				info := parser.ExitIsDoor
				if (i+writer)%2 == 1 {
					info |= parser.ExitPickproof
				}
				if !w.SetExitInfo(1001, "north", info) {
					t.Errorf("door write %d failed", i)
					return
				}
				if !w.SetRoomFlagBit(1001, 1) {
					t.Errorf("flag write %d failed", i)
					return
				}
			}
		}(writer)
	}
	wg.Wait()
	if exit := w.GetRoomInWorld(1001).Exits["north"]; exit.ExitInfo&parser.ExitIsDoor == 0 {
		t.Fatalf("final door state = %+v", exit)
	}
}

func TestReditWorldPublicationConcurrentSnapshots(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1, Flags: []string{"0", "0", "0", "0"}},
			{VNum: 1002, Name: "Room B", Zone: 1, Flags: []string{"0", "0", "0", "0"}},
		},
		Zones: []parser.Zone{{Number: 1, TopRoom: 1999}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	const iterations = 200
	var wg sync.WaitGroup
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				room, ok := w.SnapshotRoom(1001)
				if !ok || room.VNum != 1001 {
					t.Errorf("snapshot room = (%+v, %v)", room, ok)
					return
				}
				rooms := w.SnapshotRooms()
				if len(rooms) != 2 || rooms[0].VNum != 1001 || rooms[1].VNum != 1002 {
					t.Errorf("snapshot rooms = %+v", rooms)
					return
				}
				if live, ok := w.GetRoom(1001); !ok || live.VNum != 1001 {
					t.Errorf("lock-free room = (%+v, %v)", live, ok)
					return
				}
			}
		}()
	}
	for writer := 0; writer < 2; writer++ {
		wg.Add(1)
		go func(writer int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				room, ok := w.SnapshotRoom(1001)
				if !ok {
					t.Errorf("writer snapshot missing room")
					return
				}
				room.Name = fmt.Sprintf("writer-%d-%d", writer, i)
				if !w.CommitEditedRoom(room) {
					t.Errorf("CommitEditedRoom returned false")
					return
				}
				switch i % 3 {
				case 0:
					if !w.SetRoomName(1001, fmt.Sprintf("admin-name-%d", i)) {
						t.Errorf("SetRoomName returned false")
						return
					}
				case 1:
					if !w.SetRoomDescription(1001, fmt.Sprintf("admin-description-%d", i)) {
						t.Errorf("SetRoomDescription returned false")
						return
					}
				case 2:
					if !w.SetRoomSector(1001, i%16) {
						t.Errorf("SetRoomSector returned false")
						return
					}
				}
				if !w.SetRoomScript(1001, i&31, fmt.Sprintf("script-%d", writer)) {
					t.Errorf("SetRoomScript returned false")
					return
				}
			}
		}(writer)
	}
	wg.Wait()
}
