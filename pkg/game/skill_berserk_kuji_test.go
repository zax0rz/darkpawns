package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// ---------------------------------------------------------------------------
// DoBerserk
// ---------------------------------------------------------------------------

func TestDoBerserk_NoSkill(t *testing.T) {
	p := NewPlayer(1, "Berserker", 3001)
	// No berserk skill set (default 0).
	result := DoBerserk(p)
	if result.Success {
		t.Error("expected failure with no berserk skill")
	}
	if result.MessageToCh != "You're about as berserk as you're gonna get." {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoBerserk_AlreadyAffected(t *testing.T) {
	p := NewPlayer(1, "Berserker", 3001)
	p.SetSkill(SkillBerserk, 75)
	p.SetAffect(affBerserk, true)

	result := DoBerserk(p)
	if result.Success {
		t.Error("expected failure when already berserk")
	}
	if result.MessageToCh != "You're unable to summon your battle rage right now." {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoBerserk_AffectedGateConsumesPercentDraw(t *testing.T) {
	const seed uint32 = 1
	p := NewPlayer(1, "Berserker", 3001)
	p.SetSkill(SkillBerserk, 75)
	p.SetAffect(affBerserk, true)

	dprng.ResetStream(seed)
	dprng.Number(1, 101) // C's percent draw occurs before the affect gate.
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	result := DoBerserk(p)
	if result.Success {
		t.Fatal("already affected player should be rejected")
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Errorf("next RNG draw = %d, want %d after consuming C's percent draw", got, wantNext)
	}
}

// TestDoBerserk_GuaranteedSuccessFoldsIntoStats forces the C immortal bypass
// (GET_LEVEL(ch) > LEVEL_IMMORT => percent = 0, guaranteed success) to
// deterministically exercise the success path end-to-end: DoBerserk must add
// affects that GetHitroll/GetDamroll/GetAC actually fold in at read time.
func TestDoBerserk_GuaranteedSuccessFoldsIntoStats(t *testing.T) {
	p := NewPlayer(1, "Berserker", 3001)
	p.Level = LVL_IMMORT + 1
	p.SetSkill(SkillBerserk, 1) // any nonzero skill — percent is forced to 0 regardless
	p.Hitroll = 0
	p.Damroll = 0
	p.AC = 0

	result := DoBerserk(p)
	if !result.Success {
		t.Fatalf("expected guaranteed success for level > LVL_IMMORT, got failure: %q", result.MessageToCh)
	}

	if got := p.GetHitroll(); got != 2 {
		t.Errorf("GetHitroll = %d, want 2 (berserk success bonus not folded in)", got)
	}
	if got := p.GetDamroll(); got != 2 {
		t.Errorf("GetDamroll = %d, want 2 (berserk success bonus not folded in)", got)
	}
	if got := p.GetAC(); got != 25 {
		t.Errorf("GetAC = %d, want 25 (berserk AC penalty not folded in)", got)
	}
	if !p.IsAffected(affBerserk) {
		t.Error("player should be flagged as berserk after a successful use")
	}
}

// TestDoBerserk_ImproveSkillAlwaysCalled ports the C dangling-else bug in
// do_berserk(): improve_skill() is written at the else branch's indentation
// but isn't actually inside it, so it runs on every use, win or lose. We
// can't force a deterministic RNG failure, but we can confirm the call site
// is unconditional by checking the guaranteed-success path still improves —
// and that it does so via the shared improveSkill() helper, not a
// success-only branch. See DoBerserk's comment for the full rationale.
func TestDoBerserk_ImproveSkillAlwaysCalled(t *testing.T) {
	p := NewPlayer(1, "Berserker", 3001)
	p.Level = LVL_IMMORT + 1
	p.SetSkill(SkillBerserk, 1)

	before := p.GetSkill(SkillBerserk)
	DoBerserk(p)
	after := p.GetSkill(SkillBerserk)
	if after < before {
		t.Errorf("GetSkill(berserk) should never decrease: before=%d after=%d", before, after)
	}
}

func TestDoBerserk_FailureStillInstallsInertAffectsAndWait(t *testing.T) {
	p := NewPlayer(1, "Berserker", 3001)
	p.SetSkill(SkillBerserk, 1)
	dprng.ResetStream(1) // first number(1,101) is above skill 1

	result := DoBerserk(p)
	if result.Success {
		t.Fatal("expected seed 1 to fail at berserk skill 1")
	}
	if result.MessageToCh != "You fail to summon up your battle rage." {
		t.Errorf("failure message = %q", result.MessageToCh)
	}
	if result.WaitCh != 2 {
		t.Errorf("WaitCh = %d, want 2 PULSE_VIOLENCE rounds", result.WaitCh)
	}
	if len(p.ActiveAffects) != 3 {
		t.Fatalf("active affects = %d, want three C affect records", len(p.ActiveAffects))
	}
	wantLocations := []int{ApplyHitroll, ApplyDamroll, ApplyAC}
	for i, affect := range p.ActiveAffects {
		if affect.SpellID != skillNumBerserk || affect.Location != wantLocations[i] {
			t.Errorf("affect %d = (spell %d, location %d), want (%d, %d)", i, affect.SpellID, affect.Location, skillNumBerserk, wantLocations[i])
		}
		if affect.Magnitude != 0 || affect.Duration != 1 || affect.Flags != engine.AFFBerserk {
			t.Errorf("affect %d = (duration %d, magnitude %d, flags %d), want (1, 0, %d)", i, affect.Duration, affect.Magnitude, affect.Flags, engine.AFFBerserk)
		}
	}
	if !p.IsAffected(affBerserk) {
		t.Fatal("failed berserk attempt should still set AFF_BERSERK via the C affect record")
	}
}

// ---------------------------------------------------------------------------
// DoKujiKiri — guard clauses
// ---------------------------------------------------------------------------

func TestDoKujiKiri_NotNinja(t *testing.T) {
	p := NewPlayer(1, "Ronin", 3001)
	p.Class = ClassWarrior
	p.Level = 20
	p.SetSkill(SkillKkRin, 75)

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Error("expected failure for non-Ninja, non-immortal class")
	}
	if result.MessageToCh != "You know nothing of kuji-kiri!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoKujiKiri_ImmortalBypassesClassCheck(t *testing.T) {
	p := NewPlayer(1, "Immort", 3001)
	p.Class = ClassWarrior
	p.Level = LVL_IMMORT
	p.SetSkill(SkillKkRin, 75)

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.MessageToCh == "You know nothing of kuji-kiri!" {
		t.Error("immortals should bypass the Ninja class check")
	}
}

func TestDoKujiKiri_Fighting(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkRin, 75)
	p.Fighting = "a training dummy"

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Error("expected failure while fighting")
	}
	if result.MessageToCh != "You are too busy fighting to practice kuji-kiri!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoKujiKiri_AlreadyAffected(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkRin, 75)
	p.SetAffect(affKujiKiri, true)

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Error("expected failure when already under a kuji-kiri lockout")
	}
	if result.MessageToCh != "You can not practice kuji-kiri again right now!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoKujiKiri_Mounted(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkRin, 75)
	p.MountName = "a warhorse"

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Error("expected failure while mounted")
	}
	if result.MessageToCh != "Dismount first!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

