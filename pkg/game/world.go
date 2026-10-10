// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"

	"github.com/zax0rz/darkpawns/pkg/boards"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/events"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// MessageSinkFunc is the callback type for delivering messages to a player.
// The session manager sets this on World initialization so that game-layer
// SendMessage calls route through Session.send (which writePump reads).
type MessageSinkFunc func(playerName string, msg []byte)

// MovementLookFunc renders the destination room for a connected player at the
// exact point C calls look_at_room during do_simple_move.
type MovementLookFunc func(player *Player)

// CloseConnectionFunc is called when a player session should be forcibly closed
// (e.g., do_quit). Set by the session manager.
type CloseConnectionFunc func(playerName string)

// CommandExecFunc is the callback signature for executing a command on behalf of
// a player (e.g. from doOrder for charmed followers). Set by the session layer.
// Returns true if the command was dispatched.
type CommandExecFunc func(ch *Player, command string) bool

// World represents the active game world with runtime state.
//
// LOCK ORDERING: w.mu must be acquired BEFORE m.mu (Manager) if both are
// needed. Never call Manager methods while holding w.mu unless you are
// certain they don't acquire m.mu. See Manager.Register() comment for
// details.
type World struct {
	// zoneResetMu serializes boot, manual, admin and heartbeat resets.
	// It precedes brief World/Spawner locks; no lifecycle lock is acquired.
	zoneResetMu      sync.Mutex
	zoneClockStarted bool
	zoneAges         map[int]int
	zoneMinuteTicks  int
	zoneResetQueue   []int
	// OccupiedZoneRooms snapshots playing descriptors, not retained bodies.
	// Installed before heartbeat starts; called without World.mu.
	OccupiedZoneRooms func() []int

	mu sync.RWMutex

	// Snapshot manager for lock-free reads
	snapshots *SnapshotManager

	// Static world data (from parsed files)
	rooms     map[int]*parser.Room
	roomOrder []int // room VNums in C world[] RNUM/load order
	// sortedRoomVNums is the vnum-ascending copy of the room set; an index
	// into it is exactly C's room rnum (see NewWorld). Read-only after boot:
	// fixtures never add or remove rooms.
	sortedRoomVNums []int
	mobs            map[int]*parser.Mob
	objs            map[int]*parser.Obj
	zones           map[int]*parser.Zone
	parsedData      *parser.World // original parsed data, nil after boot

	// World path for reload support
	WorldPath string
	// ScriptsDir is the live Lua script tree used by the scripting engine and
	// the in-game luaedit command. It is normally <WorldPath>/scripts, but the
	// server's -scripts override may point elsewhere.
	ScriptsDir string
	// PlayerSaver, when set by the server, writes a player's record to the
	// store of record (the database login reads). nil means saves are
	// skipped — never treated as successful. See persistence_seam.go.
	PlayerSaver     PlayerSaver
	ObjectSaver     func(*Player, int) error
	PlayerStoreList func() ([]*Player, error)
	PlayerStoreEdit func(string, func(*Player)) error

	// Runtime state
	players               map[string]*Player   // keyed by player name
	activeMobs            map[int]*MobInstance // keyed by instance ID
	nextMobID             int
	nextRoomEntrySequence uint64
	// pendingPlayerExtractions is the explicit queue used by death paths that
	// mirror C's extract_char() to next-heartbeat extract_pending_chars() flow.
	pendingPlayerExtractions map[*Player]struct{}
	pendingMobileExtractions map[*MobInstance]struct{} // retained C bodies until extract_pending_chars

	// Room items: room VNum -> list of object instances
	roomItems map[int][]*ObjectInstance
	nextObjID int

	// All live object instances: instance ID -> ObjectInstance
	objectInstances map[int]*ObjectInstance

	// AI combat engine (CRIT-006: moved from global to World field)
	combatEngine CombatEngine

	// Shutdown coordination (closed once by StopAITicker)
	done     chan bool
	doneOnce sync.Once

	// Spawner
	spawner *Spawner

	// Shop manager
	shopManager common.ShopManager
	// shopKeepers maps shopkeeper mob vnum -> shop behavior bitvector, loaded
	// from the .shp files at boot (C: assign_the_shopkeepers).
	shopKeepers map[int]int

	// Event queue for timer-based scripted events
	// Source: events.c event_init() — global event_q
	EventQueue *events.EventQueue

	// Events is the typed event bus for decoupled subsystem communication.
	Events events.Bus

	// House control records — loaded by HouseBoot() during initialization
	HouseControl []HouseControl

	// Clans manager — loaded by InitClans() during initialization
	Clans *ClanManager

	// Boards system — initialized via GetOrInitBoards()
	Boards *boards.BoardSystem

	// Bans — site ban list + invalid name filter (ported from ban.c)
	Bans *BanManager

	// Whod — who-daemon display mode flags (ported from whod.c)
	WhodDisplay *Whod

	// MessageSink routes player messages through the session layer.
	// Set by the session manager on initialization. If nil, messages are silently dropped.
	MessageSink MessageSinkFunc

	// IdleCloseDescriptor runs with world/player locks released at C close_socket.
	IdleCloseDescriptor func(*Player)

	// MobileMessageSink delivers to the descriptor of this concrete NPC body.
	// Ordinary descriptorless mobiles receive no actor output.
	MobileMessageSink func(*MobInstance, []byte)

	// MovementLook is owned by the session layer because room rendering is a
	// transport concern, but movement owns its ordering relative to arrivals,
	// followers, and entry triggers.
	MovementLook MovementLookFunc

	// LibTextDir is the lib/text root (2010 static text + help), derived from
	// the -world flag's parent at boot; "lib/text" when no source dir is known.
	LibTextDir string
	// HelpTable holds all loaded help entries (from lib/text/help/).
	// Populated by LoadHelpFiles during world boot.
	HelpTable []HelpEntry

	// HelpScreen is the no-argument help text (lib/text/help/screen), page_string'd
	// by do_help on a bare `help`. Loaded once at boot (C: file_to_string_alloc of
	// HELP_PAGE_FILE into the `help` global, db.c:193).
	HelpScreen string

	// CloseConnection routes close requests through the session layer.
	CloseConn CloseConnectionFunc

	// gossipHistory records the last 25 gossip messages for the review command.
	// Matches C: struct review_t review[25] in db.c.
	gossipMu      sync.RWMutex
	gossipHistory []gossipEntry

	// DNS cache — loaded from etc/dns on first use and kept in the same
	// 257-bucket/prepend shape as C's dns_cache[] (db.h:194-201).
	dnsMu     sync.Mutex
	dnsCache  [dnsHashBuckets][]dnsEntry
	dnsLoaded bool

	// CommandExecFunc dispatches player commands through the session layer.
	// Set by the session manager. If nil, executeCommand is a no-op.
	CommandExecFunc CommandExecFunc

	// OnGossip is a callback triggered when a human player gossips.
	OnGossip func(senderName string, message string)

	// OutOfBand mirrors room renders, channel lines, and regen ticks to the
	// session layer's structured-client protocols (GMCP). Set once by the
	// session manager before the world starts ticking; nil disables it.
	OutOfBand OutOfBandObserver
}

