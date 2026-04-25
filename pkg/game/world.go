// Package game manages the game world state and player interactions.
package game

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/events"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// World represents the active game world with runtime state.
type World struct {
	mu sync.RWMutex

	// Snapshot manager for lock-free reads
	snapshots *SnapshotManager

	// Static world data (from parsed files)
	rooms      map[int]*parser.Room
	mobs       map[int]*parser.Mob
	objs       map[int]*parser.Obj
	zones      map[int]*parser.Zone
	parsedData *parser.World // original parsed data, nil after boot

	// Runtime state
	players    map[string]*Player   // keyed by player name
	activeMobs map[int]*MobInstance // keyed by instance ID
	nextMobID  int

	// Room items: room VNum -> list of object instances
	roomItems map[int][]*ObjectInstance
	nextObjID int

	// AI tick management
	aiticker *time.Ticker
	done     chan bool

	// Spawner
	spawner *Spawner

	// Shop manager
	shopManager common.ShopManager

	// Event queue for timer-based scripted events
	// Source: events.c event_init() — global event_q
	EventQueue *events.EventQueue

	// Events is the typed event bus for decoupled subsystem communication.
	Events events.Bus

	// Zone dispatcher for per-zone goroutine processing
	zoneDispatcher *ZoneDispatcher

	// Last tellers tracking for reply command
	lastTellers *lastTellersData

	// House control records — loaded by HouseBoot() during initialization
	HouseControl []HouseControl

	// Clans manager — loaded by InitClans() during initialization
	Clans *ClanManager

	// Boards system — initialized via GetOrInitBoards()
	Boards *BoardSystem
}

