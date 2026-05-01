// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/events"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// MessageSinkFunc is the callback type for delivering messages to a player.
// The session manager sets this on World initialization so that game-layer
// SendMessage calls route through Session.send (which writePump reads).
type MessageSinkFunc func(playerName string, msg []byte)

// CloseConnectionFunc is called when a player session should be forcibly closed
// (e.g., do_quit). Set by the session manager.
type CloseConnectionFunc func(playerName string)

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

	// World path for reload support
	WorldPath  string

	// Runtime state
	players    map[string]*Player   // keyed by player name
	activeMobs map[int]*MobInstance // keyed by instance ID
	nextMobID  int

	// Room items: room VNum -> list of object instances
	roomItems map[int][]*ObjectInstance
	nextObjID int

	// All live object instances: instance ID -> ObjectInstance
	objectInstances map[int]*ObjectInstance

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

	// Bans — site ban list + invalid name filter (ported from ban.c)
	Bans *BanManager

	// Whod — who-daemon display mode flags (ported from whod.c)
	WhodDisplay *Whod

	// MessageSink routes player messages through the session layer.
	// Set by the session manager on initialization. If nil, messages are silently dropped.
	MessageSink MessageSinkFunc

	// CloseConnection routes close requests through the session layer.
	CloseConn CloseConnectionFunc
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
		roomItems:        make(map[int][]*ObjectInstance),
		nextObjID:         1,
		objectInstances:  make(map[int]*ObjectInstance),
		done:        make(chan bool),
		shopManager: nil,    // Will be set via SetShopManager
		parsedData:  parsed, // Keep reference for door loading etc.
		WorldPath:   "", // Set externally for reload support
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

	// Initialize ban manager and WHOD display (ported from ban.c + whod.c)
	w.Bans = NewBanManager()
	w.WhodDisplay = NewWhod()

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

// ReplaceParsedWorld swaps the in-memory world data with a fresh parse.
// Used by the reload wizard command.
func (w *World) ReplaceParsedWorld(pw *parser.World) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.parsedData = pw
	w.rooms = make(map[int]*parser.Room)
	for i := range pw.Rooms {
		w.rooms[pw.Rooms[i].VNum] = &pw.Rooms[i]
	}
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

	p.mu.Lock()
	p.worldRef = w
	p.mu.Unlock()

	w.players[p.Name] = p
	return nil
}

// RemovePlayer removes a player from the world.
func (w *World) RemovePlayer(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.players, name)
}

// ForEachPlayer calls fn for each player in the world. Thread-safe.
func (w *World) ForEachPlayer(fn func(p *Player)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range w.players {
		fn(p)
	}
}

// ForEachPlayerInZone calls fn for each player in the given zone. Thread-safe.
func (w *World) ForEachPlayerInZone(zoneNum int, fn func(p *Player)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range w.players {
		roomVNum := p.GetRoom()
		zone := w.GetRoomZone(roomVNum)
		if zone == zoneNum {
			fn(p)
		}
	}
}

// ForEachPlayerInZoneInterface is like ForEachPlayerInZone but accepts interface{} callback
// for use from packages that can't import game types (e.g., spells).
func (w *World) ForEachPlayerInZoneInterface(zoneNum int, fn func(p interface{})) {
	w.ForEachPlayerInZone(zoneNum, func(p *Player) { fn(p) })
}

// ForEachPlayerInRoomInterface iterates players in a room with interface{} callback.
// For use from packages that can't import game types (e.g., spells).
func (w *World) ForEachPlayerInRoomInterface(roomVNum int, fn func(p interface{})) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range w.players {
		if p.GetRoom() == roomVNum {
			fn(p)
		}
	}
}

// ForEachMobInRoomInterface iterates mobs in a room with interface{} callback.
// For use from packages that can't import game types (e.g., spells).
func (w *World) ForEachMobInRoomInterface(roomVNum int, fn func(m interface{})) {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		fn(m)
	}
}

// GetRoomInWorld returns a room by VNum, or nil if not found.
// Deprecated: use GetRoom (snapshot version) instead.
func (w *World) GetRoomInWorld(vnum int) *parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rooms[vnum]
}

// Rooms returns all rooms in the world.
// GetRoomCount returns the total number of rooms in the world.
// Equivalent to top_of_world in C.
func (w *World) GetRoomCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.rooms)
}

func (w *World) Rooms() []parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var result []parser.Room
	for _, r := range w.rooms {
		result = append(result, *r)
	}
	return result
}

// sendToZone sends a message to all players in the same zone as the given room.
func (w *World) SendToZone(roomVNum int, msg string) {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return
	}
	zone := room.Zone

	// H-07: Acquire read lock before iterating w.players map.
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	w.mu.RUnlock()

	for _, p := range players {
		pr := w.GetRoomInWorld(p.RoomVNum)
		if pr != nil && pr.Zone == zone {
			p.SendMessage(msg)
		}
	}
}

// sendToAll sends a message to all online players.
// Source: comm.c send_to_all().
func (w *World) SendToAll(msg string) {
	if msg == "" {
		return
	}

	// H-07: Acquire read lock before iterating w.players map.
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	w.mu.RUnlock()

	for _, p := range players {
		p.SendMessage(msg)
	}
}

