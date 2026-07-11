package game

import "testing"

func TestMobHitGain_OddLevelTruncates(t *testing.T) {
	// C: gain = 2.5 * level assigned to int truncates the fraction.
	// Level 9: 2.5*9 = 22.5 → 22 in C. Go's lvl*5/2 = 45/2 = 22.
	m := &MobInstance{Level: 9}
	if got := MobHitGain(m); got != 22 {
		t.Errorf("MobHitGain(level 9) = %d, want 22", got)
	}
}

func TestMobHitGain_EvenLevelUnchanged(t *testing.T) {
	// Level 10: 2.5*10 = 25, no truncation difference.
	m := &MobInstance{Level: 10}
	if got := MobHitGain(m); got != 25 {
		t.Errorf("MobHitGain(level 10) = %d, want 25", got)
	}
}

func TestMobHitGain_HighLevelBeforeFourXBranch(t *testing.T) {
	// Level 22 is the last level using the 2.5x branch (lvl < 23).
	// 2.5*22 = 55, no fractional part.
	m := &MobInstance{Level: 22}
	if got := MobHitGain(m); got != 55 {
		t.Errorf("MobHitGain(level 22) = %d, want 55", got)
	}
}

func TestMobHitGain_FourXBranch(t *testing.T) {
	// Level 23+ uses the 4x branch.
	m := &MobInstance{Level: 23}
	if got := MobHitGain(m); got != 92 {
		t.Errorf("MobHitGain(level 23) = %d, want 92", got)
	}
}
