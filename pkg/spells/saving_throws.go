package spells

import (
	"math/rand"
)

// numClasses is the number of playable classes in the C source (MAGIC_USER=0, CLERIC=1, etc.)
const numClasses = 12

// maxLevel is the max level for saving throw tables
const maxLevel = 21

// savThrows[class][level-1][saveType] = target roll needed to succeed
// Ported from src/spell_parser.c sav_throws[][][].
// Rows: MAGIC_USER=0, CLERIC=1, THIEF=2, WARRIOR=3, RANGER=4,
//        PSIONIC=5, BARBARIAN=6, MYSTIC=7, ROGUE=8, DRUID=9,
//        ASSASSIN=10, PALADIN=11
// Cols per class: level 1..21
// Entries per row: [para, rod, petri, breath, spell]
var savThrows = [numClasses][maxLevel][SaveCount]int{
	// MAGIC_USER (0)
	{
		{14, 14, 13, 16, 15},  // lvl 1
		{14, 14, 13, 16, 15},  // lvl 2
		{14, 14, 13, 16, 14},  // lvl 3
		{14, 14, 13, 16, 14},  // lvl 4
		{14, 14, 13, 16, 13},  // lvl 5
		{13, 13, 11, 14, 13},  // lvl 6 (transition)
		{13, 13, 11, 14, 12},
		{13, 13, 11, 14, 12},
		{13, 13, 11, 14, 11},
		{13, 13, 11, 14, 11},
		{11, 11, 9, 12, 10},
		{11, 11, 9, 12, 10},
		{11, 11, 9, 12, 9},
		{11, 11, 9, 12, 9},
		{11, 11, 9, 12, 8},
		{10, 10, 7, 10, 8},
		{10, 10, 7, 10, 7},
		{10, 10, 7, 10, 7},
		{10, 10, 7, 10, 6},
		{10, 10, 7, 10, 6},
		{10, 10, 7, 10, 5},
	},
	// CLERIC (1)
	{
		{12, 14, 13, 16, 15},  // lvl 1
		{12, 14, 13, 16, 15},
		{12, 14, 13, 16, 15},
		{12, 14, 13, 16, 15},
		{12, 14, 13, 16, 15},
		{10, 12, 11, 14, 12},  // lvl 6
		{10, 12, 11, 14, 12},
		{10, 12, 11, 14, 12},
		{10, 12, 11, 14, 12},
		{10, 12, 11, 14, 12},
		{8, 10, 9, 12, 10},
		{8, 10, 9, 12, 10},
		{8, 10, 9, 12, 10},
		{8, 10, 9, 12, 10},
		{8, 10, 9, 12, 10},
		{6, 8, 7, 10, 8},
		{6, 8, 7, 10, 8},
		{6, 8, 7, 10, 8},
		{6, 8, 7, 10, 8},
		{6, 8, 7, 10, 8},
		{6, 8, 7, 10, 8},
	},
	// THIEF (2)
	{
		{13, 14, 13, 16, 15},
		{13, 14, 13, 16, 15},
		{13, 14, 13, 16, 15},
		{13, 14, 13, 16, 15},
		{13, 14, 13, 16, 15},
		{11, 12, 11, 14, 13},
		{11, 12, 11, 14, 13},
		{11, 12, 11, 14, 13},
		{11, 12, 11, 14, 13},
		{11, 12, 11, 14, 13},
		{9, 10, 9, 12, 11},
		{9, 10, 9, 12, 11},
		{9, 10, 9, 12, 11},
		{9, 10, 9, 12, 11},
		{9, 10, 9, 12, 11},
		{7, 8, 7, 10, 9},
		{7, 8, 7, 10, 9},
		{7, 8, 7, 10, 9},
		{7, 8, 7, 10, 9},
		{7, 8, 7, 10, 9},
		{7, 8, 7, 10, 9},
	},
	// WARRIOR (3)
	{
		{14, 15, 14, 17, 16},
		{14, 15, 14, 17, 16},
		{14, 15, 14, 17, 16},
		{14, 15, 14, 17, 16},
		{14, 15, 14, 17, 16},
		{12, 13, 11, 14, 13},
		{12, 13, 11, 14, 13},
		{12, 13, 11, 14, 13},
		{12, 13, 11, 14, 13},
		{12, 13, 11, 14, 13},
		{10, 11, 9, 12, 11},
		{10, 11, 9, 12, 11},
		{10, 11, 9, 12, 11},
		{10, 11, 9, 12, 11},
		{10, 11, 9, 12, 11},
		{8, 9, 7, 10, 9},
		{8, 9, 7, 10, 9},
		{8, 9, 7, 10, 9},
		{8, 9, 7, 10, 9},
		{8, 9, 7, 10, 9},
		{8, 9, 7, 10, 9},
	},
	// RANGER (4)
	{
		{14, 15, 14, 16, 15},
		{14, 15, 14, 16, 15},
		{14, 15, 14, 16, 15},
		{14, 15, 14, 16, 15},
		{14, 15, 14, 16, 15},
		{12, 13, 12, 14, 13},
		{12, 13, 12, 14, 13},
		{12, 13, 12, 14, 13},
		{12, 13, 12, 14, 13},
		{12, 13, 12, 14, 13},
		{10, 11, 10, 12, 11},
		{10, 11, 10, 12, 11},
		{10, 11, 10, 12, 11},
		{10, 11, 10, 12, 11},
		{10, 11, 10, 12, 11},
		{8, 9, 8, 10, 9},
		{8, 9, 8, 10, 9},
		{8, 9, 8, 10, 9},
		{8, 9, 8, 10, 9},
		{8, 9, 8, 10, 9},
		{8, 9, 8, 10, 9},
	},
	// Additional classes simplified — same structure, using CLERIC saves as placeholder
	// PSIONIC (5), BARBARIAN (6), MYSTIC (7), ROGUE (8), DRUID (9), ASSASSIN (10), PALADIN (11)
}

