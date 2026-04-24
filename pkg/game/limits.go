package game

import (
	"fmt"
	"math/rand"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Condition constants — from structs.h
const (
	CondDrunk  = 0
	CondFull   = 1
	CondThirst = 2
)

// Level/immortal constants — from structs.h LVL_*
const (
	LVL_IMMORT = 50
	LVL_IMPL   = 54
	LVL_GOD    = 50

	// Idle time limits — from limits.c
	IDLE_TO_VOID     = 20 // cycles before being pulled into void
	IDLE_DISCONNECT  = 30 // cycles before forced disconnect
	MAX_TITLE_LENGTH = 80 // from structs.h
)

// maxExpGain and maxExpLoss cap XP per single kill/death.
// Source: limits.c extern int max_exp_gain, max_exp_loss
var (
	maxExpGain = 100000
	maxExpLoss = 50000
)

// Titles is the class title string array.
// Source: class.c:1087-1099 titles[NUM_CLASSES]
var Titles = []string{
	"the Mage", "the Cleric", "the Thief", "the Warrior", "the Magus",
	"the Avatar", "the Assassin", "the Paladin", "the Ninja", "the Psionic",
	"the Ranger", "the Mystic",
}

// Position constants re-exported from combat package for use within game package.
// Source: structs.h POS_* constants (structs.h:130-138), ported to pkg/combat/formulas.go
const (
	PosDead      = combat.PosDead
	PosMortally  = combat.PosMortally
	PosIncap     = combat.PosIncap
	PosStunned   = combat.PosStunned
	PosSleeping  = combat.PosSleeping
	PosResting   = combat.PosResting
	PosSitting   = combat.PosSitting
	PosFighting  = combat.PosFighting
	PosStanding  = combat.PosStanding
)

// FieldObject represents a field object entry.
// Source: limits.c field_object_data_t field_objs[NUM_FOS]
type FieldObject struct {
	ObjVNum       int
	WornOffObjNum int
	WearOffMsg    string
}

// fieldObjs is the field objects list.
var fieldObjs []FieldObject

// Affect bit constants — from structs.h:321,335,341
const (
	AffPoison    = 11 // AFF_POISON — structs.h:321
	AffCutthroat = 25 // AFF_CUTTHROAT — structs.h:335
	AffFlaming   = 31 // AFF_FLAMING — structs.h:341
)

// ---------------------------------------------------------------------------
// isMystic — from src/utils.h IS_MYSTIC() macro
// ---------------------------------------------------------------------------
func isMystic(p *Player) bool {
	if p == nil {
		return false
	}
	return p.Class == ClassMystic
}

// ---------------------------------------------------------------------------
// isVeteran — from utils.c:358-362
// ---------------------------------------------------------------------------
// playing_time(ch).day >= 30 && GET_KILLS(ch) >= 10000
// TODO: implement when playing_time and kill count fields are added.
func isVeteran(_ *Player) bool {
	return false
}

// ---------------------------------------------------------------------------
// roomHasFlag — checks room flags
// ---------------------------------------------------------------------------
func (w *World) roomHasFlag(roomVNum int, flag string) bool {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// ManaGain — from limits.c mana_gain() (lines 59-125)
// ---------------------------------------------------------------------------
func (w *World) ManaGain(p *Player) int {
	p.mu.RLock()
	class := p.Class
	pos := p.Position
	roomVNum := p.RoomVNum
	cFull := p.Conditions[CondFull]
	cThirst := p.Conditions[CondThirst]
	veteran := isVeteran(p)
	mystic := isMystic(p)
	poisoned := p.Affects&(1<<AffPoison) != 0
	flaming := p.Affects&(1<<AffFlaming) != 0
	cutthroat := p.Affects&(1<<AffCutthroat) != 0
	p.mu.RUnlock()

	gain := 14

	if veteran {
		gain += 4
	}

	// Position calculations — limits.c:72-85
	switch pos {
	case PosSleeping:
		gain <<= 1 // doubled
	case PosResting:
		gain += gain >> 1
	case PosSitting:
		gain += gain >> 2
	}

	// Class calculations — limits.c:97-104
	if class == ClassMageUser || class == ClassCleric {
		gain <<= 1
	}
	if class == ClassMagus || class == ClassAvatar {
		gain <<= 1
	} else if class == ClassPsionic || class == ClassNinja {
		gain += gain >> 2
	} else if mystic {
		gain <<= 1
	}

	// Skill/Spell calculations — limits.c:108-115
	if poisoned {
		gain >>= 2
	}
	if flaming {
		gain >>= 2
	}
	if cutthroat {
		gain >>= 2
	}

	// Hunger or thirst — limits.c:117-118
	if cFull == 0 || cThirst == 0 {
		gain >>= 2
	}

	// ROOM_REGENROOM — limits.c:120-122
	if roomVNum > 0 && w.roomHasFlag(roomVNum, "regenroom") {
		gain += gain >> 1
	}

	return gain
}

// ---------------------------------------------------------------------------
// ManaGainNPC — from limits.c mana_gain() NPC branch
// ---------------------------------------------------------------------------
func ManaGainNPC(m *MobInstance) int {
	return m.GetLevel()
}

// ---------------------------------------------------------------------------
// HitGain — from limits.c hit_gain() (lines 128-193)
// ---------------------------------------------------------------------------
func (w *World) HitGain(p *Player) int {
	p.mu.RLock()
	class := p.Class
	pos := p.Position
	roomVNum := p.RoomVNum
	cFull := p.Conditions[CondFull]
	cThirst := p.Conditions[CondThirst]
	veteran := isVeteran(p)
	poisoned := p.Affects&(1<<AffPoison) != 0
	flaming := p.Affects&(1<<AffFlaming) != 0
	cutthroat := p.Affects&(1<<AffCutthroat) != 0
	p.mu.RUnlock()

	gain := 20

	if veteran {
		gain += 12
	}

	// Position calculations — limits.c:149-167
	switch pos {
	case PosSleeping:
		gain += gain >> 1 // ×1.5
	case PosResting:
		gain += gain >> 2 // ×1.25
	case PosSitting:
		gain += gain >> 3 // ×1.125
	}

	// Class/Level calculations — limits.c:171-173
	if class == ClassMageUser || class == ClassCleric {
		gain >>= 1
	}

	// Skill/Spell calculations — limits.c:176-192
	if poisoned {
		gain >>= 2
	}
	if flaming {
		gain >>= 2
	}
	if cutthroat {
		gain >>= 2
	}

	// Hunger or thirst — limits.c:185-186
	if cFull == 0 || cThirst == 0 {
		gain >>= 2
	}

	// ROOM_REGENROOM — limits.c:188-190
	if roomVNum > 0 && w.roomHasFlag(roomVNum, "regenroom") {
		gain += gain >> 1
	}

	return gain
}

// ---------------------------------------------------------------------------
// MobHitGain — from limits.c hit_gain() NPC branch (lines 133-137)
// ---------------------------------------------------------------------------
func MobHitGain(m *MobInstance) int {
	lvl := m.GetLevel()
	if lvl < 23 {
		return (lvl*5 + 1) / 2 // integer approximation of 2.5×level
	}
	return 4 * lvl
}

// ---------------------------------------------------------------------------
// MoveGain — from limits.c move_gain() (lines 197-253)
// ---------------------------------------------------------------------------
func (w *World) MoveGain(p *Player) int {
	p.mu.RLock()
	pos := p.Position
	roomVNum := p.RoomVNum
	cFull := p.Conditions[CondFull]
	cThirst := p.Conditions[CondThirst]
	veteran := isVeteran(p)
	poisoned := p.Affects&(1<<AffPoison) != 0
	flaming := p.Affects&(1<<AffFlaming) != 0
	cutthroat := p.Affects&(1<<AffCutthroat) != 0
	p.mu.RUnlock()

	gain := 20

	if veteran {
		gain += 4
	}

	// Position calculations — limits.c:217-235
	switch pos {
	case PosSleeping:
		gain += gain >> 1
	case PosResting:
		gain += gain >> 2
	case PosSitting:
		gain += gain >> 3
	}

	// Skill/Spell calculations — limits.c:239-247
	if poisoned || flaming {
		gain >>= 2
	}
	if cutthroat {
		gain >>= 2
	}
	if cFull == 0 || cThirst == 0 {
		gain >>= 2
	}

	// ROOM_REGENROOM — limits.c:248-250
	if roomVNum > 0 && w.roomHasFlag(roomVNum, "regenroom") {
		gain += gain >> 1
	}

	return gain
}

// ---------------------------------------------------------------------------
// MoveGainNPC — from limits.c move_gain() NPC branch
// ---------------------------------------------------------------------------
func MoveGainNPC(m *MobInstance) int {
	return m.GetLevel()
}

// ---------------------------------------------------------------------------
// GainCondition — from limits.c gain_condition() (lines 366-417)
// ---------------------------------------------------------------------------
// Applies a delta to a player's condition (hunger/thirst/drunk).
// Value -1 means "immortal" — no change applied.
// Clamps to [0, 48].
// Sends flavour messages at threshold crossings.
func GainCondition(p *Player, condition int, value int) {
	p.mu.RLock()
	cond := p.Conditions[condition]
	p.mu.RUnlock()

	if cond == -1 { // Immortal / no change
		return
	}

	intoxicated := false
	p.mu.RLock()
	if p.Conditions[CondDrunk] > 0 {
		intoxicated = true
	}
	p.mu.RUnlock()

	p.mu.Lock()
	p.Conditions[condition] += value
	if p.Conditions[condition] < 0 {
		p.Conditions[condition] = 0
	}
	if p.Conditions[condition] > 48 {
		p.Conditions[condition] = 48
	}
	newCond := p.Conditions[condition]
	p.mu.Unlock()

	// Messages only at threshold 0 or 1
	if newCond > 1 {
		return
	}

	// Also skip messages if player is writing (PLR_WRITING flag)
	// PLR_WRITING = bit 4 — check p.Flags
	p.mu.RLock()
	writing := p.Flags&(1<<4) != 0
	p.mu.RUnlock()
	if writing {
		return
	}

	var msg string
	if newCond > 0 {
		switch condition {
		case CondFull:
			msg = "Your stomach growls with hunger.\r\n"
		case CondThirst:
			msg = "You feel a bit parched.\r\n"
		case CondDrunk:
			if intoxicated {
				msg = "Your head starts to clear.\r\n"
			}
		}
	} else {
		switch condition {
		case CondFull:
			msg = "You are hungry.\r\n"
		case CondThirst:
			msg = "You are thirsty.\r\n"
		case CondDrunk:
			if intoxicated {
				msg = "You are now sober.\r\n"
			}
		}
	}

	if msg != "" {
		p.SendMessage(msg)
	}
}

// ---------------------------------------------------------------------------
// PointUpdate — from limits.c point_update() (lines 460-686)
// ---------------------------------------------------------------------------
// Main tick function called periodically. Iterates all players and NPCs,
// applies condition decay, regenerates HMV, processes poison/cutthroat
// damage, memory clearing, idle checks, and object decay.
func (w *World) PointUpdate() {
	// Snapshot players under read lock, operate without lock
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
	}
	w.mu.RUnlock()

	// --- Players ---
	for _, p := range players {
		p.mu.RLock()
		pos := p.Position
		p.mu.RUnlock()

		// Condition decay — skip if inactive (PRF_INACTIVE)
		// TODO: PRF_INACTIVE flag check using p.Flags
		GainCondition(p, CondFull, -1)
		GainCondition(p, CondDrunk, -1)
		GainCondition(p, CondThirst, -1)

		// Tattoo timer
		p.mu.Lock()
		if p.TatTimer > 0 {
			p.TatTimer--
		}
		p.mu.Unlock()

		// Dream processing for sleeping characters
		// TODO: call dream when implemented — limits.c:476
		_ = pos
		// if pos == PosSleeping { dream(p, w) }

		if pos >= PosStunned {
			p.mu.RLock()
			hp := p.Health
			maxHP := p.MaxHealth
			mana := p.Mana
			maxMana := p.MaxMana
			move := p.Move
			maxMove := p.MaxMove
			poisoned := p.Affects&(1<<AffPoison) != 0
			cutthroat := p.Affects&(1<<AffCutthroat) != 0
			curPos := p.Position
			p.mu.RUnlock()

			// HP regen
			if hp < maxHP {
				gain := w.HitGain(p)
				hp += gain
				if hp > maxHP {
					hp = maxHP
				}
				p.mu.Lock()
				p.Health = hp
				p.mu.Unlock()
			}

			// Mana regen
			if mana < maxMana {
				gain := w.ManaGain(p)
				mana += gain
				if mana > maxMana {
					mana = maxMana
				}
				p.mu.Lock()
				p.Mana = mana
				p.mu.Unlock()
			}

			// Move regen — always (even at max, original limits.c:501)
			mvGain := w.MoveGain(p)
			move += mvGain
			if move > maxMove {
				move = maxMove
			}
			p.mu.Lock()
			p.Move = move
			p.mu.Unlock()

			// Poison damage — limits.c:503-504
			if poisoned {
				p.TakeDamage(10)
			}

			// Cutthroat damage — limits.c:505-506
			if cutthroat {
				p.TakeDamage(13)
			}

			// Update position if HP has dropped low — limits.c:507-508
			if curPos <= PosStunned {
				p.mu.RLock()
				hp := p.Health
				p.mu.RUnlock()
				updatePosFromHP(p, hp)
			}
		} else if pos == PosIncap {
			// Incapacitated: 1 damage per tick — limits.c:511
			p.TakeDamage(1)
		} else if pos == PosMortally {
			// Mortally wounded: 2 damage per tick — limits.c:513
			p.TakeDamage(2)
		}

		// Memory clearing for NPCs — limits.c:516-518
		// (handled in NPC section below)

		// Idle check for players — limits.c:521-524
		w.CheckIdling(p)
	}

	// --- NPCs ---
	for _, m := range mobs {
		pos := m.GetPosition()
		roomVNum := m.GetRoomVNum()

		if pos >= PosStunned {
			if m.CurrentHP < m.MaxHP {
				gain := MobHitGain(m)
				m.CurrentHP += gain
				if m.CurrentHP > m.MaxHP {
					m.CurrentHP = m.MaxHP
				}
			}
		} else if pos == PosIncap {
			m.TakeDamage(1)
		} else if pos == PosMortally {
			m.TakeDamage(2)
		}

		// Memory clearing — limits.c:516-518
		// 1 in 99 chance of clearing mob memory
		if m.Memory != nil && rand.Intn(99) == 0 {
			clearMemory(m)
		}

		// Object decay for things in this mob's room
		_ = roomVNum
		w.decayObjectsInRoom(roomVNum)
	}
}

// clearMemory clears a mob's memory — from handler.c
func clearMemory(m *MobInstance) {
	m.Memory = nil
}

// decayObjectsInRoom decays objects in the given room.
// Ported from limits.c point_update() object section (lines 527-686).
func (w *World) decayObjectsInRoom(roomVNum int) {
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		if obj.Prototype == nil {
			continue
		}
		objVNum := obj.GetVNum()

		// Moongate — VNum defined in constants
		_ = objVNum

		// TODO: object decay (puddle/puke/dust/corpse/circle of summoning/field objects)
		// Requires: obj timer fields, special object VNums, field object table
		// See limits.c:527-686 for full implementation
	}
}

