package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// wireHeadbuttMessages loads lib/misc/messages so the skill_message path can
// emit set 141 (Headbutt). Mirrors wireKickMessages.
func wireHeadbuttMessages(t *testing.T, chName string) (cb *combat.GameCallbacks, attackerMsg *string, roomMsg *string, teardown func()) {
	t.Helper()
	orig := combat.GetCallbacks()
	messages := loadMessagesFile(t)
	var attMsg, roomOut string
	c := &combat.GameCallbacks{
		Broadcast: func(_ int, msg, _ string) { roomOut = msg },
		SendToChar: func(name, msg string) {
			if name == chName {
				attMsg = msg
			}
		},
		GetSex:   func(string) int { return 0 },
		GetHP:    func(string) int { return 10 },
		GetLevel: func(string) int { return 1 },
		IsNPC:    func(name string) bool { return false },
	}
	combat.SetCallbacks(c)
	combat.InitFightMessages(c, messages)
	teardown = func() { combat.SetCallbacks(orig) }
	return c, &attMsg, &roomOut, teardown
}

// TestDoHeadbutt_Miss_ReroutesThroughSkillMessage — miss returns
// SkillMsgType=141, Damage==0, StartCombat==true, empty MessageTo*.
func TestDoHeadbutt_Miss_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	ch.SetSkill(SkillHeadbutt, 1) // low skill → miss
	ch.Level = 10                 // mortal — a level>31 caster auto-succeeds (new_cmds.c:424)

	var result SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		ch.SetHealth(200) // reset each attempt: a stray hit applies recoil
		mob.SetPosition(combat.PosFighting)
		result = DoHeadbutt(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillHeadbuttNum {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no headbutt miss observed in 50 tries (RNG)")
	}

	if result.SkillMsgType != SkillHeadbuttNum {
		t.Errorf("miss SkillMsgType = %d, want %d (141)", result.SkillMsgType, SkillHeadbuttNum)
	}
	if result.Damage != 0 {
		t.Errorf("miss Damage = %d, want 0", result.Damage)
	}
	if !result.StartCombat {
		t.Error("miss should set StartCombat (C: damage(…,0,SKILL_HEADBUTT) → set_fighting)")
	}
	if result.WaitCh != 3 {
		t.Errorf("miss WaitCh = %d, want 3", result.WaitCh)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}

	messages := loadMessagesFile(t)
	if variants, ok := messages.Variants(SkillHeadbuttNum); !ok || len(variants) == 0 {
		t.Errorf("set 141 (Headbutt) not found in lib/misc/messages")
	}
}

