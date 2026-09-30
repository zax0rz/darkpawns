package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// WAIT_STATE's argument is pulses, including additive arithmetic (R3c/R5c).
func TestAdditiveWaitSkillBranches(t *testing.T) {
	for _, skill := range []string{SkillKick, SkillDragonKick, SkillStrike, SkillCircle} {
		for _, branch := range []string{"hit", "miss"} {
			t.Run(skill+"/"+branch, func(t *testing.T) {
				ch := NewPlayer(1, "Attacker", 1001)
				ch.SetLevel(20)
				ch.Move = 100
				ch.SetSkill(skill, 100)
				target := NewPlayer(2, "Victim", 1001)
				target.SetPosition(combat.PosStanding)
				target.SetAC(1000) // Forces the kick probability below zero.
				if branch == "miss" {
					target.SetAC(-1000)
					ch.SetSkill(skill, 1)
				}
				dprng.ResetStream(1)
				var result SkillResult
				switch skill {
				case SkillKick:
					result = DoKick(ch, target)
				case SkillDragonKick:
					result = DoDragonKick(ch, target)
				case SkillStrike:
					percent := 1
					if branch == "miss" {
						percent = 101
					}
					result = DoStrike(ch, target, percent)
				case SkillCircle:
					equipWeapon(t, ch, makeCircleWeapon())
					if branch == "hit" {
						target.SetPosition(combat.PosSleeping)
					}
					combat.WithRoller(combat.NewScriptedRoller([]int{20, 1, 1}), func() { result = DoCircle(ch, target) })
				}
				if result.Success != (branch == "hit") {
					t.Fatalf("wrong branch: %#v", result)
				}
				if result.WaitCh != 0 || result.WaitChPulses != engine.PULSE_VIOLENCE+2 {
					t.Fatalf("wait = %d rounds / %d pulses, want 0 rounds / 22 pulses", result.WaitCh, result.WaitChPulses)
				}
			})
		}
	}
}

func TestCirclePassedSkillMissWaitPulses(t *testing.T) {
	ch := NewPlayer(1, "Attacker", 1001)
	ch.SetLevel(20)
	ch.Move = 100
	ch.SetSkill(SkillCircle, 101)
	target := NewPlayer(2, "Victim", 1001)
	target.SetPosition(combat.PosStanding)
	equipWeapon(t, ch, makeCircleWeapon())
	var result SkillResult
	combat.WithRoller(combat.NewScriptedRoller([]int{1}), func() { result = DoCircle(ch, target) })
	if result.Success || !result.SkillMsgAfterDamage {
		t.Fatalf("expected passed skill roll and THAC0 miss: %#v", result)
	}
	if result.WaitCh != 0 || result.WaitChPulses != engine.PULSE_VIOLENCE+2 {
		t.Fatalf("wait = %d rounds / %d pulses, want 22 raw pulses", result.WaitCh, result.WaitChPulses)
	}
}