// SetOnGossip installs the gossip relay callback under the gossip lock; the
// grapevine client rewrites it from its reconnect goroutine (VULN-040).
func (w *World) SetOnGossip(fn func(playerName, message string)) {
	w.gossipMu.Lock()
	w.OnGossip = fn
	w.gossipMu.Unlock()
}

// SetCombatEngine sets the combat engine for AI to use.
// CRIT-006: replaces global SetAICombatEngine.
func (w *World) SetCombatEngine(ce CombatEngine) {
	w.combatEngine = ce
}

// NewWorld creates a new game world from parsed data.
func NewWorld(parsed *parser.World) (*World, error) {
	shopManager := NewShopManager()
	w := &World{
		rooms:                    make(map[int]*parser.Room),
		roomOrder:                make([]int, 0, len(parsed.Rooms)),
		mobs:                     make(map[int]*parser.Mob),
		objs:                     make(map[int]*parser.Obj),
		zones:                    make(map[int]*parser.Zone),
		players:                  make(map[string]*Player),
		activeMobs:               make(map[int]*MobInstance),
		nextMobID:                1,
		pendingPlayerExtractions: make(map[*Player]struct{}),
		pendingMobileExtractions: make(map[*MobInstance]struct{}),
		roomItems:                make(map[int][]*ObjectInstance),
		nextObjID:                1,
		objectInstances:          make(map[int]*ObjectInstance),
		done:                     make(chan bool),
		shopManager:              shopManager,
		parsedData:               parsed, // Keep reference for door loading etc.
		WorldPath:                "",     // Set externally for reload support
		ScriptsDir:               "",     // Set by the server after resolving -scripts
	}

	// Index rooms by VNum
	for i := range parsed.Rooms {
		room := &parsed.Rooms[i]
		if _, exists := w.rooms[room.VNum]; !exists {
			w.roomOrder = append(w.roomOrder, room.VNum)
		}
		w.rooms[room.VNum] = room
	}

	// Build the rnum-equivalent index: C's world[] is strictly vnum-ascending
	// (rooms load inside their zone's vnum range over an ascending zone table,
	// and real_room() binary-searches it — db.c:3083), so the sorted vnum
	// slice's indexes are exactly C room rnums.
	w.sortedRoomVNums = make([]int, len(w.roomOrder))
	copy(w.sortedRoomVNums, w.roomOrder)
	sort.Ints(w.sortedRoomVNums)

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
	// boot_db's renum_zone_table: legacy R commands take C's form and
	// commands naming a missing room, mobile or object are disabled.
	w.renumZoneTable()

	// Index shops by keeper vnum — C's assign_the_shopkeepers (shop.c:1232-1243)
	// gives every shop's keeper the shop_keeper spec, which is what
	// is_shopkeeper (mobprog.c:473) and ok_damage_shopkeeper test.
	w.shopKeepers = make(map[int]int, len(parsed.Shops))
	for i := range parsed.Shops {
		shop := &parsed.Shops[i]
		if shop.KeeperVNum >= 0 {
			w.shopKeepers[shop.KeeperVNum] = shop.Bitvector
		}
		legacyShop := &Shop{
			VNum:       shop.VNum,
			KeeperVNum: shop.KeeperVNum,
			SellTypes:  append([]int(nil), shop.Products...),
			BuyTypes:   append([]int(nil), shop.BuyTypes...),
			BuyWords:   append([]string(nil), shop.BuyWords...),
			ProfitBuy:  shop.BuyProfit,
			ProfitSell: shop.SellProfit,
			Flags:      shop.Bitvector,
			Messages:   shop.Messages,
			Temper:     shop.Temper,
			WithWho:    shop.WithWho,
			OpenHour1:  shop.OpenHour1,
			CloseHour1: shop.CloseHour1,
			OpenHour2:  shop.OpenHour2,
			CloseHour2: shop.CloseHour2,
		}
		legacyShop.Rooms = append([]int(nil), shop.Rooms...)
		if len(shop.Rooms) > 0 {
			legacyShop.RoomVNum = shop.Rooms[0]
		}
		if keeper, ok := w.mobs[shop.KeeperVNum]; ok {
			legacyShop.KeeperName = keeper.ShortDesc
		}
		shopManager.AddShop(legacyShop)
	}

	// Initialize event queue
	// Source: events.c event_init() — called in init_game() before boot_db()
	// In original: 1 pulse = 1/10 second (OPT_USEC = 100000)
	w.EventQueue = events.NewEventQueue(100 * time.Millisecond)

	// Initialize typed event bus
	w.Events = events.NewInProcessBus()

	// Start event processing loop (MobProg delayed events, etc.).
	// Source: events.c event_process() — called once per pulse in heartbeat().
	// Mob AI itself is driven solely by the game loop's OnMobileActivity
	// (PULSE_MOBILE = 4s), faithful to C's mobile_activity(); there is no
	// separate AI ticker in C (DP-1035).
	w.StartEventQueue()

	// PointUpdate is driven only by the server heartbeat, in C order after
	// weather and affects (src/comm.c:825-830), in live and DP_CLOCK modes.

	// Initialize snapshot manager and publish initial snapshot
	w.snapshots = NewSnapshotManager()
	w.snapshots.Publish(w.rooms)

	// Initialize house control and board system
	w.HouseControl = make([]HouseControl, 0)

	// Initialize ban manager and WHOD display (ported from ban.c + whod.c)
	w.Bans = NewBanManager()
	w.WhodDisplay = NewWhod()

	// Load help files from lib/text/help/. LoadHelpFiles returns the table
	// keyword-sorted (C qsort/hsort). We then append the hardcoded race help
	// entries and RE-SORT, because do_help's prefix binary search requires the
	// whole table to be sorted — appending after the sort would break it.
	// lib/text root: derived from -world's parent (see parser.World.SourceDir);
	// CWD fallback for hand-built worlds/tests. Shared by help and the
	// do_gen_ps static-text commands (session layer reads World.LibTextDir).
	w.LibTextDir = "lib/text"
	if parsed != nil && parsed.SourceDir != "" {
		w.LibTextDir = filepath.Join(parsed.SourceDir, "..", "text")
	}
	helpDir := filepath.Join(w.LibTextDir, "help")
	if loaded, err := LoadHelpFiles(helpDir); err == nil {
		w.HelpTable = append(w.HelpTable, loaded...)
	}
	w.HelpTable = append(w.HelpTable, RaceHelpEntries()...)
	sortHelpTable(w.HelpTable)
	// No-argument help screen (lib/text/help/screen), page_string'd by do_help.
	if screen, err := LoadHelpScreen(helpDir); err == nil {
		w.HelpScreen = screen
	}

	w.spawner = NewSpawner(w)
	return w, nil
}

