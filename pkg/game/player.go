package game

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// Sex constants — Go's actor encoding, deliberately different from C's SEX_*
// (structs.h: 0=neutral/1=male/2=female). C-encoded data translates at the
// Actor boundary; see MobInstance.GetSex.
const (
	SexMale    = 0
	SexFemale  = 1
	SexNeutral = 2
)

// Player represents an active player in the game.
type Player struct {
	mu sync.RWMutex

	// Identity
	ID   int
	Name string

	// Core stats
	Health    int
	MaxHealth int
	Mana      int
	MaxMana   int
	Move      int // Movement points — ported from limits.c/structs.h GET_MOVE
	MaxMove   int // Max movement points
	Practices int // Practice sessions for skill training
	Height    int `json:"height"` // Height in cm — randomized by sex at creation (db.c:3041-3047)
	Weight    int `json:"weight"` // Weight in kg — randomized by sex at creation (db.c:3041-3047)

	Level    int
	Exp      int
	Gold     int // Currency, used by Lua scripts
	BankGold int // Bank account — from structs.h GET_BANK_GOLD

	// Clan system (ported from clan.c)
	Strength int // For inventory capacity

	// Clan membership — ported from src/clan.c clan_rec/_clan / _clan_rank
	ClanID   int `json:"clan_id"`
	ClanRank int `json:"clan_rank"`

	// WriteMagic — set by gen_board spec proc to indicate session-level editor
	// should start for board writing. Value is board_type + BOARD_MAGIC.
	WriteMagic int

	// Hunger/thirst/drunk conditions — limits.c:366, structs.h:566-568
	// Index: CondDrunk=0, CondFull=1, CondThirst=2
	// Value: -1 = immortal (no change), 0 = depleted, 1-48 = current level
	Conditions [3]int

	// Affect flags bitmask — structs.h AFF_* constants
	// Bit 11 = AFF_POISON, bit 25 = AFF_CUTTHROAT, bit 31 = AFF_FLAMING
	// Source: structs.h:321,335,341
	Affects uint64

	// Player flags bitmask — structs.h PLR_* constants
	// Source: structs.h:221-244
	PlayerFlags uint64

	// ActiveAffects is a list of active spell/status effects on this player.
	// This is separate from the Affects bitmask — bitmask tracks AFF_* flags,
	// while ActiveAffects tracks spell effects with duration/stacking.
	// Used by save/load, not persisted to JSON yet.
	ActiveAffects []*engine.Affect

	// Character identity — from do_start()/roll_real_abils() in class.c
	Class int
	Race  int
	Sex   int // SexMale=0, SexFemale=1, SexNeutral=2 (Go encoding; C data translates at boundary)

	// RaceHates tracks 5 racial hatred slots (src/structs.h race_hate[5]).
	// Initialized to -1 for all slots; matching a mob's race triggers aggression.
	RaceHates [5]int

	Stats CharStats

	// SavingThrows — array of 5 saving throw values: para, rod, petri, breath, spell
	// Source: structs.h saving_throws[5]
	SavingThrows [5]int

	// MasterAffects — active spell/status effects used by the engine for affect iteration.
	// Replaces ActiveAffects for engine interaction; ActiveAffects remains for serialization.
	MasterAffects []*engine.MasterAffect

	// Combat stats
	THAC0      int // To Hit Armor Class 0
	AC         int // Armor Class
	Hitroll    int // Hitroll bonus (modified by affects, spell-based)
	Damroll    int // Damroll bonus (modified by affects, spell-based)
	DamageRoll combat.DiceRoll
	Position   int // Current position (standing, fighting, etc.)

	// Inventory and equipment
	Inventory *Inventory
	Equipment *Equipment

	// Position
	RoomVNum int // Current room

	// State
	ConnectedAt time.Time
	LastActive  time.Time

	// DP-943: atomic guard to make player death idempotent under concurrent kills.
	dying atomic.Bool

	// Birth — Unix timestamp of character creation (ch->player.time.birth).
	// Used by Age() to calculate character age in MUD years.
	Birth int64

	// PlayedDuration — total accumulated play time in real seconds (ch->player.time.played).
	// Updated on disconnect: PlayedDuration += time.Since(ConnectedAt).
	// Used by PlayingTime() for formatted play-time display.
	PlayedDuration int64

	Fighting string // Name of character being fought

	// Conditions: hunger/thirst/drunk — from limits.c
	// Range: -1 (gone) to 24 (full); clamped 0-48 in original gain_condition
	Hunger int
	Thirst int
	Drunk  int

	// Hometown index — 0=invalid sentinel, 1=Kir Drax'in, 2=Kir-Oshi, 3=Alaozar
	// Source: spec_procs3.c specReceptionist
	Hometown int

	// Alignment: -1000 (evil) to +1000 (good), 0 = neutral
	// Source: structs.h:930, utils.h:454-456
	// IS_GOOD: >= 350, IS_EVIL: <= -350, IS_NEUTRAL: between
	Alignment int

	// Skills: map of skill name → proficiency (0-100)
	// Populated by DoStart() and advance_level(). Used by Phase 3 Lua scripts.
	SkillManager *engine.SkillManager

	// Group/follow state
	// Source: act.movement.c (ch->master), structs.h AFF_GROUP flag
	Following string // Name of player being followed (ch->master in original)
	InGroup   bool   // Whether in a group (AFF_GROUP flag in original)

	// Mount state — from src/utils.c
	MountName string // Name of mount mob being ridden

	// WolfBaseMaxHP stores the player's MaxHealth before werewolf transformation.
	// Restored when the player reverts to human form to prevent the HP exploit.
	WolfBaseMaxHP int

	// Stabled mount state — from src/spec_procs2.c stableboy
	MountRentTime int64 // Unix timestamp when mount was stabled
	MountVNum     int   // VNum of stabled mount (0 = none)
	MountCostDay  int   // Gold per day to keep mount stabled

	// Player flags bitmask — PLR_* constants from structs.h
	// Bit N corresponds to PLR flag N (e.g. PLR_WEREWOLF=16, PLR_VAMPIRE=17).
	// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
	Flags uint64

	// worldRef holds a reference to the World this player belongs to.
	// Used by SendMessage to route through the session layer's MessageSink.
	// Set when the player is added to the world via AddPlayer.
	worldRef *World

	// AFK state
	AFK        bool   // Player is away from keyboard
	AFKMessage string // Optional AFK message
	lastTeller string // Runtime reply target; not persisted in the save format.

	// Character title and description
	Title       string // Character title (shown in who list)
	Description string

	// Poof messages — immortals only, persistent across logins
	PoofIn  string
	PoofOut string // Character description (shown on examine)

	// Prompt settings
	PromptOn  bool   // Whether to show a prompt
	PromptStr string // Custom prompt format (%h/%H hp, %m/%M mana, %v/%V mv)

	// Misc stats — act.other.c
	WimpLevel int `json:"wimp_level"`
	Kills     int `json:"kills"`
	PKs       int `json:"pks"`
	Deaths    int `json:"deaths"`

	// Auto-exit display toggle
	AutoExit  bool // Show exits automatically in room descriptions
	HolyLight bool // Can see in the dark (PRF_HOLYLIGHT)

	// WaitState is the WAIT_STATE cooldown in game-loop pulses (heartbeat
	// ticks). SetWaitState stores rounds*PULSE_VIOLENCE here; the per-pulse
	// drain in the heartbeat decrements it (port of comm.c:603).
	WaitState int
	RoomFlags bool // Show room vnums/sector in room descriptions (PRF_ROOMFLAGS)

	// AutoGold indicates the player auto-loots gold from killed victims (PRF_AUTOGOLD = 24).
	// Source: structs.h:#define PRF_AUTOGOLD 24
	AutoGold bool

	// AutoSplit indicates the player splits gold among group members (PRF_AUTOSPLIT = 25).
	// Source: structs.h:#define PRF_AUTOSPLIT 25
	AutoSplit bool

	// NoBroadcast indicates the player has toggled off global broadcasts (PRF_NOBROAD).
	NoBroadcast bool

	// Known spells: map of spell name → level learned
	SpellMap map[string]int

	// Tattoo — from tattoo.c
	Tattoo    int // tattoo type constant (TATTOO_*)
	TatTimer  int // hours remaining before tattoo can be used again
	IdleTimer int // ticks of inactivity — limits.c check_idling()
	WasInRoom int // previous room before void pull — limits.c GET_WAS_IN()

	// Ignore list: map of player names the player is ignoring
	IgnoredPlayers map[string]bool

	// Aliases — from src/alias.c
	// Per-player command aliases stored in data/aliases/
	Aliases []Alias

	// JailTimer — ticks remaining in jail (0 = not jailed).
	// Decremented each PointUpdate tick. When it reaches 0,
	// the player is auto-released to MortalStartRoom.
	// Source: structs.h GET_JAIL_TIMER / limits.c point_update()
	JailTimer int

	// LastDeath — timestamp of last death (unix time).
	// Used by dream.c for nightmare progression.
	LastDeath int64

	// FreezeLevel records the level of the God who froze this player
	// (C GET_FREEZE_LEV; act.wizard.c:2149). Thaw consults it to stop a
	// lower-level God from un-freezing a higher-level God's freeze. In-memory
	// only (json:"-") — not part of the save format.
	FreezeLevel int `json:"-"`
}

