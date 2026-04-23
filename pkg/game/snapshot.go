package game

import (
	"sync/atomic"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// WorldSnapshot is a read-only point-in-time view of world state.
type WorldSnapshot struct {
	Rooms map[int]*parser.Room
}

// SnapshotManager manages atomic pointer-swapped snapshots for lock-free reads.
type SnapshotManager struct {
	snapshot   atomic.Pointer[WorldSnapshot]
	generation atomic.Uint64
}

// NewSnapshotManager creates a new SnapshotManager.
func NewSnapshotManager() *SnapshotManager {
	return &SnapshotManager{}
}

// Snapshot returns the current read-only snapshot. Safe for concurrent use.
func (sm *SnapshotManager) Snapshot() *WorldSnapshot {
	return sm.snapshot.Load()
}

// Generation returns the current snapshot generation counter.
func (sm *SnapshotManager) Generation() uint64 {
	return sm.generation.Load()
}

// Publish atomically replaces the current snapshot with a new one built from rooms.
func (sm *SnapshotManager) Publish(rooms map[int]*parser.Room) {
	snap := &WorldSnapshot{
		Rooms: make(map[int]*parser.Room, len(rooms)),
	}
	for vnum, room := range rooms {
		snap.Rooms[vnum] = room
	}
	sm.snapshot.Store(snap)
	sm.generation.Add(1)
}
