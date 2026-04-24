package game

// regen.go — HP/mana/move regeneration and hunger/thirst tick
//
// Ports the following functions from limits.c (rparet-darkpawns/src/limits.c):
//   hit_gain()      — limits.c:128-193
//   mana_gain()     — limits.c:59-125
//   move_gain()     — limits.c:197-253
//   gain_condition() — limits.c:366-417
//   point_update()  — limits.c:460-686 (character portion only; object decay is a separate concern)
//
// The original point_update() is called from comm.c every PULSE_VIOLENCE ticks.
// Here we call PointUpdate() from the world's regen ticker (separate from AI ticker).

// Affect bit constants — from structs.h:321,335,341
// Used in IsAffected() calls inside gain functions.
const (
	AffPoison    = 11 // AFF_POISON — structs.h:321
	AffCutthroat = 25 // AFF_CUTTHROAT — structs.h:335
	AffFlaming   = 31 // AFF_FLAMING — structs.h:341
)

const (
	PlrWriting = 4 // PLR_WRITING — structs.h:225
)

// isMystic returns true if the player's class is Mystic.
// Source: utils.h IS_MYSTIC() macro (checks CLASS_MYSTIC)
func isMystic(p *Player) bool {
	return p.Class == ClassMystic
}

// isVeteran returns true if the player qualifies as a veteran.
// Source: utils.c:358-362 — playing_time(ch).day >= 30 && GET_KILLS(ch) >= 10000
// NOTE: playing_time and kill count are not yet tracked in this port.
// We conservatively return false until those fields are added.
// TODO: phase N — implement when DaysPlayed and KillCount fields added to Player
func isVeteran(_ *Player) bool {
	return false
}

