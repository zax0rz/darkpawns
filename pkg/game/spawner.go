// Package game manages the game world state and player interactions.
package game

import (
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"
	"github.com/zax0rz/darkpawns/pkg/dprng"

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

// FlagItemRare is ITEM_RARE — structs.h:492, bit 24 of ExtraFlags[0].
const FlagItemRare = 1 << 24

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

	// done signals the periodic reset goroutine to stop.
	done     chan struct{}
	doneOnce sync.Once
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

	// C (db.c:2116-2126) rejection-samples without an attempt cap or fallback.
	// Every rejected room consumes another draw from the shared stream.
	for {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		idx := zoneRoomNumber(0, len(rooms)-1)
		if isRoomValidForSpawn(&rooms[idx]) {
			return &rooms[idx]
		}
	}
}

// pickRandomZoneRoom selects a random valid room in the given zone (RANDZON style).
func (s *Spawner) pickRandomZoneRoom(zone int) *parser.Room {
	rooms := s.world.Rooms()
	if len(rooms) == 0 {
		return nil
	}

	// C (db.c:2133-2141) rejection-samples without an attempt cap or fallback.
	// Unlike zone79 placement, RANDZON accepts city-sector rooms.
	for {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		idx := zoneRoomNumber(0, len(rooms)-1)
		if isRoomValidForRandZon(&rooms[idx], zone) {
			return &rooms[idx]
		}
	}
}

// percentLoad returns true if an object should load based on its load probability.
// Matches C: returns TRUE if GET_OBJ_LOAD(obj) > uniform() * 100.0
func percentLoad(obj *parser.Obj) bool {
	if obj == nil {
		return true
	}
	// #nosec G404 — game RNG, not cryptographic
	return obj.LoadPercent > (float64(dprng.Uniform()) * 100.0)
}

var (
	zoneObjectPercentLoad = percentLoad
	zoneObjectInitRare    = initRare
	zoneRareNumber        = dprng.Number
	zoneRoomNumber        = dprng.Number
)

// ExecuteZoneReset executes all reset commands for a zone.
// Matches C's reset_zone() semantics including if_flag, loop, percent_load,
// MOB_RANDZON, zone79, door-state, and remove commands.
func (s *Spawner) ExecuteZoneReset(zone *parser.Zone) error {
	// Do NOT hold s.mu — spawn and global-count helpers lock internally.
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
			if !s.canSpawnMob(cmd.Arg1, cmd.Arg2) {
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
			if spawnRoom != nil && mob.HasFlag("RANDZON") {
				randRoom := s.pickRandomZoneRoom(spawnRoom.Zone)
				if randRoom != nil {
					s.moveMobToRoom(mob, randRoom.VNum)
				}
			}

		case "O": // Load object to room
			if !s.canSpawnObject(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2)
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, cmd.Arg3)
			if err != nil {
				slog.Error("error spawning object", "obj_vnum", cmd.Arg1, "error", err)
				continue
			}
			if cmd.Arg3 < 0 {
				// C floating O commands skip percent_load entirely.
				lastCmd = 1
				continue
			}
			if !zoneObjectPercentLoad(obj.Prototype) {
				slog.Debug("object not loaded per percent_load", "obj_vnum", cmd.Arg1, "load_percent", obj.Prototype.LoadPercent)
				s.extractSpawnedObject(obj)
				continue
			}
			lastCmd = 1

		case "G": // Give object to last loaded mob
			if lastMob == nil {
				slog.Warn("G command: no lastMob available")
				continue
			}
			if !s.canSpawnObject(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for mob: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "mob_inventory")
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, -1)
			if err != nil {
				slog.Error("error spawning object for mob", "obj_vnum", cmd.Arg1, "error", err, "context", "mob_inventory")
				continue
			}
			if !zoneObjectPercentLoad(obj.Prototype) {
				slog.Debug("object not loaded per percent_load (G)", "obj_vnum", cmd.Arg1, "load_percent", obj.Prototype.LoadPercent)
				s.extractSpawnedObject(obj)
				continue
			}
			lastMob.Inventory = append(lastMob.Inventory, obj)
			lastCmd = 1

		case "E": // Equip object on last loaded mob
			if lastMob == nil {
				slog.Warn("E command: no lastMob available")
				continue
			}
			if !s.canSpawnObject(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for mob equip: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "mob_equip")
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
			if !zoneObjectPercentLoad(obj.Prototype) {
				slog.Debug("object not loaded per percent_load (E)", "obj_vnum", cmd.Arg1, "load_percent", obj.Prototype.LoadPercent)
				s.extractSpawnedObject(obj)
				continue
			}
			if lastMob.Equipment == nil {
				lastMob.Equipment = make(map[int]*ObjectInstance)
			}
			lastMob.Equipment[cmd.Arg3] = obj // Arg3 = equip position
			lastCmd = 1

		case "P": // Put object in container
			if !s.canSpawnObject(cmd.Arg1, cmd.Arg2) {
				slog.Warn("cannot spawn object for container: max in world reached", "obj_vnum", cmd.Arg1, "max_in_world", cmd.Arg2, "context", "container")
				continue
			}

			obj, err := s.SpawnObject(cmd.Arg1, -1)
			if err != nil {
				slog.Error("error spawning object for container", "obj_vnum", cmd.Arg1, "error", err, "context", "container")
				continue
			}
			container := s.findObjectInstance(cmd.Arg3)
			if container == nil {
				// C leaves the newly read object floating and counted when the
				// target container is missing, without calling percent_load.
				slog.Warn("P command: container object not found", "container_vnum", cmd.Arg3)
				continue
			}
			if !zoneObjectPercentLoad(obj.Prototype) {
				slog.Debug("object not loaded per percent_load (P)", "obj_vnum", cmd.Arg1, "load_percent", obj.Prototype.LoadPercent)
				s.extractSpawnedObject(obj)
				continue
			}
			if err := s.world.MoveObjectToContainer(obj, container); err != nil {
				slog.Warn("MoveObjectToContainer failed in spawner", "obj_vnum", obj.GetVNum(), "error", err)
			}
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
			// Arg3 is runtime state: 0=open, 1=closed, 2=closed+locked.
			ext.ExitInfo = parser.ApplyDoorReset(ext.ExitInfo, cmd.Arg3)
			s.world.SetExitInfo(cmd.Arg1, roomDirNames[cmd.Arg2], ext.ExitInfo)
			lastCmd = 1

		case "R": // Remove obj/mob from room
			// Go parser convention: Arg2=vnum, Arg3=type (1=obj, 0=mob)
			removed := false
			if cmd.Arg3 == 1 { // Remove object
				removed = s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
			} else { // Remove mob
				removed = s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
			}
			if removed {
				lastCmd = 1
			}
		}
	}

	return nil
}

