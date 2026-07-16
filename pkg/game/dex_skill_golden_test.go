package game

import "testing"

// dexAppSkillsGolden is transcribed verbatim from src/constants.c:1060-1087.
var dexAppSkillsGolden = [26][5]int{
	{-99, -99, -90, -99, -60},
	{-90, -90, -60, -90, -50},
	{-80, -80, -40, -80, -45},
	{-70, -70, -30, -70, -40},
	{-60, -60, -30, -60, -35},
	{-50, -50, -20, -50, -30},
	{-40, -40, -20, -40, -25},
	{-30, -30, -15, -30, -20},
	{-20, -20, -15, -20, -15},
	{-15, -10, -10, -20, -10},
	{-10, -5, -10, -15, -5},
	{-5, 0, -5, -10, 0},
	{0, 0, 0, -5, 0},
	{0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0},
	{0, 5, 0, 0, 0},
	{5, 10, 0, 5, 5},
	{10, 15, 5, 10, 10},
	{15, 20, 10, 15, 15},
	{15, 20, 10, 15, 15},
	{20, 25, 10, 15, 20},
	{20, 25, 15, 20, 20},
	{25, 25, 15, 20, 20},
	{25, 30, 15, 25, 25},
	{25, 30, 15, 25, 25},
}

func TestDexAppSkillsGoldenAgainstCSource(t *testing.T) {
	if len(dexAppSkills) != len(dexAppSkillsGolden) {
		t.Fatalf("dexAppSkills length = %d; want %d", len(dexAppSkills), len(dexAppSkillsGolden))
	}
	for dex, want := range dexAppSkillsGolden {
		got := dexAppSkills[dex]
		if got.PPocket != want[0] || got.PLocks != want[1] || got.Traps != want[2] || got.Sneak != want[3] || got.Hide != want[4] {
			t.Errorf("dexAppSkills[%d] = {%d, %d, %d, %d, %d}; want %v",
				dex, got.PPocket, got.PLocks, got.Traps, got.Sneak, got.Hide, want)
		}
	}
}

func TestDexAppSkillClamps(t *testing.T) {
	if got := dexAppSkill(-1); got != dexAppSkills[0] {
		t.Errorf("dexAppSkill(-1) = %+v; want dex 0 row %+v", got, dexAppSkills[0])
	}
	if got := dexAppSkill(26); got != dexAppSkills[25] {
		t.Errorf("dexAppSkill(26) = %+v; want dex 25 row %+v", got, dexAppSkills[25])
	}
}
