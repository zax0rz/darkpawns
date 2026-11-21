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

// The visible half drives every supported skill attack ID plus the generic
// hit types through damage()'s default arm. Each must send the same C pain line
// once the blow exceeds one quarter of the victim's maximum hit points
// (src/fight.c:1587-1592).
func TestApplySkillDamageEmitsTheHurtLineForEveryAttackType(t *testing.T) {
	cases := []struct {
		name       string
		attackType int
	}{
		{name: "undefined", attackType: combat.TYPE_UNDEFINED},
		{name: "hit", attackType: combat.TYPE_HIT},
		{name: "backstab", attackType: SkillBackstabNum},
		{name: "bash", attackType: SkillBashNum},
		{name: "cutthroat", attackType: SkillCutthroatNum},
		{name: "bearhug", attackType: SkillBearhugNum},
		{name: "kick", attackType: SkillKickNum},
		{name: "punch", attackType: SkillPunchNum},
		{name: "bite", attackType: SkillBiteNum},
		{name: "headbutt", attackType: SkillHeadbuttNum},
		{name: "trip", attackType: SkillTripNum},
		{name: "smackheads", attackType: SkillSmackheadsNum},
		{name: "slug", attackType: SkillSlugNum},
		{name: "charge", attackType: SkillChargeNum},
		{name: "serpent kick", attackType: SkillSerpentKickNum},
		{name: "circle", attackType: SkillCircleNum},
		{name: "groinrip", attackType: SkillGroinripNum},
		{name: "disembowel", attackType: SkillDisembowelNum},
		{name: "sleeper", attackType: SkillSleeperNum},
		{name: "neckbreak", attackType: SkillNeckbreakNum},
		{name: "dragon kick", attackType: SkillDragonKickNum},
		{name: "strike", attackType: SkillStrikeNum},
		{name: "ambush", attackType: SkillAmbushNum},
		{name: "tiger punch", attackType: SkillTigerPunchNum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

			w.ApplySkillDamageWithMessage(player, mob, mob.GetMaxHP()/4+10, tc.attackType, nil)

			if got := strings.Join(victimLines, ""); !strings.Contains(got, "That really did HURT!") {
				t.Fatalf("attack type %d victim lines = %q, want the C pain line", tc.attackType, got)
			}
		})
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
