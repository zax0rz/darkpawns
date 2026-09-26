package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestDoStrike_MissRoutesThroughSkillMessage — C's do_strike miss arm calls
// damage(ch, vict, 0, SKILL_STRIKE) (new_cmds.c:1499), so the lines come from
// lib/misc/messages set 155 through the shared skill_message seam and no Go
// literal may be emitted (R1/R4). percent == prob is not < prob, so the arm is
// deterministic given the draw CmdStrike owns.
func TestDoStrike_MissRoutesThroughSkillMessage(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	ch.SetSkill(SkillStrike, 100)

	result := DoStrike(ch, mob, 100)

	if result.Success || result.Damage != 0 {
		t.Fatalf("percent==prob strike = success %v damage %d, want a miss", result.Success, result.Damage)
	}
	if result.SkillMsgType != SkillStrikeNum {
		t.Errorf("miss SkillMsgType = %d, want set %d (155)", result.SkillMsgType, SkillStrikeNum)
	}
	if !result.SkillMsgInDamage || result.DamageSkill != SkillStrike {
		t.Errorf("miss damage contract = inDamage %v skill %q, want true/%q",
			result.SkillMsgInDamage, result.DamageSkill, SkillStrike)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss emitted invented literals ch=%q vict=%q room=%q (R4)",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}
	if !result.StartCombat || result.WaitCh != 3 {
		t.Errorf("miss combat contract = start %v wait %d, want true/3 (PULSE_VIOLENCE+2)",
			result.StartCombat, result.WaitCh)
	}
}

// TestDoStrike_HitContract — percent < prob routes GET_LEVEL(ch)*.65 through
// damage(ch, vict, dam, SKILL_STRIKE) and defers improve_skill past the message
// dice (new_cmds.c:1494-1497; R3b).
func TestDoStrike_HitContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	ch.SetSkill(SkillStrike, 50)

	result := DoStrike(ch, mob, 1)

	if !result.Success {
		t.Fatalf("percent < prob strike did not succeed: %+v", result)
	}
	if want := int(float64(ch.GetLevel()) * 0.65); result.Damage != want {
		t.Errorf("hit damage = %d, want C integer conversion of level*.65 = %d", result.Damage, want)
	}
	if result.SkillMsgType != SkillStrikeNum || !result.SkillMsgInDamage || result.DamageSkill != SkillStrike {
		t.Errorf("hit damage contract = set %d inDamage %v skill %q, want %d/true/%q",
			result.SkillMsgType, result.SkillMsgInDamage, result.DamageSkill, SkillStrikeNum, SkillStrike)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillStrike {
		t.Errorf("hit DeferredImprove = %v, want [%q]", result.DeferredImprove, SkillStrike)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("hit emitted invented literals ch=%q vict=%q room=%q (R4)",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}
}

// TestDoStrike_SleepingVictimRaisesProbTo100 — C raises prob to the complete
// value for a victim at or below POS_SLEEPING (new_cmds.c:1474), which flips a
// percent that the awake skill value would have refused.
func TestDoStrike_SleepingVictimRaisesProbTo100(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	ch.SetSkill(SkillStrike, 1)

	mob.SetPosition(combat.PosStanding)
	if awake := DoStrike(ch, mob, 99); awake.Success {
		t.Errorf("awake skill-1 strike with percent 99 = hit, want miss (prob 1)")
	}

	mob.SetPosition(combat.PosSleeping)
	if sleeping := DoStrike(ch, mob, 99); !sleeping.Success {
		t.Errorf("sleeping skill-1 strike with percent 99 = miss, want hit (prob raised to 100)")
	}
}
