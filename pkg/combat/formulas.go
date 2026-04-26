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
	PosDead     = 0
	PosMortally = 1
	PosIncap    = 2
	PosStunned  = 3
	PosSleeping = 4
	PosResting  = 5
	PosSitting  = 6
	PosFighting = 7
	PosStanding = 8
)

// Class constants — from class.h (NUM_CLASSES = 13)
const (
	ClassMage     = 0 // Deprecated: use game.ClassMageUser instead
	ClassCleric   = 1
	ClassThief    = 2
	ClassWarrior  = 3
	ClassMagus    = 4
	ClassAvatar   = 5
	ClassAssassin = 6
	ClassPaladin  = 7
	ClassNinja    = 8
	ClassPsionic  = 9
	ClassRanger   = 10
	ClassMystic   = 11
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
	ToHit int
	ToDam int
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
	{-7, -7, 6}, // 0
	{-6, -6, 5}, // 1
	{-4, -4, 5}, // 2
	{-3, -3, 4}, // 3
	{-2, -2, 3}, // 4
	{-1, -1, 2}, // 5
	{0, 0, 1},   // 6
	{0, 0, 0},   // 7
	{0, 0, 0},   // 8
	{0, 0, 0},   // 9
	{0, 0, 0},   // 10
	{0, 0, 0},   // 11
	{0, 0, 0},   // 12
	{0, 0, 0},   // 13
	{0, 0, 0},   // 14
	{0, 0, -1},  // 15
	{1, 1, -2},  // 16
	{2, 2, -3},  // 17
	{2, 2, -4},  // 18
	{3, 3, -4},  // 19
	{3, 3, -4},  // 20
	{4, 4, -5},  // 21
	{4, 4, -5},  // 22
	{4, 4, -5},  // 23
	{5, 5, -6},  // 24
	{5, 5, -6},  // 25
}

// strIndex returns the str_app index for a combatant.
// Implements STRENGTH_APPLY_INDEX macro from utils.h line 440
// Source: utils.h: STRENGTH_APPLY_INDEX(ch) macro
func strIndex(c Combatant) int {
	str := c.GetStr()
	strAdd := c.GetStrAdd()

	if strAdd == 0 || str != 18 {
		return str
	}

	// Handle 18/xx exceptional strength
	if strAdd <= 50 {
		return 26 // 18/01-50
	}
	if strAdd <= 75 {
		return 27 // 18/51-75
	}
	if strAdd <= 90 {
		return 28 // 18/76-90
	}
	if strAdd <= 99 {
		return 29 // 18/91-99
	}
	return 30 // 18/100
}

// dexIndex returns the dex_app index for a combatant.
// Source: fight.c uses GET_DEX(ch) directly for dex_app index
func dexIndex(c Combatant) int {
	dex := c.GetDex()
	if dex < 0 {
		dex = 0
	} else if dex > 25 {
		dex = 25
	}
	return dex
}

// getTHAC0 returns the base THAC0 for a combatant.
// Mobs always use 20 (from fight.c line 1786).
// Players use the class/level table.
// Source: fight.c line 1784-1785
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
	class := c.GetClass()
	if class < 0 || class >= len(thaco) {
		class = ClassWarrior // Default to warrior if invalid class
	}
	return thaco[class][level]
}

// HitModifiers holds optional combat modifiers for CalculateHitChance.
// Zero values mean "not applicable."
type HitModifiers struct {
	WeaponBlessed bool // ITEM_BLESS on wielded weapon: -1 to calc_thaco
	DrunkLevel    int  // GET_COND(ch, DRUNK): +2 to calc_thaco if > 1
}