// NewPlayer creates a new player with default stats (no class/race yet).
// For new characters, call NewCharacter instead.
func NewPlayer(id int, name string, roomVNum int) *Player {
	now := time.Now()
	player := &Player{
		ID:           id,
		Name:         name,
		RoomVNum:     roomVNum,
		Health:       100,
		MaxHealth:    100,
		Mana:         100,
		MaxMana:      100,
		Level:        1,
		Exp:          0,
		Strength:     10,                                         // Default strength
		THAC0:        20,                                         // Default THAC0
		AC:           10,                                         // Default AC
		DamageRoll:   combat.DiceRoll{Num: 1, Sides: 4, Plus: 0}, // 1d4
		Position:     8,                                          // POS_STANDING
		ConnectedAt:  now,
		LastActive:   now,
		Birth:        now.Unix(), // character creation timestamp
		Fighting:     "",         // Not fighting anyone
		AFK:          false,
		AFKMessage:   "",
		AutoGold:     false, // Autogold off by default
		AutoSplit:    false, // Autosplit off by default
		Alignment:    0,     // Neutral by default
		SkillManager: engine.NewSkillManager(),
		AutoExit:     true, // Default to on, like PRF_AUTOEXIT in original
		WaitState:    0,
		JailTimer:    0,

		SpellMap: make(map[string]int),
	}

	// Initialize race-hate slots to empty (-1) per src/db.c.
	for i := range player.RaceHates {
		player.RaceHates[i] = -1
	}

	// Initialize inventory and equipment
	player.Inventory = NewInventory()
	player.Equipment = NewEquipment()
	player.Equipment.OwnerName = player.Name
	// Set default capacity (will be updated when stats are set)
	player.Inventory.SetCapacity(10, 0, 10, 1) // Default STR=10, DEX=10, level=1

	// Start fully fed/hydrated/sober — limits.c. NewCharacter overrides these
	// with rolled values, but bare NewPlayer sessions need valid conditions too.
	player.Hunger = 36
	player.Thirst = 36
	player.Drunk = 0
	player.Conditions[CondFull] = 36
	player.Conditions[CondThirst] = 36
	player.Conditions[CondDrunk] = 0

	return player
}

