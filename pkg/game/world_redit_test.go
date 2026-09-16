package game

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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
