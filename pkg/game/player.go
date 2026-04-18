package game

import (
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
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
	Inventory  *Inventory
	Equipment  *Equipment

	// Position
	RoomVNum int // Current room

	// State
	ConnectedAt time.Time
	LastActive  time.Time
	Fighting   string // Name of character being fought

	// Communication
	Send chan []byte // Channel for sending messages to player
}

// NewPlayer creates a new player with default stats (no class/race yet).
// For new characters, call NewCharacter instead.
func NewPlayer(id int, name string, roomVNum int) *Player {
	now := time.Now()
	player := &Player{
		ID:          id,
		Name:        name,
		RoomVNum:    roomVNum,
		Health:      100,
		MaxHealth:   100,
		Mana:        100,
		MaxMana:     100,
		Level:       1,
		Exp:         0,
		Strength:    10, // Default strength
		THAC0:       20, // Default THAC0
		AC:          10, // Default AC
		DamageRoll:  combat.DiceRoll{Num: 1, Sides: 4, Plus: 0}, // 1d4
		Position:    8, // POS_STANDING
		ConnectedAt: now,
		LastActive:  now,
		Fighting:    "", // Not fighting anyone
		Send:        make(chan []byte, 256),
	}
	
	// Initialize inventory and equipment
	player.Inventory = NewInventory()
	player.Equipment = NewEquipment()
	player.Inventory.SetCapacity(player.Strength)
	
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

	p.Inventory.SetCapacity(p.Strength)
	return p
}

// thaco local reference for player creation
// Full table lives in pkg/combat/formulas.go
var thaco = [12][41]int{
	{100,20,20,20,19,19,19,18,18,18,17,17,17,16,16,16,15,15,15,14,14,14,13,13,13,12,12,12,11,11,11,10,10,10,9,9,9,9,9,9,9},
	{100,20,20,20,18,18,18,16,16,16,14,14,14,12,12,12,10,10,10,8,8,8,6,6,6,4,4,4,2,2,2,1,1,1,1,1,1,1,1,1,1},
	{100,20,20,19,19,18,18,17,17,16,16,15,15,14,13,13,12,12,11,11,10,10,9,9,8,8,7,7,6,6,5,5,4,4,3,3,3,3,3,3,3},
	{100,20,19,18,17,16,15,14,13,12,11,10,9,8,7,6,5,4,3,2,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1},
	{100,20,20,20,19,19,19,18,18,18,17,17,17,16,16,16,15,15,15,14,14,14,13,13,13,12,12,12,11,11,11,10,10,10,9,9,9,9,9,9,9},
	{100,20,20,20,18,18,18,16,16,16,14,14,14,12,12,12,10,10,10,8,8,8,6,6,6,4,4,4,2,2,2,1,1,1,1,1,1,1,1,1,1},
	{100,20,20,19,19,18,18,17,17,16,16,15,15,14,13,13,12,12,11,11,10,10,9,9,8,8,7,7,6,6,5,5,4,4,3,3,3,3,3,3,3},
	{100,20,19,18,17,16,15,14,13,12,11,10,9,8,7,6,5,4,3,2,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1},
	{100,20,20,19,19,18,18,17,17,16,16,15,15,14,13,13,12,12,11,11,10,10,9,9,8,8,7,7,6,6,5,5,4,4,3,3,3,3,3,3,3},
	{100,20,20,19,18,18,17,16,16,16,15,15,14,14,14,13,12,12,10,10,9,9,8,8,7,7,6,5,5,4,4,3,3,3,2,2,1,1,1,1,1},
	{100,20,19,18,17,16,15,14,13,12,11,10,9,8,7,6,5,4,3,2,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1},
	{100,20,20,20,19,19,19,18,18,18,17,17,17,16,16,16,15,15,15,14,14,14,13,13,13,12,12,12,11,11,11,10,10,10,9,9,9,9,9,9,9},
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