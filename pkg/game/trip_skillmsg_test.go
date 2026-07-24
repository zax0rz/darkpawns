package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// wireTripMessages loads lib/misc/messages so the skill_message path can emit
// set 144 (Trip). Mirrors wireKickMessages.
func wireTripMessages(t *testing.T, chName string) (cb *combat.GameCallbacks, attackerMsg *string, roomMsg *string, teardown func()) {
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

// TestDoTrip_Miss_ReroutesThroughSkillMessage — miss returns SkillMsgType=144,
// Damage==0, StartCombat==true, SelfStumble preserved (caster falls), empty
// MessageTo*. R4: C routes the message through skill_message + starts combat.
func TestDoTrip_Miss_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	ch.SetSkill(SkillTrip, 1) // low skill → miss

	var result SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		result = DoTrip(ch, mob, w)
		if !result.Success && result.Damage == 0 {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no trip miss observed in 50 tries (RNG)")
	}

	if result.SkillMsgType != SkillTripNum {
		t.Errorf("miss SkillMsgType = %d, want %d (144)", result.SkillMsgType, SkillTripNum)
	}
	if result.Damage != 0 {
		t.Errorf("miss Damage = %d, want 0", result.Damage)
	}
	if !result.StartCombat {
		t.Error("miss should set StartCombat (C: damage(…,0,SKILL_TRIP) → set_fighting)")
	}
	if !result.SelfStumble {
		t.Error("miss should keep SelfStumble (C: GET_POS(ch)=POS_SITTING)")
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}

	messages := loadMessagesFile(t)
	if variants, ok := messages.Variants(SkillTripNum); !ok || len(variants) == 0 {
		t.Errorf("set 144 (Trip) not found in lib/misc/messages")
	}
}

// TestDoTrip_Hit_ReroutesThroughSkillMessage — hit returns SkillMsgType=144,
// Damage==(level/2)+1, TargetFalls, no hardcoded strings.
func TestDoTrip_Hit_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	wantDam := (ch.GetLevel() / 2) + 1
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		result = DoTrip(ch, mob, w)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no trip hit observed in 20 tries (RNG)")
	}

	if result.SkillMsgType != SkillTripNum {
		t.Errorf("hit SkillMsgType = %d, want %d (144)", result.SkillMsgType, SkillTripNum)
	}
	if result.Damage != wantDam {
		t.Errorf("hit Damage = %d, want %d ((level/2)+1)", result.Damage, wantDam)
	}
	if !result.Success {
		t.Error("hit should be Success")
	}
	if !result.TargetFalls {
		t.Error("hit should keep TargetFalls (C: GET_POS(victim)=POS_SITTING)")
	}
	if result.MessageToCh != "" {
		t.Errorf("hit should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}
}

// TestDoTrip_MissDrawCountAndOrder — R3: a trip miss consumes number(1,121)
// then the skill_message dice(1,N), in that order. Self-referencing stream check.
func TestDoTrip_MissDrawCountAndOrder(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillTrip, 1)

	cb, _, _, teardown := wireTripMessages(t, ch.Name)
	defer teardown()

	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillTripNum)
	if !ok {
		t.Fatal("set 144 (Trip) not in messages file")
	}
	n := len(variants)

	const seed = 7
	dprng.ResetStream(seed)
	result := DoTrip(ch, mob, w)
	if result.Success || result.Damage != 0 {
		t.Skipf("seed %d did not produce a miss (success=%v); draw-order not exercised", seed, result.Success)
	}
	if result.SkillMsgType != SkillTripNum {
		t.Fatalf("miss SkillMsgType = %d, want %d", result.SkillMsgType, SkillTripNum)
	}
	handled := cb.SkillMessage(0, ch.Name, mob.GetName(), SkillTripNum, ch.GetRoom())
	if !handled {
		t.Fatal("SkillMessage(0, ..., 144) did not handle set 144")
	}

	// Reference: number(1,121) then dice(1,N).
	dprng.ResetStream(seed)
	dprng.Number(1, 121)
	dprng.Dice(1, n)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	DoTrip(ch, mob, w)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillTripNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("trip miss draw count/order wrong: next=%d want=%d (number(1,121) then dice(1,%d))", got, wantNext, n)
	}
}

// TestDoTrip_MissMessageFromSkillMessages — R4: miss emits set-144 text, not
// the old invented "You try to trip… but fail and fall!" string.
func TestDoTrip_MissMessageFromSkillMessages(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillTrip, 1)

	cb, attMsg, _, teardown := wireTripMessages(t, ch.Name)
	defer teardown()

	var missed bool
	for i := 0; i < 100; i++ {
		result := DoTrip(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillTripNum {
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillTripNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no trip miss observed in 100 tries (RNG)")
	}
	got := *attMsg
	if strings.Contains(got, "try to trip") || strings.Contains(got, "fail and fall") {
		t.Errorf("miss emitted an OLD invented trip string (R4): %q", got)
	}
	if got == "" {
		t.Errorf("miss attacker message is empty — SkillMessage(144) did not emit")
	}
}

// TestDoTrip_SkillKnownGateUnchanged — DP-1206 regression: GetSkill(trip)==0
// still returns the bare sneaky-stuff message.
func TestDoTrip_SkillKnownGateUnchanged(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillTrip, 0)

	result := DoTrip(ch, mob, w)
	if result.Success {
		t.Error("unknown trip should not succeed")
	}
	if result.MessageToCh != "You'd better leave the sneaky stuff to the thieves." {
		t.Errorf("skill-known gate = %q, want the bare sneaky-stuff line (DP-1206)", result.MessageToCh)
	}
}
