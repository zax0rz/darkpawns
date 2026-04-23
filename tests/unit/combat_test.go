package unit

import (
	"testing"
	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestCombatDamageCalculation(t *testing.T) {
	tests := []struct {
		name           string
		attackerLevel  int
		defenderLevel  int
		weaponDamage   int
		armorRating    int
		minExpected    int
		maxExpected    int
	}{
		{
			name:           "Equal level combat",
			attackerLevel:  5,
			defenderLevel:  5,
			weaponDamage:   10,
			armorRating:    5,
			minExpected:    1,
			maxExpected:    15,
		},
		{
			name:           "Higher level attacker",
			attackerLevel:  10,
			defenderLevel:  5,
			weaponDamage:   15,
			armorRating:    5,
			minExpected:    5,
			maxExpected:    25,
		},
		{
			name:           "Higher level defender",
			attackerLevel:  5,
			defenderLevel:  10,
			weaponDamage:   10,
			armorRating:    10,
			minExpected:    0,
			maxExpected:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			damage := combat.CalculateDamage(tt.attackerLevel, tt.defenderLevel, tt.weaponDamage, tt.armorRating)
			
			if damage < tt.minExpected || damage > tt.maxExpected {
				t.Errorf("CalculateDamage() = %d, expected between %d and %d", damage, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestCombatHitChance(t *testing.T) {
	tests := []struct {
		name           string
		attackerSkill  int
		defenderSkill  int
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "Equal skill",
			attackerSkill:  50,
			defenderSkill:  50,
			expectedMin:    0.4,
			expectedMax:    0.6,
		},
		{
			name:           "Higher attacker skill",
			attackerSkill:  80,
			defenderSkill:  30,
			expectedMin:    0.7,
			expectedMax:    0.9,
		},
		{
			name:           "Higher defender skill",
			attackerSkill:  30,
			defenderSkill:  80,
			expectedMin:    0.1,
			expectedMax:    0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chance := combat.CalculateHitChance(tt.attackerSkill, tt.defenderSkill)
			
			if chance < tt.expectedMin || chance > tt.expectedMax {
				t.Errorf("CalculateHitChance() = %f, expected between %f and %f", chance, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCombatCriticalHit(t *testing.T) {
	tests := []struct {
		name           string
		attackerLuck   int
		defenderLuck   int
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "Normal luck",
			attackerLuck:   10,
			defenderLuck:   10,
			expectedMin:    0.05,
			expectedMax:    0.15,
		},
		{
			name:           "Lucky attacker",
			attackerLuck:   20,
			defenderLuck:   5,
			expectedMin:    0.15,
			expectedMax:    0.25,
		},
		{
			name:           "Unlucky attacker",
			attackerLuck:   5,
			defenderLuck:   20,
			expectedMin:    0.01,
			expectedMax:    0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chance := combat.CalculateCriticalChance(tt.attackerLuck, tt.defenderLuck)
			
			if chance < tt.expectedMin || chance > tt.expectedMax {
				t.Errorf("CalculateCriticalChance() = %f, expected between %f and %f", chance, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCombatRound(t *testing.T) {
	attacker := &combat.Combatant{
		Name:     "Attacker",
		Level:    5,
		Health:   100,
		MaxHealth: 100,
		Attack:   15,
		Defense:  10,
		Speed:    12,
	}
	
	defender := &combat.Combatant{
		Name:     "Defender",
		Level:    5,
		Health:   100,
		MaxHealth: 100,
		Attack:   12,
		Defense:  12,
		Speed:    10,
	}
	
	// Test a combat round
	round := combat.NewRound(attacker, defender)
	result := round.Execute()
	
	if result.Attacker == nil || result.Defender == nil {
		t.Error("Combat round should return both combatants")
	}
	
	if result.Attacker.Health > attacker.MaxHealth {
		t.Errorf("Attacker health %d should not exceed max health %d", result.Attacker.Health, attacker.MaxHealth)
	}
	
	if result.Defender.Health > defender.MaxHealth {
		t.Errorf("Defender health %d should not exceed max health %d", result.Defender.Health, defender.MaxHealth)
	}
	
	// Health should have changed for at least one combatant
	if result.Attacker.Health == attacker.Health && result.Defender.Health == defender.Health {
		t.Error("Combat should result in health changes")
	}
}

func TestCombatFleeChance(t *testing.T) {
	tests := []struct {
		name           string
		fleeingSpeed   int
		pursuingSpeed  int
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "Faster fleer",
			fleeingSpeed:   20,
			pursuingSpeed:  10,
			expectedMin:    0.7,
			expectedMax:    0.9,
		},
		{
			name:           "Slower fleer",
			fleeingSpeed:   10,
			pursuingSpeed:  20,
			expectedMin:    0.1,
			expectedMax:    0.3,
		},
		{
			name:           "Equal speed",
			fleeingSpeed:   15,
			pursuingSpeed:  15,
			expectedMin:    0.4,
			expectedMax:    0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chance := combat.CalculateFleeChance(tt.fleeingSpeed, tt.pursuingSpeed)
			
			if chance < tt.expectedMin || chance > tt.expectedMax {
				t.Errorf("CalculateFleeChance() = %f, expected between %f and %f", chance, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCombatExperienceCalculation(t *testing.T) {
	tests := []struct {
		name           string
		playerLevel    int
		mobLevel       int
		mobDifficulty  int
		expectedMin    int
		expectedMax    int
	}{
		{
			name:           "Equal level normal mob",
			playerLevel:    5,
			mobLevel:       5,
			mobDifficulty:  1,
			expectedMin:    50,
			expectedMax:    100,
		},
		{
			name:           "Higher level mob",
			playerLevel:    5,
			mobLevel:       10,
			mobDifficulty:  2,
			expectedMin:    150,
			expectedMax:    300,
		},
		{
			name:           "Lower level mob",
			playerLevel:    10,
			mobLevel:       5,
			mobDifficulty:  1,
			expectedMin:    10,
			expectedMax:    30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := combat.CalculateExperience(tt.playerLevel, tt.mobLevel, tt.mobDifficulty)
			
			if exp < tt.expectedMin || exp > tt.expectedMax {
				t.Errorf("CalculateExperience() = %d, expected between %d and %d", exp, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCombatLootGeneration(t *testing.T) {
	mob := &combat.Mob{
		Name:       "Test Mob",
		Level:      5,
		Difficulty: 2,
		LootTable: []combat.LootItem{
			{ItemID: "gold", Chance: 0.8, MinAmount: 1, MaxAmount: 10},
			{ItemID: "potion", Chance: 0.3, MinAmount: 1, MaxAmount: 2},
			{ItemID: "weapon", Chance: 0.1, MinAmount: 1, MaxAmount: 1},
		},
	}
	
	// Generate loot multiple times to test probabilities
	lootCounts := make(map[string]int)
	totalRuns := 1000
	
	for i := 0; i < totalRuns; i++ {
		loot := combat.GenerateLoot(mob)
		for _, item := range loot {
			lootCounts[item.ID]++
		}
	}
	
	// Check that gold drops about 80% of the time
	goldChance := float64(lootCounts["gold"]) / float64(totalRuns)
	if goldChance < 0.7 || goldChance > 0.9 {
		t.Errorf("Gold drop chance = %f, expected ~0.8", goldChance)
	}
	
	// Check that potions drop about 30% of the time
	potionChance := float64(lootCounts["potion"]) / float64(totalRuns)
	if potionChance < 0.25 || potionChance > 0.35 {
		t.Errorf("Potion drop chance = %f, expected ~0.3", potionChance)
	}
	
	// Check that weapons drop about 10% of the time
	weaponChance := float64(lootCounts["weapon"]) / float64(totalRuns)
	if weaponChance < 0.05 || weaponChance > 0.15 {
		t.Errorf("Weapon drop chance = %f, expected ~0.1", weaponChance)
	}
}

func TestCombatStatusEffects(t *testing.T) {
	combatant := &combat.Combatant{
		Name:     "Test",
		Level:    5,
		Health:   100,
		MaxHealth: 100,
		Attack:   10,
		Defense:  10,
		Speed:    10,
	}
	
	// Test poison effect
	poison := &combat.StatusEffect{
		Name:        "Poison",
		Type:        combat.EffectTypeDamageOverTime,
		Duration:    3,
		Potency:     5,
	}
	
	combatant.AddStatusEffect(poison)
	
	if len(combatant.StatusEffects) != 1 {
		t.Errorf("Expected 1 status effect, got %d", len(combatant.StatusEffects))
	}
	
	// Test effect application
	initialHealth := combatant.Health
	combatant.ApplyStatusEffects()
	
	if combatant.Health >= initialHealth {
		t.Error("Poison should reduce health")
	}
	
	// Test effect duration
	combatant.TickStatusEffects()
	if len(combatant.StatusEffects) != 1 {
		t.Error("Effect should still be active after 1 tick")
	}
	
	// Tick remaining duration
	combatant.TickStatusEffects()
	combatant.TickStatusEffects()
	
	if len(combatant.StatusEffects) != 0 {
		t.Error("Effect should expire after duration ends")
	}
}

func TestCombatWeaponProficiency(t *testing.T) {
	combatant := &combat.Combatant{
		Name:     "Test",
		Level:    5,
		Health:   100,
		MaxHealth: 100,
		Attack:   10,
		Defense:  10,
		Speed:    10,
		WeaponProficiencies: map[string]int{
			"sword": 75,
			"axe":   50,
			"mace":  25,
		},
	}
	
	tests := []struct {
		name           string
		weaponType     string
		expectedBonus  float64
	}{
		{
			name:           "High proficiency",
			weaponType:     "sword",
			expectedBonus:  0.75,
		},
		{
			name:           "Medium proficiency",
			weaponType:     "axe",
			expectedBonus:  0.5,
		},
		{
			name:           "Low proficiency",
			weaponType:     "mace",
			expectedBonus:  0.25,
		},
		{
			name:           "No proficiency",
			weaponType:     "bow",
			expectedBonus:  0.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bonus := combatant.GetWeaponBonus(tt.weaponType)
			
			if bonus != tt.expectedBonus {
				t.Errorf("GetWeaponBonus(%s) = %f, expected %f", tt.weaponType, bonus, tt.expectedBonus)
			}
		})
	}
}

func TestCombatArmorCalculation(t *testing.T) {
	combatant := &combat.Combatant{
		Name:     "Test",
		Level:    5,
		Health:   100,
		MaxHealth: 100,
		Attack:   10,
		Defense:  10,
		Speed:    10,
		Armor: []combat.ArmorPiece{
			{Type: "helmet", Defense: 2, Weight: 1},
			{Type: "chest", Defense: 5, Weight: 3},
			{Type: "gloves", Defense: 1, Weight: 0.5},
			{Type: "boots", Defense: 1, Weight: 0.5},
		},
	}
	
	totalDefense := combatant.GetTotalDefense()
	expectedDefense := 2 + 5 + 1 + 1 // 9
	
	if totalDefense != expectedDefense {
		t.Errorf("GetTotalDefense() = %d, expected %d", totalDefense, expectedDefense)
	}
	
	totalWeight := combatant.GetArmorWeight()
	expectedWeight := 1.0 + 3.0 + 0.5 + 0.5 // 5.0
	
	if totalWeight != expectedWeight {
		t.Errorf("GetArmorWeight() = %f, expected %f", totalWeight, expectedWeight)
	}
	
	// Test speed penalty from armor weight
	speedPenalty := combatant.GetSpeedPenalty()
	expectedPenalty := totalWeight * 0.1 // 5.0 * 0.1 = 0.5
	
	if speedPenalty != expectedPenalty {
		t.Errorf("GetSpeedPenalty() = %f, expected %f", speedPenalty, expectedPenalty)
	}
}