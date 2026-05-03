package game

import (
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
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

	Level int
	Exp   int
	Gold  int // Currency, used by Lua scripts
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
	Sex   int // 0=male, 1=female, 2=neutral (matching C SEX_* constants)
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

	// Hometown index — 0=default, 1=Midgaard, 2=Thalos, 3=New Thalos
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

	// Character title and description
	Title       string // Character title (shown in who list)
	Description string // Character description (shown on examine)

	// Prompt settings
	PromptOn  bool   // Whether to show a prompt
	PromptStr string // Custom prompt format (%h/%H hp, %m/%M mana, %v/%V mv)

	// Misc stats — act.other.c
	WimpLevel int `json:"wimp_level"`
	Kills     int `json:"kills"`
	PKs       int `json:"pks"`
	Deaths    int `json:"deaths"`

	// Auto-exit display toggle
	AutoExit   bool // Show exits automatically in room descriptions
	HolyLight  bool // Can see in the dark (PRF_HOLYLIGHT)

	// C-10: WAIT_STATE cooldown in PULSE_VIOLENCE ticks (1 tick = 2 seconds).
	WaitState int
	// C-11: parry stance toggle
	Parrying bool
	RoomFlags  bool // Show room vnums/sector in room descriptions (PRF_ROOMFLAGS)

	// AutoGold indicates the player auto-loots gold from killed victims (PRF_AUTOGOLD = 24).
	// Source: structs.h:#define PRF_AUTOGOLD 24
	AutoGold  bool

	// AutoSplit indicates the player splits gold among group members (PRF_AUTOSPLIT = 25).
	// Source: structs.h:#define PRF_AUTOSPLIT 25
	AutoSplit bool

	// NoBroadcast indicates the player has toggled off global broadcasts (PRF_NOBROAD).
	NoBroadcast bool

	// Known spells: map of spell name → level learned
	SpellMap map[string]int

	// Tattoo — from tattoo.c
	Tattoo   int // tattoo type constant (TATTOO_*)
	TatTimer int // hours remaining before tattoo can be used again
	IdleTimer int // ticks of inactivity — limits.c check_idling()
	WasInRoom int // previous room before void pull — limits.c GET_WAS_IN()

	// Ignore list: map of player names the player is ignoring
	IgnoredPlayers map[string]bool

	// Aliases — from src/alias.c
	// Per-player command aliases stored in data/aliases/
	Aliases []Alias

	// LastDeath — timestamp of last death (unix time).
	// Used by dream.c for nightmare progression.
	LastDeath int64
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
		Birth:       now.Unix(), // character creation timestamp
		Fighting:     "", // Not fighting anyone
		AFK:          false,
		AFKMessage:   "",
		AutoGold:     false, // Autogold off by default
		AutoSplit:    false, // Autosplit off by default
		Alignment:    0, // Neutral by default
		SkillManager: engine.NewSkillManager(),
		AutoExit:     true, // Default to on, like PRF_AUTOEXIT in original
		WaitState:    0,
		Parrying:    false,

		SpellMap: make(map[string]int),
	}

	// Initialize inventory and equipment
	player.Inventory = NewInventory()
	player.Equipment = NewEquipment()
	player.Equipment.OwnerName = player.Name
	// Set default capacity (will be updated when stats are set)
	player.Inventory.SetCapacity(10, 1) // Default DEX=10, level=1

	return player
}

// NewCharacter creates a brand new level 1 character with class/race and rolled stats.
// Implements do_start() from class.c — call this on first login.
func NewCharacter(id int, name string, class, race int) *Player {
	stats := RollRealAbils(class, race)
	p := NewPlayer(id, name, MortalStartRoom)
	p.Class = class
	p.Race = race
	p.Stats = stats
	p.Strength = stats.Str

	// do_start(): level 1, 1 exp, 10 base HP, 100 mana — from class.c line 538
	p.Level = 1
	p.Exp = 1
	p.MaxHealth = 10
	p.Health = 10
	p.MaxMana = 100
	p.Mana = 100
	p.MaxMove = 100
	p.Move = 100

	// Start fully fed/hydrated/sober — limits.c
	p.Hunger = 24
	p.Thirst = 24
	p.Drunk = 0

	// THAC0 from class table
	if class >= 0 && class < 12 {
		p.THAC0 = thaco[class][1]
	}

	// Call advance_level() for level 1 HP bonus (class.c:600-720)
	// This adds con_app[con].hitp + class-specific random HP
	p.AdvanceLevel()

	// Set inventory capacity based on DEX and level
	// Formula: 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)
	p.Inventory.SetCapacity(p.Stats.Dex, p.Level)

	// Initialize default skills
	p.SkillManager.InitializeDefaultSkills()

	// Set starting skill levels based on class — from class.c do_start()
	// Thieves and Assassins get starting thief skills
	if p.Class == ClassThief || p.Class == ClassAssassin {
		p.SetSkill("sneak", 10)
		p.SetSkill("hide", 5)
		p.SetSkill("steal", 15)
		p.SetSkill("backstab", 10)
		p.SetSkill("pick_lock", 10)
	}
	// Kender get bonus steal
	if p.Race == RaceKender {
		p.SetSkill("steal", 25)
	}
	// All classes get kick at level 1
	p.SetSkill("kick", 10)
	// Warrior-types get bash and rescue
	if p.Class == ClassWarrior || p.Class == ClassPaladin || p.Class == ClassRanger {
		p.SetSkill("bash", 10)
		p.SetSkill("rescue", 10)
	}

	return p
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
// Table from constants.c str_app[] (4th column is carry_w):
//   STR 0:0, 1:3, 2:3, 3:10, 4:25, 5:55, 6:80, 7:90, 8:100, 9:100,
//   STR 10:115, 11:115, 12:140, 13:140, 14:170, 15:170, 16:195, 17:220, 18:255
func (p *Player) MaxCarryWeight() int {
	strCarry := [...]int{0, 3, 3, 10, 25, 55, 80, 90, 100, 100, 115, 115, 140, 140, 170, 170, 195, 220, 255}
	str := p.Strength
	if str < 0 {
		return 0
	}
	if str >= len(strCarry) {
		str = len(strCarry) - 1
	}
	return strCarry[str]
}

// UpdateActivity marks the player as active.