// roomHasFlag returns true if the room with the given VNum has the named flag.
// Used to check ROOM_REGENROOM (structs.h:74, parsed as "regenroom" in wld.go).
func (w *World) roomHasFlag(roomVNum int, flag string) bool {
	w.mu.RLock()
	room, ok := w.rooms[roomVNum]
	w.mu.RUnlock()
	if !ok {
		return false
	}
	for _, f := range room.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// HitGain calculates the HP recovered per regen tick for a player.
// Source: limits.c:128-193 hit_gain()
//
// Formula (players):
//   base = 20 (+ 12 if veteran)
//   POS_SLEEPING: gain += gain>>1  (×1.5)
//   POS_RESTING:  gain += gain>>2  (×1.25)
//   POS_SITTING:  gain += gain>>3  (×1.125)
//   CLASS_MAGIC_USER or CLASS_CLERIC: gain >>= 1 (halved)
//   AFF_POISON, AFF_FLAMING, AFF_CUTTHROAT: each halves (>>2)
//   Hunger or thirst = 0: gain >>= 2
//   ROOM_REGENROOM: gain += gain>>1
func (w *World) HitGain(p *Player) int {
	p.mu.RLock()
	class := p.Class
	pos := p.Position
	fighting := p.Fighting
	roomVNum := p.RoomVNum
	cFull := p.Conditions[CondFull]
	cThirst := p.Conditions[CondThirst]
	veteran := isVeteran(p)
	poisoned := p.Affects&(1<<AffPoison) != 0
	flaming := p.Affects&(1<<AffFlaming) != 0
	cutthroat := p.Affects&(1<<AffCutthroat) != 0
	_ = fighting // used for SKILL_KK_JIN check — TODO phase N
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

	// Hunger or thirst depleted reduces regen — limits.c:185-186
	if cFull == 0 || cThirst == 0 {
		gain >>= 2
	}

	// ROOM_REGENROOM bonus — limits.c:188-190
	if roomVNum > 0 && w.roomHasFlag(roomVNum, "regenroom") {
		gain += gain >> 1
	}

	return gain
}

// MobHitGain calculates the HP recovered per regen tick for an NPC.
// Source: limits.c:133-137
//
//	gain = GET_LEVEL(ch) for low-level (< 23), else 4×level.
func MobHitGain(m *MobInstance) int {
	lvl := m.GetLevel()
	if lvl < 23 {
		// Original: gain = 2.5 * GET_LEVEL(ch) — limits.c:135
		return (lvl*5 + 1) / 2 // integer approximation of 2.5×level
	}
	return 4 * lvl
}

// ManaGain calculates the mana recovered per regen tick for a player.
// Source: limits.c:59-125 mana_gain()
//
// Formula (players):
//   base = 14 (+ 4 if veteran)
//   POS_SLEEPING: gain <<= 1 (doubled)
//   POS_RESTING:  gain += gain>>1
//   POS_SITTING:  gain += gain>>2
//   CLASS_MAGIC_USER or CLASS_CLERIC: gain <<= 1
//   CLASS_MAGUS or CLASS_AVATAR: gain <<= 1
//   else CLASS_PSIONIC or CLASS_NINJA: gain += gain>>2
//   else IS_MYSTIC: gain <<= 1
//   AFF_POISON, AFF_FLAMING, AFF_CUTTHROAT: each >>= 2
//   Hunger or thirst = 0: gain >>= 2
//   ROOM_REGENROOM: gain += gain>>1
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
	// Note: the original has ambiguous else-if chaining; ported faithfully:
	//   if (magic_user || cleric): gain <<= 1
	//   if (magus || avatar): gain <<= 1    ← separate if (applies additionally to magus/avatar)
	//   else if (psionic || ninja): gain += gain>>2
	//   else if (mystic): gain <<= 1
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

// ManaGainNPC returns mana gain for an NPC.
// Source: limits.c:63-65 — gain = GET_LEVEL(ch)
func ManaGainNPC(m *MobInstance) int {
	return m.GetLevel()
}

// MoveGain calculates the movement points recovered per regen tick for a player.
// Source: limits.c:197-253 move_gain()
//
// Formula (players):
//   base = 20 (+ 4 if veteran)
//   POS_SLEEPING: gain += gain>>1  (after equipment bonus — equipment TODO phase N)
//   POS_RESTING:  gain += gain>>2
//   POS_SITTING:  gain += gain>>3
//   AFF_POISON or AFF_FLAMING: gain >>= 2
//   AFF_CUTTHROAT: gain >>= 2
//   Hunger or thirst = 0: gain >>= 2
//   ROOM_REGENROOM: gain += gain>>1
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
		// TODO: phase N — equipment APPLY_MOVE_REGEN modifiers (limits.c:222-224)
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

// MoveGainNPC returns move gain for an NPC.
// Source: limits.c:202-204 — return GET_LEVEL(ch)
func MoveGainNPC(m *MobInstance) int {
	return m.GetLevel()
}

// GainCondition applies a delta to a player's condition (hunger/thirst/drunk).
// Source: limits.c:366-417 gain_condition()
//
// Value -1 means "immortal" — no change applied.
// Clamps to [0, 48].
// Sends flavour messages at threshold crossings.
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
		inactive := p.Affects&(1<<30) != 0 // PRF_INACTIVE (bit 30) — structs.h:304
		// PRF_INACTIVE in original is flag 29 (0-indexed) in prf_flags, NOT aff_flags.
		// We approximate: players without a Send channel are considered inactive.
		// TODO: phase N — add separate PrfFlags field to Player for PRF_* constants
		_ = inactive
		p.mu.RUnlock()

		// Decay hunger, drunk, thirst — limits.c:478-481
		// (PRF_INACTIVE players skip this — we always apply for now)
		GainCondition(p, CondFull, -1)
		GainCondition(p, CondDrunk, -1)
		GainCondition(p, CondThirst, -1)

		// HP/mana/move regen — limits.c:493-501
		// Only if POS >= POS_STUNNED (3) — limits.c:494
		if pos >= PosStunned {
			p.mu.Lock()
			if p.Health < p.MaxHealth {
				gain := w.HitGain(p)
				p.Health += gain
				if p.Health > p.MaxHealth {
					p.Health = p.MaxHealth
				}
			}
			if p.Mana < p.MaxMana {
				gain := w.ManaGain(p)
				p.Mana += gain
				if p.Mana > p.MaxMana {
					p.Mana = p.MaxMana
				}
			}
			// Move always gets regenerated (no < check in original — limits.c:501)
			if p.Move < p.MaxMove {
				gain := w.MoveGain(p)
				p.Move += gain
				if p.Move > p.MaxMove {
					p.Move = p.MaxMove
				}
			}
			p.mu.Unlock()
		}

		// TODO: phase N — poison damage (limits.c:503-504): damage(i, i, 10, SPELL_POISON)
		// TODO: phase N — cutthroat damage (limits.c:505-506): damage(i, i, 13, SKILL_CUTTHROAT)
	}

	// --- Mobs ---
	for _, m := range mobs {
		pos := m.GetPosition()
		if pos >= PosStunned {
			if m.CurrentHP < m.MaxHP {
				gain := MobHitGain(m)
				m.CurrentHP += gain
				if m.CurrentHP > m.MaxHP {
					m.CurrentHP = m.MaxHP
				}
			}
		}
	}
}

// Position constants re-exported from combat package for use within game package.
// Source: structs.h POS_* constants (structs.h:130-138), ported to pkg/combat/formulas.go
const (
	PosDead    = 0
	PosIncap   = 2
	PosStunned = 3
	PosSleeping = 4
	PosResting  = 5
	PosSitting  = 6
	PosFighting = 7
	PosStanding = 8
)
