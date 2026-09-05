package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestDoSlug_ResultContract pins do_slug's damage() boundary: both outcome
// arms use skill-message set 146, enroll combat, and wait two violence pulses.
// The miss improves the skill only after damage() returns (new_cmds.c:850-875).
func TestDoSlug_ResultContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillSlug, 100)

	var hit SkillResult
	for seed := uint32(1); seed < 100; seed++ {
		dprng.ResetStream(seed)
		hit = DoSlug(ch, mob)
		if hit.Success {
			break
		}
	}
	if !hit.Success {
		t.Fatal("no slug hit observed in 99 deterministic seeds")
	}
	if hit.Damage < ch.GetLevel()/2 || hit.Damage > ch.GetLevel()*2 {
		t.Fatalf("hit damage = %d, want C level*{1/2,1,3/2,2} range", hit.Damage)
	}
	if hit.SkillMsgType != SkillSlugNum || hit.DamageSkill != SkillSlug {
		t.Fatalf("hit message contract = set %d damage skill %q, want %d/%q", hit.SkillMsgType, hit.DamageSkill, SkillSlugNum, SkillSlug)
	}
	if !hit.StartCombat || hit.WaitCh != 2 {
		t.Fatalf("hit combat/wait = start %v wait %d, want true/2", hit.StartCombat, hit.WaitCh)
	}
	if len(hit.DeferredImprove) != 0 {
		t.Fatalf("hit deferred improvement = %v, want none", hit.DeferredImprove)
	}
	if hit.MessageToCh != "" || hit.MessageToVict != "" || hit.MessageToRoom != "" {
		t.Fatalf("hit carries invented literal messages: ch=%q victim=%q room=%q", hit.MessageToCh, hit.MessageToVict, hit.MessageToRoom)
	}

	ch.SetSkill(SkillSlug, 1)
	dprng.ResetStream(1)
	miss := DoSlug(ch, mob)
	if miss.Success || miss.Damage != 0 {
		t.Fatalf("skill-1 slug = success %v damage %d, want false/0", miss.Success, miss.Damage)
	}
	if miss.SkillMsgType != SkillSlugNum || miss.DamageSkill != SkillSlug || !miss.StartCombat || miss.WaitCh != 2 {
		t.Fatalf("miss contract = %#v, want set %d/%q/start/2", miss, SkillSlugNum, SkillSlug)
	}
	if len(miss.DeferredImprove) != 1 || miss.DeferredImprove[0] != SkillSlug {
		t.Fatalf("miss deferred improvement = %v, want [%q]", miss.DeferredImprove, SkillSlug)
	}
	if miss.MessageToCh != "" || miss.MessageToVict != "" || miss.MessageToRoom != "" {
		t.Fatalf("miss carries invented literal messages: ch=%q victim=%q room=%q", miss.MessageToCh, miss.MessageToVict, miss.MessageToRoom)
	}
}

func TestDoSlug_GateOrderAndMessages(t *testing.T) {
	t.Run("unknown skill", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		mob := spawnTargetMob(t, w)
		result := DoSlug(ch, mob)
		if result.MessageToCh != SkillUnknownMsg[SkillSlug] {
			t.Fatalf("unknown-skill message = %q, want %q", result.MessageToCh, SkillUnknownMsg[SkillSlug])
		}
	})

	t.Run("self precedes weapon and mount", func(t *testing.T) {
		_, ch := newCombatTestWorld(t)
		ch.SetSkill(SkillSlug, 100)
		weapon := makeSpikeWeapon("slug")
		equipWeapon(t, ch, weapon)
		ch.SetAffect(affMounted, true)

		result := DoSlug(ch, ch)
		if result.MessageToCh != "You curl up your fist and slug yourself in the nose! Ouch!" {
			t.Fatalf("self-target message = %q", result.MessageToCh)
		}
	})

	t.Run("wielded weapon precedes mount", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.SetSkill(SkillSlug, 100)
		mob := spawnTargetMob(t, w)
		weapon := makeSpikeWeapon("slug")
		equipWeapon(t, ch, weapon)
		ch.SetAffect(affMounted, true)

		result := DoSlug(ch, mob)
		if result.MessageToCh != "You can't make a fist while wielding a weapon!" {
			t.Fatalf("weapon-gate message = %q", result.MessageToCh)
		}
	})

	t.Run("mounted", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.SetSkill(SkillSlug, 100)
		mob := spawnTargetMob(t, w)
		ch.SetAffect(affMounted, true)

		result := DoSlug(ch, mob)
		if result.MessageToCh != "Dismount first!" {
			t.Fatalf("mounted message = %q", result.MessageToCh)
		}
	})
}

// TestDoSlug_SkillMessageDrawOrder pins C's number(1,101) → skill_message
// dice(1,N) sequence. C's improve_skill runs after damage() and is represented
// separately by DeferredImprove in the result contract.
func TestDoSlug_SkillMessageDrawOrder(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillSlug, 1)

	cb, attackerMsg, roomMsg, teardown := wireKickMessages(t, ch.Name)
	defer teardown()

	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillSlugNum)
	if !ok || len(variants) == 0 {
		t.Fatal("set 146 (Slug) not in messages file")
	}

	const seed = 1
	dprng.ResetStream(seed)
	result := DoSlug(ch, mob)
	if result.Success || result.SkillMsgType != SkillSlugNum {
		t.Fatalf("seed %d did not produce the expected slug miss: %#v", seed, result)
	}
	if !cb.SkillMessage(0, ch.Name, mob.GetName(), SkillSlugNum, ch.GetRoom()) {
		t.Fatal("SkillMessage(0, ..., 146) did not handle set 146")
	}
	if *attackerMsg != "You miss your swing at a training dummy." {
		t.Fatalf("set-146 miss attacker message = %q, want C miss text", *attackerMsg)
	}
	if !strings.Contains(*roomMsg, "ducks as TestPlayer swings") {
		t.Fatalf("set-146 miss room message = %q, want C room text", *roomMsg)
	}

	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Dice(1, len(variants))
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	DoSlug(ch, mob)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillSlugNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("slug draw order wrong: next=%d want=%d", got, wantNext)
	}
}
