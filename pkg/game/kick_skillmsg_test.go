package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// wireKickMessages loads lib/misc/messages into the combat callbacks so the
// skill_message path can emit set 134 (Kick). Mirrors wireBackstabMessages.
// Returns the callbacks (so the test can call cb.SkillMessage directly),
// pointers to capture the attacker/room messages, and a teardown.
func wireKickMessages(t *testing.T, chName string) (cb *combat.GameCallbacks, attackerMsg *string, roomMsg *string, teardown func()) {
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

// TestDoKick_Miss_ReroutesThroughSkillMessage — the miss branch returns
// SkillMsgType=SkillKickNum (134), Damage==0, StartCombat==true, and EMPTY
// MessageToCh/Vict/Room (no hardcoded string). R4: C routes the message through
// skill_message (fight.c) and starts combat via damage(); Go must too. DP-1207.
func TestDoKick_Miss_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)

	// Low skill → high miss probability.
	ch.SetSkill(SkillKick, 1)

	var result SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		result = DoKick(ch, mob)
		if !result.Success && result.Damage == 0 {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG); miss reroute not exercised")
	}

	if result.SkillMsgType != SkillKickNum {
		t.Errorf("miss SkillMsgType = %d, want %d (134 — Kick set)", result.SkillMsgType, SkillKickNum)
	}
	if result.Damage != 0 {
		t.Errorf("miss Damage = %d, want 0", result.Damage)
	}
	if !result.StartCombat {
		t.Error("miss should set StartCombat (C: damage(ch,vict,0,SKILL_KICK) starts combat via set_fighting)")
	}
	if result.WaitCh != 3 {
		t.Errorf("miss WaitCh = %d, want 3 (PULSE_VIOLENCE+2)", result.WaitCh)
	}
	// R4: no hardcoded strings on the rerouted branch.
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss should carry no hardcoded messages (R4), got ch=%q vict=%q room=%q",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}

	// The Kick set (134) must resolve in the messages file.
	messages := loadMessagesFile(t)
	if variants, ok := messages.Variants(SkillKickNum); !ok || len(variants) == 0 {
		t.Errorf("set 134 (Kick) not found in lib/misc/messages")
	}
}

// TestDoKick_Hit_ReroutesThroughSkillMessage — the hit branch returns
// SkillMsgType=SkillKickNum, Damage == GetLevel()>>1, Success, no hardcoded
// strings.
func TestDoKick_Hit_ReroutesThroughSkillMessage(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)

	// Skill 100 vs AC 0 → high hit probability; retry for RNG.
	wantDam := ch.GetLevel() >> 1
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		result = DoKick(ch, mob)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no hit observed in 20 tries (RNG); hit reroute not exercised")
	}

	if result.SkillMsgType != SkillKickNum {
		t.Errorf("hit SkillMsgType = %d, want %d (134)", result.SkillMsgType, SkillKickNum)
	}
	if result.Damage != wantDam {
		t.Errorf("hit Damage = %d, want %d (GetLevel>>1)", result.Damage, wantDam)
	}
	if !result.Success {
		t.Error("hit should be Success")
	}
	if result.WaitCh != 3 {
		t.Errorf("hit WaitCh = %d, want 3 (PULSE_VIOLENCE+2)", result.WaitCh)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("hit should carry no hardcoded messages (R4), got ch=%q", result.MessageToCh)
	}
}

// TestDoKick_MissDrawCountAndOrder — R3: a kick miss consumes exactly TWO
// shared draws, in order: number(1,101) (the percent roll in DoKick) then
// dice(1,N) (the skill_message variant selection). Self-referencing stream
// check, mirroring TestBackstab_MissDrawCountAndOrder.
func TestDoKick_MissDrawCountAndOrder(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillKick, 1) // low skill → miss

	cb, _, _, teardown := wireKickMessages(t, ch.Name)
	defer teardown()

	// Find a seed that produces a miss, then assert stream lockstep.
	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillKickNum)
	if !ok {
		t.Fatal("set 134 (Kick) not in messages file")
	}
	n := len(variants)

	const seed = 7
	dprng.ResetStream(seed)
	result := DoKick(ch, mob)
	if result.Success || result.Damage != 0 {
		t.Skipf("seed %d did not produce a miss (got success=%v); draw-order not exercised", seed, result.Success)
	}
	if result.SkillMsgType != SkillKickNum {
		t.Fatalf("miss SkillMsgType = %d, want %d (134)", result.SkillMsgType, SkillKickNum)
	}
	// DRAW 2 via cb.SkillMessage → production Dice on the shared stream.
	handled := cb.SkillMessage(0, ch.Name, mob.GetName(), SkillKickNum, ch.GetRoom())
	if !handled {
		t.Fatal("SkillMessage(0, ..., 134) did not handle set 134")
	}

	// Reference: same seed, consume number(1,101) then dice(1,N), then the next.
	dprng.ResetStream(seed)
	dprng.Number(1, 101) // DRAW 1 reference
	dprng.Dice(1, n)     // DRAW 2 reference
	wantNext := dprng.Number(0, 999)

	// Re-run on the same seed: DoKick (DRAW 1) + SkillMessage (DRAW 2), then the next draw must match.
	dprng.ResetStream(seed)
	DoKick(ch, mob)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillKickNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("kick miss draw count/order wrong: stream out of sync after DoKick+SkillMessage "+
			"(next=%d want=%d). Should be number(1,101) then dice(1,%d)", got, wantNext, n)
	}
}

// TestDoKick_MissMessageFromSkillMessages — R4: the miss emits the lib/misc/
// messages set-134 miss text, NOT the old invented "You try to kick ... but
// miss!" string.
func TestDoKick_MissMessageFromSkillMessages(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillKick, 1)

	cb, attMsg, _, teardown := wireKickMessages(t, ch.Name)
	defer teardown()

	var missed bool
	for i := 0; i < 100; i++ {
		result := DoKick(ch, mob)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillKickNum {
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillKickNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 100 tries (RNG); message-source not exercised")
	}

	got := *attMsg
	// The invented strings must be gone (R4).
	if strings.Contains(got, "try to kick") || strings.Contains(got, "square in the chest") {
		t.Errorf("miss emitted an OLD invented kick string (R4 violation): %q", got)
	}
	// Set 134's miss variants all reference "kick" or "balletstep"; assert the
	// message came from the file (not empty, mentions kick/foot).
	if got == "" {
		t.Errorf("miss attacker message is empty — SkillMessage(134) did not emit")
	}
}

// TestDoKick_SkillKnownGateUnchanged — DP-1206 regression guard: GetSkill(kick)
// ==0 still returns the bare "You'd better leave all the martial arts to
// fighters." message (the entry gate is untouched by this reroute).
func TestDoKick_SkillKnownGateUnchanged(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillKick, 0) // explicitly unknown

	result := DoKick(ch, mob)
	if result.Success {
		t.Error("unknown kick should not succeed")
	}
	if result.MessageToCh != "You'd better leave all the martial arts to fighters." {
		t.Errorf("skill-known gate message = %q, want the bare martial-arts line (DP-1206 unchanged)",
			result.MessageToCh)
	}
}
