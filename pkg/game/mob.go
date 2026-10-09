// Package game manages the game world state and player interactions.
package game

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"

	"github.com/zax0rz/darkpawns/pkg/metrics"
)

// MobInstance represents a spawned mob in the world.
type MobInstance struct {
	mu    sync.RWMutex
	world *World // immutable owner, set before the mobile is published

	// Link to prototype. Stored atomically: the pointed-to parser.Mob is
	// never mutated in place (writers clone+swap), so readers can Load()
	// without holding mu. This fixes the data race where RefreshLiveMobStrings
	// mutated shared prototype fields while look.go read them unlocked.
	prototype atomic.Pointer[parser.Mob]
	VNum      int
	ID        int // World-assigned instance ID

	// Current state
	alive    atomic.Bool // CRIT-004: fast alive check without acquiring mu
	RoomVNum int         // -1 if not in a room (carried, etc.)
	// RoomEntrySequence mirrors char_to_room's prepend order for room looks.
	// Higher values are newer arrivals and therefore appear first.
	RoomEntrySequence uint64
	CurrentHP         int
	MaxHP             int
	CurrentMana       int
	MaxMana           int
	CurrentMove       int
	MaxMove           int
	Status            string // "standing", "sleeping", "fighting", etc.
	Level             int    // Level override (0 = use prototype level)

	// AI
	// Brain *ai.Brain // Temporarily commented out to fix circular import

	// Inventory and equipment
	Inventory []*ObjectInstance
	Equipment map[int]*ObjectInstance // C WEAR_* index, never a player EquipmentSlot

	// Combat state
	Target        *MobInstance     // or Player
	fightingBody  combat.Combatant // C FIGHTING: actual runtime opponent
	combatRetired bool
	WaitState     int // PULSE_VIOLENCE ticks remaining (C GET_MOB_WAIT)

	// Memory: names of players this mob remembers attacking it
	// Source: mobact.c:262-285, remember()/forget() in mobact.c:346-407
	Memory []string

	// MountRider: name of player riding this mount — from src/utils.c
	MountRider string

	// Hunting: name of player being hunted — from src/utils.c
	Hunting      string
	HuntingID    string
	HuntingMobID int

	// CustomData stores arbitrary per-instance data (e.g., damroll bonus for brain eater)
	CustomData map[string]interface{}

	// Runtime state — typed replacement for CustomData (e.g., damroll_bonus)
	Runtime MobRuntimeState

	// Ability scores — instance-level values initialized from prototype + level boosts
	// db.c:1053-1062 applies random bonuses for mobs above level 15
	effectiveAttributes *CharStats
	affectSequence      uint64
	affectOrder         map[string]uint64
	Str                 int
	Intel               int
	Wis                 int
	Dex                 int
	Con                 int
	Cha                 int

	// Gold — instance-level with +/-20% variance from prototype (db.c:1766-1775)
	Gold           int
	BirthTime      time.Time
	Weight, Height int
	SavingThrows   [5]int

	// RaceHates tracks 5 racial hatred slots (src/structs.h race_hate[5]).
	// Initialized to -1 for all slots; matching a mob's race triggers aggression.
	RaceHates [5]int

	// Affect flags bitmask — same bit positions as AFF_* constants used by Player
	Affects uint64

	// Mob flags bitmask — from src/structs.h MOB_* defines
	Flags uint64

	// Following — name of player this mob follows (for charmed pets, etc.)
	Following         string
	followingBody     combat.Combatant
	followingSequence uint64
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

	// Initialize ability scores from prototype. C applies the level>15 random
	// stat boosts (db.c:1053-1062) ONCE, at parse/boot time on the prototype
	// (parse_simple_mob); read_mobile only copies the prototype and rolls HP
	// dice + gold. The Go parser (parser/mob.go) already applies those boosts
	// at parse time with the matching draw order, so NewMob must NOT re-roll
	// them — doing so double-applied the boost and burned 6 extra PRNG draws
	// per high-level mob spawn, desyncing the shared stream (fixes the
	// hunger-thirst stat divergence).
	str := proto.Str
	intel := proto.Int
	wis := proto.Wis
	dex := proto.Dex
	con := proto.Con
	cha := proto.Cha

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
		VNum:      proto.VNum,
		RoomVNum:  roomVNum,
		CurrentHP: hp,
		MaxHP:     hp,
		// C read_mobile initializes every ordinary mob's mana fields to 10
		// (src/db.c:1069-1072), then copies that prototype state into the
		// instance (src/db.c:1757).
		CurrentMana: 10,
		MaxMana:     10,
		CurrentMove: 50,
		MaxMove:     50,
		// C read_mobile copies the whole prototype struct (db.c:1757), so the
		// mob file's loadpos (line-11 first field) is the spawn position — a
		// sleeping loader like 2109 must spawn asleep or mobile_activity's
		// position gate lets it wander on the very first settle pulses.
		// 0 means unset (clear_char's POS_STANDING default; no real record
		// carries POS_DEAD as a loadpos).
		Status: positionStatus(proto.Position),
		// C read_mobile copies the whole prototype struct (db.c:1757), so the
		// mob file's act flags ride along. Without this, HasMobFlag reads a
		// zero mask for every world-loaded mob and C's MOB_AWARE backstab
		// guard (act.offensive.c:212) never fires.
		Flags: actionFlagBits(proto.ActionFlags),
		// ...and the mob file's AFFECTED flags ride along the same way —
		// nearly a thousand world mobs carry innate AFF bits (sanctuary,
		// invisibility, ...), and mag_affects' mob-affection gate
		// (magic.c:1387-1394) refuses spells whose bitvector the mob holds
		// innately.
		Affects:      affectFlagBits(proto.AffectFlags),
		Inventory:    make([]*ObjectInstance, 0),
		Equipment:    make(map[int]*ObjectInstance),
		fightingBody: nil,
		Memory:       make([]string, 0),
		CustomData:   make(map[string]interface{}),
		Runtime:      MobRuntimeState{},
		Str:          str,
		Intel:        intel,
		Wis:          wis,
		Dex:          dex,
		Con:          con,
		Cha:          cha,
		Gold:         gold,
		Weight:       proto.Weight, Height: proto.Height,
		BirthTime: time.Unix(nowFunc(), 0),
	}
	mob.SetProto(proto)
	strAdd := proto.StrAdd
	mob.Runtime.StrAddOverride = &strAdd
	mob.CopyBaseAttributes()

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

