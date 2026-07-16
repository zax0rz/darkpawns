// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// MobInstance represents a spawned mob in the world.
type MobInstance struct {
	mu sync.RWMutex

	// Link to prototype
	Prototype *parser.Mob
	VNum      int
	ID        int // World-assigned instance ID

	// Current state
	alive       atomic.Bool // CRIT-004: fast alive check without acquiring mu
	RoomVNum    int         // -1 if not in a room (carried, etc.)
	CurrentHP   int
	MaxHP       int
	CurrentMana int
	MaxMana     int
	Status      string // "standing", "sleeping", "fighting", etc.
	Level       int    // Level override (0 = use prototype level)

	// AI
	// Brain *ai.Brain // Temporarily commented out to fix circular import

	// Inventory and equipment
	Inventory []*ObjectInstance
	Equipment map[int]*ObjectInstance // key: equip position

	// Combat state
	Target         *MobInstance // or Player
	Fighting       bool
	FightingTarget string // Name of the target being fought
	WaitState      int    // PULSE_VIOLENCE ticks remaining (C GET_MOB_WAIT)

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

	// Ability scores — instance-level values initialized from prototype + level boosts
	// db.c:1053-1062 applies random bonuses for mobs above level 15
	Str   int
	Intel int
	Wis   int
	Dex   int
	Con   int
	Cha   int

	// Gold — instance-level with +/-20% variance from prototype (db.c:1766-1775)
	Gold int

	// RaceHates tracks 5 racial hatred slots (src/structs.h race_hate[5]).
	// Initialized to -1 for all slots; matching a mob's race triggers aggression.
	RaceHates [5]int

	// Affect flags bitmask — same bit positions as AFF_* constants used by Player
	Affects uint64

	// Mob flags bitmask — from src/structs.h MOB_* defines
	Flags uint64

	// Following — name of player this mob follows (for charmed pets, etc.)
	Following string
}

// NewMob creates a new mob instance from a prototype.
// This is called NewMob to match the existing code in world.go
func NewMob(proto *parser.Mob, roomVNum int) *MobInstance {
	// Roll HP from the mob's hit dice — C read_mobile() rolls dice(hp_num, hp_size)
	// per instance (db.c), consuming hp_num shared-PRNG draws. A prior "average"
	// shortcut drew nothing, desyncing the seeded stream on every mob spawn.
	hp := 0
	if proto.HP.Num > 0 && proto.HP.Sides > 0 {
		hp = dprng.Dice(proto.HP.Num, proto.HP.Sides) + proto.HP.Plus
	} else {
		hp = 100 // Default
	}

	// Initialize ability scores from prototype — db.c:1053-1062
	str := proto.Str
	intel := proto.Int
	wis := proto.Wis
	dex := proto.Dex
	con := proto.Con
	cha := proto.Cha

	// Random stat boosts for mobs above level 15 — db.c:1053-1062
	if proto.Level > 15 {
		statmod := proto.Level - 15
		// #nosec G404 — game RNG, not cryptographic
		str += min(dprng.Number(0, statmod), 7)
		// #nosec G404
		intel += min(dprng.Number(0, statmod), 7)
		// #nosec G404
		wis += min(dprng.Number(0, statmod), 7)
		// #nosec G404
		dex += min(dprng.Number(0, statmod), 7)
		// #nosec G404
		con += min(dprng.Number(0, statmod), 7)
		// #nosec G404
		cha += min(dprng.Number(0, statmod), 7)
	}

	// Gold variance +/-(1-20%) — db.c:1766-1774. C draws number(0,1) (the sign
	// coin-flip) BEFORE number(1,20) (the percentage), and the number(1,20) is
	// drawn inside the taken branch. Draw order is law: the two draws must match
	// C's sequence or the shared stream desyncs / mob gold values diverge.
	gold := proto.Gold
	if gold > 0 {
		// #nosec G404
		if dprng.Number(0, 1) == 0 {
			// #nosec G404
			gold += dprng.Number(1, 20) * gold / 100
		} else {
			// #nosec G404
			gold -= dprng.Number(1, 20) * gold / 100
		}
		if gold < 0 {
			gold = 0
		}
	}

	mob := &MobInstance{
		Prototype:      proto,
		VNum:           proto.VNum,
		RoomVNum:       roomVNum,
		CurrentHP:      hp,
		MaxHP:          hp,
		CurrentMana:    100,
		MaxMana:        100,
		Status:         "standing",
		Inventory:      make([]*ObjectInstance, 0),
		Equipment:      make(map[int]*ObjectInstance),
		Fighting:       false,
		FightingTarget: "",
		Memory:         make([]string, 0),
		CustomData:     make(map[string]interface{}),
		Runtime:        MobRuntimeState{},
		Str:            str,
		Intel:          intel,
		Wis:            wis,
		Dex:            dex,
		Con:            con,
		Cha:            cha,
		Gold:           gold,
	}

	mob.alive.Store(true)

	// Initialize race-hate slots to empty (-1) per src/db.c.
	for i := range mob.RaceHates {
		mob.RaceHates[i] = -1
	}

	// Create AI brain
	// mob.Brain = ai.NewBrain(mob) // Temporarily commented out

	return mob
}

