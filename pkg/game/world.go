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

	// lastTellers tracks last tell recipients per character ID.
	lastTellers *lastTellersData //nolint:unused // used via methods in act_comm.go

	// gossipHistory records the last 25 gossip messages for the review command.
	// Matches C: struct review_t review[25] in db.c.
	gossipMu      sync.RWMutex
	gossipHistory []gossipEntry
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

// GetPlayerByID finds a player by their instance ID.
func (w *World) GetPlayerByID(id int) *Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range w.players {
		if p.GetID() == id {
			return p
		}
	}
	return nil
}

// SetObjectExtraDesc stores a runtime extra description on an object instance
// matching the given vnum. The extra desc is stored in the ObjectInstance's
// CustomData so it persists for the lifetime of the instance and is picked up
// by GetExtraDescs() and GetExtraDesc().
func (w *World) SetObjectExtraDesc(vnum int, keyword string, description string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, obj := range w.objectInstances {
		if obj.VNum == vnum {
			// Get existing runtime extra descs from CustomData
			var descs []parser.ExtraDesc
			if raw, ok := obj.CustomData["extra_descs"]; ok {
				descs, _ = raw.([]parser.ExtraDesc)
			}
			if descs == nil {
				descs = make([]parser.ExtraDesc, 0)
			}
			descs = append(descs, parser.ExtraDesc{
				Keywords:    keyword,
				Description: description,
			})
			obj.SetCustomData("extra_descs", descs)
			return true
		}
	}
	return false
}

// SetObjectExtraFlag sets or removes an extra flag on the first object instance
// matching the given vnum.
func (w *World) SetObjectExtraFlag(vnum int, flag int, set bool) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	word := flag / 32
	bit := flag % 32

	for _, obj := range w.objectInstances {
		if obj.VNum == vnum {
			if set {
				obj.SetExtraFlag(word, bit)
			} else {
				obj.RemoveExtraFlag(word, bit)
			}
			return true
		}
	}
	return false
}

