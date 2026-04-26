// Package game manages the game world state and player interactions.
package game

import (
	"log/slog"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Room flag bit indices matching C's ROOM_* constants in structs.h
const (
	roomDeath   = 1
	roomNoMob   = 2
	roomPrivate = 9
	roomGodRoom = 10
	roomHouse   = 11
	roomAtrium  = 13
)

// Sector type constants matching C's SECT_* constants.
const (
	sectInside = 0
	sectCity   = 1
)

// Number of equipment wear positions, matching C's NUM_WEARS.
const numWears = 22

// maxZone79Vnum is the max mob vnum (exclusive) for zone79 random placement.
const maxZone79Vnum = 7999

// Spawner manages spawning of mobs and objects from zone reset commands.
type Spawner struct {
	mu sync.RWMutex

	// World reference
	world *World

	// Track spawned instances
	mobInstances map[int][]*MobInstance    // key: mob vnum
	objInstances map[int][]*ObjectInstance // key: obj vnum
	roomMobs     map[int][]*MobInstance    // key: room vnum
	roomObjects  map[int][]*ObjectInstance // key: room vnum

	// Zone reset timers
	zoneTimers map[int]*time.Timer // key: zone number
}

// NewSpawner creates a new spawner for the given world.
func NewSpawner(world *World) *Spawner {
	return &Spawner{
		world:        world,
		mobInstances: make(map[int][]*MobInstance),
		objInstances: make(map[int][]*ObjectInstance),
		roomMobs:     make(map[int][]*MobInstance),
		roomObjects:  make(map[int][]*ObjectInstance),
		zoneTimers:   make(map[int]*time.Timer),
	}
}

// roomDirNames maps direction indices to exit lookup keys.
var roomDirNames = []string{"north", "east", "south", "west", "up", "down"}

// hasRoomFlagBit checks if a room has a specific bit set in its flags[0] bitmask.
func hasRoomFlagBit(room *parser.Room, bit int) bool {
	if room == nil || len(room.Flags) < 1 {
		return false
	}
	v, err := strconv.Atoi(room.Flags[0])
	if err != nil {
		return false
	}
	return (v & (1 << bit)) != 0
}

// isRoomValidForSpawn checks if a room is valid for zone79 random spawn.
// Excludes: PRIVATE, GODROOM, DEATH, NOMOB, HOUSE, ATRIUM flags, SECT_CITY, zone 163.
func isRoomValidForSpawn(room *parser.Room) bool {
	if room == nil {
		return false
	}
	if hasRoomFlagBit(room, roomDeath) {
		return false
	}
	if hasRoomFlagBit(room, roomNoMob) {
		return false
	}
	if hasRoomFlagBit(room, roomPrivate) {
		return false
	}
	if hasRoomFlagBit(room, roomGodRoom) {
		return false
	}
	if hasRoomFlagBit(room, roomHouse) {
		return false
	}
	if hasRoomFlagBit(room, roomAtrium) {
		return false
	}
	if room.Sector == sectCity {
		return false
	}
	if room.Zone == 163 {
		return false
	}
	return true
}

// isRoomValidForRandZon checks if a room is valid for MOB_RANDZON placement.
// Excludes restricted rooms; must match the given zone.
func isRoomValidForRandZon(room *parser.Room, zone int) bool {
	if room == nil {
		return false
	}
	if hasRoomFlagBit(room, roomDeath) {
		return false
	}
	if hasRoomFlagBit(room, roomNoMob) {
		return false
	}
	if hasRoomFlagBit(room, roomPrivate) {
		return false
	}
	if hasRoomFlagBit(room, roomGodRoom) {
		return false
	}
	if hasRoomFlagBit(room, roomHouse) {
		return false
	}
	if hasRoomFlagBit(room, roomAtrium) {
		return false
	}
	if room.Sector == sectCity {
		return false
	}
	if room.Zone != zone {
		return false
	}
	return true
}

// pickRandomRoom selects a random valid room from all world rooms (zone79 style).
func (s *Spawner) pickRandomRoom() *parser.Room {
	rooms := s.world.Rooms()
	if len(rooms) == 0 {
		return nil
	}

	// Try random picks first
	for attempt := 0; attempt < 5; attempt++ {
		idx := rand.Intn(len(rooms))
		if isRoomValidForSpawn(&rooms[idx]) {
			return &rooms[idx]
		}
	}

	// Fallback: linear scan
	for _, room := range rooms {
		if isRoomValidForSpawn(&room) {
			return &room
		}
	}
	return nil
}

// pickRandomZoneRoom selects a random valid room in the given zone (RANDZON style).
func (s *Spawner) pickRandomZoneRoom(zone int) *parser.Room {
	rooms := s.world.Rooms()
	if len(rooms) == 0 {
		return nil
	}

	// Try random picks first
	for attempt := 0; attempt < 5; attempt++ {
		idx := rand.Intn(len(rooms))
		if isRoomValidForRandZon(&rooms[idx], zone) {
			return &rooms[idx]
		}
	}

	// Fallback: linear scan
	for _, room := range rooms {
		if isRoomValidForRandZon(&room, zone) {
			return &room
		}
	}
	return nil
}

// percentLoad returns true if an object should load based on its load probability.
// Matches C: returns TRUE if GET_OBJ_LOAD(obj) > uniform() * 100.0
func percentLoad(obj *parser.Obj) bool {
	if obj == nil {
		return true
	}
	return obj.LoadPercent > (rand.Float64() * 100.0)
}

// ExecuteZoneReset executes all reset commands for a zone.
// Matches C's reset_zone() semantics including if_flag, loop, percent_load,
// MOB_RANDZON, zone79, door-state, and remove commands.
func (s *Spawner) ExecuteZoneReset(zone *parser.Zone) error {
	// Do NOT hold s.mu — SpawnMob/SpawnObject/CanSpawn each lock internally.
	// Holding s.mu causes a deadlock.

	var lastMob *MobInstance
	lastCmd := 0 // tracks whether last non-if_flag command succeeded
	tmpCmd := 0  // saved command index for loop
	loop := 0    // remaining loop iterations

	cmdCount := len(zone.Commands)
	for cmdIdx := 0; cmdIdx < cmdCount; cmdIdx++ {
		cmd := zone.Commands[cmdIdx]

		// IfFlag logic: skip if if_flag is set but last command did NOT succeed
		if cmd.IfFlag != 0 && lastCmd == 0 {
			continue
		}

		// Non-if_flag commands reset last_cmd
		if cmd.IfFlag == 0 {
			lastCmd = 0
		}

		switch cmd.Command {
		case "*": // ignore command
			continue

		case "L": // Start/End Looping
			if cmd.Arg2 == 0 {
				// Start loop: save current position, set iterations
				tmpCmd = cmdIdx
				loop = cmd.Arg3
				lastCmd = 1
			} else {
				// End loop: decrement counter, jump back if still > 0
				loop--
				if loop > 0 {
					cmdIdx = tmpCmd
				} else {
					loop = 0
					tmpCmd = 0
				}
			}
			continue

		case "M": // Load mobile
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn mob: max in world reached", "mob_vnum", cmd.Arg1, "max_in_world", cmd.Arg2)
				continue
			}

			mob, err := s.SpawnMob(cmd.Arg1, cmd.Arg3)
			if err != nil {
				slog.Error("error spawning mob", "mob_vnum", cmd.Arg1, "error", err)
				continue
			}
			lastMob = mob
			lastCmd = 1

			// zone79 randload: mobs with vnums 7900-7998 placed in random room
			if cmd.Arg1 > 7899 && cmd.Arg1 < maxZone79Vnum {
				randRoom := s.pickRandomRoom()
				if randRoom != nil {
					s.moveMobToRoom(mob, randRoom.VNum)
				}
			}

			// MOB_RANDZON: random room within the same zone
			spawnRoom := s.world.GetRoomInWorld(cmd.Arg3)
			if spawnRoom != nil && mob.HasFlag("randzon") {
				randRoom := s.pickRandomZoneRoom(spawnRoom.Zone)
				if randRoom != nil {
					s.moveMobToRoom(mob, randRoom.VNum)
				}
			}

		case "O": // Load object to room
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2)
				continue
			}

			proto, ok := s.world.GetObjPrototype(cmd.Arg1)
			if !ok {
				slog.Warn("object prototype not found", "obj_vnum", cmd.Arg1)
				continue
			}

			if !percentLoad(proto) {
				slog.Debug("object not loaded per percent_load", "obj_vnum", cmd.Arg1, "load_percent", proto.LoadPercent)
				continue
			}

			if cmd.Arg3 >= 0 {
				// Load object to specific room
				_, err := s.SpawnObject(cmd.Arg1, cmd.Arg3)
				if err != nil {
					slog.Error("error spawning object", "obj_vnum", cmd.Arg1, "error", err)
					continue
				}
			} else {
				// arg3 < 0: create object floating (NOWHERE) — like C's obj->in_room = NOWHERE
				obj, err := s.SpawnObject(cmd.Arg1, -1)
				if err != nil {
					slog.Error("error spawning floating object", "obj_vnum", cmd.Arg1, "error", err)
					continue
				}
				obj.Location = LocNowhere()
			}
			lastCmd = 1

		case "G": // Give object to last loaded mob
			if lastMob == nil {
				slog.Warn("G command: no lastMob available")
				continue
			}
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for mob: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "mob_inventory")
				continue
			}

			proto, ok := s.world.GetObjPrototype(cmd.Arg1)
			if !ok {
				slog.Warn("object prototype not found for G command", "obj_vnum", cmd.Arg1)
				continue
			}
			if !percentLoad(proto) {
				slog.Debug("object not loaded per percent_load (G)", "obj_vnum", cmd.Arg1, "load_percent", proto.LoadPercent)
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, -1)
			if err != nil {
				slog.Error("error spawning object for mob", "obj_vnum", cmd.Arg1, "error", err, "context", "mob_inventory")
				continue
			}
			lastMob.Inventory = append(lastMob.Inventory, obj)
			lastCmd = 1

		case "E": // Equip object on last loaded mob
			if lastMob == nil {
				slog.Warn("E command: no lastMob available")
				continue
			}
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for mob equip: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "mob_equip")
				continue
			}

			proto, ok := s.world.GetObjPrototype(cmd.Arg1)
			if !ok {
				slog.Warn("object prototype not found for E command", "obj_vnum", cmd.Arg1)
				continue
			}
			if !percentLoad(proto) {
				slog.Debug("object not loaded per percent_load (E)", "obj_vnum", cmd.Arg1, "load_percent", proto.LoadPercent)
				continue
			}

			if cmd.Arg3 < 0 || cmd.Arg3 >= numWears {
				slog.Warn("invalid equipment position", "pos", cmd.Arg3)
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, -1)
			if err != nil {
				slog.Error("error spawning object for mob equip", "obj_vnum", cmd.Arg1, "error", err, "context", "mob_equip")
				continue
			}
			if lastMob.Equipment == nil {
				lastMob.Equipment = make(map[int]*ObjectInstance)
			}
			lastMob.Equipment[cmd.Arg3] = obj // Arg3 = equip position
			lastCmd = 1

		case "P": // Put object in container
			container := s.findObjectInstance(cmd.Arg3)
			if container == nil {
				slog.Warn("P command: container object not found", "container_vnum", cmd.Arg3)
				continue
			}
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for container: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "container")
				continue
			}

			proto, ok := s.world.GetObjPrototype(cmd.Arg1)
			if !ok {
				slog.Warn("object prototype not found for P command", "obj_vnum", cmd.Arg1)
				continue
			}
			if !percentLoad(proto) {
				slog.Debug("object not loaded per percent_load (P)", "obj_vnum", cmd.Arg1, "load_percent", proto.LoadPercent)
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, -1)
			if err != nil {
				slog.Error("error spawning object for container", "obj_vnum", cmd.Arg1, "error", err, "context", "container")
				continue
			}
			s.world.MoveObjectToContainer(obj, container)
			lastCmd = 1

		case "D": // Door state: arg2=direction, arg3=state (0=open, 1=closed, 2=locked)
			if cmd.Arg2 < 0 || cmd.Arg2 >= len(roomDirNames) {
				slog.Warn("Invalid door direction", "dir", cmd.Arg2, "room", cmd.Arg1)
				continue
			}
			room := s.world.GetRoomInWorld(cmd.Arg1)
			if room == nil {
				slog.Warn("Door command: room not found", "room", cmd.Arg1)
				continue
			}
			ext, ok := room.Exits[roomDirNames[cmd.Arg2]]
			if !ok {
				slog.Warn("Door command: exit not found", "room", cmd.Arg1, "dir", roomDirNames[cmd.Arg2])
				continue
			}
			// Arg3: 0=open, 1=closed, 2=locked
			ext.DoorState = cmd.Arg3
			lastCmd = 1

		case "R": // Remove obj/mob from room
			// Go parser convention: Arg2=vnum, Arg3=type (1=obj, 0=mob)
			if cmd.Arg3 == 1 { // Remove object
				s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
			} else { // Remove mob
				s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
			}
			lastCmd = 1
		}
	}

	return nil
}

