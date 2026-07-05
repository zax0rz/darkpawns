package spells

import "testing"

// spellDamageGoldenCase captures the expected dice formula for one damage spell.
// Transcribed from src/magic.c mag_damage() switch (lines 615-819).
// For each case we verify magDamageFormula() returns the expected dice count,
// die size, and flat bonus at representative caster/victim levels.
type spellDamageGoldenCase struct {
	name       string
	spellNum   int
	level      int
	victimLevel int
	isMage     bool
	hasReagent bool
	wantNum    int
	wantSides  int
	wantFlat   int
}

func TestMagDamageFormula_GoldenAgainstCSource(t *testing.T) {
	cases := []spellDamageGoldenCase{
		// --- MAGE SPELLS ---
		{"magic missile non-mage", SpellMagicMissile, 15, 10, false, false, 4, 3, 15},
		{"magic missile mage no reagent", SpellMagicMissile, 15, 10, true, false, 4, 3, 15},
		{"magic missile mage with reagent", SpellMagicMissile, 15, 10, true, true, 4, 3, 30},
		{"chill touch", SpellChillTouch, 12, 10, false, false, 5, 3, 12},
		{"burning hands", SpellBurningHands, 10, 10, false, false, 4, 5, 10},
		{"shocking grasp", SpellShockingGrasp, 10, 10, false, false, 4, 7, 10},
		{"lightning bolt", SpellLightningBolt, 15, 10, false, false, 9, 4, 15},
		{"color spray mage with reagent", SpellColorSpray, 20, 10, true, true, 9, 7, 40},
		{"fireball non-mage", SpellFireball, 20, 10, false, false, 12, 8, 40},
		{"fireball mage no reagent", SpellFireball, 20, 10, true, false, 12, 8, 60},
		{"fireball mage with reagent", SpellFireball, 20, 10, true, true, 12, 8, 80},
		{"disintegrate non-mage", SpellDisintegrate, 25, 10, false, false, 18, 8, 25},
		{"disintegrate mage with reagent", SpellDisintegrate, 25, 10, true, true, 18, 8, 100},
		{"disrupt non-mage", SpellDisrupt, 20, 10, false, false, 20, 7, 20},
		{"disrupt mage", SpellDisrupt, 20, 10, true, false, 20, 7, 60},

		// --- CLERIC SPELLS ---
		{"dispel evil", SpellDispelEvil, 18, 10, false, false, 9, 5, 5 + 18 + 9},
		{"dispel good", SpellDispelGood, 18, 10, false, false, 9, 5, 5 + 18},
		{"call lightning", SpellCallLightning, 18, 10, false, false, 10, 8, 5 + 18},
		{"harm", SpellHarm, 20, 10, false, false, 12, 8, 40},

		// --- NINJA / MAGE ---
		{"soul leech low victim", SpellSoulLeech, 30, 2, false, false, 0, 0, 100},
		{"soul leech mage with reagent", SpellSoulLeech, 30, 10, true, true, 10, 6, 60},
		{"energy drain non-mage", SpellEnergyDrain, 25, 10, false, false, 10, 6, 25},

		// --- AREA SPELLS ---
		{"earthquake", SpellEarthquake, 20, 10, false, false, 7, 7, 20},
		{"acid blast", SpellAcidBlast, 15, 10, false, false, 4, 3, 15},

		// --- PSIONIC SPELLS ---
		{"mind poke", SpellMindPoke, 10, 10, false, false, 3, 3, 10},
		{"mind attack", SpellMindAttack, 12, 10, false, false, 4, 6, 12},
		{"mind blast", SpellMindBlast, 18, 10, false, false, 9, 7, 18 + 9},
		{"psiblast", SpellPsiblast, 26, 10, false, false, 15, 13, 78},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotNum, gotSides, gotFlat, ok := magDamageFormula(tc.level, tc.victimLevel, tc.spellNum, tc.isMage, tc.hasReagent)
			if !ok {
				t.Fatalf("magDamageFormula(%s) returned ok=false", tc.name)
			}
			if gotNum != tc.wantNum || gotSides != tc.wantSides || gotFlat != tc.wantFlat {
				t.Errorf("magDamageFormula(%s) = (%dd%d + %d), want (%dd%d + %d)",
					tc.name, gotNum, gotSides, gotFlat, tc.wantNum, tc.wantSides, tc.wantFlat)
			}
		})
	}
}

// TestMagDamageFormula_NonDamageSpells verifies that spells which are NOT in
// C's mag_damage() switch return ok=false, preventing fabricated damage.
func TestMagDamageFormula_NonDamageSpells(t *testing.T) {
	nonDamage := []int{
		SpellHellfire, SpellMeteorSwarm, SpellCalliope, SpellSmokescreen, SpellMentalLapse,
		SpellFireBreath, SpellGasBreath, SpellFrostBreath, SpellAcidBreath, SpellLightningBreath,
		SpellDrowning, SpellPetrify,
	}
	for _, spellNum := range nonDamage {
		_, _, _, ok := magDamageFormula(30, 10, spellNum, false, false)
		if ok {
			t.Errorf("magDamageFormula(spell=%d) returned ok=true, want false for non-damage spell", spellNum)
		}
	}
}
