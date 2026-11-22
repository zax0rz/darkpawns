package game

import "testing"

func TestMobHitGain_OddLevelTruncates(t *testing.T) {
	// C: hit_gain assigns 2.5 * level to int, truncating the fraction
	// (src/limits.c:133-137).
	// Level 9: 2.5*9 = 22.5 → 22 in C. Go's lvl*5/2 = 45/2 = 22.
	m := &MobInstance{Level: 9}
	if got := (&World{}).MobHitGain(m); got != 22 {
		t.Errorf("MobHitGain(level 9) = %d, want 22", got)
	}
}

func TestMobHitGain_HighLevelBeforeFourXBranch(t *testing.T) {
	// Level 22 is the last level using the 2.5x branch (lvl < 23; src/limits.c:134-137).
	// 2.5*22 = 55, no fractional part.
	m := &MobInstance{Level: 22}
	if got := (&World{}).MobHitGain(m); got != 55 {
		t.Errorf("MobHitGain(level 22) = %d, want 55", got)
	}
}

func TestMobHitGain_FourXBranch(t *testing.T) {
	// Level 23+ uses the 4x branch (src/limits.c:134-137).
	m := &MobInstance{Level: 23}
	if got := (&World{}).MobHitGain(m); got != 92 {
		t.Errorf("MobHitGain(level 23) = %d, want 92", got)
	}
}