// GetSex returns the mob's sex in Go's actor encoding
// (0=male, 1=female, 2=neutral). Mob files retain C's encoding
// (0=neutral, 1=male, 2=female), so translate at the Actor boundary.
func (m *MobInstance) GetSex() int {
	if m.Runtime.SexOverride != nil {
		return *m.Runtime.SexOverride
	}
	if m.Proto() != nil {
		switch m.Proto().Sex {
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
	if m.Proto() != nil {
		return m.Proto().ShortDesc
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
	if m.world != nil {
		m.world.stopRoomFights(m)
		m.world.mu.Lock()
		m.moveRoomLocked(m.world, vnum)
		m.world.mu.Unlock()
		return
	}
	m.mu.Lock()
	m.RoomVNum = vnum
	m.mu.Unlock()
}

// moveRoomLocked applies char_from_room/char_to_room's one-light adjustment.
// Caller holds World.mu; Mob.mu follows it, with no callbacks under either.
func (m *MobInstance) moveRoomLocked(w *World, vnum int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lit := false
	for _, obj := range m.Equipment {
		if isLitLightSource(obj) {
			lit = true
			break
		}
	}
	if lit && m.RoomVNum >= 0 {
		w.adjustRoomLight(m.RoomVNum, -1)
	}
	m.RoomVNum = vnum
	if lit && vnum >= 0 {
		w.adjustRoomLight(vnum, 1)
	}
}

// GetRoomEntrySequence returns the runtime arrival order used by room looks.
func (m *MobInstance) GetRoomEntrySequence() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RoomEntrySequence
}

// GetMove returns the mob's current movement points. C initializes every
// spawned mobile to max_move=50 in read_mobile (src/db.c:1073,1758).
func (m *MobInstance) GetMove() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentMove
}

// SetMove sets the mob's current movement points.
func (m *MobInstance) SetMove(move int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentMove = move
}

// GetMaxMove returns the mob's maximum movement points.
func (m *MobInstance) GetMaxMove() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MaxMove
}