// moveMobToRoom relocates a mob instance to a different room.
func (s *Spawner) moveMobToRoom(mob *MobInstance, newRoomVNum int) {
	if mob == nil {
		return
	}

	oldRoom := mob.GetRoom()
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

	mob.SetRoom(newRoomVNum)
	if newRoomVNum >= 0 {
		s.roomMobs[newRoomVNum] = append(s.roomMobs[newRoomVNum], mob)
	}
}

// C tests each prototype's global live-instance count. Runtime instances made
// outside this Spawner (including character-creation gear) still count against
// zone command maxima.
func (s *Spawner) canSpawnMob(vnum int, maxInWorld int) bool {
	return s.world.countMobInstances(vnum) < maxInWorld
}

func (s *Spawner) canSpawnObject(vnum int, maxInWorld int) bool {
	return s.world.countObjectInstances(vnum) < maxInWorld
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
	if roomVNum >= 0 {
		// World.SpawnObject registers runtime identity and location but leaves
		// room indexing to its caller. Zone O commands are obj_to_room calls in
		// C, so the spawner must also make the object visible in room contents.
		s.world.AddItemToRoom(obj, roomVNum)
	}

	// Apply ITEM_RARE affect variance — db.c:1899-1925 init_rare() (DP-376)
	if obj.Prototype != nil && obj.Prototype.ExtraFlags[0]&FlagItemRare != 0 {
		zoneObjectInitRare(obj)
	}

	s.objInstances[objVNum] = append(s.objInstances[objVNum], obj)
	if roomVNum >= 0 {
		s.roomObjects[roomVNum] = append(s.roomObjects[roomVNum], obj)
	}
	return obj, nil
}