// PostInit performs first-time initialization of systems that depend on
// the world being fully constructed (house boot, clan init).
// Must be called after NewWorld but before starting the main loop.
func (w *World) PostInit() {
	w.Clans = InitClans("./data/clans.json")
	w.HouseBoot()
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
	if !ok {
		for key, candidate := range w.players {
			if strings.EqualFold(key, name) {
				return candidate, true
			}
		}
	}
	return p, ok
}

// GetPlayers returns a snapshot of the currently online players.
func (w *World) GetPlayers() []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	players := make([]*Player, 0, len(w.players))
	for _, player := range w.players {
		players = append(players, player)
	}
	return players
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

// SetExitInfo replaces the runtime EX_* bitfield for an exit in a room.
// It takes the routine mutation path: door operations (open/close/lock,
// zone-reset D commands, scripting) must not pay the structural insertion
// cost of a full world rebuild per call.
func (w *World) SetExitInfo(roomVNum int, direction string, info int) bool {
	return w.mutateRoom(roomVNum, func(room *parser.Room) bool {
		exit, ok := room.Exits[direction]
		if !ok {
			return false
		}
		exit.ExitInfo = info
		room.Exits[direction] = exit
		return true
	})
}

// AddPlayer adds a player to the world.
func (w *World) AddPlayer(p *Player) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for name := range w.players {
		if strings.EqualFold(name, p.Name) {
			return fmt.Errorf("player %s already online", p.Name)
		}
	}

	// Database-backed players already have persistent IDs. In a no-DB
	// differential/runtime session, new characters start at ID 0; retain that
	// first ID for fixture ownership, but give later zero-ID players distinct
	// runtime IDs so ID↔name lookups (notably house guests) cannot alias them.
	if p.ID == 0 {
		zeroIDInUse := false
		for _, existing := range w.players {
			if existing.ID == 0 {
				zeroIDInUse = true
				break
			}
		}
		if zeroIDInUse {
			for candidate := 1; ; candidate++ {
				used := false
				for _, existing := range w.players {
					if existing.ID == candidate {
						used = true
						break
					}
				}
				if !used {
					p.ID = candidate
					break
				}
			}
		}
	}

	p.mu.Lock()
	p.worldRef = w
	p.combatRetired = false
	w.nextRoomEntrySequence++
	p.RoomEntrySequence = w.nextRoomEntrySequence
	p.mu.Unlock()

	w.players[p.Name] = p
	return nil
}