// SpendMove deducts movement points after all movement gates pass.
func (m *MobInstance) SpendMove(cost int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cost < 0 || m.CurrentMove < cost {
		return false
	}
	m.CurrentMove -= cost
	return true
}

// GainMove restores movement points up to the mob's fixed maximum.
func (m *MobInstance) GainMove(gain int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentMove += gain
	if m.CurrentMove > m.MaxMove {
		m.CurrentMove = m.MaxMove
	}
}

// SetFollowing changes who the mob is following.
func (m *MobInstance) SetFollowing(leader string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Following = leader
	m.followingBody = nil
	if leader == "" {
		m.followingSequence = 0
	} else {
		m.followingSequence = nextFollowerSequence()
	}
}

func (m *MobInstance) GetFollowingSequence() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.followingSequence
}

// GetFollowing returns who the mob is following.
func (m *MobInstance) GetFollowing() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Following
}

// HasFlag reports MOB_FLAGGED(m, flag) by name. A MOB_* action flag reads the
// instance's own bitmask, which read_mobile copies from the prototype and
// which the game and scripts may change for one mobile (mob_flags,
// SET_BIT_AR(MOB_FLAGS)). A name that is not an action bit falls back to
// the prototype's flag list.
func (m *MobInstance) HasFlag(flag string) bool {
	if m == nil {
		return false
	}
	name := strings.ToUpper(strings.TrimPrefix(strings.ToUpper(flag), "MOB_"))
	if bit, ok := actionFlagBitIndex[name]; ok {
		return m.HasMobFlag(bit)
	}
	if m.Proto() == nil {
		return false
	}
	for _, f := range m.Proto().ActionFlags {
		if strings.EqualFold(strings.TrimPrefix(f, "MOB_"), name) {
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
	if m.Proto() != nil {
		return m.Proto().LongDesc
	}
	return "A generic mob is here."
}

// TakeDamage applies damage to the mob.
// Death state (alive=false, removal from activeMobs, XP awards, events) is
// handled exclusively by HandleDeath — not here. Storing alive=false here
// would pre-empt HandleDeath's CompareAndSwap guard and skip the entire
// kill-payout pipeline (XP, gold, kill counter, corpse, events).
func (m *MobInstance) TakeDamage(amount int) {
	// Every one of the 24 TakeDamage call sites — melee, spells, spec procs,
	// ambush, hunger and thirst — arrives here, which is why the metric counts
	// damage taken rather than damage dealt: this side knows who lost the hit
	// points, not who took them off. Healing arrives as a negative amount and
	// is not damage.
	if amount > 0 {
		metrics.DamageTaken("mob", amount)
	}
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
	m.mu.Lock()
	defer m.mu.Unlock()
	obj.Location = LocInventoryMob(m.ID)
	m.Inventory = append(m.Inventory, obj)
}

// RemoveFromInventory removes an object from the mob's inventory.
func (m *MobInstance) RemoveFromInventory(obj *ObjectInstance) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, item := range m.Inventory {
		if item == obj {
			m.Inventory = append(m.Inventory[:i], m.Inventory[i+1:]...)
			obj.Location = LocNowhere()
			return true
		}
	}
	return false
}

// EquipItem mirrors do_wear's obj_from_char then equip_char. Positions are C indices.
func (m *MobInstance) EquipItem(obj *ObjectInstance, position int) bool {
	if m.world != nil {
		return m.world.equipMobileFromInventory(m, obj, position)
	}

	m.mu.Lock()
	for i, item := range m.Inventory {
		if item == obj {
			m.Inventory = append(m.Inventory[:i], m.Inventory[i+1:]...)
			obj.Location = LocNowhere()
			break
		}
	}
	_, err := m.equipMobileLocked(nil, obj, position)
	equipped := m.Equipment[position] == obj
	m.mu.Unlock()
	return err == nil && equipped
}

// UnequipItem is the inventory-returning caller of unequip_char used by disarm.
func (m *MobInstance) UnequipItem(position int) *ObjectInstance {
	if m.world != nil {
		obj := m.Equipped(position)
		if obj == nil {
			return nil
		}
		if err := m.world.MoveObjectToMobInventoryFront(obj, m); err != nil {
			return nil
		}
		return obj
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	obj := m.unequipMobileLocked(nil, position)
	if obj != nil {
		obj.Location = LocInventoryMob(m.ID)
		m.Inventory = append([]*ObjectInstance{obj}, m.Inventory...)
	}
	return obj
}

// GetAC returns the mob's armor class.
func (m *MobInstance) GetAC() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Runtime.ACOverride != nil {
		return *m.Runtime.ACOverride
	}
	if m.Proto() != nil {
		return m.Proto().AC
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
	if m.Proto() != nil {
		return m.Proto().Level
	}
	return 1
}

// SetLevel overrides the mob's level (used by conjure_elemental, divine_int, etc.)
func (m *MobInstance) SetLevel(level int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Level = level
}

// ConfigureCreatedMobile applies create_mobile()'s per-instance level stats.
// C new_cmds2.c:588-618 writes these fields after read_mobile(), so they must
// remain instance-local and never mutate the shared mob prototype.
func (m *MobInstance) ConfigureCreatedMobile(level int) {
	damroll := 0
	ndd := 0
	if level > 10 {
		damroll = int(float64(level+1) / 1.50)
		ndd = int(float64(level) / 1.50)
	} else {
		damroll = (level + 1) / 2
		ndd = (level + 1) / 2
	}
	sdd := 4
	ac := 100 - (10 * level)
	hitroll := level
	maxHP := 10*level + 10
	if level > 22 {
		maxHP += 13 * (level - 22)
	}
	if level > 30 {
		maxHP += 560 * (level - 30)
	}
	zeroExp := 0

	m.mu.Lock()
	defer m.mu.Unlock()
	m.Level = level
	damroll = mobileSignedPoint(damroll, 8)
	m.Runtime.DamrollOverride = &damroll
	m.Runtime.DamrollBonus = 0
	m.Runtime.DamageNumOverride = &ndd
	m.Runtime.DamageSidesOverride = &sdd
	m.Runtime.HitrollOverride = &hitroll
	m.Runtime.ACOverride = &ac
	m.Runtime.ExpOverride = &zeroExp
	m.MaxHP = maxHP
	m.CurrentHP = maxHP
}

// SetDamroll overrides the mob's instance-local GET_DAMROLL value. C specials
// may assign points.damroll on a spawned mobile without changing its shared
// prototype (src/spec_procs2.c:1743).
func (m *MobInstance) SetDamroll(damroll int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Runtime.DamrollOverride = &damroll
}

// SetDamageDice overrides the instance-local mob_specials damage dice.
// Native specials can mutate these without changing the shared prototype.
func (m *MobInstance) SetDamageDice(num, sides int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Runtime.DamageNumOverride = &num
	m.Runtime.DamageSidesOverride = &sides
}

// AddDamrollBonus adds to the instance-local GET_DAMROLL value without
// mutating the shared mob prototype. C specials use this for permanent
// per-instance growth such as brain_eater at level 30.
func (m *MobInstance) AddDamrollBonus(bonus int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	damroll := mobileSignedPoint(m.damrollPointLocked()+bonus, 8)
	m.Runtime.DamrollOverride = &damroll
	m.Runtime.DamrollBonus = 0
}

// GetDamageRoll returns the damage dice for the mob's attacks.
func (m *MobInstance) GetDamageRoll() combat.DiceRoll {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Proto() != nil {
		num := m.Proto().Damage.Num
		sides := m.Proto().Damage.Sides
		plus := m.Proto().Damage.Plus
		if m.Runtime.DamageNumOverride != nil {
			num = *m.Runtime.DamageNumOverride
		}
		if m.Runtime.DamageSidesOverride != nil {
			sides = *m.Runtime.DamageSidesOverride
		}
		if m.Runtime.DamrollOverride != nil {
			// The mob-file damage plus is the normal Go representation of the
			// C damroll value. Once C overwrites points.damroll, it must not be
			// counted again as the dice roll's plus component.
			plus = 0
		}
		return combat.DiceRoll{
			Num:   num,
			Sides: sides,
			Plus:  plus,
		}
	}
	return combat.DiceRoll{Num: 0, Sides: 0, Plus: 0} // bare hands
}

// Combatant interface implementation

// GetTHAC0 returns the mob's THAC0.
func (m *MobInstance) GetTHAC0() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Proto() != nil {
		return m.Proto().THAC0
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

// actionFlagBitIndex maps an action flag name to its MOB_* bit, from the
// canonical ActionBitNames table (index = C MOB_* bit), so extended flags
// such as AGGR24 and LOOTS are carried onto mob instances.
var actionFlagBitIndex = func() map[string]int {
	index := make(map[string]int, len(ActionBitNames))
	for i, n := range ActionBitNames {
		index[n] = i
	}
	return index
}()

// actionFlagBits converts parsed act-flag names to the C MOB_* bitmask.
func actionFlagBits(names []string) uint64 {
	var bits uint64
	for _, n := range names {
		name := strings.TrimPrefix(strings.ToUpper(n), "MOB_")
		if bit, ok := actionFlagBitIndex[name]; ok {
			bits |= 1 << uint(bit)
		}
	}
	return bits
}

// affectFlagBitNames mirrors the parser's affect-flag name table
// (parser/mob.go affectBitNames); index = C AFF_* bit (structs.h:310-348).
// Kept in sync by TestAffectFlagBitsMatchParserTable.
var affectFlagBitNames = []string{
	"BLIND", "INVISIBLE", "DETECT_ALIGN", "DETECT_INVIS", "DETECT_MAGIC",
	"SENSE_LIFE", "WATERWALK", "SANCTUARY", "GROUP", "CURSE",
	"INFRAVISION", "POISON", "PROTECT_EVIL", "PROTECT_GOOD", "SLEEP",
	"NOTRACK", "FLESH_ALTER", "DODGE", "SNEAK", "HIDE",
	"BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI_KIRI",
	"CUTTHROAT", "FLY", "WEREWOLF", "VAMPIRE", "MOUNT",
	"INVULN", "FLAMING", "NOTHING", "HASTE", "SLOW",
	"DREAM", "WATERBREATHE", "METALSKIN", "ROBBED",
}

// affectFlagBits converts parsed affect-flag names to the C AFF_* bitmask.
// The bits are C struct positions — the same positions MobInstance.Affects
// and the spells package's mob-affection gate use.
func affectFlagBits(names []string) uint64 {
	index := make(map[string]int, len(affectFlagBitNames))
	for i, n := range affectFlagBitNames {
		index[n] = i
	}
	var bits uint64
	for _, n := range names {
		if bit, ok := index[n]; ok {
			bits |= 1 << uint(bit)
		}
	}
	return bits
}

// positionStatus maps a C POS_* loadpos constant to the Status string
// encoding. Unknown values default to standing like GetPosition does.
func positionStatus(pos int) string {
	if pos <= combat.PosDead || pos > combat.PosStanding {
		pos = combat.PosStanding
	}
	switch pos {
	case combat.PosDead:
		return "dead"
	case combat.PosMortally:
		return "mortally_wounded"
	case combat.PosIncap:
		return "incapacitated"
	case combat.PosStunned:
		return "stunned"
	case combat.PosSleeping:
		return "sleeping"
	case combat.PosResting:
		return "resting"
	case combat.PosSitting:
		return "sitting"
	case combat.PosFighting:
		return "fighting"
	default:
		return "standing"
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

// SetFightingBody sets the actual opponent without changing posture.
// C damage stands the victim after the opener has read its sleeping posture.
func (m *MobInstance) SetFightingBody(target combat.Combatant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fightingBody = target
}

func (m *MobInstance) GetFightingBody() combat.Combatant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fightingBody
}

// StopFighting clears the fighting state.
func (m *MobInstance) StopFighting() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fightingBody = nil
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
	target := m.GetFightingBody()
	if target == nil {
		return ""
	}
	return target.GetName()
}

// GetClass returns the mob's class
func (m *MobInstance) GetClass() int {
	if m.Runtime.ClassOverride != nil {
		return *m.Runtime.ClassOverride
	}
	return 0 // CLASS_MAGE
}

// GetStr returns the mob's strength (instance-level, includes level boosts).
func (m *MobInstance) GetStr() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Str
}

// GetDex returns the mob's dexterity (instance-level, includes level boosts).
func (m *MobInstance) GetDex() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Dex
}

// GetInt returns the mob's intelligence (instance-level, includes level boosts).
func (m *MobInstance) GetInt() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Int
}