// moveMobToRoom relocates a mob instance to a different room.
func (s *Spawner) moveMobToRoom(mob *MobInstance, newRoomVNum int) {
	if mob == nil {
		return
	}

	oldRoom := mob.RoomVNum
	if oldRoom >= 0 {
		if mobs, ok := s.roomMobs[oldRoom]; ok {
			for i, m := range mobs {
				if m == mob {
					s.roomMobs[oldRoom] = append(mobs[:i], mobs[i+1:]...)
					break
				}
			}
		}
	}

	mob.RoomVNum = newRoomVNum
	if newRoomVNum >= 0 {
		s.roomMobs[newRoomVNum] = append(s.roomMobs[newRoomVNum], mob)
	}
}

// CanSpawn checks if we can spawn more of a given mob/obj based on maxInWorld.
func (s *Spawner) CanSpawn(vnum int, maxInWorld int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if instances, ok := s.mobInstances[vnum]; ok {
		return len(instances) < maxInWorld
	}
	if instances, ok := s.objInstances[vnum]; ok {
		return len(instances) < maxInWorld
	}
	return maxInWorld > 0
}

// SpawnMob creates a new mob instance in the specified room.
func (s *Spawner) SpawnMob(mobVNum, roomVNum int) (*MobInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mob, err := s.world.SpawnMob(mobVNum, roomVNum)
	if err != nil {
		return nil, err
	}

	s.mobInstances[mobVNum] = append(s.mobInstances[mobVNum], mob)
	if roomVNum >= 0 {
		s.roomMobs[roomVNum] = append(s.roomMobs[roomVNum], mob)
	}
	return mob, nil
}

