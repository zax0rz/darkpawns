package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestDoBashSubcmdFixedProbAndNoImprove pins do_bash's subcmd=1 arm
// (src/act.offensive.c:427-433,476,491-492): the refusal does not return, the
// fixed 131 replaces the caster's skill, and improve_skill is skipped.
func TestDoBashSubcmdFixedProbAndNoImprove(t *testing.T) {
	mf := newMinorFixWorld(t)
	p := mf.addPlayer(t, "Wallmage", 1001, 10)
	p.SetMove(100)
	target := mf.addMob(t, 1001, 5, "rat", "a rat")
	target.SetPosition(combat.PosFighting)

	// subcmd 0 with no skill: the refusal returns before any draw.
	dprng.ResetStream(42)
	refusal := DoBash(p, target, mf.w)
	if refusal.Success || refusal.MessageToCh != "You'd better leave all the martial arts to fighters." {
		t.Fatalf("subcmd 0 result = %+v, want the refusal", refusal)
	}
	if got := dprng.Number(0, 1000000); got != func() int { dprng.ResetStream(42); return dprng.Number(0, 1000000) }() {
		t.Fatal("subcmd 0 refusal consumed a draw")
	}

	// subcmd 1: proceeds against the fixed 131 whatever the caster's skill.
	dprng.ResetStream(42)
	percent := ((5 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	wantSuccess := percent <= 131
	dprng.ResetStream(42)
	result := DoBashSubcmd(p, target, mf.w, 1)
	if result.Success != wantSuccess {
		t.Fatalf("subcmd 1 bash success = %t, want %t (percent %d vs fixed 131): %+v", result.Success, wantSuccess, percent, result)
	}
	if len(result.DeferredImprove) != 0 {
		t.Fatalf("subcmd 1 DeferredImprove = %v, want none (src/act.offensive.c:491-492)", result.DeferredImprove)
	}
	if result.MessageToCh != "" {
		t.Fatalf("body emitted %q; the caller prints the refusal (src/act.offensive.c:429-430)", result.MessageToCh)
	}
	dprng.ResetStream(42)
	_ = dprng.Number(1, 101)
	want := dprng.Number(0, 1000000)
	dprng.ResetStream(42)
	p.SetMove(100)
	_ = DoBashSubcmd(p, target, mf.w, 1)
	if got := dprng.Number(0, 1000000); got != want {
		t.Fatal("subcmd 1 bash consumed other than its one number(1,101)")
	}
}

// TestDoHeadbuttSubcmdProbDrawAndSingleImprove pins do_headbutt's subcmd=1
// arm (src/new_cmds.c:382-387,437,450,456-457): the refusal does not return,
// prob is a number(50,100) draw after the HP gate, and the success arm keeps
// only the unconditional improve_skill.
func TestDoHeadbuttSubcmdProbDrawAndSingleImprove(t *testing.T) {
	mf := newMinorFixWorld(t)
	p := mf.addPlayer(t, "Wallmage", 1001, 10)
	p.SetHP(100)
	target := mf.addMob(t, 1001, 5, "rat", "a rat")
	target.SetPosition(combat.PosFighting)

	// Replay the body's two draws to learn the expected arm.
	dprng.ResetStream(42)
	percent := dprng.Number(1, 121)
	prob := dprng.Number(50, 100)
	wantSuccess := percent <= prob

	dprng.ResetStream(42)
	result := DoHeadbuttSubcmd(p, target, mf.w, 1)
	if result.Success != wantSuccess {
		t.Fatalf("subcmd 1 headbutt success = %t, want %t (percent %d vs drawn prob %d)", result.Success, wantSuccess, percent, prob)
	}
	if result.MessageToCh != "" {
		t.Fatalf("body emitted %q; the caller prints the refusal (src/new_cmds.c:384)", result.MessageToCh)
	}
	if wantSuccess && len(result.DeferredImprove) != 1 {
		t.Fatalf("subcmd 1 DeferredImprove = %v, want exactly the unconditional call (src/new_cmds.c:450)", result.DeferredImprove)
	}
	// Exactly two draws: number(1,121) then number(50,100) (improve is deferred).
	dprng.ResetStream(42)
	_ = dprng.Number(1, 121)
	_ = dprng.Number(50, 100)
	want := dprng.Number(0, 1000000)
	dprng.ResetStream(42)
	p.SetHP(100)
	_ = DoHeadbuttSubcmd(p, target, mf.w, 1)
	if got := dprng.Number(0, 1000000); got != want {
		t.Fatal("subcmd 1 headbutt consumed other than number(1,121) + number(50,100)")
	}

	// subcmd 0 with skill keeps both improve calls on a success.
	p.SetSkill(SkillHeadbutt, 100)
	seed := minorNumberSeed(t, 1, 121, 50) // percent 50 <= skill 100: success
	dprng.ResetStream(seed)
	manual := DoHeadbutt(p, target, mf.w)
	if !manual.Success || len(manual.DeferredImprove) != 2 {
		t.Fatalf("subcmd 0 headbutt = success %t, improve %v; want success with two improves", manual.Success, manual.DeferredImprove)
	}
}

// TestDoBerserkSubcmdProbDraw pins do_berserk's subcmd=1 arm
// (src/new_cmds.c:2140-2145,2159): the refusal does not return and prob is a
// number(50,100) draw after the percent draw.
func TestDoBerserkSubcmdProbDraw(t *testing.T) {
	mf := newMinorFixWorld(t)
	p := mf.addPlayer(t, "Wallmage", 1001, 10)

	// subcmd 0 with no skill: refusal, no draw.
	dprng.ResetStream(42)
	refusal := DoBerserk(p)
	if refusal.Success || refusal.MessageToCh != "You're about as berserk as you're gonna get." {
		t.Fatalf("subcmd 0 result = %+v, want the refusal", refusal)
	}
	if got := dprng.Number(0, 1000000); got != func() int { dprng.ResetStream(42); return dprng.Number(0, 1000000) }() {
		t.Fatal("subcmd 0 refusal consumed a draw")
	}

	// subcmd 1: proceeds with the drawn prob.
	dprng.ResetStream(42)
	percent := dprng.Number(1, 101)
	prob := dprng.Number(50, 100)
	wantSuccess := percent <= prob
	dprng.ResetStream(42)
	result := DoBerserkSubcmd(p, 1)
	if result.Success != wantSuccess {
		t.Fatalf("subcmd 1 berserk success = %t, want %t (percent %d vs drawn prob %d)", result.Success, wantSuccess, percent, prob)
	}
	if result.MessageToCh == "" || result.MessageToCh == "You're about as berserk as you're gonna get." {
		t.Fatalf("subcmd 1 berserk message = %q, want the battle-rage line", result.MessageToCh)
	}
	// Draw order: number(1,101) then number(50,100); improve_skill's
	// number(1,200) follows (unconditional, src/new_cmds.c:2169). A fresh
	// caster keeps the AFF_BERSERK gate out of the replay.
	fresh := mf.addPlayer(t, "Wallmage2", 1001, 10)
	dprng.ResetStream(42)
	_ = dprng.Number(1, 101)
	_ = dprng.Number(50, 100)
	_ = dprng.Number(1, 200)
	want := dprng.Number(0, 1000000)
	dprng.ResetStream(42)
	_ = DoBerserkSubcmd(fresh, 1)
	if got := dprng.Number(0, 1000000); got != want {
		t.Fatal("subcmd 1 berserk draw order diverged from C")
	}
}
