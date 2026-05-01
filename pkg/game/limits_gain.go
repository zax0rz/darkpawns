package game

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

	// Equipment mana regen — limits.c:89-95
	// Positive modifier only applies while sleeping; negative always applies.
	gain += p.sumEquipAffect(ApplyManaRegen, pos == PosSleeping)

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

	// KK_JIN skill bonus — limits.c:144-146, +25% regen when not fighting
	if !isFighting(p) && p.HasSpellAffect(162) { // SKILL_KK_JIN
		gain += gain >> 2
	}

	// Position calculations — limits.c:149-167
	switch pos {
	case PosSleeping:
		gain += gain >> 1 // ×1.5
		// Equipment hit regen — limits.c:156-162, only while sleeping
		gain += p.sumEquipAffect(ApplyHitRegen, false)
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

	// KK_ZHEN skill bonus — limits.c:212-214, +25% regen when not fighting
	if !isFighting(p) && p.HasSpellAffect(165) { // SKILL_KK_ZHEN
		gain += gain >> 2
	}

	// Position calculations — limits.c:217-235
	switch pos {
	case PosSleeping:
		// Equipment move regen — limits.c:224-230, applied before position multiplier
		gain += p.sumEquipAffect(ApplyMoveRegen, false)
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