// updatePosFromHP updates a player's position based on their HP.
// Ported from fight.c update_pos()
func updatePosFromHP(p *Player, hp int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if hp > 0 {
		if p.Position > PosStunned {
			return
		}
		p.Position = PosStanding
		return
	}
	if hp <= -11 {
		p.Position = PosDead
	} else if hp <= -6 {
		p.Position = PosMortally
	} else if hp <= -3 {
		p.Position = PosIncap
	} else {
		p.Position = PosStunned
	}
}

// ---------------------------------------------------------------------------
// SetTitle — from limits.c set_title()
// ---------------------------------------------------------------------------
func SetTitle(p *Player, title string) {
	if title == "" {
		class := p.Class
		if class >= 0 && class < len(Titles) {
			title = Titles[class]
		} else {
			title = "the Adventurer"
		}
	}
	if len(title) > MAX_TITLE_LENGTH {
		title = title[:MAX_TITLE_LENGTH]
	}
	p.Title = title
}

// ---------------------------------------------------------------------------
// CheckAutowiz — from limits.c check_autowiz()
// ---------------------------------------------------------------------------
func CheckAutowiz(p *Player) {
	_ = p
	// Stub: the autowiz shell script is CircleMUD-specific.
	// TODO: implement wizlist update when admin system is built.
}

