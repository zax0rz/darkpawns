package game

// character.go — class/race definitions and stat rolling
// Source: class.c, structs.h

import "math/rand"

// Class constants — from structs.h / class.c
const (
	ClassMageUser = 0
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

// Race constants — from structs.h
const (
	RaceHuman    = 0
	RaceElf      = 1
	RaceDwarf    = 2
	RaceKender   = 3
	RaceMinotaur = 4
	RaceRakshasa = 5
	RaceSsaur    = 6
)

// ClassAbbrevs maps class int to 2-character abbreviation.
// Source: class.c class_abbrevs[] lines 51–65
var ClassAbbrevs = []string{
	"Mu", // ClassMageUser = 0
	"Cl", // ClassCleric   = 1
	"Th", // ClassThief    = 2
	"Wa", // ClassWarrior  = 3
	"Ma", // ClassMagus    = 4
	"Av", // ClassAvatar   = 5
	"As", // ClassAssassin = 6
	"Pa", // ClassPaladin  = 7
	"Ni", // ClassNinja    = 8
	"Ps", // ClassPsionic  = 9
	"Ra", // ClassRanger   = 10
	"My", // ClassMystic   = 11
}

// ClassNames maps class int to display name
var ClassNames = map[int]string{
	ClassMageUser: "Mage",
	ClassCleric:   "Cleric",
	ClassThief:    "Thief",
	ClassWarrior:  "Warrior",
	ClassMagus:    "Magus",
	ClassAvatar:   "Avatar",
	ClassAssassin: "Assassin",
	ClassPaladin:  "Paladin",
	ClassNinja:    "Ninja",
	ClassPsionic:  "Psionic",
	ClassRanger:   "Ranger",
	ClassMystic:   "Mystic",
}

// RaceNames maps race int to display name
var RaceNames = map[int]string{
	RaceHuman:    "Human",
	RaceElf:      "Elf",
	RaceDwarf:    "Dwarf",
	RaceKender:   "Kender",
	RaceMinotaur: "Minotaur",
	RaceRakshasa: "Rakshasa",
	RaceSsaur:    "Ssaur",
}

// CharStats holds the six base ability scores
type CharStats struct {
	Str    int
	StrAdd int // 18/xx for warriors
	Int    int
	Wis    int
	Dex    int
	Con    int
	Cha    int
}

// RollRealAbils implements roll_real_abils() from class.c lines 380-497.
//
// Original: roll 4d6 drop lowest, six times, sort descending.
// Assign to stats based on class primary stat priority, then apply race bonuses.
func RollRealAbils(class, race int) CharStats {
	// Roll 6 stats: 4d6 drop lowest, sorted descending
	table := rollStatTable()

	var s CharStats

	// Assign stats by class priority — from class.c roll_real_abils()
	switch class {
	case ClassMageUser, ClassMagus, ClassPsionic, ClassMystic:
		s.Int = table[0]
		s.Wis = table[1]
		s.Dex = table[2]
		s.Str = table[3]
		s.Con = table[4]
		s.Cha = table[5]
	case ClassCleric, ClassAvatar:
		s.Wis = table[0]
		s.Int = table[1]
		s.Str = table[2]
		s.Dex = table[3]
		s.Con = table[4]
		s.Cha = table[5]
	case ClassThief, ClassAssassin, ClassNinja:
		s.Dex = table[0]
		s.Str = table[1]
		s.Con = table[2]
		s.Int = table[3]
		s.Wis = table[4]
		s.Cha = table[5]
	case ClassWarrior, ClassPaladin, ClassRanger:
		s.Str = table[0]
		s.Dex = table[1]
		s.Con = table[2]
		s.Wis = table[3]
		s.Int = table[4]
		s.Cha = table[5]
		if s.Str == 18 {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			s.StrAdd = rand.Intn(101) // 0-100
		}
	default:
		s.Str = table[0]
		s.Dex = table[1]
		s.Con = table[2]
		s.Wis = table[3]
		s.Int = table[4]
		s.Cha = table[5]
	}

	// Race bonuses — from class.c roll_real_abils() lines 460-497
	switch race {
	case RaceHuman:
		s.Cha = min18(s.Cha + 1)
	case RaceElf:
		s.Int = min18(s.Int + 1)
		if s.Str == 18 {
			s.StrAdd = 0
		} // Elves cap at 18/00
	case RaceDwarf:
		s.Wis = min18(s.Wis + 1)
	case RaceKender:
		s.Dex = min18(s.Dex + 1)
		if s.Str == 18 {
			s.StrAdd = 0
		}
	case RaceMinotaur:
		s.Str = min18(s.Str + 1)
		if s.Str == 18 && class == ClassWarrior {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			s.StrAdd = rand.Intn(101)
		}
	case RaceRakshasa:
		s.Str = min18(s.Str + 1)
		if s.Str == 18 && class == ClassWarrior {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			s.StrAdd = rand.Intn(101)
		}
	case RaceSsaur:
		s.Con = min18(s.Con + 1)
		s.Wis = min18(s.Wis) // CAP wis at 16 (handled by min18)
		if s.Wis > 16 {
			s.Wis = 16
		}
	}

	return s
}

// rollStatTable rolls 6 stats (4d6 drop lowest) and sorts descending.
// From roll_real_abils(): "best 3 out of 4 rolls of a 6-sided die"
func rollStatTable() [6]int {
	var table [6]int
	for i := 0; i < 6; i++ {
		rolls := [4]int{
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			rand.Intn(6) + 1,
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			rand.Intn(6) + 1,
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			rand.Intn(6) + 1,
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			rand.Intn(6) + 1,
		}
		// Sum of best 3 (drop lowest)
		min := rolls[0]
		for _, r := range rolls {
			if r < min {
				min = r
			}
		}
		total := rolls[0] + rolls[1] + rolls[2] + rolls[3] - min
		// Insert sorted descending (bubble up)
		for k := 0; k < 6; k++ {
			if table[k] < total {
				table[k], total = total, table[k]
			}
		}
	}
	return table
}

func min18(v int) int {
	if v > 18 {
		return 18
	}
	return v
}

// ValidUserClassChoice implements valid_user_class_choice() from interpreter.c lines 1673-1688.
// Returns true if the race/class combination is valid for new character creation.
// Only CLASS_NINJA is restricted to RACE_HUMAN.
// Valid classes for all races: Mage, Cleric, Thief, Warrior, Psionic
// All other classes (Magus, Avatar, Assassin, Paladin, Ranger, Mystic) are remort-only.
func ValidUserClassChoice(race, class int) bool {
	switch class {
	case ClassNinja:
		if race != RaceHuman {
			return false
		}
		fallthrough
	case ClassMageUser, ClassCleric, ClassThief, ClassWarrior, ClassPsionic:
		return true
	}
	return false
}

// DoStart implements do_start() from class.c lines 501-591.
// Sets level 1 stats and returns starting item VNums to give the player.
//
// Starting items by class (class.c):
//
//	Thief:   backpack(8038)+lockpicks(8027), dagger(8036)
//	Mage:    dagger(8036), 2x obsidian(1239)
//	Ninja:   dagger(8036)
//	Warrior/Psionic: small sword(8037)
//	Others:  club(8023)
//	All:     tunic(8019), pack(8038) with bread(8010) + waterskin(8063)
type StartItems struct {
	Carried []int // vnums to give directly to player
	InPack  []int // vnums to put in pack(8038)
}

// DoStart returns starting items and base stats for a new character of the given class.
// Implements do_start() from class.c lines 501-591.
// The returned CharStats is zeroed; callers should use RollRealAbils() for actual stats.
func DoStart(class int) (StartItems, CharStats) {
	// Stats will be properly rolled — for now return zeroed
	// (caller should pass in already-rolled stats)
	var items StartItems

	// Pack always goes to player with bread + waterskin inside
	items.InPack = []int{8010, 8063} // bread, waterskin

	switch class {
	case ClassThief, ClassAssassin:
		items.Carried = append(items.Carried, 8036) // dagger
		items.InPack = append(items.InPack, 8027)   // lockpicks in pack
	case ClassMageUser, ClassMagus:
		items.Carried = append(items.Carried, 8036)       // dagger
		items.Carried = append(items.Carried, 1239, 1239) // 2x obsidian
	case ClassNinja:
		items.Carried = append(items.Carried, 8036) // dagger
	case ClassWarrior, ClassPsionic:
		items.Carried = append(items.Carried, 8037) // small sword
	default:
		items.Carried = append(items.Carried, 8023) // club
	}

	items.Carried = append(items.Carried, 8019) // tunic (all classes)
	// Pack vnum 8038 is created separately and filled with InPack items

	return items, CharStats{}
}

// GiveStartingSkills assigns starting skills to a player based on class and race.
// Source: class.c:554-570 (do_start skill assignments)
func GiveStartingSkills(p *Player) {
	// Thief/Assassin starting skills — class.c:554-562
	if p.Class == ClassThief || p.Class == ClassAssassin {
		p.SetSkill("sneak", 10)
		p.SetSkill("hide", 5)
		p.SetSkill("peek", 15)
		p.SetSkill("steal", 15)
		p.SetSkill("backstab", 10)
		p.SetSkill("pick_lock", 10)
	}

	// Kender racial skill — class.c:567
	if p.Race == RaceKender {
		p.SetSkill("steal", 25)
	}

	// Minotaur racial skill — class.c:569
	if p.Race == RaceMinotaur {
		p.SetSkill("headbutt", 25)
	}
}

