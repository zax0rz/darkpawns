package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// wireBashMessages loads lib/misc/messages so the skill_message path can emit
// set 132 (Bash). Mirrors wireKickMessages.
func wireBashMessages(t *testing.T, chName string) (cb *combat.GameCallbacks, attackerMsg *string, roomMsg *string, teardown func()) {
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

// TestDoBash_Miss_ReroutesThroughSkillMessage — miss returns SkillMsgType=132,
// Damage==0, StartCombat==true, SelfStumble preserved (caster falls), empty
// MessageTo*. R4: C routes the message through skill_message + starts combat.
func TestDoBash_Miss_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	ch.SetSkill(SkillBash, 1) // low skill → miss

	var result SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		ch.Move = 100 // reset move (SpendMove(10) drains it)
		mob.SetPosition(combat.PosFighting)
		result = DoBash(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillBashNum {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no bash miss observed in 50 tries (RNG)")
	}

	if result.SkillMsgType != SkillBashNum {
		t.Errorf("miss SkillMsgType = %d, want %d (132)", result.SkillMsgType, SkillBashNum)
	}
	if result.Damage != 0 {
		t.Errorf("miss Damage = %d, want 0", result.Damage)
	}
	if !result.StartCombat {
		t.Error("miss should set StartCombat (C: damage(…,0,SKILL_BASH) → set_fighting)")
	}
	if !result.SelfStumble {
		t.Error("miss should keep SelfStumble (C: GET_POS(ch)=POS_SITTING)")
	}
	if result.WaitCh != 2 {
		t.Errorf("miss WaitCh = %d, want 2 (PULSE_VIOLENCE*2)", result.WaitCh)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}

	messages := loadMessagesFile(t)
	if variants, ok := messages.Variants(SkillBashNum); !ok || len(variants) == 0 {
		t.Errorf("set 132 (Bash) not found in lib/misc/messages")
	}
}

// TestDoBash_Hit_ReroutesThroughSkillMessage — hit returns SkillMsgType=132,
// Damage==(level/2)+1, StartCombat, TargetFalls, DeferredImprove==[SkillBash],
// no hardcoded strings.
func TestDoBash_Hit_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	wantDam := (ch.GetLevel() / 2) + 1
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		ch.Move = 100
		mob.SetPosition(combat.PosFighting)
		result = DoBash(ch, mob, w)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no bash hit observed in 20 tries (RNG)")
	}

	if result.SkillMsgType != SkillBashNum {
		t.Errorf("hit SkillMsgType = %d, want %d (132)", result.SkillMsgType, SkillBashNum)
	}
	if result.Damage != wantDam {
		t.Errorf("hit Damage = %d, want %d ((level/2)+1)", result.Damage, wantDam)
	}
	if !result.Success {
		t.Error("hit should be Success")
	}
	if !result.StartCombat {
		t.Error("hit should set StartCombat (C: damage(…,dam,SKILL_BASH) → set_fighting)")
	}
	if !result.TargetFalls {
		t.Error("hit should keep TargetFalls (C: GET_POS(vict)=POS_SITTING)")
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillBash {
		t.Errorf("hit DeferredImprove = %v, want [SkillBash] (C: improve_skill deferred to sendSkillResult)", result.DeferredImprove)
	}
	if result.MessageToCh != "" {
		t.Errorf("hit should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}
}

// TestDoBash_MissDrawCountAndOrder — R3: a bash miss consumes number(1,101)
// then the skill_message dice(1,N), in that order. Self-referencing stream check.
func TestDoBash_MissDrawCountAndOrder(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillBash, 1)

	cb, _, _, teardown := wireBashMessages(t, ch.Name)
	defer teardown()

	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillBashNum)
	if !ok {
		t.Fatal("set 132 (Bash) not in messages file")
	}
	n := len(variants)

	const seed = 7
	ch.Move = 100
	mob.SetPosition(combat.PosFighting)
	dprng.ResetStream(seed)
	result := DoBash(ch, mob, w)
	if result.Success || result.Damage != 0 {
		t.Skipf("seed %d did not produce a miss (success=%v); draw-order not exercised", seed, result.Success)
	}
	handled := cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBashNum, ch.GetRoom())
	if !handled {
		t.Fatal("SkillMessage(0, ..., 132) did not handle set 132")
	}

	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Dice(1, n)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	ch.Move = 100
	mob.SetPosition(combat.PosFighting)
	DoBash(ch, mob, w)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBashNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("bash miss draw count/order wrong: next=%d want=%d (number(1,101) then dice(1,%d))", got, wantNext, n)
	}
}

// TestDoBash_MissMessageFromSkillMessages — R4: miss emits set-132 text, not
// the old invented "You try to bash… but miss and fall!" string.
func TestDoBash_MissMessageFromSkillMessages(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillBash, 1)

	cb, attMsg, _, teardown := wireBashMessages(t, ch.Name)
	defer teardown()

	var missed bool
	for i := 0; i < 100; i++ {
		ch.Move = 100
		mob.SetPosition(combat.PosFighting)
		result := DoBash(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillBashNum {
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBashNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no bash miss observed in 100 tries (RNG)")
	}
	got := *attMsg
	if strings.Contains(got, "miss and fall") || strings.Contains(got, "powerful bash") {
		t.Errorf("miss emitted an OLD invented bash string (R4): %q", got)
	}
	if got == "" {
		t.Errorf("miss attacker message is empty — SkillMessage(132) did not emit")
	}
}

// TestDoBash_SkillKnownGateUnchanged — DP-1206 regression: GetSkill(bash)==0
// still returns the bare martial-arts message.
func TestDoBash_SkillKnownGateUnchanged(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillBash, 0)

	result := DoBash(ch, mob, w)
	if result.Success {
		t.Error("unknown bash should not succeed")
	}
	if result.MessageToCh != "You'd better leave all the martial arts to fighters." {
		t.Errorf("skill-known gate = %q, want the bare martial-arts line (DP-1206)", result.MessageToCh)
	}
}