// RemovePlayerBody cannot delete a later same-name replacement.
func (w *World) RemovePlayerBody(p *Player) {
	if p == nil {
		return
	}
	w.mu.Lock()
	removed := w.players[p.GetName()] == p
	if removed {
		delete(w.players, p.GetName())
	}
	w.mu.Unlock()
	if removed {
		w.retireCombatBody(p)
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
//
// Deprecated: use GetRoom (snapshot version) instead.
//
// The returned pointer aliases live world state: the read lock is released
// before the caller runs, so callers must NOT mutate the room through it.
// Writes must go through the locked setters (SetRoomFlags, SetRoomFlagBit,
// SetRoomSector, SetRoomExit, SetExitInfo, ...).
func (w *World) GetRoomInWorld(vnum int) *parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rooms[vnum]
}

// RealRoomIndex returns the vnum-sorted room index (C's rnum) for vnum,
// mirroring C real_room()'s binary search over the ascending world[] array
// (db.c:3083). ok is false when the vnum has no room — C returns NOWHERE.
func (w *World) RealRoomIndex(vnum int) (index int, ok bool) {
	w.mu.RLock()
	sorted := w.sortedRoomVNums
	w.mu.RUnlock()
	i := sort.SearchInts(sorted, vnum)
	if i < len(sorted) && sorted[i] == vnum {
		return i, true
	}
	return 0, false
}

// RoomVNumByIndex converts a vnum-sorted room index (C rnum) back to its
// vnum. ok is false for an out-of-range index; callers on a valid rnum range
// never hit that branch.
func (w *World) RoomVNumByIndex(index int) (vnum int, ok bool) {
	w.mu.RLock()
	sorted := w.sortedRoomVNums
	w.mu.RUnlock()
	if index < 0 || index >= len(sorted) {
		return 0, false
	}
	return sorted[index], true
}

// isLitLightSource returns true if the object is a working light source.
// Source: src/handler.c:823-839 — ITEM_LIGHT and value[2] != 0.
func isLitLightSource(obj *ObjectInstance) bool {
	if obj == nil {
		return false
	}
	return obj.GetTypeFlag() == itemLightTypeFlag && obj.GetValue(2) != 0
}

// adjustRoomLight increments or decrements the light counter for a room.
// Called when light sources enter or leave a room.
func (w *World) adjustRoomLight(vnum int, delta int) {
	if room, ok := w.rooms[vnum]; ok {
		updated := CloneRoom(*room)
		updated.Light += delta
		w.replaceRoomLocked(updated)
	}
}

// Rooms returns all rooms in the world.
// GetRoomCount returns the total number of rooms in the world.
// Equivalent to top_of_world in C.
func (w *World) GetRoomCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.rooms)
}

