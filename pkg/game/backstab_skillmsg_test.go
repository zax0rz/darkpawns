package game

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// loadMessagesFile locates lib/misc/messages whether the test CWD is the repo
// root ("lib/misc/messages") or the package dir ("../../lib/misc/messages").
func loadMessagesFile(t *testing.T) combat.FightMessages {
	t.Helper()
	for _, path := range []string{"lib/misc/messages", "../../lib/misc/messages"} {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		messages, err := combat.LoadFightMessages(path)
		if err != nil {
			t.Fatalf("LoadFightMessages %q: %v", path, err)
		}
		return messages
	}
	t.Skip("lib/misc/messages not found from this test CWD")
	return nil
}

// wireBackstabMessages loads lib/misc/messages into the combat callbacks so the
// skill_message path can emit set 131 (Backstab). It returns the callbacks (so
// the test can call cb.SkillMessage directly), pointers to capture the attacker
// and room messages, and a teardown that restores the original callbacks. Uses
// the real messages file (read-only) so the miss text is the genuine set-131
// string, not a stub.
func wireBackstabMessages(t *testing.T, chName string) (cb *combat.GameCallbacks, attackerMsg *string, roomMsg *string, teardown func()) {
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

// TestBackstab_MissDrawCountAndOrder — R3: a backstab miss consumes exactly TWO
// draws from the shared CMWC stream, in order: number(1,101) (the skill roll in
// DoBackstab) then dice(1,N) (the skill_message variant selection). This is a
// self-referencing stream assertion: seed, run the miss path (DoBackstab +
// SkillMessage), then assert the next stream draw matches a reference that
// consumed exactly those two draws. Mirrors TestImproveSkill_DrawParity.
//
// C: act.offensive.c:220 percent=number(1,101); on miss, damage(ch,vict,0,
// SKILL_BACKSTAB) → fight.c:1023 skill_message → nr=dice(1,N). Two draws, in
// order, on the shared stream.
func TestBackstab_MissDrawCountAndOrder(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)
	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	ch.SetSkill(SkillBackstab, 1)
	mob.SetPosition(combat.PosStanding) // awake → miss branch reachable

	cb, _, _, teardown := wireBackstabMessages(t, ch.Name)
	defer teardown()

	// Seed the shared stream. Run the miss: DoBackstab draws number(1,101),
	// then cb.SkillMessage draws dice(1,N) from the SAME production stream.
	dprng.ResetStream(1)

	var result SkillResult
	missed := false
	for i := 0; i < 50; i++ {
		dprng.ResetStream(uint32(1 + i))
		result = DoBackstab(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillBackstabNum {
			// DRAW 1 (number(1,101)) consumed. Now run DRAW 2 via SkillMessage.
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBackstabNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG); draw-count not exercised")
	}

	// Self-referencing check: re-seed identically, consume the SAME two draws
	// manually, and assert the stream is in lockstep. number(1,101) is DRAW 1.
	// For DRAW 2, the variant count N for set 131 is fixed (load it to get N).
	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillBackstabNum)
	if !ok {
		t.Fatal("set 131 (Backstab) not in messages file")
	}
	n := len(variants)

	// Re-run the miss deterministically on a fresh seed and verify the stream
	// consumed exactly number(1,101) then dice(1,N) by comparing the post-state
	// of the stream against a reference that consumes exactly those two.
	const seed = 7
	dprng.ResetStream(seed)
	refDraw1 := dprng.Number(1, 101) // DRAW 1 reference
	refDraw2 := dprng.Dice(1, n)     // DRAW 2 reference
	wantNext := dprng.Number(0, 999) // next draw after both

	dprng.ResetStream(seed)
	result = DoBackstab(ch, mob, w)
	if result.Success || result.Damage != 0 {
		t.Skipf("seed %d did not produce a miss (got success=%v dam=%d); draw-order not exercised", seed, result.Success, result.Damage)
	}
	if result.SkillMsgType != SkillBackstabNum {
		t.Fatalf("miss SkillMsgType = %d, want %d (131)", result.SkillMsgType, SkillBackstabNum)
	}
	// DRAW 2 happens here, via cb.SkillMessage → production Dice on the shared stream.
	handled := cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBackstabNum, ch.GetRoom())
	if !handled {
		t.Fatal("SkillMessage(0, ..., 131) did not handle set 131")
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("miss draw count/order wrong: stream out of sync after DoBackstab+SkillMessage "+
			"(next=%d want=%d). DRAW1 should be number(1,101)=%d, DRAW2 dice(1,%d)=%d",
			got, wantNext, refDraw1, n, refDraw2)
	}
}

