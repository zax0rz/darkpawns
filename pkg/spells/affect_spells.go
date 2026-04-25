package spells

import "github.com/zax0rz/darkpawns/pkg/engine"

// MagAffects applies spell affects to a character.
// Functions named MagXxx to match C convention; constants are RoutineXxx.
func MagAffects(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil || ch == nil {
		return
	}
	_ = level
	_ = ch
	_ = savetype
	_ = world

	var aff *engine.Affect

	switch spellNum {
	case SpellChillTouch:
		dur := 4
		if magSavingThrow(victim, savetype) {
			dur = 1
		}
		aff = engine.NewAffect(engine.AffectStrength, dur, -1, "chill touch")
	case SpellBless:
		aff = engine.NewAffect(engine.AffectHitRoll, 6, 2, "bless")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectArmorClass, 6, -2, "bless")
	case SpellArmor:
		aff = engine.NewAffect(engine.AffectArmorClass, 24, -15, "armor")
	case SpellBlindness, SpellSmokescreen:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "You shake off the blinding effect.\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectHitRoll, 2, -(4), "blindness")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectBlind, 2, 40, "blindness")
		sendToVictim(victim, "You have been blinded!\r\n")
	case SpellCurse:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "The spell had no effect.\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectCurse, getLevel(ch)/2, -(3), "curse")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectDamageRoll, getLevel(ch)/2, -(3), "curse")
	case SpellInvisible:
		aff = engine.NewAffect(engine.AffectInvisible, 12+getLevel(ch)/4, 0, "invisibility")
	case SpellSanctuary:
		aff = engine.NewAffect(engine.AffectSanctuary, 4, 0, "sanctuary")
	case SpellSleep:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "You resist the spell!\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectSleep, 4+getLevel(ch)/4, 0, "sleep")
	case SpellPoison:
		if magSavingThrow(victim, savetype) {
			return
		}
		dur := (level / 2) - 2
		if dur < 1 {
			dur = 1
		}
		aff = engine.NewAffect(engine.AffectPoison, dur, -2, "poison")
	case SpellHaste:
		aff = engine.NewAffect(engine.AffectHaste, level, 0, "haste")
	case SpellSlow:
		aff = engine.NewAffect(engine.AffectSlow, level, 0, "slow")
	case SpellFly:
		aff = engine.NewAffect(engine.AffectFlying, getLevel(ch), 0, "fly")
	case SpellDetectMagic:
		aff = engine.NewAffect(engine.AffectDetectMagic, 12+level, 0, "detect magic")
	case SpellDetectInvis:
		aff = engine.NewAffect(engine.AffectDetectInvisible, 12+level, 0, "detect invis")
	case SpellInfravision:
		aff = engine.NewAffect(engine.AffectInfrared, 12+level, 0, "infravision")
	case SpellWaterBreathe:
		aff = engine.NewAffect(engine.AffectWaterBreathing, getLevel(ch), 0, "water breathe")
	default:
		return
	}

	applyAffect(victim, aff)
}

// MagPoints handles HP/MV restoration spells.
func MagPoints(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil {
		return
	}
	_ = ch
	_ = savetype
	_ = world

	hit := 0

	switch spellNum {
	case SpellCureLight:
		hit = dice(2, 8) + 1 + (level >> 2)
		sendToVictim(victim, "You feel better.\r\n")
	case SpellCureCritic:
		hit = dice(5, 8) + 3 + (level >> 2)
		sendToVictim(victim, "You feel a lot better!\r\n")
	case SpellHeal:
		hit = 100 + dice(3, 8)
		sendToVictim(victim, "A warm feeling floods your body.\r\n")
	case SpellVitality:
		hit = dice(5, 10)
		sendToVictim(victim, "You feel vitalized!\r\n")
	}

	if hit > 0 {
		healHP(victim, hit)
	}
}

// MagUnaffects removes spell affects from a target.
func MagUnaffects(level int, ch, victim interface{}, spellNum int, world interface{}) {
	if victim == nil {
		return
	}
	_ = level
	_ = ch
	_ = world

	switch spellNum {
	case SpellCureBlind, SpellHeal, SpellMassHeal:
		removeAffect(victim, SpellBlindness)
		sendToVictim(victim, "Your vision clears!\r\n")
	case SpellRemovePoison:
		removeAffect(victim, SpellPoison)
		sendToVictim(victim, "A warm feeling runs through your body!\r\n")
	case SpellRemoveCurse:
		removeAffect(victim, SpellCurse)
		sendToVictim(victim, "You don't feel so unlucky.\r\n")
	}
}

// MagGroups applies group versions of spells.
func MagGroups(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagMasses applies mass (room-wide) spells.
func MagMasses(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagAreas applies area (room-wide offensive) spells.
func MagAreas(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagSummons summons NPCs into the world.
func MagSummons(level int, ch interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = spellNum
	_ = world
}

// MagCreations creates objects.
func MagCreations(level int, ch interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = spellNum
	_ = world
}

// MagAlterObjs alters objects.
func MagAlterObjs(level int, ch, obj interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = obj
	_ = spellNum
	_ = world
}

// ExecuteManualSpell dispatches manual (ASPELL) spell implementations.
func ExecuteManualSpell(spellNum, level int, ch, cvict, ovict interface{}, arg string, world interface{}) {
	_ = level
	_ = cvict
	_ = ovict
	_ = arg
	_ = world
	sendToVictim(ch, "Spell not yet implemented.\r\n")
}

// --- helpers ---

func sendToVictim(victim interface{}, msg string) {
	type sender interface{ SendMessage(string) }
	if s, ok := victim.(sender); ok {
		s.SendMessage(msg)
	}
}

func sendToCaster(ch interface{}, msg string) {
	type sender interface{ SendMessage(string) }
	if s, ok := ch.(sender); ok {
		s.SendMessage(msg)
	}
}

func getLevel(ch interface{}) int {
	type lever interface{ GetLevel() int }
	if l, ok := ch.(lever); ok {
		return l.GetLevel()
	}
	return 1
}

func isClassMage(ch interface{}) bool {
	type classer interface{ GetClass() int }
	if c, ok := ch.(classer); ok {
		return c.GetClass() == 0 || c.GetClass() == 8
	}
	return false
}

func applyAffect(entity interface{}, aff *engine.Affect) {
	type affecter interface{ AddAffect(*engine.Affect) }
	if a, ok := entity.(affecter); ok {
		a.AddAffect(aff)
	}
}

func removeAffect(entity interface{}, spellNum int) {
	type remover interface{ RemoveAffectBySpell(int) }
	if r, ok := entity.(remover); ok {
		r.RemoveAffectBySpell(spellNum)
	}
}

func healHP(victim interface{}, amount int) {
	type healer interface {
		GetHP() int
		GetMaxHP() int
		SetHP(int)
	}
	if h, ok := victim.(healer); ok {
		newHP := h.GetHP() + amount
		if newHP > h.GetMaxHP() {
			newHP = h.GetMaxHP()
		}
		h.SetHP(newHP)
	}
}

func checkReagents(ch interface{}, spellNum, level int, reagents ...string) int {
	_ = spellNum
	_ = level
	_ = reagents
	return 0
}


