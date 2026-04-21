// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
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

// GetMobsInRoomScriptable returns mobs in a room as ScriptableMob slice.
func (w *World) GetMobsInRoomScriptable(roomVNum int) []scripting.ScriptableMob {
	mobs := w.GetMobsInRoom(roomVNum)
	out := make([]scripting.ScriptableMob, 0, len(mobs))
	for _, m := range mobs {
		out = append(out, m)
	}
	return out
}

// GetMobByVNumAndRoomScriptable returns a mob by vnum+room as ScriptableMob.
func (w *World) GetMobByVNumAndRoomScriptable(vnum int, roomVNum int) scripting.ScriptableMob {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if m.GetVNum() == vnum {
			return m
		}
	}
	return nil
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

// GiveStartingItems implements do_start() item distribution from class.c lines 506-532.
// Creates ObjectInstance items from prototypes and adds them to player inventory.
// Source: class.c do_start()
func (w *World) GiveStartingItems(p *Player) {
	// Pack (8038) is created first, filled with bread (8010) + waterskin (8063)
	// then given to player
	
	packProto, packOK := w.GetObjPrototype(8038)
	
	// Class-specific items (given directly to player)
	switch p.Class {
	case ClassThief, ClassAssassin:
		w.giveItem(p, 8036) // dagger
		if packOK {
			// lockpicks (8027) go INTO the pack — handled after pack creation
			_ = packProto // suppress unused warning, used below
		}
	case ClassMageUser, ClassMagus:
		w.giveItem(p, 8036) // dagger
		w.giveItem(p, 1239) // obsidian
		w.giveItem(p, 1239) // obsidian (2x)
	case ClassNinja:
		w.giveItem(p, 8036) // dagger
	case ClassWarrior, ClassPsionic:
		w.giveItem(p, 8037) // small sword
	default:
		w.giveItem(p, 8023) // club
	}
	
	w.giveItem(p, 8019) // tunic (all classes)
	
	// Create pack and fill it
	if packOK {
		pack := NewObjectInstance(packProto, -1)
		pack.Contains = make([]*ObjectInstance, 0)
		
		// bread + waterskin always in pack
		if bread, ok := w.GetObjPrototype(8010); ok {
			pack.Contains = append(pack.Contains, NewObjectInstance(bread, -1))
		}
		if water, ok := w.GetObjPrototype(8063); ok {
			pack.Contains = append(pack.Contains, NewObjectInstance(water, -1))
		}
		// lockpicks in pack for thieves/assassins
		if p.Class == ClassThief || p.Class == ClassAssassin {
			if picks, ok := w.GetObjPrototype(8027); ok {
				pack.Contains = append(pack.Contains, NewObjectInstance(picks, -1))
			}
		}
		
		_ = p.Inventory.AddItem(pack)
	}
}