// GetRoomVNumAtIndex returns the room VNUM at the C world's stable RNUM
// position. C spell_teleport draws a room index, while Go gameplay addresses
// rooms by VNUM; this keeps the shared spell path's draw range and destination
// selection faithful.
func (w *World) GetRoomVNumAtIndex(index int) (int, bool) {
	rooms := w.Rooms()
	if index < 0 || index >= len(rooms) {
		return 0, false
	}
	return rooms[index].VNum, true
}

// GetPlayerCount returns the number of online players.
func (w *World) GetPlayerCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.players)
}

// GetAllPlayers returns all online players.
func (w *World) GetAllPlayers() []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		result = append(result, p)
	}
	return result
}

// GetHelpFileCount returns the number of loaded help entries.
func (w *World) GetHelpFileCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.HelpTable)
}

// VisiblePlayersForMortal returns the in-game, non-NPC characters that a
// level-1 mortal would see in `who`. It applies the same visibility
// boundary cmdWho uses — whoTargetVisible in pkg/session/cmd_info.go
// gates on game.CanSee(viewer, target); here the viewer is a bare
// level-1 mortal, so wizinvis immortals, invisible and hidden characters
// are excluded. External presence surfaces (Grapevine heartbeat, MSSP
// PLAYERS) share this rule so a hidden immortal is never revealed to an
// outside network or crawler. Guests are ordinary in-world characters
// and are included.
func (w *World) VisiblePlayersForMortal() []*Player {
	viewer := &Player{Level: 1}
	players := w.GetAllPlayers()
	visible := make([]*Player, 0, len(players))
	for _, p := range players {
		if p.IsNPC() || p.Name == "" {
			continue
		}
		if !CanSee(viewer, p) {
			continue
		}
		visible = append(visible, p)
	}
	return visible
}