// SetAlive marks the mob as alive or dead.
func (m *MobInstance) SetAlive(v bool) {
	m.alive.Store(v)
}

// GetID returns the world-assigned instance ID.
func (m *MobInstance) GetID() int {
	return m.ID
}

// NewMobInstance is an alias for NewMob for compatibility.
func NewMobInstance(proto *parser.Mob, roomVNum int) *MobInstance {
	return NewMob(proto, roomVNum)
}

// GetSex returns the mob's sex in Go's actor encoding
// (0=male, 1=female, 2=neutral). Mob files retain C's encoding
// (0=neutral, 1=male, 2=female), so translate at the Actor boundary.
func (m *MobInstance) GetSex() int {
	if m.Prototype != nil {
		switch m.Prototype.Sex {
		case 1:
			return 0
		case 2:
			return 1
		default:
			return 2
		}
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

// SetFollowing changes who the mob is following.
func (m *MobInstance) SetFollowing(leader string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Following = leader
}

// GetFollowing returns who the mob is following.
func (m *MobInstance) GetFollowing() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Following
}

// HasFlag checks if the mob has a specific flag.
func (m *MobInstance) HasFlag(flag string) bool {
	if m == nil || m.Prototype == nil || len(m.Prototype.ActionFlags) == 0 {
		return false
	}
	for _, f := range m.Prototype.ActionFlags {
		if f == flag {
			return true
		}
	}
	return false
}

// Attack makes the mob attack a player.
func (m *MobInstance) Attack(player *Player, world *World) error {
	// Simple attack implementation
	damage := 10 // Default damage
	player.TakeDamage(damage)

	// Send messages
	player.SendMessage(fmt.Sprintf("%s attacks you for %d damage!\n", m.GetShortDesc(), damage))

	// Notify other players in the room
	players := world.GetPlayersInRoom(m.RoomVNum)
	for _, p := range players {
		if p != player {
			p.SendMessage(fmt.Sprintf("%s attacks %s!\n", m.GetShortDesc(), player.Name))
		}
	}

	// Transition into the wounded band or POS_DEAD from the new HP, and run the
	// death pipeline at POS_DEAD (HP <= -11) so this path can't leave a player
	// stranded at negative HP with no death — fight.c update_pos (DP-1021).
	if combat.UpdatePositionAfterDamage(player, world.woundBroadcast) == combat.PosDead {
		world.HandleDeath(player, m, -1)
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
// Death state (alive=false, removal from activeMobs, XP awards, events) is
// handled exclusively by HandleDeath — not here. Storing alive=false here
// would pre-empt HandleDeath's CompareAndSwap guard and skip the entire
// kill-payout pipeline (XP, gold, kill counter, corpse, events).
func (m *MobInstance) TakeDamage(amount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentHP -= amount
	// Allow HP into the wounded band; floor at -11 (POS_DEAD threshold,
	// fight.c update_pos). Position/death transitions are owned by callers via
	// combat.UpdatePositionAfterDamage / HandleDeath, not here (DP-1021).
	if m.CurrentHP < -11 {
		m.CurrentHP = -11
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

// IsAlive returns true if the mob is alive (atomic, no lock needed).
func (m *MobInstance) IsAlive() bool {
	return m.alive.Load()
}

// GetMana returns the mob's current mana.
func (m *MobInstance) GetMana() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentMana
}

// SetMana sets the mob's current mana.
func (m *MobInstance) SetMana(v int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentMana = v
}

// GetMaxMana returns the mob's maximum mana.
func (m *MobInstance) GetMaxMana() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MaxMana
}

// SetMaxMana sets the mob's maximum mana.
func (m *MobInstance) SetMaxMana(v int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MaxMana = v
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
	// Unequip existing item in this slot first
	if existing, ok := m.Equipment[position]; ok {
		existing.Location = LocNowhere()
		m.AddToInventory(existing)
	}

	// Remove from inventory if present
	m.RemoveFromInventory(obj)

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

// GetLevel returns the mob's effective level (override or prototype).
func (m *MobInstance) GetLevel() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Level > 0 {
		return m.Level
	}
	if m.Prototype != nil {
		return m.Prototype.Level
	}
	return 1
}

// SetLevel overrides the mob's level (used by conjure_elemental, divine_int, etc.)
func (m *MobInstance) SetLevel(level int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Level = level
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

// SetMaxHP sets the mob's maximum health.
func (m *MobInstance) SetMaxHP(hp int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MaxHP = hp
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
	case "mortally_wounded":
		return combat.PosMortally
	case "incapacitated":
		return combat.PosIncap
	case "stunned":
		return combat.PosStunned
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

// SetPosition sets the mob's position using the same int constants as Player.
func (m *MobInstance) SetPosition(pos int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch pos {
	case combat.PosDead:
		m.Status = "dead"
	case combat.PosMortally:
		m.Status = "mortally_wounded"
	case combat.PosIncap:
		m.Status = "incapacitated"
	case combat.PosStunned:
		m.Status = "stunned"
	case combat.PosSleeping:
		m.Status = "sleeping"
	case combat.PosResting:
		m.Status = "resting"
	case combat.PosSitting:
		m.Status = "sitting"
	case combat.PosFighting:
		m.Status = "fighting"
	default:
		m.Status = "standing"
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

// GetWaitState returns the mob's remaining wait state in PULSE_VIOLENCE ticks.
// Source: utils.h GET_MOB_WAIT(ch)
func (m *MobInstance) GetWaitState() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.WaitState
}

// SetWaitState sets the mob's wait state cooldown.
// Source: utils.h WAIT_STATE(ch, PULSE_VIOLENCE * n)
func (m *MobInstance) SetWaitState(ticks int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WaitState = ticks
}

// DecrementWaitState reduces the mob's wait state by one tick.
// Called each combat round (perform_violence) in C.
func (m *MobInstance) DecrementWaitState() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.WaitState > 0 {
		m.WaitState--
	}
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
	m.Fighting = false
	m.FightingTarget = ""
	// Re-derive position from HP rather than forcing "standing": a mob beaten
	// into the wounded band must stay downed when it stops fighting. Mirrors
	// Merc stop_fighting (reset to default_pos, then update_pos). DP-1021.
	m.Status = mobStatusFromHP(m.CurrentHP)
}

// mobStatusFromHP maps HP to the Status string for the wounded band, matching
// combat.GetPositionFromHP for positive HP → standing and the negative bands.
func mobStatusFromHP(hp int) string {
	if hp > 0 {
		return "standing"
	}
	if hp <= -11 {
		return "dead"
	}
	if hp <= -6 {
		return "mortally_wounded"
	}
	if hp <= -3 {
		return "incapacitated"
	}
	return "stunned"
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

// GetStr returns the mob's strength (instance-level, includes level boosts).
func (m *MobInstance) GetStr() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Str
}

// GetDex returns the mob's dexterity (instance-level, includes level boosts).
func (m *MobInstance) GetDex() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Dex
}

// GetInt returns the mob's intelligence (instance-level, includes level boosts).
func (m *MobInstance) GetInt() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Intel
}

// GetWis returns the mob's wisdom (instance-level, includes level boosts).
func (m *MobInstance) GetWis() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Wis
}

// GetCon returns the mob's constitution (instance-level, includes level boosts).
func (m *MobInstance) GetCon() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Con
}

// GetCha returns the mob's charisma (instance-level, includes level boosts).
func (m *MobInstance) GetCha() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Cha
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
	return m.Gold
}