// GetWis returns the mob's wisdom (instance-level, includes level boosts).
func (m *MobInstance) GetWis() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Wis
}

// GetCon returns the mob's constitution (instance-level, includes level boosts).
func (m *MobInstance) GetCon() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Con
}

// GetCha returns the mob's charisma (instance-level, includes level boosts).
func (m *MobInstance) GetCha() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().Cha
}

// GetHitroll returns the mob's file-derived hitroll plus equipment bonuses.
// C parses the mob-file THAC0 field as points.hitroll = 20 - thac0
// (src/db.c:1064), while hit() itself always starts NPCs at THAC0 20.
func (m *MobInstance) GetHitroll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Runtime.HitrollOverride != nil {
		return *m.Runtime.HitrollOverride
	}
	total := 0
	if m.Proto() != nil {
		total = 20 - m.Proto().THAC0
	}
	return total
}

// GetDamroll returns the mob's damroll bonus from equipment
// Sums APPLY_DAMROLL (location 19) from all equipped items.
func (m *MobInstance) GetDamroll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0
	if m.Runtime.DamrollOverride != nil {
		total = *m.Runtime.DamrollOverride
	}
	total += m.Runtime.DamrollBonus
	return total
}

// GetStrAdd returns the mob's strength add
func (m *MobInstance) GetStrAdd() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effectiveAttributesLocked().StrAdd
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

