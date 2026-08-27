package game

import "testing"

// how_good bands from src/spec_procs.c:108 — every string carries a LEADING space.
func TestHowGoodBands(t *testing.T) {
	cases := []struct {
		pct  int
		want string
	}{
		{0, " (not learned)"},
		{1, " (awful)"},
		{10, " (awful)"},
		{11, " (bad)"},
		{20, " (bad)"},
		{21, " (poor)"},
		{40, " (poor)"},
		{41, " (average)"},
		{55, " (average)"},
		{56, " (fair)"},
		{70, " (fair)"},
		{71, " (good)"},
		{80, " (good)"},
		{81, " (very good)"},
		{85, " (very good)"},
		{86, " (superb)"},
		{98, " (superb)"},
		{99, " (MASTER)"},
		{100, " (MASTER)"},
	}
	for _, c := range cases {
		if got := howGood(c.pct); got != c.want {
			t.Errorf("howGood(%d) = %q, want %q", c.pct, got, c.want)
		}
	}
}

// SPLSKL per class from prac_params[PRAC_TYPE] (class.c:261) via prac_types[].
func TestSplSkl(t *testing.T) {
	cases := map[int]string{
		ClassMageUser: "spell",
		ClassCleric:   "spell",
		ClassThief:    "skill",
		ClassWarrior:  "skill",
		ClassAvatar:   "art",
		ClassPsionic:  "art",
		ClassMystic:   "art",
	}
	for class, want := range cases {
		if got := SplSkl(class); got != want {
			t.Errorf("SplSkl(%d) = %q, want %q", class, got, want)
		}
	}
	if got := SplSkl(-1); got != "skill" {
		t.Errorf("SplSkl(out-of-range) = %q, want fallback %q", got, "skill")
	}
}

// FindSkillNum resolves display names to numbers (find_skill_num).
func TestFindSkillNum(t *testing.T) {
	cases := map[string]int{
		"kick": 134,
		"bash": 132,
	}
	for name, want := range cases {
		if got := FindSkillNum(name); got != want {
			t.Errorf("FindSkillNum(%q) = %d, want %d", name, got, want)
		}
	}
	if got := FindSkillNum("nonesuchskill"); got != -1 {
		t.Errorf("FindSkillNum(unknown) = %d, want -1", got)
	}
}

// ClassSkillMinLevel mirrors spell_info[num].min_level[class].
func TestClassSkillMinLevel(t *testing.T) {
	if got := ClassSkillMinLevel(ClassWarrior, 134); got != 1 { // kick
		t.Errorf("warrior kick min level = %d, want 1", got)
	}
	if got := ClassSkillMinLevel(ClassWarrior, 132); got != 3 { // bash
		t.Errorf("warrior bash min level = %d, want 3", got)
	}
	// C's spello() defaults an unassigned class's min_level to LVL_IMMORT
	// (spell_parser.c:1154) — mortals can never reach it, but an immortal of
	// any class passes the cast gate.
	if got := ClassSkillMinLevel(ClassWarrior, 32); got != 31 { // magic missile — spello default
		t.Errorf("warrior magic-missile min level = %d, want 31 (spello LVL_IMMORT default)", got)
	}
	if got := ClassSkillMinLevel(ClassWarrior, 9999); got != 999 { // not in the catalog
		t.Errorf("unknown skill min level = %d, want 999", got)
	}
}