// SetGold updates the mob's instance gold.
func (m *MobInstance) SetGold(gold int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Gold = gold
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

// HasMobFlag returns true if the given MOB flag bit is set.
func (m *MobInstance) HasMobFlag(bit int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if bit < 0 || bit >= 64 {
		return false
	}
	return m.Flags&(1<<uint(bit)) != 0
}

// SetMobFlag sets the given MOB flag bit.
func (m *MobInstance) SetMobFlag(bit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bit < 0 || bit >= 64 {
		return
	}
	m.Flags |= 1 << uint(bit)
}

// GetHunting returns the mob's current hunting target name.
func (m *MobInstance) GetHunting() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Hunting
}

// IsHunting returns true if the mob has an active hunting target.
func (m *MobInstance) IsHunting() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Hunting != ""
}

// ClearHunting clears the mob's hunting target.
func (m *MobInstance) ClearHunting() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Hunting = ""
	m.HuntingID = ""
}

// SetHunting — defined in deferred_fight_fns.go (full implementation with nil guard)
// func (m *MobInstance) SetHunting(targetName string) — kept there

// GetTarget returns the mob's current combat target.
func (m *MobInstance) GetTarget() *MobInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Target
}

// SetTarget sets the mob's current combat target.
func (m *MobInstance) SetTarget(target *MobInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Target = target
}