// NewCharacter creates a brand new level 1 character with class/race and rolled stats.
// Implements do_start() from class.c — call this on first login.
func NewCharacter(id int, name string, class, race int) *Player {
	return newCharacter(id, name, class, race, 0, RollRealAbils(class, race))
}

// NewCharacterWithStats creates a level-1 character from the stats and sex
// already accepted during nanny character creation. C rolls abilities before
// do_start(); consuming another roll here would shift every later RNG draw.
func NewCharacterWithStats(id int, name string, class, race, sex int, stats CharStats) *Player {
	return newCharacter(id, name, class, race, sex, stats)
}

func newCharacter(id int, name string, class, race, sex int, stats CharStats) *Player {
	p := NewPlayer(id, name, MortalStartRoom)
	p.Class = class
	p.Race = race
	p.Sex = sex
	if class >= 0 && class < len(Titles) {
		SetTitle(p, Titles[class])
	} else {
		SetTitle(p, "the Adventurer")
	}
	p.Stats = stats
	p.Strength = stats.Str

	// do_start(): level 1, 1 exp, 10 base HP, 100 mana — from class.c line 538
	p.Level = 1
	p.Exp = 1
	p.MaxHealth = 10
	p.Health = 10
	p.MaxMana = 100
	p.Mana = 100
	p.MaxMove = 82
	p.Move = 82
	p.AC = 100

	// Start fully fed/hydrated/sober — limits.c
	p.Hunger = 36
	p.Thirst = 36
	p.Drunk = 0
	p.Conditions[CondFull] = 36
	p.Conditions[CondThirst] = 36
	p.Conditions[CondDrunk] = 0

	// Wimp level: HP threshold for auto-flee — class.c:588
	p.WimpLevel = 5

	// Default preference flags from do_start() — class.c:585-589. These set the
	// fresh-character PRF defaults that do_toggle's grid (act.informative.c)
	// reports as "ON": Hit Pnt / Move / Mana display, plus autoexits. AUTOEXIT is
	// already ON via NewPlayer.AutoExit; the three display bits live in the PRF
	// bitmask (PrfDisphp/Dispmmana/Dispmove), which the toggle grid reads, so set
	// them here. Without these the oracle do_toggle grid diverges from C on a
	// fresh mortal (all three display as OFF in Go vs ON in C).
	p.SetPlrFlag(PrfDisphp, true)
	p.SetPlrFlag(PrfDispmmana, true)
	p.SetPlrFlag(PrfDispmove, true)

	// Starting practices — class.c:590
	p.Practices = 2

	// Random height/weight by sex — db.c:3041-3047
	if p.Sex == 0 { // SEX_MALE = 0
		p.Weight = dprng.Number(120, 180) // 120-180
		p.Height = dprng.Number(160, 200) // 160-200
	} else {
		p.Weight = dprng.Number(100, 160) // 100-160
		p.Height = dprng.Number(150, 180) // 150-180
	}

	// THAC0 from class table
	if class >= 0 && class < 12 {
		p.THAC0 = thaco[class][1]
	}

	// NOTE: advance_level() is NOT called here. C calls do_start()→
	// advance_level() only `if (!GET_LEVEL(ch))` (interpreter.c:2214), i.e. for
	// a real level-1 mortal — never for the first-player God (already LVL_IMPL)
	// nor a DB-loaded character (level restored from the record). Calling it
	// unconditionally in the constructor consumed 2 phantom RNG draws on the God
	// path (DP-1212), desyncing the shared stream by +2 vs C. Callers that need
	// the level-1 HP/move bonus (the char-creation mortal path) must call
	// AdvanceLevel() explicitly after deciding God-ness. The God path
	// (BootstrapFirstPlayerGod) and the DB-load path (RecordToPlayer) set their
	// own stats and must NOT call it.

	// Set inventory capacity and carry weight based on STR, DEX and level
	// CAN_CARRY_N = 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)
	// CAN_CARRY_W = str_app[str].carry_w
	p.Inventory.SetCapacity(p.Stats.Str, p.Stats.StrAdd, p.Stats.Dex, p.Level)

	// Initialize default skills
	p.SkillManager.InitializeDefaultSkills()

	return p
}

