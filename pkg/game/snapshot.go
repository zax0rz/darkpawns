package game

import (
	"sync/atomic"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// WorldSnapshot is a read-only point-in-time view of room topology.
// Created by World.PublishSnapshot() and served to readers via World.Snapshot().
// Readers get zero-lock access to a consistent view of room layout.
//
// Scope: Room topology only (rooms, exits, flags). Player/mob/item/combat state
// is protected by World.mu and does not require snapshotting — the RWLock path
// is sufficient for the expected player count. Full world snapshots are unnecessary
// for crash recovery; CircleMUD-style periodic player saves handle that.
type WorldSnapshot struct {
	Rooms map[int]*parser.Room
}

// SnapshotManager manages atomic pointer-swapped snapshots for lock-free reads.
// It provides generation tracking and safe concurrent access patterns.
type SnapshotManager struct {
	snapshot   atomic.Pointer[WorldSnapshot]
	generation atomic.Uint64
}

// NewSnapshotManager creates a new SnapshotManager.
func NewSnapshotManager() *SnapshotManager {
	return &SnapshotManager{}
}

// Snapshot returns the current read-only snapshot.
// Safe for concurrent use — no locks held.
func (sm *SnapshotManager) Snapshot() *WorldSnapshot {
	return sm.snapshot.Load()
}

// Publish atomically replaces the current snapshot with a new one
// built from the provided rooms map. Must be called while holding (or after
// releasing) the World write lock.
//
// The rooms map is shallow-copied — Room structs are configuration data,
// effectively read-only after world initialization.
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

// SnapshotGeneration returns the current snapshot generation counter.
// Readers can use this to detect whether a new snapshot has been published.
func (sm *SnapshotManager) SnapshotGeneration() uint64 {
	return sm.generation.Load()
}

// GetRoomFromSnapshot returns a room by VNum from the current snapshot, lock-free.
func (w *World) GetRoom(vnum int) (*parser.Room, bool) {
	snap := w.snapshots.Snapshot()
	if snap == nil {
		return nil, false
	}
	room, ok := snap.Rooms[vnum]
	return room, ok
}
