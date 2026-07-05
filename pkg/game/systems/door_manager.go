// Package world provides door management for Dark Pawns.
package systems

import (
	"fmt"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DoorManager manages all doors in the world.
type DoorManager struct {
	mu    sync.RWMutex
	doors map[string]*Door // key: "fromRoom:direction" e.g., "3001:north"
}

// NewDoorManager creates a new DoorManager.
func NewDoorManager() *DoorManager {
	return &DoorManager{
		doors: make(map[string]*Door),
	}
}

// key generates a unique key for a door.
func (dm *DoorManager) key(fromRoom int, direction string) string {
	return fmt.Sprintf("%d:%s", fromRoom, direction)
}

// mirrorToReverseDoor updates the reciprocal exit to match door's state.
// Caller must hold dm.mu.
func (dm *DoorManager) mirrorToReverseDoor(door *Door) {
	revDir := reverseDirection(door.Direction)
	if revDir == "" {
		return
	}
	revKey := dm.key(door.ToRoom, revDir)
	revDoor, ok := dm.doors[revKey]
	if !ok || revDoor.ToRoom != door.FromRoom {
		return
	}
	revDoor.mu.Lock()
	defer revDoor.mu.Unlock()
	revDoor.Closed = door.Closed
	revDoor.Locked = door.Locked
	revDoor.Hp = door.Hp
}

// AddDoor adds a door to the manager.
func (dm *DoorManager) AddDoor(door *Door) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(door.FromRoom, door.Direction)
	dm.doors[key] = door
}

// GetDoor returns a snapshot of the door at fromRoom/direction.
func (dm *DoorManager) GetDoor(fromRoom int, direction string) (DoorSnapshot, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return DoorSnapshot{}, false
	}
	return door.Snapshot(), true
}

// GetDoorBetween returns a snapshot of the door connecting two rooms.
func (dm *DoorManager) GetDoorBetween(room1, room2 int) (DoorSnapshot, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, door := range dm.doors {
		if (door.FromRoom == room1 && door.ToRoom == room2) ||
			(door.FromRoom == room2 && door.ToRoom == room1) {
			return door.Snapshot(), true
		}
	}

	return DoorSnapshot{}, false
}

// RemoveDoor removes a door from the manager.
func (dm *DoorManager) RemoveDoor(fromRoom int, direction string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	delete(dm.doors, key)
}

// LoadDoorsFromWorld loads doors from parsed world data.
func (dm *DoorManager) LoadDoorsFromWorld(world *parser.World) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for i := range world.Rooms {
		for direction, exit := range world.Rooms[i].Exits {
			door := NewDoor(world.Rooms[i].VNum, exit.ToRoom, direction, exit.DoorState, exit.Key)
			dm.doors[dm.key(world.Rooms[i].VNum, direction)] = door
		}
	}
}

// GetDoorsInRoom returns snapshots of all doors in a room.
func (dm *DoorManager) GetDoorsInRoom(roomVNum int) []DoorSnapshot {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var doors []DoorSnapshot
	for _, door := range dm.doors {
		// Check if key starts with roomVNum:
		if door.FromRoom == roomVNum {
			doors = append(doors, door.Snapshot())
		}
	}

	return doors
}

// GetVisibleDoorsInRoom returns snapshots of only visible doors in a room.
func (dm *DoorManager) GetVisibleDoorsInRoom(roomVNum int) []DoorSnapshot {
	allDoors := dm.GetDoorsInRoom(roomVNum)
	var visibleDoors []DoorSnapshot

	for _, door := range allDoors {
		if door.CanSee() {
			visibleDoors = append(visibleDoors, door)
		}
	}

	return visibleDoors
}

// CanPass checks if a player can pass through a door.
func (dm *DoorManager) CanPass(fromRoom int, direction string) (bool, string) {
	door, ok := dm.GetDoor(fromRoom, direction)
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	if !door.IsPassable() {
		if door.Locked {
			return false, "The door is locked."
		}
		return false, "The door is closed."
	}

	return true, ""
}

// OpenDoor attempts to open a door.
func (dm *DoorManager) OpenDoor(fromRoom int, direction string) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Open()
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// CloseDoor attempts to close a door.
func (dm *DoorManager) CloseDoor(fromRoom int, direction string) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Close()
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// LockDoor attempts to lock a door.
func (dm *DoorManager) LockDoor(fromRoom int, direction string, keyVNum int) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Lock(keyVNum)
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// UnlockDoor attempts to unlock a door.
func (dm *DoorManager) UnlockDoor(fromRoom int, direction string, keyVNum int) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Unlock(keyVNum)
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// PickDoor attempts to pick a door lock.
func (dm *DoorManager) PickDoor(fromRoom int, direction string, skill int) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Pick(skill)
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// BashDoor attempts to bash a door.
func (dm *DoorManager) BashDoor(fromRoom int, direction string, strength int) (bool, string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := dm.key(fromRoom, direction)
	door, ok := dm.doors[key]
	if !ok {
		return false, "There is no door there."
	}

	if !door.CanSee() {
		return false, "There is no door there."
	}

	ok, msg := door.Bash(strength)
	if ok {
		dm.mirrorToReverseDoor(door)
	}
	return ok, msg
}

// ResetDoors resets all doors to their default state.
func (dm *DoorManager) ResetDoors() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, door := range dm.doors {
		door.Reset()
	}
}

// GetDoorStatus returns the status of a door.
func (dm *DoorManager) GetDoorStatus(fromRoom int, direction string) (string, bool) {
	door, ok := dm.GetDoor(fromRoom, direction)
	if !ok {
		return "", false
	}

	return door.GetStatus(), true
}

// Count returns the total number of doors.
func (dm *DoorManager) Count() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return len(dm.doors)
}
