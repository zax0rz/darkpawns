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
	Level     int
	Exp       int
	Gold      int // Currency, used by Lua scripts
	Strength  int // For inventory capacity

	// Character identity — from do_start()/roll_real_abils() in class.c
	Class int
	Race  int
	Stats CharStats

	// Combat stats
	THAC0      int // To Hit Armor Class 0
	AC         int // Armor Class
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
	Fighting    string // Name of character being fought

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

	// Player flags bitmask — PLR_* constants from structs.h
	// Bit N corresponds to PLR flag N (e.g. PLR_WEREWOLF=16, PLR_VAMPIRE=17).
	// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
	Flags uint64

	// Communication
	Send chan []byte // Channel for sending messages to player
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
		Fighting:     "", // Not fighting anyone
		Send:         make(chan []byte, 256),
		Alignment:    0, // Neutral by default
		SkillManager: engine.NewSkillManager(),
	}

	// Initialize inventory and equipment
	player.Inventory = NewInventory()
	player.Equipment = NewEquipment()
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

// UpdateActivity marks the player as active.
func (p *Player) UpdateActivity() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastActive = time.Now()
}

// SetRoom changes the player's current room.
func (p *Player) SetRoom(vnum int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RoomVNum = vnum
}

// GetRoom returns the player's current room VNum.
func (p *Player) GetRoom() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RoomVNum
}

// Combatant interface implementation

// GetLevel returns the player's level.
func (p *Player) GetLevel() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Level
}

// GetTHAC0 returns the player's THAC0.
func (p *Player) GetTHAC0() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.THAC0
}

// GetAC returns the player's Armor Class including equipment bonuses.
func (p *Player) GetAC() int {
	p.mu.RLock()
	baseAC := p.AC
	p.mu.RUnlock()

	// Add equipment AC bonus
	if p.Equipment != nil {
		baseAC -= p.Equipment.GetArmorClass() // Lower AC is better
	}

	return baseAC
}

// GetHP returns the player's current health.
func (p *Player) GetHP() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

// GetMaxHP returns the player's maximum health.
func (p *Player) GetMaxHP() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MaxHealth
}

// GetDamageRoll returns the player's damage dice including weapon.
func (p *Player) GetDamageRoll() combat.DiceRoll {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if wielding a weapon
	if p.Equipment != nil {
		num, sides := p.Equipment.GetWeaponDamage()
		if num > 0 && sides > 0 {
			return combat.DiceRoll{Num: num, Sides: sides, Plus: 0}
		}
	}

	// Return bare hands damage
	return p.DamageRoll
}

// IsNPC returns false for players.
func (p *Player) IsNPC() bool {
	return false
}

// GetPosition returns the player's current position.
func (p *Player) GetPosition() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Position
}

// SetPosition sets the player's position.
func (p *Player) SetPosition(pos int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Position = pos
}

// GetFighting returns who the player is fighting.
func (p *Player) GetFighting() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Fighting
}

// SetFighting sets who the player is fighting.
func (p *Player) SetFighting(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Fighting = target
}

// TakeDamage applies damage to the player.
func (p *Player) TakeDamage(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health -= amount
	if p.Health < 0 {
		p.Health = 0
	}
}

// GetAlignment returns the player's alignment score.
func (p *Player) GetAlignment() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Alignment
}

// IsGood returns true if alignment >= 350 (utils.h:454)
func (p *Player) IsGood() bool { return p.GetAlignment() >= 350 }

// IsEvil returns true if alignment <= -350 (utils.h:455)
func (p *Player) IsEvil() bool { return p.GetAlignment() <= -350 }

// IsNeutral returns true if not good and not evil (utils.h:456)
func (p *Player) IsNeutral() bool { return !p.IsGood() && !p.IsEvil() }

// SetSkill sets a skill level (0-100).
func (p *Player) SetSkill(name string, level int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.SkillManager == nil {
		p.SkillManager = engine.NewSkillManager()
	}
	// Create or update skill in manager
	skill := p.SkillManager.GetSkill(name)
	if skill == nil {
		// Create new skill with default values
		skill = engine.NewSkill(name, name, engine.SkillTypeUtility, 3)
		p.SkillManager.RegisterSkill(skill)
	}
	skill.Learned = true
	skill.Level = level
	if level > 0 {
		skill.Learned = true
	}
}

// GetSkill returns a skill level (0 if not set).
func (p *Player) GetSkill(name string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.SkillManager == nil {
		return 0
	}
	return p.SkillManager.GetSkillLevel(name)
}

// LoseExp deducts experience from the player, floored at 0.
func (p *Player) LoseExp(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Exp -= amount
	if p.Exp < 0 {
		p.Exp = 0
	}
}

// Heal restores health to the player.
func (p *Player) Heal(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health += amount
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
}

// GetName returns the player's name.
func (p *Player) GetName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Name
}

// SendMessage sends a message to the player.
func (p *Player) SendMessage(msg string) {
	select {
	case p.Send <- []byte(msg):
	default:
		// Channel full, drop message
	}
}

// StopFighting clears the fighting target.
func (p *Player) StopFighting() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Fighting = ""
}

// IsFighting returns true if the player is in combat.
func (p *Player) IsFighting() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Fighting != ""
}

// GetClass returns the player's class (Phase 2c addition)
// Source: fight.c uses GET_CLASS(ch) macro
func (p *Player) GetClass() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Class
}

// GetStr returns the player's strength (Phase 2c addition)
// Source: fight.c uses GET_STR(ch) macro
func (p *Player) GetStr() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Str
}

// GetDex returns the player's dexterity (Phase 2c addition)
// Source: fight.c uses GET_DEX(ch) macro
func (p *Player) GetDex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Dex
}

// GetInt returns the player's intelligence (Phase 2c addition)
// Source: fight.c uses GET_INT(ch) macro
func (p *Player) GetInt() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Int
}

// GetWis returns the player's wisdom (Phase 2c addition)
// Source: fight.c uses GET_WIS(ch) macro
func (p *Player) GetWis() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Wis
}

// GetHitroll returns the player's hitroll bonus (Phase 2c addition)
// Source: fight.c uses GET_HITROLL(ch) macro
// TODO: Phase 3 - implement equipment bonuses
func (p *Player) GetHitroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0 // No equipment bonuses yet
}

// GetDamroll returns the player's damroll bonus (Phase 2c addition)
// Source: fight.c uses GET_DAMROLL(ch) macro
// TODO: Phase 3 - implement equipment bonuses
func (p *Player) GetDamroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0 // No equipment bonuses yet
}

// GetStrAdd returns the player's strength add (exceptional strength)
// Source: utils.h GET_ADD(ch) macro
func (p *Player) GetStrAdd() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.StrAdd
}

// Scripting interface implementations

func (p *Player) GetID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ID
}

func (p *Player) GetHealth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

func (p *Player) SetHealth(health int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health = health
}

func (p *Player) GetMaxHealth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MaxHealth
}

func (p *Player) GetGold() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Gold
}

func (p *Player) SetGold(gold int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Gold = gold
}

func (p *Player) GetRace() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Race
}

func (p *Player) GetRoomVNum() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RoomVNum
}

// GetFlags returns the raw PLR flags bitmask.
// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
func (p *Player) GetFlags() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Flags
}

// SetPlrFlag sets or clears PLR flag bit N on this player.
// Source: utils.h PLR_FLAGS() macro.
func (p *Player) SetPlrFlag(bit int, val bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if bit < 0 || bit >= 64 {
		return
	}
	if val {
		p.Flags |= 1 << uint(bit)
	} else {
		p.Flags &^= 1 << uint(bit)
	}
}