// GetExp returns the instance experience value, including create_mobile's
// explicit zero override, falling back to the prototype for ordinary mobs.
func (m *MobInstance) GetExp() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Runtime.ExpOverride != nil {
		return *m.Runtime.ExpOverride
	}
	if m.Proto() != nil {
		return m.Proto().Exp
	}
	return 0
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
	return m.Proto()
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
	m.HuntingMobID = 0
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
func (m *MobInstance) IsFighting() bool { return m.GetFightingBody() != nil }

// GetFightingTarget returns the name of the target being fought.
func (m *MobInstance) GetFightingTarget() string { return m.GetFighting() }

// GetAlignment returns the mob's alignment from its prototype.
func (m *MobInstance) GetAlignment() int {
	return m.alignmentLocked()
}

func (m *MobInstance) alignmentLocked() int {
	if m.Runtime.AlignmentOverride != nil {
		return *m.Runtime.AlignmentOverride
	}
	if m.Proto() != nil {
		return m.Proto().Alignment
	}
	return 0
}

// Proto returns the mob's prototype snapshot. The returned
// *parser.Mob is immutable — writers clone+swap via SetProto — so
// callers can safely dereference it without holding the instance lock.
func (m *MobInstance) Proto() *parser.Mob {
	return m.prototype.Load()
}

