// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// MobInstance represents a spawned mob in the world.
type MobInstance struct {
	mu sync.RWMutex

	// Link to prototype
	Prototype *parser.Mob
	VNum      int
	ID        int    // World-assigned instance ID

	// Current state
	RoomVNum  int // -1 if not in a room (carried, etc.)
	CurrentHP int
	MaxHP     int
	Status    string // "standing", "sleeping", "fighting", etc.

	// AI
	// Brain *ai.Brain // Temporarily commented out to fix circular import

	// Inventory and equipment
	Inventory []*ObjectInstance
	Equipment map[int]*ObjectInstance // key: equip position

	// Combat state
	Target         *MobInstance // or Player
	Fighting       bool
	FightingTarget string // Name of the target being fought

	// Memory: names of players this mob remembers attacking it
	// Source: mobact.c:262-285, remember()/forget() in mobact.c:346-407
	Memory []string

	// MountRider: name of player riding this mount — from src/utils.c
	MountRider string

	// Hunting: name of player being hunted — from src/utils.c
	Hunting   string
	HuntingID string

	// CustomData stores arbitrary per-instance data (e.g., damroll bonus for brain eater)
	CustomData map[string]interface{}

	// Runtime state — typed replacement for CustomData (e.g., damroll_bonus)
	Runtime MobRuntimeState

	// Affect flags bitmask — same bit positions as AFF_* constants used by Player
	Affects uint64

	// Following — name of player this mob follows (for charmed pets, etc.)
	Following string
}

// NewMob creates a new mob instance from a prototype.
// This is called NewMob to match the existing code in world.go
func NewMob(proto *parser.Mob, roomVNum int) *MobInstance {
	// Calculate HP from dice roll
	hp := 0
	if proto.HP.Num > 0 && proto.HP.Sides > 0 {
		// Simple average calculation for now
		hp = (proto.HP.Num * (proto.HP.Sides + 1) / 2) + proto.HP.Plus
	} else {
		hp = 100 // Default
	}

	mob := &MobInstance{
		Prototype:      proto,
		VNum:           proto.VNum,
		RoomVNum:       roomVNum,
		CurrentHP:      hp,
		MaxHP:          hp,
		Status:         "standing",
		Inventory:      make([]*ObjectInstance, 0),
		Equipment:      make(map[int]*ObjectInstance),
		Fighting:       false,
		FightingTarget: "",
		CustomData:     make(map[string]interface{}),
		Runtime:        MobRuntimeState{},
	}

	// Create AI brain
	// mob.Brain = ai.NewBrain(mob) // Temporarily commented out

	return mob
}

// GetID returns the world-assigned instance ID.
func (m *MobInstance) GetID() int {
	return m.ID
}

// NewMobInstance is an alias for NewMob for compatibility.
func NewMobInstance(proto *parser.Mob, roomVNum int) *MobInstance {
	return NewMob(proto, roomVNum)
}

// GetSex returns the mob's sex (0=male, 1=female, 2=neutral).
func (m *MobInstance) GetSex() int {
	if m.Prototype != nil {
		return m.Prototype.Sex
	}
	return 2 // neutral default
}

// GetShortDesc returns the mob's short description.
func (m *MobInstance) GetShortDesc() string {
	if m.Prototype != nil {
		return m.Prototype.ShortDesc
	}
	return "a generic mob"
}

// GetRoom returns the mob's current room.
func (m *MobInstance) GetRoom() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RoomVNum
}

// SetRoom sets the mob's current room.
func (m *MobInstance) SetRoom(vnum int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RoomVNum = vnum
}

// HasFlag checks if the mob has a specific flag.
// This is a simplified implementation.
func (m *MobInstance) HasFlag(flag string) bool {
	// In a real implementation, we'd check the prototype's action flags
	// For now, return false for all flags
	return false
}