// executeMobCommand makes a mob execute a game command.
// Source: scripts.c lua_action() → command_interpreter().
// Mobs don't have sessions, so we log the command for now.
// When mob script task system is implemented, this should dispatch through
// a proper task queue instead of inline execution.
func (w *World) executeMobCommand(mobVNum int, cmdStr string) {
	w.mu.RLock()
	var mob *MobInstance
	for _, m := range w.activeMobs {
		if m.GetVNum() == mobVNum {
			mob = m
			break
		}
	}
	w.mu.RUnlock()

	if mob == nil {
		slog.Debug("executeMobCommand: mob not found", "vnum", mobVNum, "command", cmdStr)
		return
	}

	slog.Debug("mob executes command", "mob_vnum", mobVNum, "mob_name", mob.GetName(), "command", cmdStr)
	// The actual command handler will parse cmdStr and route to game commands
	// (mobSay, mobEmote, mobForce, mobDamage, etc.)
}

// IsRoomDark returns true if the given room VNum is dark.
// Based on utils.h IS_DARK() macro — checks ROOM_DARK flag.
func (w *World) IsRoomDark(roomVNum int) bool {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}
	// Check ROOM_DARK flag (bit 0)
	return room.HasFlag(0)
}

// GetRoomZone returns the zone number for a given room VNum.
func (w *World) GetRoomZone(roomVNum int) int {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return -1
	}
	return room.Zone
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
	// H-11: Split into two phases to avoid blocking SendMessage while holding the write lock.

	// Phase 1: Create mob under write lock.
	w.mu.Lock()
	proto, ok := w.mobs[vnum]
	if !ok {
		w.mu.Unlock()
		return nil, fmt.Errorf("mob prototype %d not found", vnum)
	}

	_, ok = w.rooms[roomVNum]
	if !ok {
		w.mu.Unlock()
		return nil, fmt.Errorf("room %d not found", roomVNum)
	}

	mob := NewMob(proto, roomVNum)
	mob.ID = w.nextMobID
	w.activeMobs[w.nextMobID] = mob
	w.nextMobID++

	// Copy players in the room while holding the lock.
	var targets []*Player
	for _, player := range w.players {
		if player.GetRoom() == roomVNum {
			targets = append(targets, player)
		}
	}
	w.mu.Unlock()

	// Phase 2: Notify outside the lock (SendMessage may block on channel buffer).
	for _, player := range targets {
		player.SendMessage(fmt.Sprintf("%s appears.\n", mob.GetShortDesc()))
	}

	return mob, nil
}

// SpawnMobInstance is an alias for SpawnMob for compatibility.
func (w *World) SpawnMobInstance(vnum int, roomVNum int) (*MobInstance, error) {
	return w.SpawnMob(vnum, roomVNum)
}

// SpawnMobWithLevelI creates a mob with overridden level, returns interface{} for spell layer.
func (w *World) SpawnMobWithLevelI(vnum int, roomVNum int, level int) (interface{}, error) {
	mob, err := w.SpawnMob(vnum, roomVNum)
	if err != nil {
		return nil, err
	}
	mob.SetLevel(level)
	return mob, nil
}

// LookAtRoomSimple sends a basic room description to a player via interface{}.
// Used by mindsight which temporarily transfers the caster to another room.
func (w *World) LookAtRoomSimple(roomVNum int, sender interface{}) {
	sm, ok := sender.(interface{ SendMessage(string) })
	if !ok {
		return
	}

	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		sm.SendMessage("You see nothing but void.\r\n")
		return
	}

	sm.SendMessage(fmt.Sprintf("%s\r\n", room.Name))
	if room.Description != "" {
		sm.SendMessage(room.Description + "\r\n")
	}

	// List characters in room
	for _, m := range w.GetMobsInRoom(roomVNum) {
		sm.SendMessage(fmt.Sprintf("%s is here.\r\n", m.GetShortDesc()))
	}
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if pn, ok := sender.(interface{ GetName() string }); ok {
			if pn.GetName() != p.GetName() {
				sm.SendMessage(fmt.Sprintf("%s is here.\r\n", p.GetName()))
			}
		}
	}
}

// extractMob removes a mob instance from the world (extract_char equivalent).
func (w *World) ExtractMob(mob *MobInstance) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for id, m := range w.activeMobs {
		if m == mob {
			delete(w.activeMobs, id)
			break
		}
	}
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
	obj.ID = w.nextObjID
	w.nextObjID++
	obj.Location = LocRoom(roomVNum)
	w.objectInstances[obj.ID] = obj
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

// GetMobByID returns a mob instance by its world-assigned ID.
func (w *World) GetMobByID(id int) (*MobInstance, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	mob, ok := w.activeMobs[id]
	return mob, ok
}

// GetMobByName finds an active mob by partial name match (case-insensitive).
func (w *World) GetMobByName(name string) *MobInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	nameLower := strings.ToLower(name)
	for _, mob := range w.activeMobs {
		if strings.Contains(strings.ToLower(mob.GetName()), nameLower) {
			return mob
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

// CharTransfer moves a character (player or mob) from one room to another.
// This is the Go equivalent of C's char_from_room + char_to_room.
// It stops fighting if the target is in a different room, and moves mounts with riders.
// Returns an error if the target room doesn't exist.
// Source: src/handler.c char_from_room/char_to_room
