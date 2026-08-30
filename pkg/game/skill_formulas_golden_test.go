package game

import (
	"math"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newFidelityTestWorld creates a minimal world and a player for skill formula testing.
func newFidelityTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Fidelity Test Room", Zone: 1},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 10
	ch.Class = ClassWarrior
	ch.Race = RaceHuman
	ch.Inventory = NewInventory()
	ch.Equipment = NewEquipment()
	ch.Stats.Str = 18 // Gives StrToDam = 2
	ch.Stats.Dex = 18
	ch.Stats.Int = 18
	ch.Stats.Wis = 18
	ch.Stats.Con = 18
	ch.SetDamroll(2)
	ch.Move = 100

	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	return w, ch
}

// makeFidelityWeapon creates a piercing weapon (TYPE_PIERCE=11).
func makeFidelityWeapon(vnum int, subtype int) *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      vnum,
			Keywords:  "test weapon",
			ShortDesc: "a test weapon",
			TypeFlag:  5,                        // ITEM_WEAPON
			WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
			Values:    [4]int{0, 1, 1, subtype}, // weapon damage 1d1, subtype
		},
	}
}

// mockFidelityCombatEngine is used for DoRescue testing.
type mockFidelityCombatEngine struct {
	startedAttacker combat.Combatant
	startedTarget   combat.Combatant
	stoppedTarget   string
}

func (m *mockFidelityCombatEngine) StartCombat(ch combat.Combatant, target combat.Combatant) error {
	m.startedAttacker = ch
	m.startedTarget = target
	return nil
}

func (m *mockFidelityCombatEngine) StopCombat(name string) {
	m.stoppedTarget = name
}