// initRare applies random stat variance to a rare item's applies.
// Ports db.c init_rare() — each apply has 20% chance of +/-1 (damroll/hitroll) or +/-5 (AC).
func initRare(obj *ObjectInstance) {
	if obj.Prototype == nil || len(obj.Prototype.Affects) == 0 {
		return
	}
	affects := make([]parser.ObjAffect, len(obj.Prototype.Affects))
	copy(affects, obj.Prototype.Affects)
	for i, a := range affects {
		if a.Location == 0 {
			continue
		}
		// #nosec G404 — game RNG, not cryptographic
		if zoneRareNumber(1, 100) > 20 {
			continue
		}
		mod := 0
		switch a.Location {
		case 19, 18: // APPLY_DAMROLL, APPLY_HITROLL
			mod = 1
		case 17: // APPLY_AC
			mod = 5
		}
		// C consumes the sign draw even for an unhandled apply location, where
		// mod remains zero.
		if zoneRareNumber(0, 1) == 0 {
			mod = -mod
		}
		affects[i].Modifier += mod
	}
	obj.SetAffectsOverride(affects)
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

// extractSpawnedObject undoes SpawnObject after a failed percent_load gate.
// C's extract_obj decrements obj_index[].number; removing every spawner index
// here keeps max-in-world checks at the same command-boundary count.
func (s *Spawner) extractSpawnedObject(obj *ObjectInstance) {
	if obj == nil {
		return
	}

	s.mu.Lock()
	if instances := s.objInstances[obj.VNum]; len(instances) > 0 {
		for i, candidate := range instances {
			if candidate == obj {
				s.objInstances[obj.VNum] = append(instances[:i], instances[i+1:]...)
				break
			}
		}
	}
	roomVNum := obj.GetRoomVNum()
	if instances := s.roomObjects[roomVNum]; len(instances) > 0 {
		for i, candidate := range instances {
			if candidate == obj {
				s.roomObjects[roomVNum] = append(instances[:i], instances[i+1:]...)
				break
			}
		}
	}
	s.mu.Unlock()

	s.world.ExtractObject(obj, roomVNum)
}

// removeObjectFromRoom removes an object instance from a room.
func (s *Spawner) removeObjectFromRoom(roomVNum, objVNum int) bool {
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
				// Clean up global state — db.c zone reset 'R' (DP-373)
				s.world.ExtractObject(obj, roomVNum)
				return true
			}
		}
	}
	return false
}

// removeMobFromRoom removes a mob instance from a room.
func (s *Spawner) removeMobFromRoom(roomVNum, mobVNum int) bool {
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
				// Clean up global state — db.c zone reset 'R' (DP-373)
				s.world.ExtractMob(mob)
				return true
			}
		}
	}
	return false
}

// RegisterObjectInstance adds a deserialized object to the spawner's tracking.
// Called when loading player saves so the spawner respects max-in-world limits.
func (s *Spawner) RegisterObjectInstance(obj *ObjectInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if obj == nil || obj.Prototype == nil {
		return
	}
	s.objInstances[obj.Prototype.VNum] = append(s.objInstances[obj.Prototype.VNum], obj)
}

// RemoveMobInstance decrements the spawner's instance count for a dead mob,
// allowing it to respawn on the next zone reset.
func (s *Spawner) RemoveMobInstance(mobVNum int, mob *MobInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if instances, ok := s.mobInstances[mobVNum]; ok {
		for i, m := range instances {
			if m == mob {
				s.mobInstances[mobVNum] = append(instances[:i], instances[i+1:]...)
				break
			}
		}
	}

	// Also remove from roomMobs tracking
	if mob.GetRoom() >= 0 {
		if roomInstances, ok := s.roomMobs[mob.GetRoom()]; ok {
			for i, m := range roomInstances {
				if m == mob {
					s.roomMobs[mob.GetRoom()] = append(roomInstances[:i], roomInstances[i+1:]...)
					break
				}
			}
		}
	}
}

// StartPeriodicResets starts the periodic zone reset timer.
func (s *Spawner) StartPeriodicResets(interval time.Duration) {
	if dpclock.Frozen() {
		return
	}
	s.done = make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.resetEmptyZones()
				if s.world != nil {
					s.world.RebuildSpecRooms()
				}
			case <-s.done:
				ticker.Stop()
				return
			}
		}
	}()
}

// StopPeriodicResets signals the periodic reset goroutine to exit cleanly.
// Safe to call multiple times.
func (s *Spawner) StopPeriodicResets() {
	if s.done == nil {
		return
	}
	s.doneOnce.Do(func() {
		close(s.done)
	})
}

// resetEmptyZones resets zones that have no active players in them.
// Matches C behavior: db.c's zone_point_update() resets a zone when its
// timer fires AND no PCs are present (is_zone_empty check).
func (s *Spawner) resetEmptyZones() {
	// Do NOT hold s.mu here — ExecuteZoneReset handles its own locking internally.
	// Also need to check player rooms which requires world lock.

	zones := s.world.GetAllZones()
	for _, zone := range zones {
		if s.zoneHasPlayers(zone.Number) {
			continue
		}
		if err := s.ExecuteZoneReset(zone); err != nil {
			slog.Warn("periodic zone reset failed", "zone", zone.Number, "error", err)
		}
	}
}

// zoneHasPlayers returns true if any player is in a room belonging to the given zone.
func (s *Spawner) zoneHasPlayers(zoneNum int) bool {
	hasPlayer := false
	s.world.ForEachPlayerInZone(zoneNum, func(p *Player) {
		hasPlayer = true
	})
	return hasPlayer
}