func init() {
	// Fill remaining classes with their saving throw tables.
	// For now, copy from nearest analog class.

	// PSIONIC (5) — use magic-user saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[5][lvl] = savThrows[0][lvl]
	}
	// BARBARIAN (6) — use warrior saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[6][lvl] = savThrows[3][lvl]
	}
	// MYSTIC (7) — use cleric saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[7][lvl] = savThrows[1][lvl]
	}
	// ROGUE (8) — use thief saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[8][lvl] = savThrows[2][lvl]
	}
	// DRUID (9) — use cleric saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[9][lvl] = savThrows[1][lvl]
	}
	// ASSASSIN (10) — use thief saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[10][lvl] = savThrows[2][lvl]
	}
	// PALADIN (11) — use warrior saves
	for lvl := 0; lvl < maxLevel; lvl++ {
		savThrows[11][lvl] = savThrows[3][lvl]
	}
}

// GetSavingThrow gets the target saving throw for a character of given class and level.
// class: character class (0-11)
// level: character level (1-50+, clamped to maxLevel)
// saveType: PARALYSIS=0, ROD=1, PETRIFY=2, BREATH=3, SPELL=4
// Returns the target number the d20 roll must meet or exceed to save.
func GetSavingThrow(class, level int, saveType SavingThrowType) int {
	if class < 0 || class >= numClasses {
		class = 0
	}
	idx := level - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= maxLevel {
		idx = maxLevel - 1
	}
	if saveType < 0 || int(saveType) >= SaveCount {
		saveType = SaveSpell
	}
	return savThrows[class][idx][saveType]
}

// CheckSavingThrow rolls a d20 and returns true if the character saves against
// the given save type. (roll >= target = save succeeds)
// ch must implement:
//
//	GetClass() int
//	GetLevel() int
func CheckSavingThrow(ch interface{}, saveType SavingThrowType) bool {
	caster := getClassGetter(ch)
	if caster == nil {
		return false
	}
	target := GetSavingThrow(caster.GetClass(), caster.GetLevel(), saveType)
	roll := rand.Intn(20) + 1
	return roll >= target
}

// classGetter is a minimal interface for saving throw checks.
type classGetter interface {
	GetClass() int
	GetLevel() int
}

func getClassGetter(ch interface{}) classGetter {
	if ch == nil {
		return nil
	}
	cg, ok := ch.(classGetter)
	if !ok {
		return nil
	}
	return cg
}

// Dice rolls N dice of S sides and returns the total (e.g. dice(2,6) = 2d6).
func Dice(num, sides int) int {
	if num <= 0 || sides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}