// SpawnObject creates a new object instance.
func (s *Spawner) SpawnObject(objVNum, roomVNum int) (*ObjectInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, err := s.world.SpawnObject(objVNum, roomVNum)
	if err != nil {
		return nil, err
	}

	s.objInstances[objVNum] = append(s.objInstances[objVNum], obj)
	if roomVNum >= 0 {
		s.roomObjects[roomVNum] = append(s.roomObjects[roomVNum], obj)
	}
	return obj, nil
}

// GetMobsInRoom returns all mob instances in a room.
func (s *Spawner) GetMobsInRoom(roomVNum int) []*MobInstance {
	return s.world.GetMobsInRoom(roomVNum)
}

// GetObjectsInRoom returns all object instances in a room.
func (s *Spawner) GetObjectsInRoom(roomVNum int) []*ObjectInstance {
	return s.world.GetItemsInRoom(roomVNum)
}

// findObjectInstance finds an object instance by vnum (simple implementation).
func (s *Spawner) findObjectInstance(objVNum int) *ObjectInstance {
	if instances, ok := s.objInstances[objVNum]; ok && len(instances) > 0 {
		return instances[0]
	}
	return nil
}

// removeObjectFromRoom removes an object instance from a room.
func (s *Spawner) removeObjectFromRoom(roomVNum, objVNum int) {
	if instances, ok := s.roomObjects[roomVNum]; ok {
		for i, obj := range instances {
			if obj.VNum == objVNum {
				s.roomObjects[roomVNum] = append(instances[:i], instances[i+1:]...)
				if objInstances, ok2 := s.objInstances[objVNum]; ok2 {
					for j, obj2 := range objInstances {
						if obj2 == obj {
							s.objInstances[objVNum] = append(objInstances[:j], objInstances[j+1:]...)
							break
						}
					}
				}
				break
			}
		}
	}
}

// removeMobFromRoom removes a mob instance from a room.
func (s *Spawner) removeMobFromRoom(roomVNum, mobVNum int) {
	if instances, ok := s.roomMobs[roomVNum]; ok {
		for i, mob := range instances {
			if mob.VNum == mobVNum {
				s.roomMobs[roomVNum] = append(instances[:i], instances[i+1:]...)
				if mobInstances, ok2 := s.mobInstances[mobVNum]; ok2 {
					for j, mob2 := range mobInstances {
						if mob2 == mob {
							s.mobInstances[mobVNum] = append(mobInstances[:j], mobInstances[j+1:]...)
							break
						}
					}
				}
				break
			}
		}
	}
}

// StartPeriodicResets starts the periodic zone reset timer.
func (s *Spawner) StartPeriodicResets(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.resetEmptyZones()
		}
	}()
}

// resetEmptyZones resets zones that are empty (no players or mobs).
func (s *Spawner) resetEmptyZones() {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("Periodic zone reset check")
}