// SetExitDoorState sets the door state for an exit in a room.
func (w *World) SetExitDoorState(roomVNum int, direction string, state int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	room, ok := w.rooms[roomVNum]
	if !ok {
		return false
	}
	exit, ok := room.Exits[direction]
	if !ok {
		return false
	}
	exit.DoorState = state
	room.Exits[direction] = exit
	return true
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

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	switch cmd {
	case "say":
		w.mobSayTo(mob, args)

	case "emote":
		// emote broadcasts to room: "<name> <message>"
		msg := fmt.Sprintf("%s %s", mob.GetName(), args)
		for _, p := range w.GetPlayersInRoom(mob.GetRoom()) {
			p.SendMessage(msg)
		}

	case "gossip":
		w.mobGossip(mob, args)

	case "tell":
		tellParts := strings.SplitN(args, " ", 2)
		if len(tellParts) == 2 {
			w.mobTellPlayer(mob, tellParts[0], tellParts[1])
		} else {
			slog.Debug("executeMobCommand: tell missing target or message", "args", args)
		}

	case "shout":
		w.mobShout(mob, args)

	case "auction":
		w.mobAuction(mob, args)

	case "north", "east", "south", "west", "up", "down":
		dirMap := map[string]int{"north": 0, "east": 1, "south": 2, "west": 3, "up": 4, "down": 5}
		w.mobPerformMove(mob, dirMap[cmd])

	case "follow":
		mob.SetFollowing(args)

	case "open":
		openParts := strings.Fields(args)
		if len(openParts) > 0 {
			dirMap := map[string]int{"north": 0, "east": 1, "south": 2, "west": 3, "up": 4, "down": 5,
				"n": 0, "e": 1, "s": 2, "w": 3, "u": 4, "d": 5}
			if dir, ok := dirMap[strings.ToLower(openParts[0])]; ok {
				keyword := ""
				if len(openParts) > 1 {
					keyword = strings.Join(openParts[1:], " ")
				}
				w.mobOpenDoor(mob, dir, keyword)
			}
		}

	case "kill", "murder":
		target := w.findPlayerByName(args)
		if target != nil {
			w.mobAttackPlayer(mob, target)
		} else {
			slog.Debug("executeMobCommand: kill target not found", "target", args)
		}

	case "drop":
		// Mob drops item(s) to the room. "drop all" drops everything.
		if args == "all" {
			for _, obj := range mob.Inventory {
				w.AddItemToRoom(obj, mob.GetRoomVNum())
			}
			mob.Inventory = mob.Inventory[:0]
		} else {
			for i, obj := range mob.Inventory {
				if obj.Prototype != nil && strings.Contains(strings.ToLower(obj.Prototype.ShortDesc), strings.ToLower(args)) {
					mob.Inventory = append(mob.Inventory[:i], mob.Inventory[i+1:]...)
					w.AddItemToRoom(obj, mob.GetRoomVNum())
					break
				}
			}
		}

	case "get":
		// Mob picks up item from room. "get all" picks up everything.
		roomItems := w.GetItemsInRoom(mob.GetRoomVNum())
		if args == "all" {
			for _, obj := range roomItems {
				w.RemoveItemFromRoom(obj, mob.GetRoomVNum())
				mob.Inventory = append(mob.Inventory, obj)
			}
		} else {
			for _, obj := range roomItems {
				if obj.Prototype != nil && strings.Contains(strings.ToLower(obj.Prototype.ShortDesc), strings.ToLower(args)) {
					w.RemoveItemFromRoom(obj, mob.GetRoomVNum())
					mob.Inventory = append(mob.Inventory, obj)
					break
				}
			}
		}

	case "give":
		// Mob gives item to a player in the room.
		// Usage: give <item> <player>
		parts := strings.Fields(args)
		if len(parts) >= 2 {
			itemName := parts[0]
			targetName := strings.Join(parts[1:], " ")
			target := w.FindPlayerInRoom(mob.GetRoomVNum(), targetName)
			if target != nil {
				for i, obj := range mob.Inventory {
					if obj.Prototype != nil && strings.Contains(strings.ToLower(obj.Prototype.ShortDesc), strings.ToLower(itemName)) {
						mob.Inventory = append(mob.Inventory[:i], mob.Inventory[i+1:]...)
						if target.Inventory != nil {
							if err := target.Inventory.AddItem(obj); err != nil {
								slog.Debug("give: AddItem error", "error", err)
							}
						}
						break
					}
				}
			}
		}

	case "ride":
		target := w.findPlayerByName(args)
		if target != nil {
			w.ExecRide(target, "ride")
		}

	case "dismount":
		target := w.findPlayerByName(mob.GetName())
		if target == nil {
			target = w.findPlayerByName(args)
		}
		if target != nil {
			w.ExecRide(target, "dismount")
		}

	case "social":
		if len(parts) > 1 {
			socialName := strings.ToLower(parts[1])
			w.doMobSocial(mob, socialName, strings.Join(parts[2:], " "))
		}

	default:
		// Check if the command itself is a social
		if social, found := Socials[cmd]; found {
			w.doMobSocial(mob, cmd, args)
			_ = social
		} else {
			slog.Debug("executeMobCommand: unknown command", "command", cmd)
		}
	}
}

// doMobSocial performs a social emote on behalf of a mob.
func (w *World) doMobSocial(mob *MobInstance, cmd string, targetName string) {
	social, found := Socials[cmd]
	if !found {
		return
	}

	var target *Player
	if targetName != "" {
		target = w.findPlayerByName(targetName)
	}

	if target != nil {
		// Social with target
		w.actToRoomMob(mob, social.Messages[2], target)    // char to vict
		w.actToRoomMob(mob, social.Messages[3], target)    // room to vict (exclude mob & vict)
		if len(social.Messages) > 4 {
			w.actToRoomMob(mob, social.Messages[4], target) // vict to char
		}
	} else if targetName != "" {
		// Target not found
		mob.SendMessage(social.Messages[5])
	} else {
		// Social without target
		mob.SendMessage(social.Messages[0])                // char_auto (to mob itself)
		w.actToRoomMob(mob, social.Messages[1], nil)       // room (exclude mob)
	}
}