// ---------------------------------------------------------------------------
// FindExp — from class.c find_exp()
// ---------------------------------------------------------------------------
func FindExp(class int, level int) int {
	var modifier float64

	switch class {
	case ClassMageUser:
		modifier = 0.3
	case ClassCleric:
		modifier = 0.4
	case ClassWarrior:
		modifier = 0.7
	case ClassThief:
		modifier = 0.1
	case ClassMagus, ClassMystic:
		modifier = 1.5
	case ClassAvatar:
		modifier = 1.6
	case ClassAssassin:
		modifier = 1.2
	case ClassPaladin, ClassRanger:
		modifier = 1.9
	case ClassNinja, ClassPsionic:
		modifier = 0.6
	default:
		modifier = 1.0
	}

	switch {
	case level <= 0:
		return 1
	case level == 1:
		return 1500
	case level == 2:
		return 3000
	case level == 3:
		return 6000
	case level == 4:
		return 11000
	case level == 5:
		return 21000
	case level == 6:
		return 42000
	case level == 7:
		return 80000
	case level == 8:
		return 155000
	case level == 9:
		return 300000
	case level == 10:
		return 450000
	case level == 11:
		return 650000
	case level == 12:
		return 870000
	default:
		return 900000 + ((level-13)*level*20000) + (level*level*1000) + int(modifier*10000*float64(level))
	}
}

