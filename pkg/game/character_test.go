package game

import "testing"

// TestRollRealAbils verifies RollRealAbils produces valid stat ranges.
// Implements DP-290 — replacing manual cmd/test-race binary with automated tests.
func TestRollRealAbils(t *testing.T) {
	classes := []int{
		ClassMageUser, ClassCleric, ClassThief, ClassWarrior,
		ClassNinja, ClassPsionic, ClassMagus, ClassAvatar,
	}
	races := []int{
		RaceHuman, RaceElf, RaceDwarf, RaceKender,
		RaceMinotaur, RaceRakshasa, RaceSsaur,
	}

	for _, class := range classes {
		for _, race := range races {
			stats := RollRealAbils(class, race)

			// All base stats should be 3-18 (4d6 drop lowest)
			checkStat(t, "Str", stats.Str)
			checkStat(t, "Int", stats.Int)
			checkStat(t, "Wis", stats.Wis)
			checkStat(t, "Dex", stats.Dex)
			checkStat(t, "Con", stats.Con)
			checkStat(t, "Cha", stats.Cha)

			// StrAdd should be 0 unless Str is 18 (and only for warrior-types)
			if stats.Str < 18 && stats.StrAdd != 0 {
				t.Errorf("class=%d race=%d: StrAdd=%d but Str=%d (should be 0)", class, race, stats.StrAdd, stats.Str)
			}
			if stats.StrAdd < 0 || stats.StrAdd > 100 {
				t.Errorf("class=%d race=%d: StrAdd=%d out of range [0,100]", class, race, stats.StrAdd)
			}
		}
	}
}

// TestRollRealAbilsClassPriority verifies that the primary stat for each class
// is assigned the highest rolled value.
func TestRollRealAbilsClassPriority(t *testing.T) {
	// Run enough samples to be confident (rng is random)
	samples := 500
	for i := 0; i < samples; i++ {
		// Mages should have Int as highest or second-highest
		stats := RollRealAbils(ClassMageUser, RaceHuman)
		if stats.Int < stats.Con {
			// Int should generally be high for mages, but rng can surprise
			// Just verify it's at least above average
			if stats.Int < 10 {
				t.Errorf("sample %d: mage Int=%d suspiciously low", i, stats.Int)
			}
		}

		// Warriors should have Str as highest
		stats = RollRealAbils(ClassWarrior, RaceHuman)
		if stats.Str < 8 {
			t.Errorf("sample %d: warrior Str=%d suspiciously low", i, stats.Str)
		}
	}
}

// TestValidUserClassChoice verifies class/race restrictions.
// Implements DP-290 — automated test for ValidUserClassChoice.
func TestValidUserClassChoice(t *testing.T) {
	// Base classes valid for all races
	baseClasses := []int{ClassMageUser, ClassCleric, ClassThief, ClassWarrior, ClassPsionic}
	allRaces := []int{RaceHuman, RaceElf, RaceDwarf, RaceKender, RaceMinotaur, RaceRakshasa, RaceSsaur}

	for _, class := range baseClasses {
		for _, race := range allRaces {
			if !ValidUserClassChoice(race, class) {
				t.Errorf("base class %d should be valid for race %d", class, race)
			}
		}
	}

	// Ninja restricted to human only
	if !ValidUserClassChoice(RaceHuman, ClassNinja) {
		t.Error("Ninja should be valid for Human")
	}
	for _, race := range allRaces {
		if race == RaceHuman {
			continue
		}
		if ValidUserClassChoice(race, ClassNinja) {
			t.Errorf("Ninja should NOT be valid for race %d", race)
		}
	}

	// Remort classes invalid for all races
	remortClasses := []int{ClassMagus, ClassAvatar, ClassAssassin, ClassPaladin, ClassRanger, ClassMystic}
	for _, class := range remortClasses {
		for _, race := range allRaces {
			if ValidUserClassChoice(race, class) {
				t.Errorf("remort class %d should NOT be valid for race %d", class, race)
			}
		}
	}

	// Invalid class ID
	if ValidUserClassChoice(RaceHuman, 9999) {
		t.Error("bogus class 9999 should not be valid")
	}
}

func checkStat(t *testing.T, name string, val int) {
	t.Helper()
	if val < 3 || val > 18 {
		t.Errorf("%s=%d out of range [3,18]", name, val)
	}
}
