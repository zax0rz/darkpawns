package spells

import "testing"

// spellHealingGoldenCase captures the expected dice formula for one healing/movement spell.
// Transcribed from src/magic.c mag_points() switch (lines 1765-1824).
type spellHealingGoldenCase struct {
	name              string
	spellNum          int
	level             int
	isPsionicOrMystic bool
	wantHitNum        int
	wantHitSides      int
	wantHitFlat       int
	wantMoveNum       int
	wantMoveSides     int
	wantMoveFlat      int
}

func TestMagPointsFormula_GoldenAgainstCSource(t *testing.T) {
	cases := []spellHealingGoldenCase{
		// Cure spells scale with caster level via (level >> 2).
		{"cure light L1", SpellCureLight, 1, false, 2, 8, 1 + 0, 0, 0, 0},
		{"cure light L10", SpellCureLight, 10, false, 2, 8, 1 + 2, 0, 0, 0},
		{"cure light L20", SpellCureLight, 20, false, 2, 8, 1 + 5, 0, 0, 0},
		{"cure critic L1", SpellCureCritic, 1, false, 5, 8, 3 + 0, 0, 0, 0},
		{"cure critic L20", SpellCureCritic, 20, false, 5, 8, 3 + 5, 0, 0, 0},

		// Heal / cell adjustment branch on caster class.
		{"heal non-psi", SpellHeal, 25, false, 3, 8, 100, 0, 0, 0},
		{"heal psionic/mystic", SpellHeal, 25, true, 2, 8, 90, 0, 0, 0},
		{"cell adjustment non-psi", SpellCellAdjustment, 25, false, 3, 8, 100, 0, 0, 0},
		{"cell adjustment psionic/mystic", SpellCellAdjustment, 25, true, 2, 8, 90, 0, 0, 0},

		{"mass heal", SpellMassHeal, 30, false, 0, 0, 200, 0, 0, 0},

		{"vitality", SpellVitality, 20, false, 5, 10, 0, 10, 10, 0},
		{"invigorate", SpellInvigorate, 20, false, 0, 0, 0, 10, 10, 0},

		{"lay hands L10", SpellLayHands, 10, false, 3, 10, 0, 0, 0, 0},
		{"lay hands L30", SpellLayHands, 30, false, 3, 30, 0, 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := magPointsFormula(tc.level, tc.spellNum, tc.isPsionicOrMystic)
			if !ok {
				t.Fatalf("magPointsFormula(%s) returned ok=false", tc.name)
			}
			if got.hitNum != tc.wantHitNum || got.hitSides != tc.wantHitSides || got.hitFlat != tc.wantHitFlat {
				t.Errorf("magPointsFormula(%s) hit = (%dd%d + %d), want (%dd%d + %d)",
					tc.name, got.hitNum, got.hitSides, got.hitFlat, tc.wantHitNum, tc.wantHitSides, tc.wantHitFlat)
			}
			if got.moveNum != tc.wantMoveNum || got.moveSides != tc.wantMoveSides || got.moveFlat != tc.wantMoveFlat {
				t.Errorf("magPointsFormula(%s) move = (%dd%d + %d), want (%dd%d + %d)",
					tc.name, got.moveNum, got.moveSides, got.moveFlat, tc.wantMoveNum, tc.wantMoveSides, tc.wantMoveFlat)
			}
		})
	}
}

// TestMagPointsFormula_NonHealingSpells verifies that spells which are NOT in
// C's mag_points() switch return ok=false.
func TestMagPointsFormula_NonHealingSpells(t *testing.T) {
	nonHealing := []int{SpellMagicMissile, SpellFireball, SpellBlindness, SpellSanctuary}
	for _, spellNum := range nonHealing {
		_, ok := magPointsFormula(30, spellNum, false)
		if ok {
			t.Errorf("magPointsFormula(spell=%d) returned ok=true, want false for non-healing spell", spellNum)
		}
	}
}