func (w *World) Rooms() []parser.Room {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]parser.Room, 0, len(w.rooms))
	seen := make(map[int]struct{}, len(w.roomOrder))
	for _, vnum := range w.roomOrder {
		if room, ok := w.rooms[vnum]; ok {
			result = append(result, *room)
			seen[vnum] = struct{}{}
		}
	}

	// Rooms added outside parsed world data have no C RNUM equivalent. Include
	// them in stable vnum order so test fixtures and live edits remain visible
	// without reintroducing map-order nondeterminism.
	extraVNums := make([]int, 0, len(w.rooms)-len(result))
	for vnum := range w.rooms {
		if _, ok := seen[vnum]; !ok {
			extraVNums = append(extraVNums, vnum)
		}
	}
	sort.Ints(extraVNums)
	for _, vnum := range extraVNums {
		result = append(result, *w.rooms[vnum])
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

	if mob == nil || !mob.IsAlive() {
		w.mu.RUnlock()
		slog.Debug("executeMobCommand: mob not found or dead", "vnum", mobVNum, "command", cmdStr)
		return
	}

	slog.Debug("mob executes command", "mob_vnum", mobVNum, "mob_name", mob.GetName(), "command", cmdStr)

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		w.mu.RUnlock()
		return
	}

	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	w.mu.RUnlock()

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
		if leader, ok := w.followingActor(args).(combat.Combatant); ok {
			mob.SetFollowingBody(leader)
		} else {
			mob.SetFollowing(args)
		}

	case "open":
		openParts := strings.Fields(args)
		if len(openParts) > 0 {
			dirMap := map[string]int{
				"north": 0, "east": 1, "south": 2, "west": 3, "up": 4, "down": 5,
				"n": 0, "e": 1, "s": 2, "w": 3, "u": 4, "d": 5,
			}
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
				// The action() drop reaches C's perform_drop, whose room
				// placement is obj_to_room (act.item.c:504) — prepend.
				w.AddItemToRoomFront(obj, mob.GetRoomVNum())
			}
			mob.Inventory = mob.Inventory[:0]
		} else {
			for i, obj := range mob.Inventory {
				if obj.Prototype != nil && strings.Contains(strings.ToLower(obj.Prototype.ShortDesc), strings.ToLower(args)) {
					mob.Inventory = append(mob.Inventory[:i], mob.Inventory[i+1:]...)
					w.AddItemToRoomFront(obj, mob.GetRoomVNum())
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
							} else {
								// C's perform_give ends in
								// obj_to_char(obj, vict)
								// (src/act.item.c:696): the recipient is
								// flagged (handler.c:569-571).
								target.MarkCrashNeeded()
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
			w.npcSocial(mob, socialName, strings.Join(parts[2:], " "))
		}

	default:
		// Check if the command itself is a social
		if _, found := Socials[cmd]; found {
			w.npcSocial(mob, cmd, args)
		} else {
			slog.Debug("executeMobCommand: unknown command", "command", cmd)
		}
	}
}

// IsRoomDark returns true if the given room VNum is dark.
// Based on utils.h IS_DARK() macro (utils.h:254-259):
//
//	IS_DARK(room) = !world[room].light && (ROOM_FLAGGED(room, ROOM_DARK) ||
//	    (SECT(room) != SECT_INSIDE && SECT(room) != SECT_CITY && night))
//
// City streets are lit at night; only other outdoor sectors go dark.
func (w *World) IsRoomDark(roomVNum int) bool {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}
	// If room has active light sources, it's never dark
	if room.IsLight() {
		return false
	}
	// Check ROOM_DARK flag (bit 0)
	if room.HasFlag(0) {
		return true
	}
	// Rooms that are neither inside nor city are dark at night (SunDark or
	// SunSet).
	if room.Sector != SECT_INSIDE && room.Sector != SECT_CITY {
		sunlight := GetSunlight()
		if sunlight == SunDark || sunlight == SunSet {
			return true
		}
	}
	return false
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
		// Read through the player's own lock: SetRoom runs on session
		// goroutines under p.mu only, so a raw p.RoomVNum read here would
		// race movement even though the map iteration is covered by w.mu.
		if p.GetRoom() == roomVNum {
			players = append(players, p)
		}
	}
	return players
}

// RoomEcho broadcasts a message to all players in the given room,
// optionally excluding one player by name. It satisfies the boards.BoardWorld
// interface for the extracted board system.
func (w *World) RoomEcho(roomVNum int, message string, excludeName string) {
	actToRoom(w, roomVNum, message, excludeName)
}

// sectorMoveCost returns the movement-point cost for a sector type, reading the
// shared movementLoss table (src/constants.c movement_loss[]). Out-of-range
// sectors fall back to the INSIDE cost rather than panicking; C never indexes
// OOB, but malformed zone data could.
func sectorMoveCost(sector int) int {
	if sector < 0 || sector >= len(movementLoss) {
		return movementLoss[SECT_INSIDE]
	}
	return movementLoss[sector]
}

// StopAITicker stops the AI tick loop and the point-update ticker (they share
// the World's done channel). Safe to call multiple times.
func (w *World) StopAITicker() {
	if w.done == nil {
		return
	}
	w.doneOnce.Do(func() {
		close(w.done)
	})
}