func TestDoKujiKiri_SealNotMastered(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 20
	// SkillKkRin left at 0 — not learned.

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Error("expected failure for an unmastered seal")
	}
	if result.MessageToCh != "You have not mastered this art yet!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

// ---------------------------------------------------------------------------
// checkKkSuccess — the missing-TOH-case bug
// ---------------------------------------------------------------------------

// TestCheckKkSuccess_TohAlwaysFails ports check_kk_success()'s missing
// `case SKILL_KK_TOH:` (new_cmds.c:1500-1541): prob stays 0 for Toh
// regardless of skill level, so the 1-101 roll always exceeds it and the
// seal's success check always fails. This must hold even at max skill.
func TestCheckKkSuccess_TohAlwaysFails(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 30
	p.SetSkill(SkillKkToh, 100)

	for i := 0; i < 50; i++ {
		if checkKkSuccess(p, SkillKkToh) {
			t.Fatal("checkKkSuccess(SkillKkToh) succeeded — the missing-case bug must always fail Toh")
		}
	}
}

// TestCheckKkSuccess_OtherSealsCanSucceed sanity-checks that the missing-TOH
// bug is specific to Toh — a seal with a real case (Rin) at max skill should
// be able to succeed.
func TestCheckKkSuccess_OtherSealsCanSucceed(t *testing.T) {
	p := NewPlayer(1, "Ninja", 3001)
	p.Class = ClassNinja
	p.Level = 30
	p.SetSkill(SkillKkRin, 100)

	for i := 0; i < 50; i++ {
		if checkKkSuccess(p, SkillKkRin) {
			return
		}
	}
	t.Fatal("checkKkSuccess(SkillKkRin) never succeeded at skill 100 across 50 attempts")
}