// TestSkillFormulas_Statistical runs 10,000 statistical iterations for all 10 offensive skills
// and asserts that the success rate aligns with mathematical formulas within a 3.0% margin of error.
func TestSkillFormulas_Statistical(t *testing.T) {
	w, ch := newFidelityTestWorld(t)

	// Create and add a mock player victim
	victim := NewPlayer(2, "Victim", 1001)
	victim.AC = 100
	victim.Level = 15
	victim.Position = 8 // POS_STANDING
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// 1. BASH
	t.Run("Bash", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillBash, 50)
			ch.Move = 100
			victim.Position = 8 // POS_STANDING

			res := DoBash(ch, victim, w)
			if res.Success {
				successes++
				if res.Damage != 6 { // level/2 + 1 = 5 + 1 = 6
					t.Errorf("Expected Bash success damage to be 6, got %d", res.Damage)
				}
				if !res.TargetFalls || !res.StunTarget {
					t.Errorf("Expected Bash success to set TargetFalls and StunTarget")
				}
			} else {
				if !res.SelfStumble {
					t.Errorf("Expected Bash failure to set SelfStumble")
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 60.0 / 101.0 // 59.4%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Bash success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// 2. KICK
	t.Run("Kick", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillKick, 50)
			victim.Position = 8 // POS_STANDING

			res := DoKick(ch, victim)
			if res.Success {
				successes++
				if res.Damage != 5 { // level/2 = 5
					t.Errorf("Expected Kick success damage to be 5, got %d", res.Damage)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 56.0 / 101.0 // 55.44%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Kick success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// 3. TRIP
	t.Run("Trip", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillTrip, 50)
			victim.Position = 8 // POS_STANDING

			res := DoTrip(ch, victim, w)
			if res.Success {
				successes++
				if res.Damage != 6 { // level/2 + 1 = 6
					t.Errorf("Expected Trip success damage to be 6, got %d", res.Damage)
				}
				if !res.TargetFalls {
					t.Errorf("Expected Trip success to set TargetFalls")
				}
			} else {
				if !res.SelfStumble {
					t.Errorf("Expected Trip failure to set SelfStumble")
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 45.0 / 121.0 // 37.19%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Trip success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// 4. HEADBUTT
	t.Run("Headbutt", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillHeadbutt, 50)
			ch.Health = 100
			victim.Position = 8 // POS_STANDING

			res := DoHeadbutt(ch, victim, w)
			if res.Success {
				successes++
				if res.Damage != 10 { // level = 10
					t.Errorf("Expected Headbutt success damage to be 10, got %d", res.Damage)
				}
				// Recoil should have reduced ch health (10 / 4 = 2 recoil)
				if ch.Health != 98 {
					t.Errorf("Expected ch health to be 98 after recoil; got %d", ch.Health)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 50.0 / 121.0 // 41.32%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Headbutt success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// Setup a piercing weapon for Backstab/Circle/Disembowel
	piercing := makeFidelityWeapon(9001, 11) // TYPE_PIERCE = 11
	ch.Inventory.Items = append(ch.Inventory.Items, piercing)
	if err := ch.Equipment.Equip(piercing, ch.Inventory); err != nil {
		t.Fatalf("Failed to equip piercing weapon: %v", err)
	}

	// 5. BACKSTAB
	// Backstab success is a two-stage roll (DP-1033): the skill roll must pass,
	// THEN the THAC0 to-hit roll (hit(ch, vict, SKILL_BACKSTAB) in C). Both
	// must succeed for damage.
	//   skill roll: percent ∈ [1,101] <= 50  → 50/101
	//   to-hit:     calc_thaco=4, victim_ac=6 → only natural 1 misses → 19/20
	//   combined:   (50/101) × (19/20) ≈ 47.03%
	t.Run("Backstab", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillBackstab, 50)
			victim.Position = 8 // POS_STANDING
			victim.SetFighting("")

			res := DoBackstab(ch, victim, w)
			if res.Success {
				successes++
				// expected damage: (weaponDam (1) + damroll (2) + strToDam (2)) * backstab_mult(10) (3.0) = 5 * 3 = 15
				if res.Damage != 15 {
					t.Errorf("Expected Backstab success damage to be 15, got %d", res.Damage)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := (50.0 / 101.0) * (19.0 / 20.0) // 47.03%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Backstab success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// 6. CIRCLE
	t.Run("Circle", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillCircle, 50)
			victim.Position = 8 // POS_STANDING
			victim.SetFighting("")
			ch.SetFighting("")

			res := DoCircle(ch, victim)
			if res.Success {
				successes++
				// expected damage: (weaponDam (1) + damroll (2) + strToDam (2)) * (backstab_mult(10)/3) (1) = 5 * 1 = 5
				if res.Damage != 5 {
					t.Errorf("Expected Circle success damage to be 5, got %d", res.Damage)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := (50.0 / 101.0) * (19.0 / 20.0) // circle roll plus hit() d20
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Circle success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// 7. RESCUE
	t.Run("Rescue", func(t *testing.T) {
		// Set up attacker
		attacker := NewPlayer(3, "Attacker", 1001)
		w.AddPlayer(attacker)

		successes := 0
		const iterations = 10000
		mockEngine := &mockFidelityCombatEngine{}

		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillRescue, 50)
			attacker.SetFighting(victim.GetName())

			res := DoRescue(ch, victim, w, mockEngine)
			if res.Success {
				successes++
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 50.0 / 101.0 // 49.50%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Rescue success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// Wield a sword/lance for Charge
	_ = ch.Equipment.Unequip(SlotWield, ch.Inventory)
	sword := makeFidelityWeapon(9002, 3) // TYPE_SLASH = 3 (sword)
	ch.Inventory.Items = append(ch.Inventory.Items, sword)
	if err := ch.Equipment.Equip(sword, ch.Inventory); err != nil {
		t.Fatalf("Failed to equip sword: %v", err)
	}

	// 8. CHARGE
	t.Run("Charge", func(t *testing.T) {
		successes := 0
		const iterations = 10000

		// Charge success checks use combat.GetRoller() under the hood,
		// but since it resolves to the production roller which uses math/rand/v2,
		// it behaves statistically identical to rand.IntN.
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillCharge, 50)
			victim.Position = 8 // POS_STANDING
			victim.AC = 100

			res := DoCharge(ch, victim)
			if res.Success {
				successes++
				// expected damage: 2 * weaponRoll = 2 * 1 = 2
				if res.Damage != 2 {
					t.Errorf("Expected Charge success damage to be 2, got %d", res.Damage)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 60.0 / 101.0 // 59.4%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("Charge success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})

	// Wield piercing weapon back for Disembowel
	_ = ch.Equipment.Unequip(SlotWield, ch.Inventory)
	ch.Inventory.Items = append(ch.Inventory.Items, piercing)
	if err := ch.Equipment.Equip(piercing, ch.Inventory); err != nil {
		t.Fatalf("Failed to equip piercing weapon: %v", err)
	}

	// 9. DISEMBOWEL
	t.Run("Disembowel", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillDisembowel, 50)
			// C bypasses the skill-roll failure arm for a sleeping victim;
			// this keeps this legacy formula check focused on the hit damage.
			victim.Position = combat.PosSleeping

			res := DoDisembowel(ch, victim)
			if res.Success {
				successes++
				// expected damage: level * 2 + damroll = 10 * 2 + 2 = 22
				if res.Damage != 22 {
					t.Errorf("Expected Disembowel success damage to be 22, got %d", res.Damage)
				}
			}
		}
		if successes != iterations {
			t.Errorf("Disembowel sleeping-target success count = %d; expected %d", successes, iterations)
		}
	})

	// 10. DRAGONKICK
	t.Run("DragonKick", func(t *testing.T) {
		successes := 0
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			ch.SetSkill(SkillDragonKick, 50)
			ch.Move = 100
			victim.Position = 8 // POS_STANDING
			victim.AC = 100

			res := DoDragonKick(ch, victim)
			if res.Success {
				successes++
				// expected damage: level * 1.5 = 10 * 1.5 = 15
				if res.Damage != 15 {
					t.Errorf("Expected DragonKick success damage to be 15, got %d", res.Damage)
				}
			}
		}
		rate := float64(successes) / float64(iterations)
		expected := 60.0 / 101.0 // 59.4%
		if math.Abs(rate-expected) > 0.03 {
			t.Errorf("DragonKick success rate = %f; expected ~%f (+/- 3%%)", rate, expected)
		}
	})
}
