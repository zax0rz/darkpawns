package game

// dexSkillType mirrors struct dex_skill_type from src/structs.h:1250-1256.
type dexSkillType struct {
	PPocket int
	PLocks  int
	Traps   int
	Sneak   int
	Hide    int
}

// dexAppSkills is the C dex_app_skill[] table from src/constants.c:1060-1087.
// It is indexed by dexterity and is used by thief utility skills.
var dexAppSkills = [...]dexSkillType{
	{-99, -99, -90, -99, -60}, // dex = 0
	{-90, -90, -60, -90, -50}, // dex = 1
	{-80, -80, -40, -80, -45},
	{-70, -70, -30, -70, -40},
	{-60, -60, -30, -60, -35},
	{-50, -50, -20, -50, -30}, // dex = 5
	{-40, -40, -20, -40, -25},
	{-30, -30, -15, -30, -20},
	{-20, -20, -15, -20, -15},
	{-15, -10, -10, -20, -10},
	{-10, -5, -10, -15, -5}, // dex = 10
	{-5, 0, -5, -10, 0},
	{0, 0, 0, -5, 0},
	{0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0}, // dex = 15
	{0, 5, 0, 0, 0},
	{5, 10, 0, 5, 5},
	{10, 15, 5, 10, 10}, // dex = 18
	{15, 20, 10, 15, 15},
	{15, 20, 10, 15, 15}, // dex = 20
	{20, 25, 10, 15, 20},
	{20, 25, 15, 20, 20},
	{25, 25, 15, 20, 20},
	{25, 30, 15, 25, 25},
	{25, 30, 15, 25, 25}, // dex = 25
}

// dexAppSkill returns the clamped dexterity row. C indexes the table directly,
// but Go stat affects can temporarily take a score outside the normal 0..25
// range, so the accessor keeps every caller in bounds.
func dexAppSkill(dex int) dexSkillType {
	if dex < 0 {
		dex = 0
	} else if dex >= len(dexAppSkills) {
		dex = len(dexAppSkills) - 1
	}
	return dexAppSkills[dex]
}
