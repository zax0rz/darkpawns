package spells

import (
	"math/rand"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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

	// Check peaceful room for violent spells
	if si.IsViolent() && roomIsPeaceful(caster, world) {
		type sender interface{ SendMessage(string) }
		if s, ok := caster.(sender); ok {
			s.SendMessage("This room is peaceful; you cannot cast violent spells here.\r\n")
		}
		return false
	}

	// Determine saving throw type based on cast type
	var savetype SavingThrowType
	switch castType {
	case CastWand:
		savetype = SaveParalysis
	case CastStaff:
		savetype = SaveBreath
	case CastScroll:
		savetype = SaveSpell
	case CastPotion:
		savetype = SaveParalysis
	default:
		savetype = SaveSpell
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
	type wI interface { GetRoomInWorld(vnum int) *parser.Room }
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
	type wI interface { GetRoomInWorld(vnum int) *parser.Room }
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
	if num <= 0 || sides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		total += rand.Intn(sides) + 1
	}
	return total
}