// CalculateHitChance implements the original hit() logic from fight.c lines 1783–1830.
//
// Original formula:
//
//	calc_thaco = thaco[class][level]           (players) or 20 (mobs)
//	calc_thaco -= str_app[str_index].tohit
//	calc_thaco -= GET_HITROLL(ch)              (fight.c line 1812)
//	calc_thaco -= (INT-13)/1.5                 (fight.c line 1813)
//	calc_thaco -= (WIS-13)/1.5                 (fight.c line 1814)
//	diceroll = number(1,20)
//	victim_ac = GET_AC(victim)/10
//	if AWAKE: victim_ac += dex_app[dex].defensive
//	victim_ac = max(-10, victim_ac)
//	MISS if: diceroll < 20 AND AWAKE AND (diceroll==1 OR calc_thaco-diceroll > victim_ac)
//	HIT  otherwise
func CalculateHitChance(attacker, defender Combatant, mods HitModifiers) bool {
	calcThaco := getTHAC0(attacker)
	calcThaco -= strApp[strIndex(attacker)].ToHit
	calcThaco -= attacker.GetHitroll() // fight.c line 1812

	// Blessed weapon THAC0 bonus - fight.c line 1796
	if mods.WeaponBlessed {
		calcThaco -= 1
	}

	// Drunk THAC0 penalty - fight.c line 1806
	if mods.DrunkLevel > 1 {
		calcThaco += 2
	}

	// INT and WIS THAC0 reduction - fight.c lines 1813-1814
	intBonus := int(float64(attacker.GetInt()-13) / 1.5)
	wisBonus := int(float64(attacker.GetWis()-13) / 1.5)
	calcThaco -= intBonus
	calcThaco -= wisBonus

	diceroll := rand.Intn(20) + 1

	victimAC := defender.GetAC() / 10
	// Assume defender is awake (position >= PosSleeping)
	if defender.GetPosition() > PosSleeping {
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
	awake := defender.GetPosition() > PosSleeping
	if awake && (diceroll == 1 || (calcThaco-diceroll) > victimAC) {
		return false // miss
	}
	return true // hit
}

// getMinusDam implements get_minusdam() from fight.c lines 1722-1760.
// Reduces damage based on victim's AC.
// Source: fight.c get_minusdam() function
func getMinusDam(dam int, ac int) int {
	pcmod := 2.0 // Player character modifier

	// Note: In original, lower AC is better (negative values).
	// The function checks if ac > X, meaning less negative (worse armor).
	if ac > 90 {
		return dam
	}
	if ac > 80 {
		return dam - int(float64(dam)*(0.01*pcmod))
	}
	if ac > 70 {
		return dam - int(float64(dam)*(0.02*pcmod))
	}
	if ac > 60 {
		return dam - int(float64(dam)*(0.03*pcmod))
	}
	if ac > 50 {
		return dam - int(float64(dam)*(0.04*pcmod))
	}
	if ac > 40 {
		return dam - int(float64(dam)*(0.05*pcmod))
	}
	if ac > 30 {
		return dam - int(float64(dam)*(0.06*pcmod))
	}
	if ac > 20 {
		return dam - int(float64(dam)*(0.07*pcmod))
	}
	if ac > 10 {
		return dam - int(float64(dam)*(0.08*pcmod))
	}
	if ac > 0 {
		return dam - int(float64(dam)*(0.10*pcmod))
	}
	if ac > -10 {
		return dam - int(float64(dam)*(0.11*pcmod))
	}
	if ac > -20 {
		return dam - int(float64(dam)*(0.12*pcmod))
	}
	if ac > -30 {
		return dam - int(float64(dam)*(0.13*pcmod))
	}
	if ac > -40 {
		return dam - int(float64(dam)*(0.14*pcmod))
	}
	if ac > -50 {
		return dam - int(float64(dam)*(0.15*pcmod))
	}
	if ac > -60 {
		return dam - int(float64(dam)*(0.16*pcmod))
	}
	if ac > -70 {
		return dam - int(float64(dam)*(0.17*pcmod))
	}
	if ac > -80 {
		return dam - int(float64(dam)*(0.18*pcmod))
	}
	if ac > -90 {
		return dam - int(float64(dam)*(0.19*pcmod))
	}
	if ac > -95 {
		return dam - int(float64(dam)*(0.20*pcmod))
	}
	if ac > -110 {
		return dam - int(float64(dam)*(0.21*pcmod))
	}
	if ac > -130 {
		return dam - int(float64(dam)*(0.22*pcmod))
	}
	if ac > -150 {
		return dam - int(float64(dam)*(0.23*pcmod))
	}

	if ac > -170 {
		return dam - int(float64(dam)*(0.24*pcmod))
	}
	if ac > -190 {
		return dam - int(float64(dam)*(0.25*pcmod))
	}
	if ac > -210 {
		return dam - int(float64(dam)*(0.26*pcmod))
	}
	if ac > -230 {
		return dam - int(float64(dam)*(0.27*pcmod))
	}
	if ac > -250 {
		return dam - int(float64(dam)*(0.28*pcmod))
	}
	if ac > -270 {
		return dam - int(float64(dam)*(0.29*pcmod))
	}
	if ac > -290 {
		return dam - int(float64(dam)*(0.30*pcmod))
	}
	if ac > -310 {
		return dam - int(float64(dam)*(0.31*pcmod))
	}

	// ac <= -310
	return dam - int(float64(dam)*(0.32*pcmod))
}

// CalculateDamage implements the original damage calculation from fight.c lines 1840–1858.
//
// Original formula:
//
//	dam = str_app[str_index].todam
//	dam += GET_DAMROLL(ch)             (fight.c line 1840)
//	if player+wielding weapon: dam += dice(weapon_val1, weapon_val2)
//	if mob: dam += dice(damnodice, damsizedice)
//	if player+no weapon: dam += number(0, level/3)
//	if victim position < PosFighting: dam *= 1 + (PosFighting-pos)/3
//	dam = get_minusdam(dam, victim)    (fight.c line 1882)
func CalculateDamage(attacker, defender Combatant, weaponDamage DiceRoll, attackType AttackType) int {
	dam := strApp[strIndex(attacker)].ToDam
	dam += attacker.GetDamroll() // fight.c line 1840

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
	// Source: fight.c line 1859: dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3
	// Note: C uses integer division, so sitting/resting = no change (1/3=0, 2/3=0),
	// sleeping = x2 (3/3=1), stunned = x2, incap = x2, mortally = x3.
	// Comments in C claim 1.33/1.66 etc but integer math doesn't produce those.
	defPos := defender.GetPosition()
	if defPos < PosFighting {
		delta := PosFighting - defPos
		dam *= 1 + delta/3
	}

	// Apply AC damage reduction (get_minusdam) - fight.c line 1882
	// Only for normal weapon hits, not spells
	if attackType == AttackNormal {
		dam = getMinusDam(dam, defender.GetAC())
	}

	// Minimum 1 damage
	if dam < 1 {
		dam = 1
	}

	return dam
}

// GetAttacksPerRound implements perform_violence() attack count from fight.c lines 1904–1945.
// Source: fight.c perform_violence() function
func GetAttacksPerRound(c Combatant, hasHaste, hasSlow bool) int {
	attacks := 1
	if c.IsNPC() {
		// Mob attack calculation - fight.c lines 1904-1922
		level := c.GetLevel()
		attacks = 4

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
	} else {
		// Player attack calculation - fight.c lines 1924-1945
		attacks = 1
		level := c.GetLevel()
		class := c.GetClass()

		// Warriors/Paladins/Rangers: +1 at level 10+ (60% + level% chance)
		if (class == ClassWarrior || class == ClassPaladin || class == ClassRanger) &&
			level > 10 && rand.Intn(100) < (60+level) {
			attacks++
		}

		// Ninjas/Avatars: +1 at level 12+ (60% + level% chance)
		if (class == ClassNinja || class == ClassAvatar) &&
			level > 12 && rand.Intn(100) < (60+level) {
			attacks++
		}

		// Thieves/Assassins: +1 at level 15+ (30% + level% chance)
		if (class == ClassThief || class == ClassAssassin) &&
			level > 15 && rand.Intn(100) < (30+level) {
			attacks++
		}

		// All players: +1 at level 25+ (75% chance)
		if level > 25 && rand.Intn(100) < 75 {
			attacks++
		}

		// All players: +1 at level 30+ OR !number(0,500)
		if level > 30 || rand.Intn(501) == 0 {
			attacks++
		}

		// All players: +2 at level 39+
		if level > 39 {
			attacks += 2
		}
	}

	// Haste/Slow effects - fight.c lines 1922, 1944-1945
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
// backstabMult returns the backstab damage multiplier.
// Source: class.c:720-729 backstab_mult()
// Original: if level < LVL_IMMORT return (level*0.2)+1; else return 20
// LVL_IMMORT = 31 in this codebase (structs.h)
func backstabMult(level int) float64 {
	if level <= 0 {
		return 1.0
	}
	if level >= 31 { // LVL_IMMORT — structs.h
		return 20.0
	}
	return float64(level)*0.2 + 1.0
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