// TestDoHeadbutt_Hit_ReroutesThroughSkillMessage — hit returns SkillMsgType=141,
// Damage==GetLevel() (full), TargetFalls preserved, no hardcoded strings.
func TestDoHeadbutt_Hit_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	wantDam := ch.GetLevel()
	hpBefore := ch.GetHP()
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		mob.SetPosition(combat.PosFighting)
		result = DoHeadbutt(ch, mob, w)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no headbutt hit observed in 20 tries (RNG)")
	}

	if result.SkillMsgType != SkillHeadbuttNum {
		t.Errorf("hit SkillMsgType = %d, want %d (141)", result.SkillMsgType, SkillHeadbuttNum)
	}
	if result.Damage != wantDam {
		t.Errorf("hit Damage = %d, want %d (GetLevel)", result.Damage, wantDam)
	}
	if !result.Success {
		t.Error("hit should be Success")
	}
	if !result.TargetFalls {
		t.Error("hit should keep TargetFalls (C: victim sits if above POS_STUNNED)")
	}
	// Recoil was applied to the caster (level/4 without a helm).
	if ch.GetHP() >= hpBefore {
		t.Errorf("hit should apply recoil to caster: hp %d → %d", hpBefore, ch.GetHP())
	}
	if result.MessageToCh != "" {
		t.Errorf("hit should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}
}

// TestDoHeadbutt_MissDrawCountAndOrder — R3: a headbutt miss consumes
// number(1,121) then the skill_message dice(1,N), in that order. The miss path
// does NOT call improve_skill, so it's the clean two-draw sequence (unlike the
// hit, which adds two number(1,200) improve draws after the dice).
func TestDoHeadbutt_MissDrawCountAndOrder(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillHeadbutt, 1)

	cb, _, _, teardown := wireHeadbuttMessages(t, ch.Name)
	defer teardown()

	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillHeadbuttNum)
	if !ok {
		t.Fatal("set 141 (Headbutt) not in messages file")
	}
	n := len(variants)

	// Find a seed that misses. A mortal-level caster (level <= LVL_IMMORT) is
	// required — level > 31 auto-succeeds (new_cmds.c:424). Reset HP each try.
	ch.Level = 10
	var missSeed uint32
	foundMiss := false
	for s := uint32(1); s < 30; s++ {
		ch.SetHealth(200)
		mob.SetPosition(combat.PosFighting)
		dprng.ResetStream(s)
		result := DoHeadbutt(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillHeadbuttNum {
			missSeed = s
			foundMiss = true
			break
		}
	}
	if !foundMiss {
		t.Skip("no headbutt miss observed in 29 seeds (RNG); miss draw-order not exercised")
	}

	dprng.ResetStream(missSeed)
	ch.SetHealth(200)
	mob.SetPosition(combat.PosFighting)
	result := DoHeadbutt(ch, mob, w)
	if result.Success || result.Damage != 0 {
		t.Skipf("seed %d did not produce a miss on re-run (non-deterministic?)", missSeed)
	}
	handled := cb.SkillMessage(0, ch.Name, mob.GetName(), SkillHeadbuttNum, ch.GetRoom())
	if !handled {
		t.Fatal("SkillMessage(0, ..., 141) did not handle set 141")
	}

	// Reference: number(1,121) then dice(1,N).
	dprng.ResetStream(missSeed)
	dprng.Number(1, 121)
	dprng.Dice(1, n)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(missSeed)
	ch.SetHealth(200)
	mob.SetPosition(combat.PosFighting)
	DoHeadbutt(ch, mob, w)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillHeadbuttNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("headbutt miss draw count/order wrong: next=%d want=%d (number(1,121) then dice(1,%d))", got, wantNext, n)
	}
}

