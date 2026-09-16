package game

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CloneRoom returns an independent room value suitable for an OLC working
// copy. Room topology is edited off the live world and swapped in atomically
// only when the C REDIT confirmation accepts it.
func CloneRoom(room parser.Room) parser.Room {
	copyRoom := room
	copyRoom.Flags = append([]string(nil), room.Flags...)
	if room.Exits != nil {
		copyRoom.Exits = make(map[string]parser.Exit, len(room.Exits))
		for direction, exit := range room.Exits {
			copyRoom.Exits[direction] = exit
		}
	} else {
		copyRoom.Exits = make(map[string]parser.Exit)
	}
	copyRoom.ExtraDescs = append([]parser.ExtraDesc(nil), room.ExtraDescs...)
	return copyRoom
}

// SnapshotRoom returns a deep copy of one room while holding the world lock.
// It is the safe setup boundary for descriptor-owned OLC state.
func (w *World) SnapshotRoom(vnum int) (parser.Room, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	room, ok := w.rooms[vnum]
	if !ok || room == nil {
		return parser.Room{}, false
	}
	return CloneRoom(*room), true
}

// SnapshotRooms returns deep room copies from one locked point in time. The
// result is ordered by VNUM because C's room table and redit disk writer both
// walk the ascending world[] array.
func (w *World) SnapshotRooms() []parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()

	rooms := make([]parser.Room, 0, len(w.rooms))
	for _, room := range w.rooms {
		if room == nil {
			continue
		}
		rooms = append(rooms, CloneRoom(*room))
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].VNum < rooms[j].VNum })
	return rooms
}

// CommitEditedRoom atomically replaces or inserts a room after REDIT's
// "save internally" confirmation. It swaps the room pointer and publishes a
// new topology snapshot, so readers holding an older snapshot never observe a
// partially edited room. The parsed-world copy is also updated so a restart
// or a subsequent world export sees the accepted in-memory definition.
func (w *World) CommitEditedRoom(room parser.Room) bool {
	room = CloneRoom(room)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.commitEditedRoomLocked(room)
}

// updateRoom applies one short administrative/runtime room mutation to a
// copied room while holding the same lock as REDIT publication. This keeps
// the lock-free topology snapshot immutable for readers that overlap a web
// admin write or an active room editor.
func (w *World) updateRoom(vnum int, update func(*parser.Room)) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok || room == nil {
		return false
	}
	copyRoom := CloneRoom(*room)
	update(&copyRoom)
	return w.commitEditedRoomLocked(copyRoom)
}

func (w *World) commitEditedRoomLocked(room parser.Room) bool {
	room = CloneRoom(room)

	if w.rooms == nil {
		w.rooms = make(map[int]*parser.Room)
	}
	_, existed := w.rooms[room.VNum]

	if w.parsedData != nil {
		parsedRooms := make([]parser.Room, len(w.parsedData.Rooms), len(w.parsedData.Rooms)+1)
		copy(parsedRooms, w.parsedData.Rooms)
		parsedIndex := -1
		for i := range parsedRooms {
			if parsedRooms[i].VNum == room.VNum {
				parsedIndex = i
				break
			}
		}
		if parsedIndex < 0 {
			parsedRooms = append(parsedRooms, room)
			existed = false
		} else {
			parsedRooms[parsedIndex] = room
		}
		w.parsedData.Rooms = parsedRooms

		// The backing array is copied before replacement. Repoint every parsed
		// room so old snapshots and escaped read pointers stay immutable.
		updated := make(map[int]*parser.Room, len(w.rooms)+1)
		for i := range w.parsedData.Rooms {
			updated[w.parsedData.Rooms[i].VNum] = &w.parsedData.Rooms[i]
		}
		for vnum, oldRoom := range w.rooms {
			if _, ok := updated[vnum]; !ok {
				updated[vnum] = oldRoom
			}
		}
		w.rooms = updated
	} else {
		w.rooms[room.VNum] = &room
	}

	if !existed {
		w.roomOrder = append(w.roomOrder, room.VNum)
		sort.Ints(w.roomOrder)
	}
	w.sortedRoomVNums = append([]int(nil), w.roomOrder...)
	for vnum := range w.rooms {
		if !containsInt(w.sortedRoomVNums, vnum) {
			w.sortedRoomVNums = append(w.sortedRoomVNums, vnum)
		}
	}
	sort.Ints(w.sortedRoomVNums)
	if w.snapshots != nil {
		w.snapshots.Publish(w.rooms)
	}
	return true
}

// SetRoomScript replaces the live script fields used by REDIT's script menu.
// C shallow-copies room scripts into OLC, so these two fields intentionally
// bypass the working room and become visible before a room save/quit.
func (w *World) SetRoomScript(vnum, flags int, name string) bool {
	return w.updateRoom(vnum, func(room *parser.Room) {
		room.ScriptName = name
		room.ScriptFunctions = flags
	})
}

func containsInt(values []int, target int) bool {
	index := sort.SearchInts(values, target)
	return index < len(values) && values[index] == target
}