// giveItem creates an ObjectInstance from a prototype vnum and adds it to player inventory.
func (w *World) giveItem(p *Player, vnum int) {
	proto, ok := w.GetObjPrototype(vnum)
	if !ok {
		return
	}
	obj := NewObjectInstance(proto, -1)
	_ = p.Inventory.AddItem(obj)
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

// ScriptableWorld interface implementation

// GetPlayersInRoomScriptable returns all players in a given room as ScriptablePlayer.
func (w *World) GetPlayersInRoomScriptable(roomVNum int) []scripting.ScriptablePlayer {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var players []scripting.ScriptablePlayer
	for _, p := range w.players {
		if p.RoomVNum == roomVNum {
			players = append(players, p)
		}
	}
	return players
}



// GetObjPrototypeScriptable returns an object prototype by vnum as ScriptableObject.
func (w *World) GetObjPrototypeScriptable(vnum int) scripting.ScriptableObject {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	obj, ok := w.objs[vnum]
	if !ok {
		return nil
	}
	// Create a wrapper that implements ScriptableObject
	return &scriptableObjWrapper{obj: obj}
}

// AddItemToRoomScriptable adds an item to a room.
func (w *World) AddItemToRoomScriptable(obj scripting.ScriptableObject, roomVNum int) error {
	// For now, just log - we'd need to convert ScriptableObject to ObjectInstance
	log.Printf("[SCRIPT] Would add object vnum %d to room %d", obj.GetVNum(), roomVNum)
	return nil
}

// HandleNonCombatDeathScriptable handles player death from non-combat damage.
func (w *World) HandleNonCombatDeathScriptable(player scripting.ScriptablePlayer) {
	log.Printf("[SCRIPT] Player %s would die from non-combat damage", player.GetName())
	// TODO: Implement death handling
}

// ScriptableWorld interface implementation via adapter

// WorldScriptableAdapter adapts World to ScriptableWorld
type WorldScriptableAdapter struct {
	world *World
}

// NewWorldScriptableAdapter creates a WorldScriptableAdapter.
func NewWorldScriptableAdapter(w *World) *WorldScriptableAdapter {
	return &WorldScriptableAdapter{world: w}
}

func (a *WorldScriptableAdapter) GetPlayersInRoom(roomVNum int) []scripting.ScriptablePlayer {
	return a.world.GetPlayersInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) GetMobsInRoom(roomVNum int) []scripting.ScriptableMob {
	return a.world.GetMobsInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) GetMobByVNumAndRoom(vnum int, roomVNum int) scripting.ScriptableMob {
	return a.world.GetMobByVNumAndRoomScriptable(vnum, roomVNum)
}

func (a *WorldScriptableAdapter) GetObjPrototype(vnum int) scripting.ScriptableObject {
	return a.world.GetObjPrototypeScriptable(vnum)
}

func (a *WorldScriptableAdapter) AddItemToRoom(obj scripting.ScriptableObject, roomVNum int) error {
	return a.world.AddItemToRoomScriptable(obj, roomVNum)
}

func (a *WorldScriptableAdapter) HandleNonCombatDeath(player scripting.ScriptablePlayer) {
	// ScriptablePlayer is an interface — find the actual Player by name
	if p, ok := a.world.GetPlayer(player.GetName()); ok {
		a.world.HandleNonCombatDeath(p)
	}
}

// scriptableObjWrapper wraps parser.Obj to implement ScriptableObject
type scriptableObjWrapper struct {
	obj *parser.Obj
}

func (w *scriptableObjWrapper) GetVNum() int {
	return w.obj.VNum
}

func (w *scriptableObjWrapper) GetKeywords() string {
	return w.obj.Keywords
}

func (w *scriptableObjWrapper) GetShortDesc() string {
	return w.obj.ShortDesc
}

func (w *scriptableObjWrapper) GetCost() int {
	return w.obj.Cost
}

func (w *scriptableObjWrapper) GetTimer() int {
	return 0 // parser.Obj has no Timer field — runtime timer tracked on ObjectInstance
}

func (w *scriptableObjWrapper) SetTimer(timer int) {
	// TODO: timer mutation on parser.Obj not supported — Phase 3 tracks timer on ObjectInstance
}
// FireMobFightScript fires the "fight" trigger on a mob after a combat round.
// Called by the combat engine's ScriptFightFunc after each round.
// Source: mobact.c — mob_activity() calls Lua fight trigger during violence.
func (w *World) FireMobFightScript(mobName string, targetName string, roomVNum int) {
	if ScriptEngine == nil {
		return
	}

	// Find the mob by name in the room
	w.mu.RLock()
	var mob *MobInstance
	for _, m := range w.activeMobs {
		if m.GetRoom() == roomVNum && m.GetName() == mobName && m.HasScript("fight") {
			mob = m
			break
		}
	}
	// Find the target player
	var target scripting.ScriptablePlayer
	for _, p := range w.players {
		if p.GetName() == targetName {
			target = p
			break
		}
	}
	w.mu.RUnlock()

	if mob == nil {
		return
	}

	ctx := mob.CreateScriptContext(nil, nil, "")
	if target != nil {
		if p, ok := target.(*Player); ok {
			ctx.Ch = p
		}
	}
	ctx.World = NewWorldScriptableAdapter(w)
	ctx.RoomVNum = roomVNum

	if _, err := mob.RunScript("fight", ctx); err != nil {
		// Script errors are non-fatal — log and continue
		_ = err
	}
}
