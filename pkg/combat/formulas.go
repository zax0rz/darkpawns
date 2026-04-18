package combat

import (
	"math/rand"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// AttackType represents different types of attacks
type AttackType int

const (
	AttackNormal AttackType = iota
	AttackBackstab
	AttackCircle
	AttackDisembowel
)

// Position constants (from fight.c)
const (
	POS_DEAD      = 0
	POS_MORTALLY  = 1
	POS_INCAP     = 2
	POS_STUNNED   = 3
	POS_SLEEPING  = 4
	POS_RESTING   = 5
	POS_SITTING   = 6
	POS_FIGHTING  = 7
	POS_STANDING  = 8
)

// Combatant represents a character in combat (player or mob)
type Combatant interface {
	GetLevel() int
	GetTHAC0() int
	GetAC() int
	GetHP() int
	GetMaxHP() int
	GetDamageRoll() parser.DiceRoll
	IsNPC() bool
	GetPosition() int
}

// CalculateHitChance calculates if an attack hits based on THAC0 vs AC
// Returns true if the attack hits
func CalculateHitChance(attacker, defender Combatant) bool {
	// Simple THAC0 vs AC calculation
	// In D&D: roll d20, add modifiers, compare to THAC0
	// For simplicity: attackRoll = d20 + (attacker level / 3)
	// Hit if attackRoll >= defender's AC
	
	rand.Seed(time.Now().UnixNano())
	attackRoll := rand.Intn(20) + 1 + (attacker.GetLevel() / 3)
	
	// Lower AC is better, so we need to invert the comparison
	// In D&D: attackRoll must be >= (THAC0 - AC)
	// For now, use simplified: attackRoll >= (20 - defender.GetAC())
	
	// AC ranges from -10 (best) to 10 (worst) in Dark Pawns
	// Convert to positive scale where higher is better
	acAdjusted := 10 - defender.GetAC()
	
	return attackRoll >= acAdjusted
}

// CalculateDamage calculates damage based on the original C formula
func CalculateDamage(attacker, defender Combatant, weaponDamage parser.DiceRoll, attackType AttackType) int {
	dam := 0
	
	// Base damage from strength (simplified)
	// In original: dam = str_app[STRENGTH_APPLY_INDEX(ch)].todam
	// For now, use level-based approximation
	dam += attacker.GetLevel() / 2
	
	// Add damage roll
	damRoll := attacker.GetDamageRoll()
	dam += RollDice(damRoll.Num, damRoll.Sides) + damRoll.Plus
	
	// Weapon damage for players
	if !attacker.IsNPC() && weaponDamage.Num > 0 {
		dam += RollDice(weaponDamage.Num, weaponDamage.Sides) + weaponDamage.Plus
	} else if attacker.IsNPC() {
		// NPCs use their damage dice
		mobDamage := attacker.GetDamageRoll()
		dam += RollDice(mobDamage.Num, mobDamage.Sides) + mobDamage.Plus
	} else {
		// Bare hands for players
		dam += rand.Intn(attacker.GetLevel()/3 + 1)
	}
	
	// Position multipliers
	defenderPos := defender.GetPosition()
	if defenderPos < POS_FIGHTING {
		multiplier := 1.0 + float64(POS_FIGHTING-defenderPos)/3.0
		dam = int(float64(dam) * multiplier)
	}
	
	// Special attack multipliers
	switch attackType {
	case AttackBackstab:
		mult := backstabMultiplier(attacker.GetLevel())
		dam = int(float64(dam) * mult)
	case AttackCircle:
		mult := backstabMultiplier(attacker.GetLevel()) / 3.0
		dam = int(float64(dam) * mult)
	case AttackDisembowel:
		dam = (attacker.GetLevel() * 2) + RollDice(damRoll.Num, damRoll.Sides) + damRoll.Plus
	}
	
	// Minimum 1 damage
	if dam < 1 {
		dam = 1
	}
	
	return dam
}

// GetAttacksPerRound calculates how many attacks a mob gets per round
func GetAttacksPerRound(mob Combatant, hasHaste, hasSlow bool) int {
	level := mob.GetLevel()
	attacks := 4
	
	// Base attacks based on level
	if level >= 31 {
		attacks = 5
	} else if level <= 30 {
		attacks = 4
	}
	if level <= 27 {
		attacks = 3
	}
	if level <= 20 {
		attacks = 2
	}
	if level <= 10 {
		attacks = 1
	}
	
	// Random bonus attack chance
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(901) < level {
		attacks++
	}
	
	// Haste/slow effects
	if hasHaste {
		attacks++
	}
	if hasSlow {
		attacks--
	}
	
	// Minimum 1 attack
	if attacks < 1 {
		attacks = 1
	}
	
	return attacks
}

// backstabMultiplier calculates backstab damage multiplier based on level
func backstabMultiplier(level int) float64 {
	// Simplified backstab multiplier
	// In original: returns values like 2.0, 2.5, 3.0 based on level
	if level >= 30 {
		return 3.0
	} else if level >= 20 {
		return 2.5
	} else if level >= 10 {
		return 2.0
	}
	return 1.5
}

// RollDice rolls NdS+P dice
func RollDice(num, sides int) int {
	if num <= 0 || sides <= 0 {
		return 0
	}
	
	total := 0
	for i := 0; i < num; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}