// IsDying reports whether the player is currently in the death-handling path.
// Used by DP-943 idempotency guard.
func (p *Player) IsDying() bool {
	return p.dying.Load()
}

// thaco local reference for player creation
// Full table lives in pkg/combat/formulas.go
var thaco = [12][41]int{
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	{100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	{100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	{100, 20, 20, 19, 18, 18, 17, 16, 16, 16, 15, 15, 14, 14, 14, 13, 12, 12, 10, 10, 9, 9, 8, 8, 7, 7, 6, 5, 5, 4, 4, 3, 3, 3, 2, 2, 1, 1, 1, 1, 1},
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
}

// --------------------------------------------------------------------------
// RLock/RUnlock — exported RWMutex read-lock for cross-package access
// (Lock/Unlock are in player_affects.go)
// --------------------------------------------------------------------------

func (p *Player) RLock()   { p.mu.RLock() }
func (p *Player) RUnlock() { p.mu.RUnlock() }

// CarriedWeight returns the total weight of all items carried (inventory + equipment).
func (p *Player) CarriedWeight() int {
	weight := 0
	if p.Inventory != nil {
		weight += p.Inventory.GetWeight()
	}
	if p.Equipment != nil {
		for _, item := range p.Equipment.Slots {
			if item != nil {
				weight += item.GetTotalWeight()
			}
		}
	}
	return weight
}

// MaxCarryWeight returns the maximum weight this player can carry.
// Source: utils.h CAN_CARRY_W(ch) = str_app[STRENGTH_APPLY_INDEX(ch)].carry_w
func (p *Player) MaxCarryWeight() int {
	return combat.CarryWeight(p.GetStr(), p.GetStrAdd())
}

// MaxCarryItems returns CAN_CARRY_N from the player's current stats.
// Source: utils.h CAN_CARRY_N.
func (p *Player) MaxCarryItems() int {
	return 5 + (p.GetDex() >> 1) + (p.GetLevel() >> 1)
}

// MaxWieldWeight returns the maximum weight this player can wield.
// Source: act.item.c:1492 — str_app[STRENGTH_APPLY_INDEX(ch)].wield_w
// Table from constants.c str_app[] (4th column is wield_w):
//
//	STR 0:0, 1:1, 2:2, 3:3, 4:4, 5:5, 6:6, 7:7, 8:8, 9:9, 10:10,
//	11:11, 12:12, 13:13, 14:14, 15:15, 16:16, 17:18, 18:20,
//	19:40, 20:40, 21:40, 22:40, 23:40, 24:40, 25:40,
//	18/01-50:22, 18/51-75:24, 18/76-90:26, 18/91-99:28, 18/100:30
func (p *Player) MaxWieldWeight() int {
	// Index 0-25 = STR 0-25, 26-30 = 18/01-50 through 18/100
	strWield := [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 20, 40, 40, 40, 40, 40, 40, 40, 22, 24, 26, 28, 30}
	str := p.Strength
	if str < 0 {
		return 0
	}
	if str >= len(strWield) {
		str = len(strWield) - 1
	}
	return strWield[str]
}

// HasBoat returns true if the player has a boat item in inventory or equipment.
// C source: act.movement.c has_boat() — checks ITEM_BOAT wear flag.
func (p *Player) HasBoat() bool {
	if p.Inventory == nil {
		return false
	}
	for _, obj := range p.Inventory.Items {
		if obj.Prototype != nil && hasWearFlag(obj.Prototype.WearFlags, 15) { // ITEM_WEAR_BOAT = bit 15
			return true
		}
	}
	return false
}
