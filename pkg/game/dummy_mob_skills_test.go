package game

import "testing"

func TestDummyMobSkillsUseSharedCState(t *testing.T) {
	w := &World{}
	rin := skillNumKkRin
	const other = 999999

	if got := w.DummyMobSkill(rin); got != 0 {
		t.Fatalf("initial shared skill = %d, want C's zero-initialized value", got)
	}
	w.SetDummyMobSkill(rin, 37)
	if got := w.DummyMobSkill(rin); got != 37 {
		t.Fatalf("set shared skill = %d, want 37", got)
	}
	if got := w.DummyMobSkill(other); got != 0 {
		t.Fatalf("unwritten shared skill = %d, want 0", got)
	}

	w.RestoreDummyMobSkills()
	if got := w.DummyMobSkill(rin); got != 100 {
		t.Fatalf("restored skill = %d, want 100", got)
	}
	if got := w.DummyMobSkill(other); got != 100 {
		t.Fatalf("other restored skill = %d, want 100", got)
	}
	w.SetDummyMobSkill(rin, 0)
	if got := w.DummyMobSkill(rin); got != 0 {
		t.Fatalf("script-overridden skill = %d, want 0", got)
	}
	if got := w.DummyMobSkill(other); got != 100 {
		t.Fatalf("unmodified restored skill = %d, want 100", got)
	}
}
