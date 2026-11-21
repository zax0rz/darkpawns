package game

import "testing"

// Tier-2 fidelity golden test (deterministic — no RNG, no game-code change).
//
// These tests pin the XP-per-level curve from C's find_exp
// (src/class.c:1089-1180). It is the fidelity ground truth:
// FindExp MUST reproduce these exactly, or leveling progression in the Go port diverges
// from the original game.
//
// The C function returns a fixed value for levels 0–12, then applies a formula for 13+.
// We test both: the fixed values directly, and the formula output for each class modifier.
// Index is [class][level]; level 0 is the sentinel (1), levels 1–12 are fixed, 13–40 use formula.

// findExpFixed are C's hardcoded XP values for levels 0–12
// (src/class.c:1131-1158).
var findExpFixed = [13]int{
	1,      // level 0
	1500,   // level 1
	3000,   // level 2
	6000,   // level 3
	11000,  // level 4
	21000,  // level 5
	42000,  // level 6
	80000,  // level 7
	155000, // level 8
	300000, // level 9
	450000, // level 10
	650000, // level 11
	870000, // level 12
}

// classModifiers maps class constants to the multipliers in src/class.c:1093-1128.
var classModifiers = map[int]float64{
	ClassMageUser: 0.3,
	ClassCleric:   0.4,
	ClassWarrior:  0.7,
	ClassThief:    0.1,
	ClassMagus:    1.5,
	ClassMystic:   1.5,
	ClassAvatar:   1.6,
	ClassAssassin: 1.2,
	ClassPaladin:  1.9,
	ClassRanger:   1.9,
	ClassNinja:    0.6,
	ClassPsionic:  0.6,
}

// classNamesGame maps class constants to names for test output.
var classNamesGame = map[int]string{
	ClassMageUser: "Mage",
	ClassCleric:   "Cleric",
	ClassWarrior:  "Warrior",
	ClassThief:    "Thief",
	ClassMagus:    "Magus",
	ClassMystic:   "Mystic",
	ClassAvatar:   "Avatar",
	ClassAssassin: "Assassin",
	ClassPaladin:  "Paladin",
	ClassRanger:   "Ranger",
	ClassNinja:    "Ninja",
	ClassPsionic:  "Psionic",
}

// TestFindExp_GoldenFixedLevels asserts FindExp reproduces the C hardcoded values for levels 0–12.
// These are class-independent (same value for every class).
func TestFindExp_GoldenFixedLevels(t *testing.T) {
	for level, want := range findExpFixed {
		got := FindExp(ClassWarrior, level) // class doesn't matter for fixed levels
		if got != want {
			t.Errorf("FindExp(Warrior, %d) = %d, want %d (per src/class.c)", level, got, want)
		}
	}
}

// TestFindExp_GoldenFormulaLevel13Plus asserts FindExp reproduces C's formula
// for levels 13–40 across every class (src/class.c:1177-1178):
//
//	return 900000 + ((level-13) * level * 20000) + (level*level*1000) + (modifier*10000*level)
//
// We compute the expected value from the C formula and compare.
func TestFindExp_GoldenFormulaLevel13Plus(t *testing.T) {
	for class, modifier := range classModifiers {
		name := classNamesGame[class]
		for level := 13; level <= 40; level++ {
			// C formula: 900000 + ((level-13) * level * 20000) + (level*level*1000) + (modifier*10000*level)
			want := 900000 + ((level - 13) * level * 20000) + (level * level * 1000) + int(modifier*10000*float64(level))
			got := FindExp(class, level)
			if got != want {
				t.Errorf("FindExp(%s, %d) = %d, want %d (per src/class.c formula, modifier=%.1f)",
					name, level, got, want, modifier)
			}
		}
	}
}