// Attack makes the mob attack a player.
func (m *MobInstance) Attack(player *Player, world *World) error {
	// Simple attack implementation
	damage := 10 // Default damage
	player.TakeDamage(damage)

	// Send messages
	player.Send <- []byte(fmt.Sprintf("%s attacks you for %d damage!\n", m.GetShortDesc(), damage))

	// Notify other players in the room
	players := world.GetPlayersInRoom(m.RoomVNum)
	for _, p := range players {
		if p != player {
			p.Send <- []byte(fmt.Sprintf("%s attacks %s!\n", m.GetShortDesc(), player.Name))
		}
	}

	return nil
}

// Update runs the mob's AI update.
func (m *MobInstance) Update(world *World) error {
	// if m.Brain != nil {
	// 	return m.Brain.Update(m, world)
	// }
	return nil
}

// GetLongDesc returns the mob's long description.
func (m *MobInstance) GetLongDesc() string {
	if m.Prototype != nil {
		return m.Prototype.LongDesc
	}
	return "A generic mob is here."
}

// TakeDamage applies damage to the mob.
func (m *MobInstance) TakeDamage(amount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentHP -= amount
	if m.CurrentHP < 0 {
		m.CurrentHP = 0
		m.Status = "dead"
	}
}

// Heal restores HP to the mob.
func (m *MobInstance) Heal(amount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentHP += amount
	if m.CurrentHP > m.MaxHP {
		m.CurrentHP = m.MaxHP
	}
}

// IsAlive returns true if the mob is alive.
func (m *MobInstance) IsAlive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentHP > 0
}

// AddToInventory adds an object to the mob's inventory.
func (m *MobInstance) AddToInventory(obj *ObjectInstance) {
	obj.Location = LocInventoryMob(m.GetID())
	m.Inventory = append(m.Inventory, obj)
}

// RemoveFromInventory removes an object from the mob's inventory.
func (m *MobInstance) RemoveFromInventory(obj *ObjectInstance) bool {
	for i, item := range m.Inventory {
		if item == obj {
			m.Inventory = append(m.Inventory[:i], m.Inventory[i+1:]...)
			obj.Location = LocNowhere()
			return true
		}
	}
	return false
}

// EquipItem equips an object on the mob.
func (m *MobInstance) EquipItem(obj *ObjectInstance, position int) bool {
	// First remove from inventory if present
	removed := m.RemoveFromInventory(obj)
	if !removed {
		// Object wasn't in inventory, maybe it was on the ground
		// For now, just equip it
	}

	obj.Location = LocEquippedMob(m.GetID(), EquipmentSlot(position))
	m.Equipment[position] = obj
	return true
}

// UnequipItem removes an equipped object.
func (m *MobInstance) UnequipItem(position int) *ObjectInstance {
	if obj, ok := m.Equipment[position]; ok {
		delete(m.Equipment, position)
		obj.Location = LocNowhere()
		m.AddToInventory(obj)
		return obj
	}
	return nil
}

// GetAC returns the mob's armor class.
func (m *MobInstance) GetAC() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Prototype != nil {
		return m.Prototype.AC
	}
	return 0
}

// GetLevel returns the mob's level.
func (m *MobInstance) GetLevel() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Prototype != nil {
		return m.Prototype.Level
	}
	return 1
}

// GetDamageRoll returns the damage dice for the mob's attacks.
func (m *MobInstance) GetDamageRoll() combat.DiceRoll {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Prototype != nil {
		return combat.DiceRoll{
			Num:   m.Prototype.Damage.Num,
			Sides: m.Prototype.Damage.Sides,
			Plus:  m.Prototype.Damage.Plus,
		}
	}
	return combat.DiceRoll{Num: 0, Sides: 0, Plus: 0} // bare hands
}

// Combatant interface implementation

// GetTHAC0 returns the mob's THAC0.
func (m *MobInstance) GetTHAC0() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Prototype != nil {
		return m.Prototype.THAC0
	}
	return 20 // Default
}

// GetHP returns the mob's current health.
func (m *MobInstance) GetHP() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentHP
}

// GetMaxHP returns the mob's maximum health.
func (m *MobInstance) GetMaxHP() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MaxHP
}

// IsNPC returns true for mobs.
func (m *MobInstance) IsNPC() bool {
	return true
}

