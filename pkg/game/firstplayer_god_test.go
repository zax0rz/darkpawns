package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestBootstrapFirstPlayerGod_StatsAndSkillsAndConditions — the init_char
// first-player block (db.c:3016-3024, 3059-3074): a crowned first player gets
// level LVL_IMPL (40), exp 7,000,000, max_hit 500 / max_mana 100 / max_move 82,
// every skill at 100, and all three conditions -1. Byte/stat-exact vs C.
func TestBootstrapFirstPlayerGod_StatsAndSkillsAndConditions(t *testing.T) {
	p := NewPlayer(1, "FirstGod", 1001)
	p.Class = ClassThief // class doesn't matter — God gets all skills at 100

	BootstrapFirstPlayerGod(p)
	if p.GetAutoExit() {
		t.Error("first-player God should retain C init_char's autoexit-off default")
	}
	for _, flag := range []int{PrfDisphp, PrfDispmmana, PrfDispmove} {
		if p.GetFlags()&(1<<uint(flag)) != 0 {
			t.Errorf("first-player God should retain C init_char's display-off default for flag %d", flag)
		}
	}
	if p.WimpLevel != 0 {
		t.Errorf("first-player God wimp level = %d, want C init_char default 0", p.WimpLevel)
	}

	if got := p.GetLevel(); got != LVL_IMPL {
		t.Errorf("God level = %d, want %d (LVL_IMPL)", got, LVL_IMPL)
	}
	if got := p.GetExp(); got != 7000000 {
		t.Errorf("God exp = %d, want 7000000", got)
	}
	if got, want := p.GetMaxHP(), 500; got != want {
		t.Errorf("God max_hit = %d, want %d", got, want)
	}
	if got, want := p.GetMaxMana(), 100; got != want {
		t.Errorf("God max_mana = %d, want %d", got, want)
	}
	if got, want := p.GetMaxMove(), 82; got != want {
		t.Errorf("God max_move = %d, want %d", got, want)
	}

	// Every skill at 100. Spot-check a representative spread across the catalog
	// (a low-numbered spell, a combat skill, a high-numbered breath).
	for _, skill := range []string{"holy ward", "backstab", "kick", "lightning breath", "circle", "disarm"} {
		if got := p.GetSkill(skill); got != 100 {
			t.Errorf("God skill %q = %d, want 100", skill, got)
		}
	}

	// Conditions -1 (immortal: no hunger/thirst/drunk).
	if got, want := p.Hunger, -1; got != want {
		t.Errorf("God hunger = %d, want %d", got, want)
	}
	if got, want := p.Thirst, -1; got != want {
		t.Errorf("God thirst = %d, want %d", got, want)
	}
	if got, want := p.Drunk, -1; got != want {
		t.Errorf("God drunk = %d, want %d", got, want)
	}
	if got := p.Conditions[CondFull]; got != -1 {
		t.Errorf("God Conditions[CondFull] = %d, want -1", got)
	}
	if got := p.Conditions[CondThirst]; got != -1 {
		t.Errorf("God Conditions[CondThirst] = %d, want -1", got)
	}
}

// TestBootstrapFirstPlayerGod_NoRNGPerturbation — R3: the God block is pure
// assignment (no number() draws), so the shared CMWC stream must be unchanged
// across it. Self-referencing: seed, run the block, assert the next draw
// matches a reference stream that consumed nothing in between. Mirrors the
// cast_cmds / improveSkill draw-parity tests.
func TestBootstrapFirstPlayerGod_NoRNGPerturbation(t *testing.T) {
	p := NewPlayer(1, "FirstGod", 1001)

	const seed = 7
	dprng.ResetStream(seed)
	wantNext := dprng.Number(0, 999) // reference: nothing consumed yet

	dprng.ResetStream(seed)
	BootstrapFirstPlayerGod(p)
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("God block perturbed the RNG stream: next=%d want=%d (the block must draw zero)", got, wantNext)
	}
}
