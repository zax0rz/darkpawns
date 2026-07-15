package session

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
		0:  "spell", // mage
		1:  "spell", // cleric
		2:  "skill", // thief
		3:  "skill", // warrior
		5:  "art",   // avatar (BOTH)
		9:  "art",   // psionic (BOTH)
		11: "art",   // mystic (BOTH)
	}
	for class, want := range cases {
		if got := splSkl(class); got != want {
			t.Errorf("splSkl(%d) = %q, want %q", class, got, want)
		}
	}
	if got := splSkl(-1); got != "skill" {
		t.Errorf("splSkl(out-of-range) = %q, want fallback %q", got, "skill")
	}
}
