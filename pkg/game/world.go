// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"sync"
	"time"


	"github.com/zax0rz/darkpawns/pkg/parser"
)

// World represents the active game world with runtime state.
type World struct {
	mu sync.RWMutex

	// Static world data (from parsed files)
	rooms map[int]*parser.Room
	mobs  map[int]*parser.Mob
	objs  map[int]*parser.Obj
	zones map[int]*parser.Zone

	// Runtime state
	players map[string]*Player // keyed by player name
	activeMobs map[int]*MobInstance    // keyed by instance ID
	nextMobID  int
	
	// Room items: room VNum -> list of object instances
	roomItems map[int][]*ObjectInstance
	nextObjID int
	
	// AI tick management
	aiticker *time.Ticker
	done     chan bool
	
	// Spawner
	spawner *Spawner
}

// NewWorld creates a new game world from parsed data.
func NewWorld(parsed *parser.World) (*World, error) {
	w := &World{
		rooms:      make(map[int]*parser.Room),
		mobs:       make(map[int]*parser.Mob),
		objs:       make(map[int]*parser.Obj),
		zones:      make(map[int]*parser.Zone),
		players:    make(map[string]*Player),
		activeMobs: make(map[int]*MobInstance),
		nextMobID:  1,
		roomItems:  make(map[int][]*ObjectInstance),
		nextObjID:   1,
		done:       make(chan bool),
	}

	// Index rooms by VNum
	for i := range parsed.Rooms {
		room := &parsed.Rooms[i]
		w.rooms[room.VNum] = room
	}

	// Index mobs by VNum
	for i := range parsed.Mobs {
		mob := &parsed.Mobs[i]
		w.mobs[mob.VNum] = mob
	}

	// Index objects by VNum
	for i := range parsed.Objs {
		obj := &parsed.Objs[i]
		w.objs[obj.VNum] = obj
	}

	// Index zones by number
	for i := range parsed.Zones {
		zone := &parsed.Zones[i]
		w.zones[zone.Number] = zone
	}

	// Start AI ticker
	w.StartAITicker()

	return w, nil
}

// GetRoom returns a room by VNum.
func (w *World) GetRoom(vnum int) (*parser.Room, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	room, ok := w.rooms[vnum]
	return room, ok
}

// GetPlayer returns a player by name.
func (w *World) GetPlayer(name string) (*Player, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	p, ok := w.players[name]
	return p, ok
}

// AddPlayer adds a player to the world.
func (w *World) AddPlayer(p *Player) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.players[p.Name]; exists {
		return fmt.Errorf("player %s already online", p.Name)
	}

	w.players[p.Name] = p
	return nil
}

// RemovePlayer removes a player from the world.
func (w *World) RemovePlayer(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.players, name)
}

// GetPlayersInRoom returns all players in a given room.
func (w *World) GetPlayersInRoom(roomVNum int) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var players []*Player
	for _, p := range w.players {
		if p.RoomVNum == roomVNum {
			players = append(players, p)
		}
	}
	return players
}

// MovePlayer moves a player to a new room if the exit exists.
func (w *World) MovePlayer(p *Player, direction string) (*parser.Room, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	currentRoom, ok := w.rooms[p.RoomVNum]
	if !ok {
		return nil, fmt.Errorf("player in invalid room %d", p.RoomVNum)
	}

	exit, ok := currentRoom.Exits[direction]
	if !ok {
		return nil, fmt.Errorf("no exit %s", direction)
	}

	newRoom, ok := w.rooms[exit.ToRoom]
	if !ok {
		return nil, fmt.Errorf("exit leads to invalid room %d", exit.ToRoom)
	}

	p.RoomVNum = newRoom.VNum
	return newRoom, nil
}

// StopAITicker stops the AI tick loop.
func (w *World) StopAITicker() {
	if w.done != nil {
		close(w.done)
	}
}

// SpawnMob spawns a mob in the world.
func (w *World) SpawnMob(vnum int, roomVNum int) (*MobInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	proto, ok := w.mobs[vnum]
	if !ok {
		return nil, fmt.Errorf("mob prototype %d not found", vnum)
	}

	_, ok = w.rooms[roomVNum]
	if !ok {
		return nil, fmt.Errorf("room %d not found", roomVNum)
	}

	mob := NewMob(proto, roomVNum)
	// In a real implementation, we'd assign a unique instance ID
	// For now, we'll use the nextMobID
	w.activeMobs[w.nextMobID] = mob
	w.nextMobID++

	// Notify players in the room
	players := w.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.Send <- []byte(fmt.Sprintf("%s appears.\n", mob.GetShortDesc()))
	}

	return mob, nil
}