// TestBackstab_MissMessageFromSkillMessages — R4: the miss emits the
// lib/misc/messages set-131 miss text ("avoids your backstab"), NOT the old
// invented strings ("notices you" / "strikes deep"). This proves the reroute.
func TestBackstab_MissMessageFromSkillMessages(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)
	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	ch.SetSkill(SkillBackstab, 1)
	mob.SetPosition(combat.PosStanding)

	cb, attMsg, _, teardown := wireBackstabMessages(t, ch.Name)
	defer teardown()

	missed := false
	for i := 0; i < 100; i++ {
		result := DoBackstab(ch, mob, w)
		if !result.Success && result.Damage == 0 && result.SkillMsgType == SkillBackstabNum {
			cb.SkillMessage(0, ch.Name, mob.GetName(), SkillBackstabNum, ch.GetRoom())
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 100 tries (RNG); message-source not exercised")
	}

	got := *attMsg
	if !strings.Contains(got, "backstab") {
		t.Errorf("miss attacker message should mention backstab, got %q", got)
	}
	// The invented strings must be gone (R4).
	if strings.Contains(strings.ToLower(got), "notices you") {
		t.Errorf("miss emitted the OLD invented 'notices you' string (R4 violation): %q", got)
	}
	if strings.Contains(strings.ToLower(got), "strikes deep") {
		t.Errorf("miss emitted the OLD invented 'strikes deep' string (R4 violation): %q", got)
	}
	// The set-131 miss text is "avoids your backstab" (attacker form).
	if !strings.Contains(got, "avoids your backstab") {
		t.Errorf("miss should emit set-131 miss text ('avoids your backstab'), got %q", got)
	}
}

// TestBackstab_HitAppliesDamageOnce — the hit branch routes damage through
// SkillMessage + the DoSpellDamage pipeline; HP must be applied exactly once
// (no double-apply from both a direct TakeDamage and DoSpellDamage).
func TestBackstab_HitAppliesDamageOnce(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	ch.Level = 20
	mob := spawnTargetMob(t, w)
	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	ch.Stats.Str = 25 // high str → meaningful damage

	// A sleeping target auto-succeeds the skill roll (percent>prob only fails
	// when AWAKE), then the to-hit roll runs. Retry until a hit lands.
	cb, attMsg, _, teardown := wireBackstabMessages(t, ch.Name)
	defer teardown()

	var result SkillResult
	hit := false
	for i := 0; i < 40; i++ {
		mob.SetPosition(combat.PosSleeping)
		// Reset mob HP each attempt so we measure a single hit's effect.
		mob.SetHealth(mob.GetMaxHP())
		result = DoBackstab(ch, mob, w)
		if result.Success && result.Damage > 0 && result.SkillMsgType == SkillBackstabNum {
			// Emit the hit message via the skill_message path (as sendSkillResult does).
			cb.SkillMessage(result.Damage, ch.Name, mob.GetName(), SkillBackstabNum, ch.GetRoom())
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no hit observed in 40 tries (RNG); double-apply not exercised")
	}

	dam := result.Damage
	hpBefore := mob.GetMaxHP()

	// Simulate sendSkillResult's damage application: DoSpellDamage is the single
	// HP pipeline (DP-942). The hit must reduce HP by exactly dam, not 2*dam.
	w.DoSpellDamage(ch, mob, dam, "")

	hpAfter := mob.GetHP()
	applied := hpBefore - hpAfter
	if applied != dam {
		t.Errorf("hit applied HP %d, want exactly %d (double-apply check); hp %d→%d",
			applied, dam, hpBefore, hpAfter)
	}
	// The hit message must come from skill_message (the hit_msg), not the old
	// "strikes deep" invented string.
	if strings.Contains(strings.ToLower(*attMsg), "strikes deep") {
		t.Errorf("hit emitted the OLD invented 'strikes deep' string (R4): %q", *attMsg)
	}
}
