// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Spawner manages spawning of mobs and objects from zone reset commands.
type Spawner struct {
	mu sync.RWMutex

	// World reference
	world *World

	// Track spawned instances
	mobInstances    map[int][]*MobInstance    // key: mob vnum
	objInstances    map[int][]*ObjectInstance // key: obj vnum
	roomMobs        map[int][]*MobInstance    // key: room vnum
	roomObjects     map[int][]*ObjectInstance // key: room vnum

	// Zone reset timers
	zoneTimers map[int]*time.Timer // key: zone number
}

// NewSpawner creates a new spawner for the given world.
func NewSpawner(world *World) *Spawner {
	return &Spawner{
		world:           world,
		mobInstances:    make(map[int][]*MobInstance),
		objInstances:    make(map[int][]*ObjectInstance),
		roomMobs:        make(map[int][]*MobInstance),
		roomObjects:     make(map[int][]*ObjectInstance),
		zoneTimers:      make(map[int]*time.Timer),
	}
}

// StartZoneResets executes all zone resets on server start.
func (s *Spawner) StartZoneResets() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get zones from world (need to access world's internal data)
	// For now, we'll assume we can get zones somehow
	// In a real implementation, we'd need to expose zones from World
	
	// This is a placeholder - actual implementation would iterate through zones
	// and call ExecuteZoneReset for each
	
	return nil
}

// ExecuteZoneReset executes all reset commands for a zone.
func (s *Spawner) ExecuteZoneReset(zone *parser.Zone) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Track the last mob spawned for G and E commands
	var lastMob *MobInstance

	for _, cmd := range zone.Commands {
		switch cmd.Command {
		case "M": // Load mobile
			// Check if we can spawn more of this mob
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				fmt.Printf("Cannot spawn mob %d: max in world (%d) reached\n", cmd.Arg1, cmd.Arg2)
				continue
			}
			
			mob, err := s.SpawnMob(cmd.Arg1, cmd.Arg3)
			if err != nil {
				// Log error but continue with other commands
				fmt.Printf("Error spawning mob %d: %v\n", cmd.Arg1, err)
				continue
			}
			lastMob = mob

		case "O": // Load object to room
			// Check if we can spawn more of this object
			if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
				fmt.Printf("Cannot spawn object %d: max in world (%d) reached\n", cmd.Arg1, cmd.Arg2)
				continue
			}
			
			_, err := s.SpawnObject(cmd.Arg1, cmd.Arg3)
			if err != nil {
				fmt.Printf("Error spawning object %d: %v\n", cmd.Arg1, err)
			}

		case "G": // Give object to last loaded mob
			if lastMob != nil {
				// Check if we can spawn more of this object
				if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
					fmt.Printf("Cannot spawn object %d for mob: max in world (%d) reached\n", cmd.Arg1, cmd.Arg2)
					continue
				}
				
				obj, err := s.SpawnObject(cmd.Arg1, -1) // -1 means give to mob, not room
				if err != nil {
					fmt.Printf("Error spawning object %d for mob: %v\n", cmd.Arg1, err)
					continue
				}
				lastMob.Inventory = append(lastMob.Inventory, obj)
			}

		case "E": // Equip object on last loaded mob
			if lastMob != nil {
				// Check if we can spawn more of this object
				if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
					fmt.Printf("Cannot spawn object %d for mob equip: max in world (%d) reached\n", cmd.Arg1, cmd.Arg2)
					continue
				}
				
				obj, err := s.SpawnObject(cmd.Arg1, -1)
				if err != nil {
					fmt.Printf("Error spawning object %d for mob equip: %v\n", cmd.Arg1, err)
					continue
				}
				// Simple equipment - just add to equipment map
				if lastMob.Equipment == nil {
					lastMob.Equipment = make(map[int]*ObjectInstance)
				}
				lastMob.Equipment[cmd.Arg3] = obj // Arg3 is equip position
			}

		case "P": // Put object in container
			// Find container object
			container := s.findObjectInstance(cmd.Arg3)
			if container != nil {
				// Check if we can spawn more of this object
				if !s.CanSpawn(cmd.Arg1, cmd.Arg2) {
					fmt.Printf("Cannot spawn object %d for container: max in world (%d) reached\n", cmd.Arg1, cmd.Arg2)
					continue
				}
				
				obj, err := s.SpawnObject(cmd.Arg1, -1)
				if err != nil {
					fmt.Printf("Error spawning object %d for container: %v\n", cmd.Arg1, err)
					continue
				}
				obj.Container = container
				container.Contains = append(container.Contains, obj)
			}

		case "D": // Door state
			// TODO: Implement door state changes
			fmt.Printf("Door command for room %d, dir %d, state %d\n", cmd.Arg1, cmd.Arg2, cmd.Arg3)

		case "R": // Remove obj/mob from room
			if cmd.Arg3 == 1 { // Remove object
				s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
			} else { // Remove mob
				s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
			}
		}
	}

	return nil
}

// CanSpawn checks if we can spawn more of a given mob/obj based on maxInWorld.
func (s *Spawner) CanSpawn(vnum int, maxInWorld int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check mob instances
	if instances, ok := s.mobInstances[vnum]; ok {
		return len(instances) < maxInWorld
	}

	// Check object instances
	if instances, ok := s.objInstances[vnum]; ok {
		return len(instances) < maxInWorld
	}

	// No instances yet, so we can spawn
	return maxInWorld > 0
}



// SpawnMob creates a new mob instance in the specified room.
func (s *Spawner) SpawnMob(mobVNum, roomVNum int) (*MobInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use world's SpawnMob method
	mob, err := s.world.SpawnMob(mobVNum, roomVNum)
	if err != nil {
		return nil, err
	}

	// Track the spawned mob
	s.mobInstances[mobVNum] = append(s.mobInstances[mobVNum], mob)
	if roomVNum >= 0 {
		s.roomMobs[roomVNum] = append(s.roomMobs[roomVNum], mob)
	}

	return mob, nil
}

// SpawnObject creates a new object instance in the specified room or gives it to a mob.
func (s *Spawner) SpawnObject(objVNum, roomVNum int) (*ObjectInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use world's SpawnObject method
	obj, err := s.world.SpawnObject(objVNum, roomVNum)
	if err != nil {
		return nil, err
	}

	// Track the spawned object
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
	// TODO: Implement object tracking in world
	// For now, return empty slice
	return []*ObjectInstance{}
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
				// Remove from room
				s.roomObjects[roomVNum] = append(instances[:i], instances[i+1:]...)
				
				// Remove from objInstances
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
				// Remove from room
				s.roomMobs[roomVNum] = append(instances[:i], instances[i+1:]...)
				
				// Remove from mobInstances
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

	// This would iterate through zones and check if they're empty
	// For now, just a placeholder
	fmt.Println("Periodic zone reset check")
}