// ---------------------------------------------------------------------------
// ExpNeededForLevel — from class.c exp_needed_for_level()
// ---------------------------------------------------------------------------
func ExpNeededForLevel(p *Player) int {
	return FindExp(p.Class, p.Level)
}

// ---------------------------------------------------------------------------
// GainExp — from limits.c gain_exp()
// ---------------------------------------------------------------------------
func (w *World) GainExp(p *Player, gain int) {
	if p == nil {
		return
	}

	if p.IsNPC() {
		p.Exp += gain
		return
	}

	if p.Level < 1 || p.Level >= LVL_IMMORT {
		return
	}

	if gain > 0 {
		if gain > maxExpGain {
			gain = maxExpGain
		}

		maxExp := FindExp(p.Class, p.Level+1) - p.Exp
		if gain > maxExp-1 {
			gain = maxExp - 1
			if gain < 1 {
				gain = 1
			}
		}

		p.Exp += gain

		if p.Level < LVL_IMPL-1 && p.Exp >= ExpNeededForLevel(p) {
			// TODO: AFF_FLESH_ALTER handling
			p.Level++
			p.AdvanceLevel()
			sendToChar(p, fmt.Sprintf("You advance to level %d!\r\n", p.Level))
		}
	} else if gain < 0 {
		if gain < -maxExpLoss {
			gain = -maxExpLoss
		}
		p.Exp += gain
		if p.Exp < 0 {
			p.Exp = 0
		}
	}
}