// SpawnMobInstance is an alias for SpawnMob for compatibility.
func (w *World) SpawnMobInstance(vnum int, roomVNum int) (*MobInstance, error) {
	return w.SpawnMob(vnum, roomVNum)
}

// SpawnObject spawns an object in the specified room.
func (w *World) SpawnObject(objVNum, roomVNum int) (*ObjectInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	proto, ok := w.objs[objVNum]
	if !ok {
		return nil, fmt.Errorf("object prototype %d not found", objVNum)
	}

	obj := NewObjectInstance(proto, roomVNum)
	// TODO: Track object instances in world
	return obj, nil
}

// GetMobsInRoom returns all mobs in a given room.
func (w *World) GetMobsInRoom(roomVNum int) []*MobInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var mobs []*MobInstance
	for _, mob := range w.activeMobs {
		if mob.GetRoom() == roomVNum {
			mobs = append(mobs, mob)
		}
	}
	return mobs
}

// GetItemsInRoom returns all items in a given room.
func (w *World) GetItemsInRoom(roomVNum int) []*ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.roomItems[roomVNum]
}

// AddItemToRoom adds an item to a room.
func (w *World) AddItemToRoom(item *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roomItems[roomVNum] = append(w.roomItems[roomVNum], item)
}

// RemoveItemFromRoom removes an item from a room.
func (w *World) RemoveItemFromRoom(item *ObjectInstance, roomVNum int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	items := w.roomItems[roomVNum]
	for i, it := range items {
		if it == item {
			w.roomItems[roomVNum] = append(items[:i], items[i+1:]...)
			return true
		}
	}
	return false
}

// GetMobPrototype returns a mob prototype by VNum.
func (w *World) GetMobPrototype(vnum int) (*parser.Mob, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	mob, ok := w.mobs[vnum]
	return mob, ok
}

// GetObjPrototype returns an object prototype by VNum.
func (w *World) GetObjPrototype(vnum int) (*parser.Obj, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	obj, ok := w.objs[vnum]
	return obj, ok
}

// GetZone returns a zone by number.
func (w *World) GetZone(number int) (*parser.Zone, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zone, ok := w.zones[number]
	return zone, ok
}

// GetAllZones returns all zones.
func (w *World) GetAllZones() []*parser.Zone {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zones := make([]*parser.Zone, 0, len(w.zones))
	for _, zone := range w.zones {
		zones = append(zones, zone)
	}
	return zones
}

// StartZoneResets starts all zone resets.
func (w *World) StartZoneResets() error {
	if w.spawner == nil {
		w.spawner = NewSpawner(w)
	}
	
	zones := w.GetAllZones()
	for _, zone := range zones {
		if err := w.spawner.ExecuteZoneReset(zone); err != nil {
			return fmt.Errorf("zone %d reset failed: %w", zone.Number, err)
		}
	}
	return nil
}

// StartPeriodicResets starts periodic zone reset checks.
func (w *World) StartPeriodicResets(interval time.Duration) {
	if w.spawner == nil {
		w.spawner = NewSpawner(w)
	}
	w.spawner.StartPeriodicResets(interval)
}

// GetSpawner returns the world's spawner.
func (w *World) GetSpawner() *Spawner {
	return w.spawner
}

// OnPlayerEnterRoom handles player entering a room (for aggressive mobs).
// Returns true if combat was initiated.
func (w *World) OnPlayerEnterRoom(player *Player, roomVNum int, ce CombatEngine) bool {
	mobs := w.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		// Check if mob is aggressive
		isAggressive := false
		if mob.Prototype != nil {
			for _, flag := range mob.Prototype.ActionFlags {
				if flag == "aggressive" {
					isAggressive = true
					break
				}
			}
		}

		if isAggressive && !player.IsFighting() {
			// Check if mob is already fighting
			if !ce.IsFighting(mob.GetName()) {
				go func(m *MobInstance) {
					if err := ce.StartCombat(m, player); err != nil {
						// Combat might fail if already fighting, that's ok
					}
				}(mob)
				return true
			}
		}
	}
	return false
}

// Stats returns world statistics.
func (w *World) Stats() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return fmt.Sprintf(
		"World: %d rooms, %d mobs (%d active), %d objects, %d zones, %d players online",
		len(w.rooms), len(w.mobs), len(w.activeMobs), len(w.objs), len(w.zones), len(w.players),
	)
}