// TestDoHeadbutt_HitDrawSequenceIncludesImproveSkill — R3b: the headbutt HIT
// path draws, in C order (new_cmds.c:437-457), number(1,121) [skill roll] →
// dice(1,N) [skill_message, inside C's damage()] → number(1,200) [improve #1]
// → number(1,200) [improve #2]. The improves are deferred to sendSkillResult
// via SkillResult.DeferredImprove (DP-1212), so this test drives the same
// sequence the sender runs: DoHeadbutt, then SkillMessage, then the deferred
// improves. An earlier version of this test asserted the OLD Go order
// (improve → dice), which was built from the implementation and encoded the
// bug (R5a); it is now reversed to the C order.
func TestDoHeadbutt_HitDrawSequenceIncludesImproveSkill(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	cb, _, _, teardown := wireHeadbuttMessages(t, ch.Name)
	defer teardown()

	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillHeadbuttNum)
	if !ok {
		t.Fatal("set 141 (Headbutt) not in messages file")
	}
	n := len(variants)

	// Find a seed that hits.
	var hitSeed uint32
	found := false
	for s := uint32(1); s < 30; s++ {
		dprng.ResetStream(s)
		result := DoHeadbutt(ch, mob, w)
		if result.Success {
			hitSeed = s
			found = true
			break
		}
	}
	if !found {
		t.Skip("no headbutt hit observed in 29 seeds (RNG); hit draw-sequence not exercised")
	}

	// Run the hit the way sendSkillResult does (DP-1212): DoHeadbutt draws only
	// number(1,121) and returns DeferredImprove=[headbutt, headbutt];
	// SkillMessage draws dice(1,N) where C's damage() calls skill_message; the
	// two improves run LAST, in C order.
	dprng.ResetStream(hitSeed)
	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("seed %d: re-run did not hit (non-deterministic?)", hitSeed)
	}
	if len(result.DeferredImprove) != 2 ||
		result.DeferredImprove[0] != SkillHeadbutt || result.DeferredImprove[1] != SkillHeadbutt {
		t.Fatalf("hit DeferredImprove = %v, want [%s %s] (C improves twice)",
			result.DeferredImprove, SkillHeadbutt, SkillHeadbutt)
	}
	cb.SkillMessage(result.Damage, ch.Name, mob.GetName(), SkillHeadbuttNum, ch.GetRoom())
	for _, skill := range result.DeferredImprove {
		improveSkill(ch, skill)
	}

	// Reference the SAME sequence in C order: number(1,121), then dice(1,N),
	// then the two improve number(1,200)s (skill >= 97 here → gate-only draws).
	dprng.ResetStream(hitSeed)
	dprng.Number(1, 121) // percent
	dprng.Dice(1, n)     // skill_message dice — BEFORE the improves (C order)
	dprng.Number(1, 200) // improve #1 gate
	dprng.Number(1, 200) // improve #2 gate
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(hitSeed)
	result = DoHeadbutt(ch, mob, w) // consumes number(1,121) only
	cb.SkillMessage(result.Damage, ch.Name, mob.GetName(), SkillHeadbuttNum, ch.GetRoom())
	for _, skill := range result.DeferredImprove {
		improveSkill(ch, skill)
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("headbutt HIT draw sequence wrong: next=%d want=%d. Expected C order: "+
			"number(1,121) → dice(1,%d) [skill_message] → 2×number(1,200) [improve]", got, wantNext, n)
	}
}

// TestDoHeadbutt_MissMessageFromSkillMessages — R4: miss emits set-141 text,
// not the old invented "You try to headbutt… but miss!" string.
func TestDoHeadbutt_MissMessageFromSkillMessages(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillHeadbutt, 1)

	cb, attMsg, _, teardown := wireHeadbuttMessages(t, ch.Name)
	defer teardown()

	ch.Level = 10 // mortal — level > 31 auto-succeeds
	var missed bool
	for i := 0; i < 100; i++ {
		ch.SetHealth(200) // reset: a stray hit applies recoil
		mob.SetPosition(combat.PosFighting)
		result := DoHeadbutt(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillHeadbuttNum {
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillHeadbuttNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no headbutt miss observed in 100 tries (RNG)")
	}
	got := *attMsg
	// The OLD invented strings were "but miss!" and "sickening crack" — those
	// are gone. Note: set 141's miss text IS "You try to headbutt $N, but $E
	// ducks!" — that's the genuine file message, NOT an invention.
	if strings.Contains(got, "but miss!") || strings.Contains(got, "sickening crack") {
		t.Errorf("miss emitted an OLD invented headbutt string (R4): %q", got)
	}
	if got == "" {
		t.Errorf("miss attacker message is empty — SkillMessage(141) did not emit")
	}
}

// TestDoHeadbutt_SkillKnownGateUnchanged — DP-1206 regression: GetSkill
// (headbutt)==0 still returns the bare qualified message.
func TestDoHeadbutt_SkillKnownGateUnchanged(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillHeadbutt, 0)

	result := DoHeadbutt(ch, mob, w)
	if result.Success {
		t.Error("unknown headbutt should not succeed")
	}
	if result.MessageToCh != "You aren't qualified to headbutt anyone!\r\n" {
		t.Errorf("skill-known gate = %q, want the qualified line (DP-1206)", result.MessageToCh)
	}
}