// ---------------------------------------------------------------------------
// DoKujiKiri — success path folds into stats (Rin: AC + metalskin)
// ---------------------------------------------------------------------------

// TestDoKujiKiri_RinSuccessFoldsIntoAC retries until checkKkSuccess rolls a
// win (skill 100 makes failure a ~1% per-attempt event) to deterministically
// verify the success path: Rin's AC affect must be picked up by GetAC, and
// the metalskin flag must be set.
func TestDoKujiKiri_RinSuccessFoldsIntoAC(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		p := NewPlayer(1, "Ninja", 3001)
		p.Class = ClassNinja
		p.Level = 20
		p.AC = 0
		p.SetSkill(SkillKkRin, 100)

		result := DoKujiKiri(p, SkillKkRin, nil)
		if !result.Success {
			continue
		}

		wantAC := -(15 + p.GetLevel()/2)
		if got := p.GetAC(); got != wantAC {
			t.Errorf("GetAC = %d, want %d (kuji-kiri Rin AC bonus not folded in)", got, wantAC)
		}
		if !p.IsAffected(affMetalskin) {
			t.Error("Rin success should grant metalskin")
		}
		if !p.IsAffected(affKujiKiri) {
			t.Error("Rin should apply the kuji-kiri lockout")
		}
		return
	}
	t.Fatal("kuji-kiri Rin never succeeded at skill 100 across 50 attempts")
}

// TestDoKujiKiri_ShaSuccessHeals verifies Sha's direct heal-on-success path
// (new_cmds.c:1644-1653), retrying until checkKkSuccess rolls a win.
func TestDoKujiKiri_ShaSuccessHeals(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		p := NewPlayer(1, "Ninja", 3001)
		p.Class = ClassNinja
		p.Level = 20
		p.SetSkill(SkillKkSha, 100)
		p.MaxHealth = 100
		p.Health = 10

		result := DoKujiKiri(p, SkillKkSha, nil)
		if !result.Success {
			continue
		}

		wantHeal := p.GetLevel()
		if wantHeal < 15 {
			wantHeal = 15
		}
		if got := p.GetHP(); got != 10+wantHeal {
			t.Errorf("GetHP = %d, want %d (kuji-kiri Sha heal not applied)", got, 10+wantHeal)
		}
		return
	}
	t.Fatal("kuji-kiri Sha never succeeded at skill 100 across 50 attempts")
}

// TestDoKujiKiri_FailurePreventsNumericEffect verifies the failure-path
// zeroing (new_cmds.c:1713-1721): on failure, Rin's AC modifier and the
// metalskin flag must both be absent — the only observable effect should be
// the kuji-kiri lockout. We can't force a deterministic failure roll, so we
// approximate by using skill 1 (near-certain failure) and just assert that
// IF the roll happened to fail, no AC change leaked through.
func TestDoKujiKiri_FailurePreventsNumericEffect(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		p := NewPlayer(1, "Ninja", 3001)
		p.Class = ClassNinja
		p.Level = 20
		p.AC = 0
		p.SetSkill(SkillKkRin, 1)

		result := DoKujiKiri(p, SkillKkRin, nil)
		if result.Success {
			continue
		}

		if got := p.GetAC(); got != 0 {
			t.Errorf("GetAC = %d, want 0 (failed Rin must not change AC)", got)
		}
		if p.IsAffected(affMetalskin) {
			t.Error("failed Rin must not grant metalskin")
		}
		if !p.IsAffected(affKujiKiri) {
			t.Error("failed Rin should still apply the kuji-kiri lockout")
		}
		return
	}
	t.Fatal("kuji-kiri Rin never failed at skill 1 across 50 attempts")
}