// actToRoomMob sends a social message to the room, formatting $n and $N tokens.
func (w *World) actToRoomMob(mob *MobInstance, msg string, target *Player) {
	msg = strings.ReplaceAll(msg, "$n", mob.GetName())
	// Gender-aware pronouns
	switch mob.GetSex() {
	case 0:
		msg = strings.ReplaceAll(msg, "$m", "him")
		msg = strings.ReplaceAll(msg, "$s", "his")
		msg = strings.ReplaceAll(msg, "$e", "he")
	case 1:
		msg = strings.ReplaceAll(msg, "$m", "her")
		msg = strings.ReplaceAll(msg, "$s", "her")
		msg = strings.ReplaceAll(msg, "$e", "she")
	default:
		msg = strings.ReplaceAll(msg, "$m", "it")
		msg = strings.ReplaceAll(msg, "$s", "its")
		msg = strings.ReplaceAll(msg, "$e", "it")
	}
	if target != nil {
		msg = strings.ReplaceAll(msg, "$N", target.Name)
		msg = strings.ReplaceAll(msg, "$M", target.Name)
		target.SendMessage(msg)
	} else {
		players := w.GetPlayersInRoom(mob.GetRoom())
		for _, p := range players {
			if p.Name != mob.GetName() {
				p.SendMessage(msg)
			}
		}
	}
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

	// Door check — exit must be open to pass
	if exit.DoorState > 0 {
		p.SendMessage("The door is closed.\r\n")
		return nil, fmt.Errorf("door closed")
	}

	newRoom, ok := w.rooms[exit.ToRoom]
	if !ok {
		return nil, fmt.Errorf("exit leads to invalid room %d", exit.ToRoom)
	}

	// Boat requirement for WATER_NOSWIM
	// C source: act.movement.c:126-129 — needs boat for source or dest WATER_NOSWIM
	if (currentRoom.Sector == 7 || newRoom.Sector == 7) { // SECT_WATER_NOSWIM
		if !p.HasBoat() {
			p.SendMessage("You need a boat to go there.\r\n")
			return nil, fmt.Errorf("no boat")
		}
	}

	// Room tunnel limit — only 1 PC allowed
	// C source: act.movement.c:189-191 — ROOM_TUNNEL = bit 8
	if roomHasFlagBit(newRoom.Flags, 8) {
		pcCount := 0
		for _, other := range w.players {
			if other.RoomVNum == newRoom.VNum {
				pcCount++
			}
		}
		if pcCount >= 1 {
			p.SendMessage("There isn’t enough room there!\r\n")
			return nil, fmt.Errorf("room tunnel full")
		}
	}

	// Movement point cost
	moveCost := (sectorMoveCost(currentRoom.Sector) + sectorMoveCost(newRoom.Sector)) / 2
	if p.GetMove() < moveCost {
		p.SendMessage("You are too exhausted.\r\n")
		return nil, fmt.Errorf("too exhausted")
	}
	p.SetMove(p.GetMove() - moveCost)

	p.RoomVNum = newRoom.VNum
	return newRoom, nil
}

