package combat

import (
	"math/rand"
)

// AttackType represents different types of attacks
type AttackType int

const (
	AttackNormal AttackType = iota
	AttackBackstab
	AttackCircle
	AttackDisembowel
)

// Position constants — from structs.h
const (
	POS_DEAD     = 0
	POS_MORTALLY = 1
	POS_INCAP    = 2
	POS_STUNNED  = 3
	POS_SLEEPING = 4
	POS_RESTING  = 5
	POS_SITTING  = 6
	POS_FIGHTING = 7
	POS_STANDING = 8
)

// Class constants — from class.h (NUM_CLASSES = 13)
const (
	CLASS_MAGE      = 0
	CLASS_CLERIC    = 1
	CLASS_THIEF     = 2
	CLASS_WARRIOR   = 3
	CLASS_MAGUS     = 4
	CLASS_AVATAR    = 5
	CLASS_ASSASSIN  = 6
	CLASS_PALADIN   = 7
	CLASS_NINJA     = 8
	CLASS_PSIONIC   = 9
	CLASS_RANGER    = 10
	CLASS_MYSTIC    = 11
)

// thaco — from class.c: thaco[NUM_CLASSES][LVL_IMPL+1]
// Index 0 is unused (level 0), levels 1–40 are valid.
var thaco = [12][41]int{
	// MAGE
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18,
		17, 17, 17, 16, 16, 16, 15, 15, 15, 14,
		14, 14, 13, 13, 13, 12, 12, 12, 11, 11,
		11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	// CLERIC
	{100, 20, 20, 20, 18, 18, 18, 16, 16, 16,
		14, 14, 14, 12, 12, 12, 10, 10, 10, 8,
		8, 8, 6, 6, 6, 4, 4, 4, 2, 2,
		2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	// THIEF
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16,
		16, 15, 15, 14, 13, 13, 12, 12, 11, 11,
		10, 10, 9, 9, 8, 8, 7, 7, 6, 6,
		5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	// WARRIOR
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12,
		11, 10, 9, 8, 7, 6, 5, 4, 3, 2,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	// MAGUS
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18,
		17, 17, 17, 16, 16, 16, 15, 15, 15, 14,
		14, 14, 13, 13, 13, 12, 12, 12, 11, 11,
		11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	// AVATAR
	{100, 20, 20, 20, 18, 18, 18, 16, 16, 16,
		14, 14, 14, 12, 12, 12, 10, 10, 10, 8,
		8, 8, 6, 6, 6, 4, 4, 4, 2, 2,
		2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	// ASSASSIN
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16,
		16, 15, 15, 14, 13, 13, 12, 12, 11, 11,
		10, 10, 9, 9, 8, 8, 7, 7, 6, 6,
		5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	// PALADIN
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12,
		11, 10, 9, 8, 7, 6, 5, 4, 3, 2,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	// NINJA
	{100, 20, 20, 19, 19, 18, 18, 17, 17, 16,
		16, 15, 15, 14, 13, 13, 12, 12, 11, 11,
		10, 10, 9, 9, 8, 8, 7, 7, 6, 6,
		5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	// PSIONIC
	{100, 20, 20, 19, 18, 18, 17, 16, 16, 16,
		15, 15, 14, 14, 14, 13, 12, 12, 10, 10,
		9, 9, 8, 8, 7, 7, 6, 5, 5, 4,
		4, 3, 3, 3, 2, 2, 1, 1, 1, 1, 1},
	// RANGER
	{100, 20, 19, 18, 17, 16, 15, 14, 13, 12,
		11, 10, 9, 8, 7, 6, 5, 4, 3, 2,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	// MYSTIC
	{100, 20, 20, 20, 19, 19, 19, 18, 18, 18,
		17, 17, 17, 16, 16, 16, 15, 15, 15, 14,
		14, 14, 13, 13, 13, 12, 12, 12, 11, 11,
		11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
}

// strApp mirrors str_app[] from constants.c
// Fields: {tohit, todam, carry_w, carry_n}
type strAppType struct {
	ToHit  int
	ToDam  int
}

// strApp — from constants.c str_app[], indices 0–30
// We only need tohit and todam fields.
var strApp = []strAppType{
	{-5, -4}, // 0
	{-5, -4}, // 1
	{-3, -2}, // 2
	{-3, -1}, // 3
	{-2, -1}, // 4
	{-2, -1}, // 5
	{-1, 0},  // 6
	{-1, 0},  // 7
	{0, 0},   // 8
	{0, 0},   // 9
	{0, 0},   // 10
	{0, 0},   // 11
	{0, 0},   // 12
	{0, 0},   // 13
	{0, 0},   // 14
	{0, 0},   // 15
	{0, 1},   // 16
	{1, 1},   // 17
	{1, 2},   // 18
	{3, 7},   // 19
	{3, 8},   // 20
	{4, 9},   // 21
	{4, 10},  // 22
	{5, 11},  // 23
	{6, 12},  // 24
	{7, 14},  // 25
	{1, 3},   // 18/01-50
	{2, 3},   // 18/51-75
	{2, 4},   // 18/76-90
	{2, 5},   // 18/91-99
	{3, 6},   // 18/100
}

// dexApp mirrors dex_app[] from constants.c
// Fields: {reaction, miss_att, defensive}
type dexAppType struct {
	Reaction  int
	MissAtt   int
	Defensive int
}

// dexApp — from constants.c dex_app[], indices 0–25
var dexApp = []dexAppType{
	{-7, -7, 6},  // 0
	{-6, -6, 5},  // 1
	{-4, -4, 5},  // 2
	{-3, -3, 4},  // 3
	{-2, -2, 3},  // 4
	{-1, -1, 2},  // 5
	{0, 0, 1},    // 6
	{0, 0, 0},    // 7
	{0, 0, 0},    // 8
	{0, 0, 0},    // 9
	{0, 0, 0},    // 10
	{0, 0, 0},    // 11
	{0, 0, 0},    // 12
	{0, 0, 0},    // 13
	{0, 0, 0},    // 14
	{0, 0, -1},   // 15
	{1, 1, -2},   // 16
	{2, 2, -3},   // 17
	{2, 2, -4},   // 18
	{3, 3, -4},   // 19
	{3, 3, -4},   // 20
	{4, 4, -5},   // 21
	{4, 4, -5},   // 22
	{4, 4, -5},   // 23
	{5, 5, -6},   // 24
	{5, 5, -6},   // 25
}

// strIndex returns the str_app index for a combatant.
// For simplicity at Phase 2, players/mobs don't have STR/ADD stats yet,
// so we default to index 10 (str=10, no bonus/penalty).
// TODO: expose STR and ADD via Combatant interface in Phase 3+.
func strIndex(c Combatant) int {
	return 10
}

// dexIndex returns the dex_app index for a combatant.
// Defaults to 10 (no bonus/penalty) until DEX is exposed.
// TODO: expose DEX via Combatant interface in Phase 3+.
func dexIndex(c Combatant) int {
	return 10
}

// getTHAC0 returns the base THAC0 for a combatant.
// Mobs always use 20 (from fight.c line 1786).
// Players use the class/level table. Since Class isn't on the interface yet,
// we default players to WARRIOR (best THAC0) as a placeholder.
// TODO: expose Class via Combatant interface in Phase 3+.
func getTHAC0(c Combatant) int {
	if c.IsNPC() {
		return 20
	}
	level := c.GetLevel()
	if level < 1 {
		level = 1
	}
	if level > 40 {
		level = 40
	}
	// Default to WARRIOR table until Class is exposed
	return thaco[CLASS_WARRIOR][level]
}

// CalculateHitChance implements the original hit() logic from fight.c lines 1783–1830.
//
// Original formula:
//   calc_thaco = thaco[class][level]           (players) or 20 (mobs)
//   calc_thaco -= str_app[str_index].tohit
//   calc_thaco -= GET_HITROLL(ch)              (TODO: not yet on interface)
//   calc_thaco -= (INT-13)/1.5                 (TODO: not yet on interface)
//   calc_thaco -= (WIS-13)/1.5                 (TODO: not yet on interface)
//   diceroll = number(1,20)
//   victim_ac = GET_AC(victim)/10
//   if AWAKE: victim_ac += dex_app[dex].defensive
//   victim_ac = max(-10, victim_ac)
//   MISS if: diceroll < 20 AND AWAKE AND (diceroll==1 OR calc_thaco-diceroll > victim_ac)
//   HIT  otherwise
func CalculateHitChance(attacker, defender Combatant) bool {
	calcThaco := getTHAC0(attacker)
	calcThaco -= strApp[strIndex(attacker)].ToHit

	diceroll := rand.Intn(20) + 1

	victimAC := defender.GetAC() / 10
	// Assume defender is awake (position >= POS_SLEEPING)
	if defender.GetPosition() > POS_SLEEPING {
		dex := dexIndex(defender)
		if dex >= 0 && dex < len(dexApp) {
			victimAC += dexApp[dex].Defensive
		}
	}
	if victimAC < -10 {
		victimAC = -10
	}

	// Natural 20 always hits; otherwise miss if thaco-roll > victim_ac
	if diceroll == 20 {
		return true
	}
	awake := defender.GetPosition() > POS_SLEEPING
	if awake && (diceroll == 1 || (calcThaco-diceroll) > victimAC) {
		return false // miss
	}
	return true // hit
}

// CalculateDamage implements the original damage calculation from fight.c lines 1840–1858.
//
// Original formula:
//   dam = str_app[str_index].todam
//   dam += GET_DAMROLL(ch)             (TODO: not yet on interface)
//   if player+wielding weapon: dam += dice(weapon_val1, weapon_val2)
//   if mob: dam += dice(damnodice, damsizedice)
//   if player+no weapon: dam += number(0, level/3)
//   if victim position < POS_FIGHTING: dam *= 1 + (POS_FIGHTING-pos)/3
func CalculateDamage(attacker, defender Combatant, weaponDamage DiceRoll, attackType AttackType) int {
	dam := strApp[strIndex(attacker)].ToDam

	// Weapon/bare hands damage
	damRoll := attacker.GetDamageRoll()
	if attacker.IsNPC() {
		// Mob: dice(damnodice, damsizedice)
		dam += RollDice(damRoll.Num, damRoll.Sides) + damRoll.Plus
	} else if weaponDamage.Num > 0 && weaponDamage.Sides > 0 {
		// Player wielding weapon
		dam += RollDice(weaponDamage.Num, weaponDamage.Sides) + weaponDamage.Plus
	} else {
		// Bare hands: number(0, level/3)
		dam += rand.Intn(attacker.GetLevel()/3 + 1)
	}

	// Position multiplier — from fight.c comment:
	//   sitting  x1.33, resting x1.66, sleeping x2.00,
	//   stunned  x2.33, incap   x2.66, mortally  x3.00
	defPos := defender.GetPosition()
	if defPos < POS_FIGHTING {
		dam = dam * (1 + (POS_FIGHTING-defPos)) / 3
	}

	// Minimum 1 damage
	if dam < 1 {
		dam = 1
	}

	return dam
}

// GetAttacksPerRound implements perform_violence() mob attack count from fight.c lines 1904–1922.
// Player attack calculation is not yet implemented (requires class info).
func GetAttacksPerRound(mob Combatant, hasHaste, hasSlow bool) int {
	level := mob.GetLevel()
	attacks := 4

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

	// Random bonus: number(0, 900) < level
	if rand.Intn(901) < level {
		attacks++
	}

	if hasHaste {
		attacks++
	}
	if hasSlow {
		attacks--
	}
	if attacks < 1 {
		attacks = 1
	}
	return attacks
}

// backstabMult returns the backstab multiplier from backstab_mult() in skills.c.
// TODO: port the actual function when we add skills in Phase 3.
func backstabMult(level int) float64 {
	if level >= 30 {
		return 3.0
	} else if level >= 20 {
		return 2.5
	} else if level >= 10 {
		return 2.0
	}
	return 1.5
}

// RollDice rolls num d-sides dice and returns the sum.
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
