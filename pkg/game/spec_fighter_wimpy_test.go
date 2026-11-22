package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// C's damage() default arm includes a separate player wimp-level gate
// (src/fight.c:1601-1610):
//
//	if (!IS_NPC(victim) && GET_WIMP_LEV(victim) && (victim != ch) &&
//	    GET_POS(victim) >= POS_FIGHTING && GET_HIT(victim) < GET_WIMP_LEV(victim)) {
//	    send_to_char("You wimp out, and attempt to flee!\r\n", victim);
//	    ...do_retreat or do_flee...
//	}
//
// The native mob damage seam used by SPECIAL(fighter) must carry that gate
// through damage's surviving-victim arm. These tests enter through
// mobSkillDamage, then distinguish the player wimp-level threshold below.
func TestMobSkillSurvivalRunsDamageWimpyBlocks(t *testing.T) {
	w, victim := newCombatTestWorld(t)
	attacker := spawnTargetMob(t, w)

	previous := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(previous) })

	var lines []string
	fled := false
	cb := w.WireCombatCallbacks()
	capture := func(body combat.Combatant, msg string) {
		if body == victim {
			lines = append(lines, msg)
		}
	}
	cb.SendText = capture
	cb.SendToChar = capture
	cb.DoFlee = func(body combat.Combatant) {
		if body == victim {
			fled = true
		}
	}
	combat.SetCallbacks(cb)

	// 20 hit points is under the wimp level of 30, and the victim is fighting.
	victim.SetMaxHP(100)
	victim.SetHP(20)
	victim.WimpLevel = 30
	victim.SetPosition(combat.PosFighting)

	if !w.mobSkillDamage(attacker, victim, 1, SkillHeadbuttNum) {
		t.Fatal("mob skill damage did not reach the surviving-victim arm")
	}

	if got := strings.Join(lines, ""); !strings.Contains(got, "You wimp out, and attempt to flee!") {
		t.Fatalf("victim lines = %q, want C's wimpy line", got)
	}
	if !fled {
		t.Fatal("the wimpy arm did not run do_flee for a player under its wimp level")
	}
}

// The wimpy block is gated on the victim being below its wimp level: at or above
// it the arm must stay silent, so the test above cannot pass on an
// unconditional flee.
func TestMobSkillSurvivalSkipsWimpyAboveTheWimpLevel(t *testing.T) {
	w, victim := newCombatTestWorld(t)
	attacker := spawnTargetMob(t, w)

	previous := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(previous) })

	var lines []string
	fled := false
	cb := w.WireCombatCallbacks()
	capture := func(body combat.Combatant, msg string) {
		if body == victim {
			lines = append(lines, msg)
		}
	}
	cb.SendText = capture
	cb.SendToChar = capture
	cb.DoFlee = func(body combat.Combatant) {
		if body == victim {
			fled = true
		}
	}
	combat.SetCallbacks(cb)

	victim.SetMaxHP(100)
	victim.SetHP(20) // below the wounded band, but above the wimp level after damage
	victim.WimpLevel = 10
	victim.SetPosition(combat.PosFighting)

	if !w.mobSkillDamage(attacker, victim, 1, SkillHeadbuttNum) {
		t.Fatal("mob skill damage did not reach the surviving-victim arm")
	}

	got := strings.Join(lines, "")
	if !strings.Contains(got, "You wish that your wounds would stop BLEEDING so much!") {
		t.Fatalf("victim lines = %q, want the wounded-band line from the survival arm", got)
	}
	if strings.Contains(got, "You wimp out") {
		t.Fatalf("victim lines = %q, want no wimpy line above the wimp level", got)
	}
	if fled {
		t.Fatal("fled above the wimp level")
	}
}