// GetMemory returns a copy of the mob's memory list.
// Mutations go through Remember() and Forget() in deferred_fight_fns.go.
func (m *MobInstance) GetMemory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.Memory))
	copy(out, m.Memory)
	return out
}

// ClearMemory clears the mob's entire memory list.
func (m *MobInstance) ClearMemory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Memory = nil
}

// GetMountRider returns the name of the player riding this mount.
func (m *MobInstance) GetMountRider() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MountRider
}

// SetMountRider sets the name of the player riding this mount.
func (m *MobInstance) SetMountRider(rider string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MountRider = rider
}

// GetHuntingID returns the ID of the player being hunted.
func (m *MobInstance) GetHuntingID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.HuntingID
}

// SetHuntingID sets the ID of the player being hunted.
func (m *MobInstance) SetHuntingID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HuntingID = id
}

// GetAffects returns the mob's affect flags bitmask.
func (m *MobInstance) GetAffects() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Affects
}

// SetAffectFlags replaces the mob's entire affect flags bitmask.
func (m *MobInstance) SetAffectFlags(flags uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Affects = flags
}

// HasAffect checks if a specific affect bit is set.
func (m *MobInstance) HasAffect(bit int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Affects&(1<<bit) != 0
}

// ClearAffect clears a specific affect bit.
func (m *MobInstance) ClearAffect(bit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Affects &= ^(1 << bit)
}

// GetMobFlags returns the mob's flags bitmask.
func (m *MobInstance) GetMobFlags() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Flags
}

// ClearMobFlag clears a specific mob flag bit.
func (m *MobInstance) ClearMobFlag(bit int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bit < 0 || bit >= 64 {
		return
	}
	m.Flags &= ^(1 << uint(bit))
}

// IsFighting returns whether the mob is currently in combat.
func (m *MobInstance) IsFighting() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Fighting
}

// GetFightingTarget returns the name of the target being fought.
func (m *MobInstance) GetFightingTarget() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.FightingTarget
}

// GetAlignment returns the mob's alignment from its prototype.
func (m *MobInstance) GetAlignment() int {
	if m.Prototype != nil {
		return m.Prototype.Alignment
	}
	return 0
}

// SetName sets the mob instance's short description (name).
func (m *MobInstance) SetName(name string) {
	if m.Prototype != nil {
		m.Prototype.ShortDesc = name
	}
}

// AddAffect adds an engine.Affect to the mob's affect flags.
// For mobs, affects are tracked as bitmask flags on AffectFlags.
func (m *MobInstance) AddAffect(aff *engine.Affect) {
	// Mobs use affect flags (bitmask) rather than a list.
	// Map AffectType to the corresponding AFF_* bit and set it.
	// Store in CustomData for tracking; the affect tick system
	// will handle duration-based removal.
	if m.CustomData == nil {
		m.CustomData = make(map[string]interface{})
	}
	key := fmt.Sprintf("affect_%d", aff.SpellID)
	m.CustomData[key] = aff

	// Set AFF_* bitmask bits from the affect's Flags field.
	if aff.Flags != 0 {
		for engFlag, cBit := range EngineFlagToAffBit {
			if aff.Flags&engFlag != 0 {
				m.SetAffected(cBit)
			}
		}
	}
}

// RemoveAffectBySpell removes affects matching the given spell number from the mob.
func (m *MobInstance) RemoveAffectBySpell(spellNum int) {
	if m.CustomData == nil {
		return
	}
	key := fmt.Sprintf("affect_%d", spellNum)
	if aff, ok := m.CustomData[key].(*engine.Affect); ok && aff.Flags != 0 {
		for engFlag, cBit := range EngineFlagToAffBit {
			if aff.Flags&engFlag != 0 {
				m.RemoveAffected(cBit)
			}
		}
	}
	delete(m.CustomData, key)
}