// SetProto atomically replaces the mob's prototype snapshot.
// Callers must pass a freshly cloned *parser.Mob, never a shared pointer
// that will be mutated afterwards.
func (m *MobInstance) SetProto(p *parser.Mob) {
	m.prototype.Store(p)
}

// SetName sets the mob instance's short description (name).
func (m *MobInstance) SetName(name string) {
	if proto := m.Proto(); proto != nil {
		clone := *proto
		clone.ShortDesc = name
		m.SetProto(&clone)
	}
}

// AddAffect adds an engine.Affect to the mob's affect flags.
// For mobs, affects are tracked as bitmask flags on AffectFlags.
func (m *MobInstance) AddAffect(aff *engine.Affect) {
	m.mu.Lock()
	if m.CustomData == nil {
		m.CustomData = make(map[string]interface{})
	}
	if m.affectOrder == nil {
		m.affectOrder = make(map[string]uint64)
	}
	key := fmt.Sprintf("affect_%d", aff.SpellID)
	for n := 1; m.CustomData[key] != nil; n++ {
		key = fmt.Sprintf("affect_%d_%d", aff.SpellID, n)
	}
	m.affectSequence++
	m.CustomData[key] = aff
	m.affectOrder[key] = m.affectSequence
	m.mu.Unlock()
	for engFlag, cBit := range EngineFlagToAffBit {
		if aff.Flags&engFlag != 0 {
			m.SetAffected(cBit)
		}
	}
	m.AffectTotal()
}

