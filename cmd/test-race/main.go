package main

import (
	"fmt"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func main() {
	// Test all race bonuses
	races := []struct {
		name string
		id   int
	}{
		{"Human", game.RaceHuman},
		{"Elf", game.RaceElf},
		{"Dwarf", game.RaceDwarf},
		{"Kender", game.RaceKender},
		{"Minotaur", game.RaceMinotaur},
		{"Rakshasa", game.RaceRakshasa},
		{"Ssaur", game.RaceSsaur},
	}

	classes := make([]struct {
		name string
		id   int
	}, 0, len(game.ClassNames))
	for id, name := range game.ClassNames {
		classes = append(classes, struct {
			name string
			id   int
		}{name, id})
	}
	sort.Slice(classes, func(i, j int) bool {
		return classes[i].id < classes[j].id
	})

	// Build reverse lookup from game.ClassNames for test cases
	allClassNames := game.ClassNames

	fmt.Println("Testing race bonuses for each class:")
	fmt.Println("=====================================")

	for _, class := range classes {
		fmt.Printf("\nClass: %s\n", class.name)
		fmt.Println("Race\t\tSTR\tINT\tWIS\tDEX\tCON\tCHA\tStrAdd")
		fmt.Println("----\t\t---\t---\t---\t---\t---\t---\t------")

		for _, race := range races {
			// Roll stats multiple times to see patterns
			var totalStats game.CharStats
			samples := 1000

			for i := 0; i < samples; i++ {
				stats := game.RollRealAbils(class.id, race.id)
				totalStats.Str += stats.Str
				totalStats.Int += stats.Int
				totalStats.Wis += stats.Wis
				totalStats.Dex += stats.Dex
				totalStats.Con += stats.Con
				totalStats.Cha += stats.Cha
				totalStats.StrAdd += stats.StrAdd
			}

			// Calculate averages
			avgStr := float64(totalStats.Str) / float64(samples)
			avgInt := float64(totalStats.Int) / float64(samples)
			avgWis := float64(totalStats.Wis) / float64(samples)
			avgDex := float64(totalStats.Dex) / float64(samples)
			avgCon := float64(totalStats.Con) / float64(samples)
			avgCha := float64(totalStats.Cha) / float64(samples)
			avgStrAdd := float64(totalStats.StrAdd) / float64(samples)

			fmt.Printf("%-12s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n",
				race.name, avgStr, avgInt, avgWis, avgDex, avgCon, avgCha, avgStrAdd)
		}
	}

	// Test class/race restrictions
	fmt.Println("\n\nTesting class/race restrictions:")
	fmt.Println("================================")

	testCases := []struct {
		race  int
		class int
		valid bool
	}{
		{game.RaceHuman, game.ClassNinja, true},       // Ninja: Human only
		{game.RaceElf, game.ClassNinja, false},        // Ninja: non-Human rejected
		{game.RaceDwarf, game.ClassNinja, false},      // Ninja: Dwarf rejected
		{game.RaceHuman, game.ClassMageUser, true},    // Mage: all races
		{game.RaceMinotaur, game.ClassMageUser, true}, // Mage: Minotaur allowed
		{game.RaceRakshasa, game.ClassWarrior, true},  // Warrior: all races
		{game.RaceSsaur, game.ClassCleric, true},      // Cleric: all races
		{game.RaceHuman, game.ClassThief, true},       // Thief: all races
		{game.RaceHuman, game.ClassPsionic, true},     // Psionic: all races
		{game.RaceHuman, game.ClassMagus, false},      // Magus: remort-only
		{game.RaceHuman, game.ClassAvatar, false},     // Avatar: remort-only
		{game.RaceHuman, game.ClassAssassin, false},   // Assassin: remort-only
		{game.RaceHuman, game.ClassPaladin, false},    // Paladin: remort-only
		{game.RaceHuman, game.ClassRanger, false},     // Ranger: remort-only
		{game.RaceHuman, game.ClassMystic, false},     // Mystic: remort-only
	}

	for _, tc := range testCases {
		valid := game.ValidUserClassChoice(tc.race, tc.class)
		raceName := "Unknown"

		// Get race name
		for _, r := range races {
			if r.id == tc.race {
				raceName = r.name
				break
			}
		}

		// Get class name from game.ClassNames (covers all classes)
		className := allClassNames[tc.class]
		if className == "" {
			className = "Unknown"
		}

		status := "✓"
		if valid != tc.valid {
			status = "✗"
		}

		fmt.Printf("%s %-8s %-8s: expected %v, got %v\n",
			status, raceName, className, tc.valid, valid)
	}
}
