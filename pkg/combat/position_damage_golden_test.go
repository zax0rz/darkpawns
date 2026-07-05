package combat

import (
	"testing"
)

// TestPositionDamageMultiplier verifies that CalculateDamage correctly applies
// position-based damage multipliers.
// NOTE: Go uses float64 division (intentional deviation from C's integer division
// in fight.c:1854) to preserve the design intent of fractional multipliers.
func TestPositionDamageMultiplier(t *testing.T) {
	tests := []struct {
		position int
		wantDam  int
	}{
		{position: PosStanding, wantDam: 100},
		{position: PosFighting, wantDam: 100},
		{position: PosSitting, wantDam: 133},
		{position: PosResting, wantDam: 166},
		{position: PosSleeping, wantDam: 200},
		{position: PosStunned, wantDam: 233},
		{position: PosIncap, wantDam: 266},
		{position: PosMortally, wantDam: 300},
		{position: PosDead, wantDam: 333},
	}

	for _, tt := range tests {
		attacker := &mockCombatant{
			npc:     false,
			str:     8, // str index 8 has ToDam = 0
			damroll: 98,
		}
		defender := &mockCombatant{
			ac:       100, // 0% reduction
			position: tt.position,
		}

		weaponDamage := DiceRoll{Num: 2, Sides: 1} // rolls exactly 2 damage
		got := CalculateDamage(attacker, defender, weaponDamage, AttackNormal)

		if got != tt.wantDam {
			t.Errorf("CalculateDamage with defender position %d = %d; want %d",
				tt.position, got, tt.wantDam)
		}
	}
}
