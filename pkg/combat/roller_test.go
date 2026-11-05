package combat

import (
	"testing"
)

func TestScriptedRoller(t *testing.T) {
	// A scripted roller should return values in order, and loop if needed.
	r := NewScriptedRoller([]int{10, 20, 30})

	// Test Number
	if got := r.Number(1, 20); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
	if got := r.Number(1, 20); got != 20 {
		t.Errorf("expected 20, got %d", got)
	}
	if got := r.Number(1, 20); got != 30 {
		t.Errorf("expected 30, got %d", got)
	}
	// Loop back
	if got := r.Number(1, 20); got != 10 {
		t.Errorf("expected 10 on loop, got %d", got)
	}

	// Test Dice (num=2: sum of next two elements, which are 20 and 30)
	if got := r.Dice(2, 6); got != 50 {
		t.Errorf("expected 50 (20+30), got %d", got)
	}

	// Test IntN (next element is 10)
	if got := r.IntN(100); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestSeededRoller(t *testing.T) {
	// SeededRollers with the same seed should return identical sequences.
	r1 := NewSeededRoller(42)
	r2 := NewSeededRoller(42)

	for i := 0; i < 50; i++ {
		n1 := r1.Number(1, 100)
		n2 := r2.Number(1, 100)
		if n1 != n2 {
			t.Fatalf("sequence mismatch at iteration %d: %d != %d", i, n1, n2)
		}
	}
}

func TestDeterministicHitChance(t *testing.T) {
	// CalculateHitChance logic:
	//   calcThaco = getTHAC0(attacker) - strModifier - hitroll - intBonus - wisBonus
	//   If diceroll == 20: returns true (always hits)
	//   If diceroll == 1: returns false (always misses if defender is awake)
	//   Otherwise: returns false (miss) if awake and (calcThaco - diceroll > victimAC)
	//   Else: returns true (hit)

	attacker := &mockCombatant{npc: false, class: ClassWarrior, level: 20} // getTHAC0 = 1
	defender := &mockCombatant{position: PosStanding, ac: 100}             // victimAC = 10

	// Set up a scripted roller returning 1 (natural 1 = always miss)
	rMiss := NewScriptedRoller([]int{1})
	WithRoller(rMiss, func() {
		hit := CalculateHitChance(attacker, defender, HitModifiers{})
		if hit {
			t.Error("expected miss with diceroll 1")
		}
	})

	// Set up a scripted roller returning 20 (natural 20 = always hit)
	rHit := NewScriptedRoller([]int{20})
	WithRoller(rHit, func() {
		hit := CalculateHitChance(attacker, defender, HitModifiers{})
		if !hit {
			t.Error("expected hit with diceroll 20")
		}
	})

	// The case above hits by arithmetic alone (calc_thaco 22 - 20 = 2 does not
	// exceed victim_ac 16), so it cannot see the natural-20 rule disappear.
	// src/fight.c:1825-1826 misses only while diceroll < 20; the roll of 20
	// returns a hit before the miss predicate is reached. Force a fixture the
	// predicate WOULD reject: NPC calc_thaco 24 (20 base + 2 INT + 2 WIS)
	// minus 20 is 4, which is greater than the clamped victim_ac of -10
	// (src/fight.c:1822).
	n20Attacker := &mockCombatant{npc: true, level: 1, str: 10, intVal: 10, wis: 10}
	n20Defender := &mockCombatant{npc: true, position: PosStanding, ac: -1000, dex: 10}
	WithRoller(NewScriptedRoller([]int{20}), func() {
		if !CalculateHitChance(n20Attacker, n20Defender, HitModifiers{}) {
			t.Error("expected natural 20 to hit a defender the miss predicate rejects (fight.c:1825)")
		}
	})
}

func TestDeterministicDamage(t *testing.T) {
	attacker := &mockCombatant{npc: false, level: 10, str: 18, strAdd: 0, damroll: 5, damageRoll: DiceRoll{}}
	defender := &mockCombatant{position: PosStanding, ac: 100} // ac=100 -> no getMinusDam reduction
	weapon := DiceRoll{Num: 3, Sides: 6}

	// CalculateDamage logic:
	//   dam = strApp[18].ToDam(2) + attacker.GetDamroll()(5) + RollDice(3, 6)
	// We roll 3d6. Let's script the dice outcomes to be: 2, 4, 6 (sum = 12)
	rScript := NewScriptedRoller([]int{2, 4, 6})

	WithRoller(rScript, func() {
		dam := CalculateDamage(attacker, defender, weapon, AttackNormal)
		expectedDam := 2 + 5 + 12 // 19
		if dam != expectedDam {
			t.Errorf("expected scripted damage to be %d, got %d", expectedDam, dam)
		}
	})
}
