package combat

import "testing"

// Tier-2 fidelity golden test (deterministic — no RNG, no game-code change).
//
// thacoGolden is the THAC0 to-hit table transcribed VERBATIM from the original C source
// (src/class.c: `const int thaco[NUM_CLASSES][LVL_IMPL+1]`). It is the fidelity ground truth:
// getTHAC0 (and the Go `thaco` table behind it) MUST reproduce these exactly, or every to-hit
// roll in the Go port diverges from the original game — the classic silent "numbers off" bug.
//
// Index is [class][level]; level 0 is the unused sentinel (100), levels 1–40 are valid.
// Class order matches the C enum / the Go Class* constants: 0 MAGE … 11 MYSTIC.
var thacoGolden = [12][41]int{
	/* MAGE */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	/* CLERIC */ {100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* THIEF */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* WARRIOR */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* MAGUS */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	/* AVATAR */ {100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* ASSASSIN */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* PALADIN */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* NINJA */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* PSIONIC */ {100, 20, 20, 19, 18, 18, 17, 16, 16, 16, 15, 15, 14, 14, 14, 13, 12, 12, 10, 10, 9, 9, 8, 8, 7, 7, 6, 5, 5, 4, 4, 3, 3, 3, 2, 2, 1, 1, 1, 1, 1},
	/* RANGER */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* MYSTIC */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
}

var thacoClassNames = [12]string{
	"MAGE", "CLERIC", "THIEF", "WARRIOR", "MAGUS", "AVATAR",
	"ASSASSIN", "PALADIN", "NINJA", "PSIONIC", "RANGER", "MYSTIC",
}

// TestTHAC0_GoldenAgainstCSource asserts getTHAC0 reproduces the C thaco table for every
// (class, level). A failure means the Go port's to-hit numbers diverge from the original.
func TestTHAC0_GoldenAgainstCSource(t *testing.T) {
	for class := 0; class < 12; class++ {
		for level := 1; level <= 40; level++ {
			got := getTHAC0(&mockCombatant{class: class, level: level})
			if want := thacoGolden[class][level]; got != want {
				t.Errorf("THAC0 %s L%d = %d, want %d (per src/class.c)",
					thacoClassNames[class], level, got, want)
			}
		}
	}
}

// TestTHAC0_Clamps pins getTHAC0's non-table behavior: NPCs are flat 20, and out-of-range
// levels clamp to [1,40] (matching the original's bounds, not a panic).
func TestTHAC0_Clamps(t *testing.T) {
	if g := getTHAC0(&mockCombatant{npc: true}); g != 20 {
		t.Errorf("NPC THAC0 = %d, want 20", g)
	}
	if g := getTHAC0(&mockCombatant{class: ClassWarrior, level: 0}); g != thacoGolden[ClassWarrior][1] {
		t.Errorf("level<1 clamp = %d, want %d (level 1)", g, thacoGolden[ClassWarrior][1])
	}
	if g := getTHAC0(&mockCombatant{class: ClassWarrior, level: 99}); g != thacoGolden[ClassWarrior][40] {
		t.Errorf("level>40 clamp = %d, want %d (level 40)", g, thacoGolden[ClassWarrior][40])
	}
}