// NewWorld creates a new game world from parsed data.
func NewWorld(parsed *parser.World) (*World, error) {
	w := &World{
		rooms:       make(map[int]*parser.Room),
		mobs:        make(map[int]*parser.Mob),
		objs:        make(map[int]*parser.Obj),
		zones:       make(map[int]*parser.Zone),
		players:     make(map[string]*Player),
		activeMobs:  make(map[int]*MobInstance),
		nextMobID:   1,
		roomItems:   make(map[int][]*ObjectInstance),
		nextObjID:   1,
		done:        make(chan bool),
		shopManager: nil,    // Will be set via SetShopManager
		parsedData:  parsed, // Keep reference for door loading etc.
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

	// Initialize event queue
	// Source: events.c event_init() — called in init_game() before boot_db()
	// In original: 1 pulse = 1/10 second (OPT_USEC = 100000)
	w.EventQueue = events.NewEventQueue(100 * time.Millisecond)

	// Initialize typed event bus
	w.Events = events.NewInProcessBus()

	// Initialize zone dispatcher (per-zone goroutine processing)
	// Interval matches game pulse rate (~100ms)
	w.zoneDispatcher = NewZoneDispatcher(w, 100*time.Millisecond)

	// Start AI ticker
	w.StartAITicker()

	// Start point update ticker (regen + hunger/thirst) — limits.c point_update()
	// Called every ~30 seconds (Dark Pawns may have faster ticks than stock CircleMUD)
	w.StartPointUpdateTicker(30 * time.Second)

	// Initialize snapshot manager and publish initial snapshot
	w.snapshots = NewSnapshotManager()
	w.snapshots.Publish(w.rooms)

	// Initialize house control and board system
	w.HouseControl = make([]HouseControl, 0)

	return w, nil
}

// PostInit performs first-time initialization of systems that depend on
// the world being fully constructed (house boot, clan init).
// Must be called after NewWorld but before starting the main loop.
func (w *World) PostInit() {
	w.Clans = InitClans("./data/clans.json")
	w.HouseBoot()
}

// GetSnapshotManager returns the world's snapshot manager.
func (w *World) GetSnapshotManager() *SnapshotManager {
	return w.snapshots
}

// GetParsedWorld returns the original parsed world data used to create this world.
// Returns nil if the world was not created from parsed data.
func (w *World) GetParsedWorld() *parser.World {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.parsedData
}

// GetRoom returns a room by VNum.
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

// GetRoomInWorld returns a room by VNum, or nil if not found.
// Deprecated: use GetRoom (snapshot version) instead.
func (w *World) GetRoomInWorld(vnum int) *parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rooms[vnum]
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

// MovePlayer moves a player to a new room if the exit exists and doors permit.
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
	w.activeMobs[w.nextMobID] = mob
	w.nextMobID++

	// Notify players in the room — use internal unlocked access since we hold w.mu.Lock()
	for _, player := range w.players {
		if player.GetRoom() == roomVNum {
			player.Send <- []byte(fmt.Sprintf("%s appears.\n", mob.GetShortDesc()))
		}
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

// GetAllMobs returns all active mobs in the world.
func (w *World) GetAllMobs() []*MobInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()

	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
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

// GetShopManager returns the shop manager.
func (w *World) GetShopManager() common.ShopManager {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.shopManager
}

// SetShopManager sets the shop manager.
func (w *World) SetShopManager(manager common.ShopManager) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.shopManager = manager
}

// GetShopByKeeper returns a shop by keeper NPC VNum.
// Uses the concrete *ShopManager if available.
func (w *World) GetShopByKeeper(vnum int) (*Shop, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Try the concrete ShopManager first
	if sm, ok := w.shopManager.(*ShopManager); ok {
		shop := sm.GetShopByKeeper(vnum)
		return shop, shop != nil
	}

	return nil, false
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

// StartZoneDispatcher starts per-zone goroutines for resets and AI.
func (w *World) StartZoneDispatcher() {
	if w.zoneDispatcher != nil {
		w.zoneDispatcher.Start()
	}
}

// StopZoneDispatcher gracefully stops all zone goroutines.
func (w *World) StopZoneDispatcher() {
	if w.zoneDispatcher != nil {
		w.zoneDispatcher.Stop()
	}
}

// GetZoneDispatcher returns the zone dispatcher.
func (w *World) GetZoneDispatcher() *ZoneDispatcher {
	return w.zoneDispatcher
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
// If obj is a scriptableObjInstanceWrapper the live ObjectInstance is used directly;
// otherwise a new instance is created from the object prototype.
func (w *World) AddItemToRoomScriptable(obj scripting.ScriptableObject, roomVNum int) error {
	item := objectInstanceFromScriptable(obj)
	if item == nil {
		// Fallback: create a new instance from prototype
		proto, ok := w.GetObjPrototype(obj.GetVNum())
		if !ok {
			return fmt.Errorf("AddItemToRoom: prototype vnum %d not found", obj.GetVNum())
		}
		item = NewObjectInstance(proto, roomVNum)
	}
	item.RoomVNum = roomVNum
	w.AddItemToRoom(item, roomVNum)
	return nil
}

// HandleNonCombatDeathScriptable handles player death from non-combat damage.
func (w *World) HandleNonCombatDeathScriptable(player scripting.ScriptablePlayer) {
	slog.Debug("player would die from non-combat damage", "player", player.GetName())
	// TODO: Implement death handling
}

// HandleSpellDeathScriptable handles death caused by a spell.
func (w *World) HandleSpellDeathScriptable(victimName string, spellNum int, roomVNum int) {
	slog.Info("spell death", "victim", victimName, "spell_num", spellNum, "room_vnum", roomVNum)

	// First, check if it's a player
	if player, ok := w.GetPlayer(victimName); ok {
		// Player died to spell
		// We need to handle this as a combat death with spell attack type
		// For now, we'll call handlePlayerDeath directly
		// TODO: Find the actual killer (caster) if possible
		w.handlePlayerDeath(player, true, spellNum)
		return
	}

	// Check if it's a mob
	w.mu.RLock()
	for _, mob := range w.activeMobs {
		if mob.GetShortDesc() == victimName && mob.GetRoom() == roomVNum {
			w.mu.RUnlock()
			w.handleMobDeath(mob, spellNum)
			return
		}
	}
	w.mu.RUnlock()
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

func (a *WorldScriptableAdapter) HandleSpellDeath(victimName string, spellNum int, roomVNum int) {
	a.world.HandleSpellDeathScriptable(victimName, spellNum, roomVNum)
}

// SendTell delivers a private tell message to a named online player.
// Source: act.comm.c do_tell().
func (a *WorldScriptableAdapter) SendTell(targetName, message string) {
	if p, ok := a.world.GetPlayer(targetName); ok {
		p.SendMessage(message)
	}
}

func (a *WorldScriptableAdapter) GetItemsInRoom(roomVNum int) []scripting.ScriptableObject {
	return a.world.GetItemsInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) HasItemByVNum(charName string, vnum int) bool {
	return a.world.HasItemByVNumScriptable(charName, vnum)
}

func (a *WorldScriptableAdapter) RemoveItemFromRoom(vnum int, roomVNum int) scripting.ScriptableObject {
	return a.world.RemoveItemFromRoomByVNum(vnum, roomVNum)
}

func (a *WorldScriptableAdapter) RemoveItemFromChar(charName string, vnum int) scripting.ScriptableObject {
	return a.world.RemoveItemFromCharByVNum(charName, vnum)
}

func (a *WorldScriptableAdapter) GiveItemToChar(charName string, obj scripting.ScriptableObject) error {
	return a.world.GiveItemToCharScriptable(charName, obj)
}

// CreateEvent schedules a timed event on the world's event queue.
// Source: scripts.c lua_create_event() — create_event(source, target, obj, argument, trigger, delay, type)
func (a *WorldScriptableAdapter) CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
	if a.world.EventQueue == nil {
		slog.Error("cannot create event: EventQueue is nil")
		return 0
	}

	// In the original C code, delay is in PULSE_VIOLENCE units (2 seconds = 20 pulses).
	// The Lua scripts pass small integers like 1, 6, 10 meaning "N * PULSE_VIOLENCE".
	// We convert to pulses: 1 delay unit = 20 pulses = 2 seconds.
	// Source: scripts.c line 306: event->count = PULSE_VIOLENCE * time
	// Source: structs.h: PULSE_VIOLENCE = (2 RL_SEC) = 20 pulses
	pulseDelay := int64(delay) * 20

	return a.world.EventQueue.Create(pulseDelay, source, target, obj, argument, trigger, eventType,
		func(ctx context.Context, src, tgt, o, arg int, trig string, et int) int64 {
			// When the event fires, dispatch the Lua trigger on the mob.
			// Source: events.c event_process() — calls the_event->func(event_obj)
			// The original lua_create_event stored a script_event struct with
			// me, obj, room, fname, type and called run_script() when fired.
			a.world.dispatchScriptEvent(src, tgt, o, arg, trig, et)
			return 0
		})
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

// scriptableObjInstanceWrapper wraps ObjectInstance to implement ScriptableObject.
// Used by item-transfer Lua functions (objfrom/objto) to carry live instances.
type scriptableObjInstanceWrapper struct {
	item *ObjectInstance
}

func (w *scriptableObjInstanceWrapper) GetVNum() int         { return w.item.VNum }
func (w *scriptableObjInstanceWrapper) GetKeywords() string  { return w.item.GetKeywords() }
func (w *scriptableObjInstanceWrapper) GetShortDesc() string { return w.item.GetShortDesc() }
func (w *scriptableObjInstanceWrapper) GetCost() int         { return w.item.GetCost() }
func (w *scriptableObjInstanceWrapper) GetTimer() int        { return w.item.GetTimer() }
func (w *scriptableObjInstanceWrapper) SetTimer(t int)       { w.item.SetTimer(t) }

// objectInstanceFromScriptable extracts the underlying ObjectInstance from a
// scriptableObjInstanceWrapper, returning nil for other ScriptableObject types.
func objectInstanceFromScriptable(obj scripting.ScriptableObject) *ObjectInstance {
	if w, ok := obj.(*scriptableObjInstanceWrapper); ok {
		return w.item
	}
	return nil
}

// GetItemsInRoomScriptable returns all items in a room as []ScriptableObject.
func (w *World) GetItemsInRoomScriptable(roomVNum int) []scripting.ScriptableObject {
	w.mu.RLock()
	defer w.mu.RUnlock()
	items := w.roomItems[roomVNum]
	result := make([]scripting.ScriptableObject, 0, len(items))
	for _, item := range items {
		result = append(result, &scriptableObjInstanceWrapper{item: item})
	}
	return result
}

// HasItemByVNumScriptable returns true if the named player has an item with the given vnum.
func (w *World) HasItemByVNumScriptable(charName string, vnum int) bool {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return false
	}
	p.Inventory.mu.RLock()
	defer p.Inventory.mu.RUnlock()
	for _, item := range p.Inventory.Items {
		if item.VNum == vnum {
			return true
		}
	}
	return false
}

// RemoveItemFromRoomByVNum removes the first item with the given vnum from the room.
// Returns the removed item as ScriptableObject, or nil if not found.
func (w *World) RemoveItemFromRoomByVNum(vnum int, roomVNum int) scripting.ScriptableObject {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := w.roomItems[roomVNum]
	for i, item := range items {
		if item.VNum == vnum {
			w.roomItems[roomVNum] = append(items[:i], items[i+1:]...)
			item.RoomVNum = -1
			return &scriptableObjInstanceWrapper{item: item}
		}
	}
	return nil
}

// RemoveItemFromCharByVNum removes the first item with the given vnum from the named player.
// Returns the removed item as ScriptableObject, or nil if not found.
func (w *World) RemoveItemFromCharByVNum(charName string, vnum int) scripting.ScriptableObject {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return nil
	}
	item, found := p.Inventory.RemoveItemByVNum(vnum)
	if !found {
		return nil
	}
	return &scriptableObjInstanceWrapper{item: item}
}

// GiveItemToCharScriptable adds a ScriptableObject to the named player's inventory.
func (w *World) GiveItemToCharScriptable(charName string, obj scripting.ScriptableObject) error {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return fmt.Errorf("player %q not found", charName)
	}
	item := objectInstanceFromScriptable(obj)
	if item == nil {
		return fmt.Errorf("GiveItemToChar: object is not an ObjectInstance")
	}
	return p.Inventory.AddItem(item)
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

// dispatchScriptEvent dispatches a Lua trigger when a scheduled event fires.
// This is the callback registered with EventQueue.Create() for script events.
// Based on the original lua_create_event() in scripts.c lines 247-316.
//
// The original stored: me, ch, obj, room, fname (trigger), type
// and called run_script() when the event fired.
func (w *World) dispatchScriptEvent(source, target, objVNum, argument int, trigger string, eventType int) {
	if ScriptEngine == nil {
		return
	}

	// Find the mob by instance ID (source)
	w.mu.RLock()
	var mob *MobInstance
	if source > 0 {
		mob = w.activeMobs[source]
	}
	w.mu.RUnlock()

	if mob == nil {
		// Mob may have died or been extracted — event is a no-op
		// This matches original behavior where extract_char cleans up events
		return
	}

	// Build script context
	ctx := mob.CreateScriptContext(nil, nil, "")
	ctx.World = NewWorldScriptableAdapter(w)
	ctx.RoomVNum = mob.GetRoom()

	// If target is a player name hash/ID, try to resolve it
	// In the original, target was a char_data pointer. In our Go version,
	// we store the target as an int (could be player ID or mob ID).
	// For now, we only support mob-source events firing on themselves.
	// TODO: Resolve target player by ID when target > 0 and target != source

	// Run the trigger function in the mob's script
	// The trigger name is the Lua function to call (e.g., "port", "jail", "bane_one")
	if mob.HasScript(trigger) {
		if _, err := mob.RunScript(trigger, ctx); err != nil {
			slog.Error("script error", "mob_vnum", mob.GetVNum(), "trigger", trigger, "error", err)
		}
	} else {
		// The mob's prototype may not have the trigger bit set, but the
		// function might still exist in the Lua file — try anyway
		if _, err := ScriptEngine.RunScript(ctx, mob.Prototype.ScriptName, trigger); err != nil {
			slog.Error("script error", "mob_vnum", mob.GetVNum(), "trigger", trigger, "error", err)
		}
	}
}