// ---------------------------------------------------------------------------
// GainExpRegardless — from limits.c gain_exp_regardless()
// ---------------------------------------------------------------------------
func (w *World) GainExpRegardless(p *Player, gain int) {
	if p == nil {
		return
	}

	p.Exp += gain
	if p.Exp < 0 {
		p.Exp = 0
	}

	if p.IsNPC() {
		return
	}

	numLevels := 0
	for p.Level < LVL_IMPL && p.Exp >= ExpNeededForLevel(p) {
		// TODO: AFF_FLESH_ALTER handling
		p.Level++
		numLevels++
		p.AdvanceLevel()
	}

	if numLevels > 0 {
		if numLevels == 1 {
			sendToChar(p, "You rise a level!\r\n")
		} else {
			sendToChar(p, fmt.Sprintf("You rise %d levels!\r\n", numLevels))
		}
		CheckAutowiz(p)
	}
}

// ---------------------------------------------------------------------------
// CheckIdling — from limits.c check_idling()
// ---------------------------------------------------------------------------
func (w *World) CheckIdling(p *Player) {
	if p == nil {
		return
	}

	p.mu.RLock()
	level := p.Level
	roomVNum := p.RoomVNum
	p.mu.RUnlock()

	// Idle time tracking is handled externally via UpdateActivity/LastActive.
	// For immortals and NPCs, skip idle handling.
	if level >= LVL_IMMORT || p.IsNPC() {
		return
	}

	// TODO: Full idle handling when char_data.timer, was_in, and desc fields
	// are implemented. Requires:
	//   - p.IdleTimer field
	//   - WasIn room tracking
	//   - char_from_room / char_to_room for void room (VNum 1, 3)
	//   - close_socket for disconnected players
	//   - Crash_rentsave / Crash_idlesave
	//   - extract_char
	_ = roomVNum
}


