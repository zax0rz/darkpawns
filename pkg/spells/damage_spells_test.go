package spells

import "testing"

// ---------------------------------------------------------------------------
// DP-938: MagDamage must not silently fabricate damage for spells that have
// no case in C's mag_damage() switch (src/magic.c:615-819). Hellfire,
// MeteorSwarm, and Calliope are real MANUAL_SPELL routines ported separately
// (castHellfire/castMeteorSwarm/castCalliope — see area_spells_test.go).
// MentalLapse and Smokescreen are non-damage spells (mob-aggro reset and a
// blindness affect, respectively) handled elsewhere. Petrify is a
// look-triggered instant-kill in the Medusa special mob procedure, Drowning
// is a flat environmental HP tick, and the five breath spells plus the
// fabricated "DragonBreath" have no C mag_damage/mag_areas case at all — they
// deal zero damage in the original game. None of the 13 should route through
// MagDamage's dice-table fallback.
// ---------------------------------------------------------------------------

func TestMagDamage_DoesNotFabricateRemovedSpells(t *testing.T) {
	fabricated := []struct {
		name     string
		spellNum int
	}{
		{"Hellfire", SpellHellfire},
		{"MeteorSwarm", SpellMeteorSwarm},
		{"Calliope", SpellCalliope},
		{"Smokescreen", SpellSmokescreen},
		{"MentalLapse", SpellMentalLapse},
		{"FireBreath", SpellFireBreath},
		{"GasBreath", SpellGasBreath},
		{"FrostBreath", SpellFrostBreath},
		{"AcidBreath", SpellAcidBreath},
		{"LightningBreath", SpellLightningBreath},
		{"DragonBreath", SpellDragonBreath},
		{"Drowning", SpellDrowning},
		{"Petrify", SpellPetrify},
	}

	for _, tc := range fabricated {
		t.Run(tc.name, func(t *testing.T) {
			ch := &mockSpellsChar{name: "Caster", level: 40, class: 99}
			victim := &mockSpellsChar{name: "Victim", level: 20, maxHP: 100, hp: 100}

			MagDamage(40, ch, victim, tc.spellNum, int(SaveSpell), nil)

			if victim.hp != 100 {
				t.Errorf("DP-938: MagDamage dealt %d fabricated damage for %s (spell %d); expected 0 (not a real mag_damage case in C)",
					100-victim.hp, tc.name, tc.spellNum)
			}
		})
	}
}