// GetStatus returns the mob's status string.
func (m *MobInstance) GetStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Status
}

// SetStatus sets the mob's status string.
func (m *MobInstance) SetStatus(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Status = status
}

// GetPosition returns the mob's current position.
func (m *MobInstance) GetPosition() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Convert status string to position constant
	switch m.Status {
	case "dead":
		return combat.PosDead
	case "sleeping":
		return combat.PosSleeping
	case "resting":
		return combat.PosResting
	case "sitting":
		return combat.PosSitting
	case "fighting":
		return combat.PosFighting
	case "standing":
		return combat.PosStanding
	default:
		return combat.PosStanding // Default to standing
	}
}

// GetName returns the mob's short description as its name.
func (m *MobInstance) GetName() string {
	return m.GetShortDesc()
}

// SendMessage sends a message to the mob (no-op for mobs, but needed for interface).
func (m *MobInstance) SendMessage(msg string) {
	// Mobs don't receive messages
}

// SetFighting sets who the mob is fighting.
func (m *MobInstance) SetFighting(target string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Status = "fighting"
	m.Fighting = true
	m.FightingTarget = target
}

// StopFighting clears the fighting state.
func (m *MobInstance) StopFighting() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Status = "standing"
	m.Fighting = false
	m.FightingTarget = ""
}

// GetFighting returns who the mob is fighting (empty string if not fighting).
func (m *MobInstance) GetFighting() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Fighting || m.Status == "fighting" {
		return m.FightingTarget
	}
	return ""
}

// GetClass returns the mob's class
func (m *MobInstance) GetClass() int {
	return 0 // CLASS_MAGE
}

// GetStr returns the mob's strength
func (m *MobInstance) GetStr() int {
	return 10
}

// GetDex returns the mob's dexterity
func (m *MobInstance) GetDex() int {
	return 10
}

// GetInt returns the mob's intelligence
func (m *MobInstance) GetInt() int {
	return 10
}

// GetWis returns the mob's wisdom
func (m *MobInstance) GetWis() int {
	return 10
}

// GetHitroll returns the mob's hitroll bonus from equipment
// Sums APPLY_HITROLL (location 18) from all equipped items.
func (m *MobInstance) GetHitroll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0
	for _, item := range m.Equipment {
		if item == nil || item.Prototype == nil {
			continue
		}
		for _, aff := range item.Prototype.Affects {
			if aff.Location == 18 {
				total += aff.Modifier
			}
		}
	}
	return total
}

// GetDamroll returns the mob's damroll bonus from equipment
// Sums APPLY_DAMROLL (location 19) from all equipped items.
func (m *MobInstance) GetDamroll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0
	for _, item := range m.Equipment {
		if item == nil || item.Prototype == nil {
			continue
		}
		for _, aff := range item.Prototype.Affects {
			if aff.Location == 19 {
				total += aff.Modifier
			}
		}
	}
	return total
}

// GetStrAdd returns the mob's strength add
func (m *MobInstance) GetStrAdd() int {
	return 0
}

// Scripting interface implementations

func (m *MobInstance) GetVNum() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.VNum
}

func (m *MobInstance) GetHealth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentHP
}

func (m *MobInstance) SetHealth(health int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentHP = health
}

func (m *MobInstance) GetMaxHealth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MaxHP
}

func (m *MobInstance) GetGold() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Prototype != nil {
		return m.Prototype.Gold
	}
	return 0
}

// IsAffected returns true if the given AFF bit is set on the mob.
func (m *MobInstance) IsAffected(bit int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Affects&(1<<bit) != 0
}

// SetAffected sets the given AFF bit on the mob.
func (m *MobInstance) SetAffected(bit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Affects |= (1 << bit)
}

// RemoveAffected clears the given AFF bit on the mob.
func (m *MobInstance) RemoveAffected(bit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Affects &^= (1 << bit)
}

func (m *MobInstance) GetRoomVNum() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RoomVNum
}

func (m *MobInstance) GetPrototype() scripting.ScriptableMobPrototype {
	return m.Prototype
}
