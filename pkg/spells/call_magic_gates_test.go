package spells

import "testing"

// TestCallMagicItemCastIgnoresFightingPosition pins DP-1481 shape 4. C's
// call_magic() applies only the sitting refusal (src/spell_parser.c:431-434), so
// a fighting caster still zaps a wand, reads a scroll or quaffs a potion even
// when the spell's own minimum position is higher; that minimum position is
// do_cast's command gate alone (src/spell_parser.c:855-870). Before this fix the
// command gate leaked into every item cast, so once the caster was fighting
// every later `use` answered "You can't concentrate enough!".
func TestCallMagicItemCastIgnoresFightingPosition(t *testing.T) {
	info := GetSpellInfo(SpellBless)
	if info == nil {
		t.Fatal("SPELL_BLESS has no SpellInfo")
	}
	if info.MinPosition <= PosFighting {
		t.Fatalf("SPELL_BLESS minimum position = %v, want above POS_FIGHTING (%v)", info.MinPosition, PosFighting)
	}

	caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosFighting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}

	if !CallMagic(caster, victim, nil, SpellBless, 31, CastWand, nil) {
		t.Fatal("a fighting caster's wand cast was refused")
	}
	if len(victim.activeAffects) == 0 {
		t.Error("the wand's spell did not reach the victim")
	}
}

// TestCallMagicStillRefusesSittingCaster is the other half of the same C gate:
// call_magic's own sitting refusal must survive (src/spell_parser.c:431-434).
func TestCallMagicStillRefusesSittingCaster(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosSitting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}

	if CallMagic(caster, victim, nil, SpellBless, 31, CastWand, nil) {
		t.Error("a sitting caster's wand cast was allowed")
	}
	if len(victim.activeAffects) != 0 {
		t.Error("the sitting caster's spell still reached the victim")
	}
}