// JoinAffect chooses the newest matching record, C's affected-list head
// (src/handler.c:476-495), without depending on Go map iteration order.
func (m *MobInstance) JoinAffect(aff *engine.Affect, addDuration, addMagnitude bool) {
	m.mu.Lock()
	var current *engine.Affect
	var selected string
	var sequence uint64
	for key, value := range m.CustomData {
		candidate, ok := value.(*engine.Affect)
		if !ok || candidate.SpellID != aff.SpellID || candidate.Location != aff.Location {
			continue
		}
		order := m.affectOrder[key]
		if current == nil || order > sequence || (order == sequence && key > selected) {
			current = candidate
			selected = key
			sequence = order
		}
	}
	if current != nil {
		if addDuration {
			aff.Duration += current.Duration
		}
		if addMagnitude {
			aff.Magnitude += current.Magnitude
		}
		delete(m.CustomData, selected)
		delete(m.affectOrder, selected)
	}
	m.mu.Unlock()
	aff.ExpiresAt = time.Now().Add(time.Duration(aff.Duration) * engine.TickDuration)
	m.AddAffect(aff)
}

// RemoveAffectBySpell removes every location for this spell, as affect_from_char
// does in src/handler.c:449-456. A missing spell does not run affect_total.
func (m *MobInstance) RemoveAffectBySpell(spellNum int) {
	m.mu.Lock()
	var removed []*engine.Affect
	var remainingFlags uint64
	for key, value := range m.CustomData {
		aff, ok := value.(*engine.Affect)
		if !ok {
			continue
		}
		if aff.SpellID != spellNum {
			remainingFlags |= aff.Flags
			continue
		}
		removed = append(removed, aff)
		delete(m.CustomData, key)
		delete(m.affectOrder, key)
	}
	m.mu.Unlock()
	for _, aff := range removed {
		for engFlag, cBit := range EngineFlagToAffBit {
			if aff.Flags&engFlag != 0 {
				m.RemoveAffected(cBit)
			}
		}
	}
	if len(removed) > 0 {
		for engFlag, cBit := range EngineFlagToAffBit {
			if remainingFlags&engFlag != 0 {
				m.SetAffected(cBit)
			}
		}
		m.AffectTotal()
	}
}

// HitModifiers returns combat modifiers for the mob's equipped weapon.
// Source: fight.c:1793 (ITEM_BLESS weapon bonus). Mobs do not have a drunk condition.
func (m *MobInstance) HitModifiers() combat.HitModifiers {
	var blessed bool
	if weapon := m.Equipped(mobWearWield); weapon != nil && weapon.GetTypeFlag() == ITEM_WEAPON {
		blessed = weapon.HasExtraFlag(0, itemExtraBless)
	}
	return combat.HitModifiers{
		WeaponBlessed: blessed,
	}
}

// SetAlignment sets GET_ALIGNMENT on this instance.
func (m *MobInstance) SetAlignment(align int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Runtime.AlignmentOverride = &align
}

// GetSavingThrow exposes C points.saving_throws to the existing spell accessor.
func (m *MobInstance) GetSavingThrow(kind int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if kind < 0 || kind >= len(m.SavingThrows) {
		return 0
	}
	return m.SavingThrows[kind]
}

// GetDamrollPoint reads C points.damroll, including the mob-file dice plus.
// Combatant keeps that plus in GetDamageRoll until the first runtime write.
func (m *MobInstance) GetDamrollPoint() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.damrollPointLocked()
}

func (m *MobInstance) damrollPointLocked() int {
	base := 0
	if proto := m.Proto(); proto != nil {
		base = proto.Damage.Plus
	}
	if m.Runtime.DamrollOverride != nil {
		base = *m.Runtime.DamrollOverride
	}
	return mobileSignedPoint(base+m.Runtime.DamrollBonus, 8)
}

// SetFollowingBody preserves the selected leader across duplicate descriptions.
func (m *MobInstance) SetFollowingBody(body combat.Combatant) {
	name := ""
	if body != nil {
		name = body.GetName()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Following = name
	m.followingBody = body
	if body == nil {
		m.followingSequence = 0
	} else {
		m.followingSequence = nextFollowerSequence()
	}
}

func (b *MobInstance) CombatRetired() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.combatRetired
}

func (b *MobInstance) SetCombatRetired(retired bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.combatRetired = retired
}