// sectorMoveCost returns movement point cost for a sector type.
// Ported from act.movement.c movement_loss[] table.
func sectorMoveCost(sector int) int {
	switch sector {
	case 0: // SECT_INSIDE
		return 1
	case 1: // SECT_CITY
		return 1
	case 2: // SECT_FIELD
		return 2
	case 3: // SECT_FOREST
		return 3
	case 4: // SECT_HILLS
		return 4
	case 5: // SECT_MOUNTAIN
		return 6
	case 6: // SECT_WATER_SWIM
		return 4
	case 7: // SECT_WATER_NOSWIM
		return 4
	case 8: // SECT_UNDERWATER
		return 4
	case 9: // SECT_FLYING
		return 1
	default:
		return 1
	}
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

// GetAllObjects returns all active object instances in the world.
func (w *World) GetAllObjects() []*ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()

	objs := make([]*ObjectInstance, 0, len(w.objectInstances))
	for _, o := range w.objectInstances {
		objs = append(objs, o)
	}
	return objs
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

// ---------------------------------------------------------------------------
// World implementations — STATE MUTATIONS & LOOKUPS
// (lua_batch2_mutations.go)
// ---------------------------------------------------------------------------

// EquipChar equips an object (found by vnum in the character's inventory) on
// the named character. For mobs, equips at slot determined by prototype wear
// flags. For players, uses Equipment.equip which determines the correct slot.
func (w *World) EquipChar(charName string, isMob bool, objVNum int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if isMob {
		for _, m := range w.activeMobs {
			if m.GetName() == charName {
				for _, obj := range m.Inventory {
					if obj.VNum == objVNum {
					// Equipment slot determination requires object WearFlags mapping —
					// deferred until equipment system fully wires obj prototype slots.
						// (parser.Obj.WearFlags exists; equipment system not yet wired)
						return false
					}
				}
				return false
			}
		}
	} else {
		if p, ok := w.players[charName]; ok {
			if p.Inventory == nil {
				return false
			}
			item, found := p.Inventory.removeItemByVNum(objVNum)
			if !found {
				return false
			}
			if p.Equipment == nil {
				p.Equipment = NewEquipment()
				p.Equipment.OwnerName = p.Name
			}
			err := p.Equipment.equip(item, p.Inventory)
			return err == nil
		}
	}
	return false
}

// EquipMobByVNum finds a mob by vnum and room, removes the object from its
// inventory, and equips it. Used by the scripting engine's equip_char().
func (w *World) EquipMobByVNum(mobVNum, roomVNum, objVNum int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, m := range w.activeMobs {
		if m.VNum == mobVNum && m.GetRoomVNum() == roomVNum {
			for i, obj := range m.Inventory {
				if obj.VNum == objVNum {
					m.Inventory = append(m.Inventory[:i], m.Inventory[i+1:]...)
					if obj.Prototype != nil && len(m.Equipment) == 0 {
						m.Equipment = make(map[int]*ObjectInstance)
					}
					if obj.Prototype != nil {
						slot := wearFlagToIntSlot(obj.Prototype.WearFlags)
						if slot >= 0 {
							m.Equipment[slot] = obj
							return true
						}
					}
					return false
				}
			}
			return false
		}
	}
	return false
}

// wearFlagToIntSlot maps object wear flags to equipment slot position (int).
func wearFlagToIntSlot(wf [4]int) int {
	var flags int
	for _, w := range wf {
		flags |= w
	}
	switch {
	case flags&(1<<13) != 0: // ITEM_WEAR_WIELD
		return 13
	case flags&(1<<0) != 0: // ITEM_WEAR_TAKE
		return 0
	default:
		return -1
	}
}

// SetFollower sets the following target for a character.
func (w *World) SetFollower(followerName, leaderName string, followerIsMob bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if followerIsMob {
		for _, m := range w.activeMobs {
			if m.GetName() == followerName {
				m.SetFollowing(leaderName)
				return nil
			}
		}
	} else {
		if p, ok := w.players[followerName]; ok {
			p.Following = leaderName
			return nil
		}
	}
	return fmt.Errorf("follower %q not found", followerName)
}

// MountPlayer sets a player's mount name.
func (w *World) MountPlayer(playerName, mountName string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if p, ok := w.players[playerName]; ok {
		p.MountName = mountName
		return nil
	}
	return fmt.Errorf("player %q not found", playerName)
}

// DismountPlayer clears a player's mount.
func (w *World) DismountPlayer(playerName string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if p, ok := w.players[playerName]; ok {
		p.MountName = ""
		return nil
	}
	return fmt.Errorf("player %q not found", playerName)
}

// ClearAffects removes all affects from a character.
func (w *World) ClearAffects(charName string, isMob bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if isMob {
		for _, m := range w.activeMobs {
			if m.GetName() == charName {
				m.ClearHunting()
				return
			}
		}
	} else {
		if p, ok := w.players[charName]; ok {
			p.MasterAffects = nil
			p.ActiveAffects = nil
			p.Affects = 0
			return
		}
	}
}

// CanCarryObject returns true if the named player can carry the object.
// Checks inventory capacity and carry-weight limit.
func (w *World) CanCarryObject(charName string, objVNum int) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	p, ok := w.players[charName]
	if !ok {
		return false
	}
	proto, ok := w.GetObjPrototype(objVNum)
	if !ok {
		return false
	}
	if p.Inventory != nil && p.Inventory.IsFull() {
		return false
	}
	if p.CarriedWeight()+proto.Weight > p.MaxCarryWeight() {
		return false
	}
	return true
}

// IsCorpseObj returns true if the object prototype is a corpse.
// In the original C, IS_CORPSE checks ITEM_CONTAINER with val[3] == 1.
// The Go port defines ITEM_CORPSE as type 37 in item_helpers.go.
func (w *World) IsCorpseObj(objVNum int) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	proto, ok := w.GetObjPrototype(objVNum)
	if !ok {
		return false
	}
	return proto.TypeFlag == ITEM_CORPSE
}

// SetHunting sets a character's hunting target.
func (w *World) SetHunting(hunterName, preyName string, hunterIsMob bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if hunterIsMob {
		for _, m := range w.activeMobs {
			if m.GetName() == hunterName {
				m.SetHunting(preyName)
				return
			}
		}
	}
	// Players don't have hunting state in current implementation.
}

// IsHunting returns true if the character is hunting.
func (w *World) IsHunting(charName string, isMob bool) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if isMob {
		for _, m := range w.activeMobs {
			if m.GetName() == charName {
				return m.GetHunting() != ""
			}
		}
	}
	return false
}
