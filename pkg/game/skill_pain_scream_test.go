package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// C's damage() default arm (src/fight.c:1587-1592) burns one number(0,2) draw
// whenever the blow exceeds a quarter of the victim's maximum hit points, for
// EVERY attack type:
//
//	if (dam > (GET_MAX_HIT(victim) / 4)) {
//	    act("That really did HURT!", FALSE, victim, 0, 0, TO_CHAR);
//	    if (!number(0,2))
//	        act("$N screams in pain!", FALSE, ch, 0, victim, TO_NOTVICT);
//	}
//
// The skill seam burned that draw only for charge, so every later draw in the
// round sat one position early — the headbutt vehicle in
// scenarios/skill-pain-scream-draw-depth.txt is the transcript proof.
func TestApplySkillDamageConsumesThePainScreamDraw(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMaxHP(200)

	// 60 is over a quarter of 200 and leaves 140 HP, so the victim survives
	// above POS_STUNNED and takes the default arm.
	dam := mob.GetMaxHP()/4 + 10

	dprng.ResetStream(1)
	_ = dprng.Number(0, 2)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(1)
	w.ApplySkillDamageWithMessage(player, mob, dam, SkillHeadbuttNum, nil)

	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("draw after the skill hit = %d, want %d: the number(0,2) pain draw was not consumed", got, wantNext)
	}
}

// The same arm's visible half: a non-charge skill that does more than a quarter
// of the victim's maximum hit points prints "That really did HURT!" to the
// victim (src/fight.c:1589).
func TestApplySkillDamageEmitsTheHurtLineForEveryAttackType(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMaxHP(200)

	previous := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(previous) })

	var victimLines []string
	cb := w.WireCombatCallbacks()
	// The pain line is a body text message: pkg/combat's sendCombatMessage
	// prefers the SendText hook and falls back to body.SendMessage, so capture
	// both routes rather than assuming one.
	capture := func(body combat.Combatant, msg string) {
		if body == mob {
			victimLines = append(victimLines, msg)
		}
	}
	cb.SendText = capture
	cb.SendToChar = capture
	combat.SetCallbacks(cb)

	w.ApplySkillDamageWithMessage(player, mob, mob.GetMaxHP()/4+10, SkillHeadbuttNum, nil)

	if got := strings.Join(victimLines, ""); !strings.Contains(got, "That really did HURT!") {
		t.Fatalf("victim lines = %q, want the C pain line", got)
	}
}

// Below the quarter threshold C draws nothing and prints nothing, so the seam
// must stay silent: the draw above is gated, not unconditional.
func TestApplySkillDamageSkipsThePainArmUnderAQuarter(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMaxHP(200)

	dam := mob.GetMaxHP()/4 - 10 // under the threshold, still above POS_STUNNED

	dprng.ResetStream(1)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(1)
	w.ApplySkillDamageWithMessage(player, mob, dam, SkillHeadbuttNum, nil)

	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("draw after a sub-quarter hit = %d, want %d: the pain draw must be gated on dam > max_hit/4", got, wantNext)
	}
}