// SpawnMob spawns a mob in the world.
func (w *World) SpawnMob(vnum int, roomVNum int) (*MobInstance, error) {
	return w.spawnMob(vnum, roomVNum)
}

// spawnMob is read_mobile followed by char_to_room, both silent in C: a
// zone reset, do_load or a special procedure that loads a mobile prints
// whatever its own code prints. (The port used to tell everyone in the
// room "<mob> appears." on a zone reset; C has no such line, R4.)
func (w *World) spawnMob(vnum int, roomVNum int) (*MobInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	proto, ok := w.mobs[vnum]
	if !ok {
		return nil, fmt.Errorf("mob prototype %d not found", vnum)
	}
	if _, ok := w.rooms[roomVNum]; !ok {
		return nil, fmt.Errorf("room %d not found", roomVNum)
	}

	mob := NewMob(proto, roomVNum)
	mob.world = w
	mob.ID = w.nextMobID
	w.nextRoomEntrySequence++
	mob.RoomEntrySequence = w.nextRoomEntrySequence
	w.activeMobs[w.nextMobID] = mob
	w.nextMobID++
	return mob, nil
}

// spawnMobQuiet is spawnMob; kept for its callers.
func (w *World) spawnMobQuiet(vnum int, roomVNum int) (*MobInstance, error) {
	return w.spawnMob(vnum, roomVNum)
}

// SpawnMobQuiet is SpawnMob. C do_load uses read_mobile followed by
// char_to_room, and its caller supplies the command's narration
// (act.wizard.c:1315-1321).
func (w *World) SpawnMobQuiet(vnum int, roomVNum int) (*MobInstance, error) {
	return w.spawnMobQuiet(vnum, roomVNum)
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
	found := false
	for id, m := range w.activeMobs {
		if m == mob {
			m.mu.Lock()
			m.Flags |= 1 << uint(MobFlagExtract)
			m.mu.Unlock()
			w.pendingMobileExtractions[m] = struct{}{}
			delete(w.activeMobs, id)
			found = true
			break
		}
	}
	w.mu.Unlock()
	if found {
		w.retireCombatBody(mob)
	}
}

