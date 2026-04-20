// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// MobInstance represents a spawned mob in the world.
type MobInstance struct {
	// Link to prototype
	Prototype *parser.Mob
	VNum      int

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
	Target    *MobInstance // or Player
	Fighting  bool

	// Memory: names of players this mob remembers attacking it
	// Source: mobact.c:262-285, remember()/forget() in mobact.c:346-407
	Memory []string
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
		Prototype: proto,
		VNum:      proto.VNum,
		RoomVNum:  roomVNum,
		CurrentHP: hp,
		MaxHP:     hp,
		Status:    "standing",
		Inventory: make([]*ObjectInstance, 0),
		Equipment: make(map[int]*ObjectInstance),
		Fighting:  false,
	}

	// Create AI brain
	// mob.Brain = ai.NewBrain(mob) // Temporarily commented out

	return mob
}

// NewMobInstance is an alias for NewMob for compatibility.
func NewMobInstance(proto *parser.Mob, roomVNum int) *MobInstance {
	return NewMob(proto, roomVNum)
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
	return m.RoomVNum
}

// SetRoom sets the mob's current room.
func (m *MobInstance) SetRoom(vnum int) {
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
	m.CurrentHP -= amount
	if m.CurrentHP < 0 {
		m.CurrentHP = 0
		m.Status = "dead"
	}
}

// Heal restores HP to the mob.
func (m *MobInstance) Heal(amount int) {
	m.CurrentHP += amount
	if m.CurrentHP > m.MaxHP {
		m.CurrentHP = m.MaxHP
	}
}

// IsAlive returns true if the mob is alive.
func (m *MobInstance) IsAlive() bool {
	return m.CurrentHP > 0
}

// AddToInventory adds an object to the mob's inventory.
func (m *MobInstance) AddToInventory(obj *ObjectInstance) {
	obj.Carrier = m
	m.Inventory = append(m.Inventory, obj)
}

// RemoveFromInventory removes an object from the mob's inventory.
func (m *MobInstance) RemoveFromInventory(obj *ObjectInstance) bool {
	for i, item := range m.Inventory {
		if item == obj {
			m.Inventory = append(m.Inventory[:i], m.Inventory[i+1:]...)
			obj.Carrier = nil
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

	obj.EquippedOn = m
	obj.EquipPosition = position
	m.Equipment[position] = obj
	return true
}

// UnequipItem removes an equipped object.
func (m *MobInstance) UnequipItem(position int) *ObjectInstance {
	if obj, ok := m.Equipment[position]; ok {
		delete(m.Equipment, position)
		obj.EquippedOn = nil
		obj.EquipPosition = -1
		m.AddToInventory(obj)
		return obj
	}
	return nil
}

// GetAC returns the mob's armor class.
func (m *MobInstance) GetAC() int {
	if m.Prototype != nil {
		return m.Prototype.AC
	}
	return 0
}

// GetLevel returns the mob's level.
func (m *MobInstance) GetLevel() int {
	if m.Prototype != nil {
		return m.Prototype.Level
	}
	return 1
}

// GetDamageRoll returns the damage dice for the mob's attacks.
func (m *MobInstance) GetDamageRoll() combat.DiceRoll {
	if m.Prototype != nil {
		return combat.DiceRoll{
			Num:   m.Prototype.Damage.Num,
			Sides: m.Prototype.Damage.Sides,
			Plus:  m.Prototype.Damage.Plus,
		}
	}
	return combat.DiceRoll{Num: 1, Sides: 4, Plus: 0} // 1d4 default
}

// Combatant interface implementation

// GetTHAC0 returns the mob's THAC0.
func (m *MobInstance) GetTHAC0() int {
	if m.Prototype != nil {
		return m.Prototype.THAC0
	}
	return 20 // Default
}

// GetHP returns the mob's current health.
func (m *MobInstance) GetHP() int {
	return m.CurrentHP
}

// GetMaxHP returns the mob's maximum health.
func (m *MobInstance) GetMaxHP() int {
	return m.MaxHP
}

// IsNPC returns true for mobs.
func (m *MobInstance) IsNPC() bool {
	return true
}

// GetPosition returns the mob's current position.
func (m *MobInstance) GetPosition() int {
	// Convert status string to position constant
	switch m.Status {
	case "dead":
		return 0 // POS_DEAD
	case "sleeping":
		return 4 // POS_SLEEPING
	case "resting":
		return 5 // POS_RESTING
	case "sitting":
		return 6 // POS_SITTING
	case "fighting":
		return 7 // POS_FIGHTING
	case "standing":
		return 8 // POS_STANDING
	default:
		return 8 // Default to standing
	}
}

// GetName returns the mob's short description as its name.
func (m *MobInstance) GetName() string {
	return m.GetShortDesc()
}

// SendMessage sends a message to the mob (no-op for mobs, but needed for interface).
func (m *MobInstance) SendMessage(msg string) {
	// Mobs don't receive messages, but we could log this
}

// SetFighting sets who the mob is fighting.
func (m *MobInstance) SetFighting(target string) {
	m.Status = "fighting"
	// Target is stored as a string name; we'd need a lookup mechanism
	// For now, just mark as fighting
}

// StopFighting clears the fighting state.
func (m *MobInstance) StopFighting() {
	m.Status = "standing"
	m.Fighting = false
}

// GetFighting returns who the mob is fighting (empty string if not fighting).
func (m *MobInstance) GetFighting() string {
	if m.Status == "fighting" {
		return "someone" // Simplified; would need proper target tracking
	}
	return ""
}

// GetClass returns the mob's class (Phase 2c addition)
// Mobs don't have classes in Dark Pawns, return 0 (CLASS_MAGE) as default
// Source: fight.c - mobs don't use class-based THAC0
func (m *MobInstance) GetClass() int {
	return 0 // CLASS_MAGE
}

// GetStr returns the mob's strength (Phase 2c addition)
// Mobs don't have STR stats, return 10 (average) as default
// Source: fight.c - mobs don't use str_app[]
func (m *MobInstance) GetStr() int {
	return 10
}

// GetDex returns the mob's dexterity (Phase 2c addition)
// Mobs don't have DEX stats, return 10 (average) as default
// Source: fight.c - mobs don't use dex_app[]
func (m *MobInstance) GetDex() int {
	return 10
}

// GetInt returns the mob's intelligence (Phase 2c addition)
// Mobs don't have INT stats, return 10 (average) as default
// Source: fight.c - mobs don't use INT for THAC0
func (m *MobInstance) GetInt() int {
	return 10
}

// GetWis returns the mob's wisdom (Phase 2c addition)
// Mobs don't have WIS stats, return 10 (average) as default
// Source: fight.c - mobs don't use WIS for THAC0
func (m *MobInstance) GetWis() int {
	return 10
}

// GetHitroll returns the mob's hitroll bonus (Phase 2c addition)
// Mobs don't have hitroll, return 0 as default
// Source: fight.c - mobs don't use GET_HITROLL()
func (m *MobInstance) GetHitroll() int {
	return 0
}

// GetDamroll returns the mob's damroll bonus (Phase 2c addition)
// Mobs don't have damroll, return 0 as default
// Source: fight.c - mobs don't use GET_DAMROLL()
func (m *MobInstance) GetDamroll() int {
	return 0
}

// GetStrAdd returns the mob's strength add (exceptional strength)
// Mobs don't have exceptional strength, return 0 as default
func (m *MobInstance) GetStrAdd() int {
	return 0
}