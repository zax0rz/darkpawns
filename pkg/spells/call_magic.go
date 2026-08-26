package spells

import (
	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// lvlImmort mirrors C LVL_IMMORT (31); duplicated here to avoid an import cycle
// with pkg/game (same pattern as pkg/combat/fight_core.go).
const lvlImmort = 31

// CallMagic is the central spell dispatch function, ported from src/spell_parser.c call_magic().
//
// Parameters:
//   - caster: the entity casting the spell (implements GetClass/GetLevel/GetRoom/GetPosition/etc.)
//   - cvict: the character target (can be nil)
//   - ovict: the object target (can be nil)
//   - spellNum: the spell number
//   - level: effective level of the spell (usually caster level, from item for scrolls/potions)
//   - castType: CAST_SPELL, CAST_WAND, CAST_STAFF, CAST_POTION, CAST_SCROLL
//   - world: game world reference (interface{} to avoid circular imports)
//
// Returns true if the spell was executed.
func CallMagic(caster, cvict, ovict interface{}, spellNum, level int, castType CastType, world interface{}) bool {
	si := GetSpellInfo(spellNum)
	if si == nil {
		return false
	}

	// Check room for NOMAGIC flag
	if roomHasNoMagic(caster, world) {
		type sender interface{ SendMessage(string) }
		if s, ok := caster.(sender); ok {
			s.SendMessage("A magical force prevents you from casting here.\r\n")
		}
		return false
	}

	// Check position
	if !checkPosition(caster, si) {
		return false
	}

	// Peaceful room blocks violent OR damaging spells for mortal casters —
	// C spell_parser.c:436-450. The caster sees a "flash of white light"
	// message (violent magic, or "power" for psionic/mystic), the room sees the
	// light appear, and the spell is aborted. Immortals are exempt.
	if roomIsPeaceful(caster, world) && (si.IsViolent() || si.HasRoutine(RoutineDamage)) {
		if getLevel(caster) < lvlImmort {
			type sender interface{ SendMessage(string) }
			if s, ok := caster.(sender); ok {
				if isClassPsionicOrMystic(getClass(caster)) {
					s.SendMessage("A flash of white light fills the room, dispelling your violent power!\r\n")
				} else {
					s.SendMessage("A flash of white light fills the room, dispelling your violent magic!\r\n")
				}
			}
			sendAffectRoom(caster, caster, "White light from no particular source suddenly fills the room, then vanishes.\r\n", world)
			return false
		}
	}

	// Determine saving throw type based on cast type
	var savetype SavingThrowType
	switch castType {
	case CastWand, CastStaff, CastScroll, CastPotion:
		savetype = SaveRodStaff // C source: spell_parser.c:454-458 — all items use SAVING_ROD
	case CastSpell:
		savetype = SaveSpell
	default:
		savetype = SaveBreath // C source: spell_parser.c:464-467 — breath/default use SAVING_BREATH
	}

	// Route based on spell routines
	if si.HasRoutine(RoutineDamage) {
		MagDamage(level, caster, cvict, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineAffects) {
		MagAffects(level, caster, cvict, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineUnaffects) {
		MagUnaffects(level, caster, cvict, spellNum, world)
	}

	if si.HasRoutine(RoutinePoints) {
		MagPoints(level, caster, cvict, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineAlterObjs) {
		MagAlterObjs(level, caster, ovict, spellNum, world)
	}

	if si.HasRoutine(RoutineGroups) {
		MagGroups(level, caster, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineMasses) {
		MagMasses(level, caster, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineAreas) {
		MagAreas(level, caster, spellNum, int(savetype), world)
	}

	if si.HasRoutine(RoutineSummons) {
		MagSummons(level, caster, spellNum, world)
	}

	if si.HasRoutine(RoutineCreations) {
		MagCreations(level, caster, spellNum, world)
	}

	if si.HasRoutine(RoutineManual) {
		ExecuteManualSpell(spellNum, level, caster, cvict, ovict, "", world)
	}

	return true
}

// roomHasNoMagic checks if the room has the NOMAGIC flag set.
func roomHasNoMagic(ch interface{}, world interface{}) bool {
	type rg interface{ GetRoomVNum() int }
	c, ok := ch.(rg)
	if !ok {
		return false
	}
	type wI interface{ GetRoomInWorld(vnum int) *parser.Room }
	w, ok := world.(wI)
	if !ok {
		return false
	}
	room := w.GetRoomInWorld(c.GetRoomVNum())
	if room == nil {
		return false
	}
	return room.HasFlag(RoomNoMagic)
}

// checkPosition verifies the caster is in a valid position to cast.
func checkPosition(ch interface{}, si *SpellInfo) bool {
	type poser interface{ GetPosition() int }
	p, ok := ch.(poser)
	if !ok {
		return false
	}

	pos := p.GetPosition()
	type sender interface{ SendMessage(string) }

	switch {
	case pos == int(PosDead):
		if s, ok := ch.(sender); ok {
			s.SendMessage("You can't cast spells while dead!\r\n")
		}
		return false
	case pos < int(si.MinPosition):
		if s, ok := ch.(sender); ok {
			s.SendMessage("You can't concentrate enough!\r\n")
		}
		return false
	}

	return true
}

// roomIsPeaceful checks if the room has the PEACEFUL flag set.
func roomIsPeaceful(ch interface{}, world interface{}) bool {
	type rg interface{ GetRoomVNum() int }
	c, ok := ch.(rg)
	if !ok {
		return false
	}
	type wI interface{ GetRoomInWorld(vnum int) *parser.Room }
	w, ok := world.(wI)
	if !ok {
		return false
	}
	room := w.GetRoomInWorld(c.GetRoomVNum())
	if room == nil {
		return false
	}
	return room.HasFlag(RoomPeaceful)
}

// magSavingThrow performs a saving throw check based on level, class, and save type.
// Returns true if the target saves.
func magSavingThrow(ch interface{}, saveType int) bool {
	return CheckSavingThrow(ch, SavingThrowType(saveType))
}

// magAttackModifier returns the attack type name for a given attack type index.
// Ported from src/magic.c mag_attack_modifier().
func MagAttackModifier(attackType int) (singular, plural string) {
	if attackType > 0 && attackType < len(AttackTypes) {
		at := AttackTypes[attackType]
		if at.Singular != "" {
			return at.Singular, at.Plural
		}
	}
	return "hit", "hits"
}

// dice rolls N dice of S sides (NdS).
func dice(num, sides int) int {
	return dprng.Dice(num, sides)
}