// mobileObjectOwnerLocked retains C's character pointer for object ownership
// and Lua handles until extract_pending_chars, even after combat retirement.
// Caller holds World.mu (read or write); ordinary active lookup is unchanged.
func (w *World) mobileObjectOwnerLocked(id int) *MobInstance {
	if m := w.activeMobs[id]; m != nil {
		return m
	}
	for m := range w.pendingMobileExtractions {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// registerExistingObject adopts an already-constructed object into the
// registry, assigning it the next object ID. For restore paths that build
// via world-free helpers (house loads) and need the registry identity
// after the fact. Caller must hold no other locks.
func (w *World) registerExistingObject(obj *ObjectInstance) *ObjectInstance {
	if obj == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	obj.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[obj.ID] = obj
	return obj
}

// NewObjectFromProto creates a REGISTERED object from a prototype: it enters
// the world's objectInstances registry with a nextObjID identity, so
// container moves, world-wide scans, and extraction see it. This is the
// constructor for every live game object; use SpawnObject when only a vnum
// is at hand. NewObjectInstance (unregistered, ID 0) is for ephemeral
// display/comparison probes and tests only — C's read_object links every
// object into object_list (R1), and the port matches that here.
func (w *World) NewObjectFromProto(proto *parser.Obj, roomVNum int) *ObjectInstance {
	if proto == nil {
		return nil
	}
	return w.newObjectInstance(proto, roomVNum)
}

// SpawnObject spawns an object in the specified room.
func (w *World) SpawnObject(objVNum, roomVNum int) (*ObjectInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	proto, ok := w.objs[objVNum]
	if !ok {
		return nil, fmt.Errorf("object prototype %d not found", objVNum)
	}

	obj := w.newObjectInstanceLocked(proto, roomVNum)
	return obj, nil
}

// newObjectInstance creates a world-owned runtime object with a stable ID.
// Object movement relies on this registry to resolve container locations.
func (w *World) newObjectInstance(proto *parser.Obj, roomVNum int) *ObjectInstance {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.newObjectInstanceLocked(proto, roomVNum)
}

func (w *World) newObjectInstanceLocked(proto *parser.Obj, roomVNum int) *ObjectInstance {
	obj := NewObjectInstance(proto, roomVNum)
	obj.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[obj.ID] = obj
	return obj
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
	// Snapshot the mob list under the world lock, then query each mob's room
	// OUTSIDE the world lock. mob.GetRoom() acquires m.mu.RLock(); holding
	// w.mu.RLock() concurrently creates a nested-lock scenario: if a pending
	// writer queues on w.mu between the outer RLock and mob.GetRoom()'s inner
	// RLock, mob.GetRoom() blocks while the outer RLock is held — deadlock.
	w.mu.RLock()
	allMobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, mob := range w.activeMobs {
		allMobs = append(allMobs, mob)
	}
	w.mu.RUnlock()

	var mobs []*MobInstance
	for _, mob := range allMobs {
		if mob.GetRoom() == roomVNum {
			mobs = append(mobs, mob)
		}
	}
	return mobs
}

// COrderedRoomMobs returns the room's mobiles in C's world[room].people order:
// the most recent arrival first, because char_to_room() prepends to the list.
// Ties fall back to name so the order is deterministic for two mobs that
// entered in the same tick (R3). C's special() walks exactly this list for
// mobile specials and mobile oncmd scripts (src/interpreter.c:1452-1466);
// GetMobsInRoom walks a map and must not be used where the order is
// observable.
func (w *World) COrderedRoomMobs(roomVNum int) []*MobInstance {
	mobs := w.GetMobsInRoom(roomVNum)
	sort.SliceStable(mobs, func(i, j int) bool {
		si, sj := mobs[i].GetRoomEntrySequence(), mobs[j].GetRoomEntrySequence()
		if si != sj {
			return si > sj
		}
		return mobs[i].GetName() < mobs[j].GetName()
	})
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
	if isMob {
		if m := w.GetMobByName(charName); m != nil {
			return w.equipMobileInventoryVNum(m, objVNum)
		}
		return false
	}
	w.mu.Lock()
	p := w.players[charName]
	if p != nil && p.Equipment == nil {
		p.Equipment = NewEquipment()
		p.Equipment.OwnerName = p.Name
		p.Equipment.afterChange = p.AffectTotal
	}
	w.mu.Unlock()
	if p == nil || p.Inventory == nil {
		return false
	}
	item, found := p.Inventory.removeItemByVNum(objVNum)
	if !found {
		return false
	}
	if err := p.Equipment.Equip(item, p.Inventory); err != nil {
		return false
	}
	p.checkEquipmentStats()
	return true
}

// EquipMobByVNum finds a mob by vnum and room, removes the object from its
// inventory, and equips it. Used by the scripting engine's equip_char().
func (w *World) EquipMobByVNum(mobVNum, roomVNum, objVNum int) bool {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if m.GetVNum() == mobVNum {
			return w.equipMobileInventoryVNum(m, objVNum)
		}
	}
	return false
}

func (w *World) equipMobileInventoryVNum(m *MobInstance, vnum int) bool {
	m.mu.RLock()
	var obj *ObjectInstance
	for _, item := range m.Inventory {
		if item.GetVNum() == vnum {
			obj = item
			break
		}
	}
	m.mu.RUnlock()
	if obj == nil {
		return false
	}
	pos := findEqPos(obj, "")
	return pos >= 0 && m.EquipItem(obj, pos)
}

// SetFollower sets the following target for a character.
func (w *World) SetFollower(followerName, leaderName string, followerIsMob bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var leader combat.Combatant
	if p := w.players[leaderName]; p != nil {
		leader = p
	} else {
		for _, m := range w.activeMobs {
			if m.GetName() == leaderName {
				leader = m
				break
			}
		}
	}
	if followerIsMob {
		for _, m := range w.activeMobs {
			if m.GetName() == followerName {
				if leader != nil {
					m.SetFollowingBody(leader)
				} else {
					m.SetFollowing(leaderName)
				}
				return nil
			}
		}
	} else {
		if p, ok := w.players[followerName]; ok {
			if leader != nil {
				p.SetFollowingBody(leader)
			} else {
				p.SetFollowing(leaderName)
			}
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
			p.mu.Lock()
			hadAffects := len(p.ActiveAffects) > 0 || len(p.MasterAffects) > 0
			p.MasterAffects = nil
			p.ActiveAffects = nil
			p.Affects = 0
			p.mu.Unlock()
			if hadAffects {
				p.AffectTotal()
			}
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
	if p.Inventory != nil && p.Inventory.GetItemCount() >= p.MaxCarryItems() {
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
