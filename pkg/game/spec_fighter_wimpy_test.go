package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// C's damage() default arm calls two wimpy blocks, for every attack type
// (src/fight.c:1596-1611):
//
//	if (GET_HIT(victim) < (GET_MAX_HIT(victim) / 4)) {
//	    ...wounded-band line...
//	    if (MOB_FLAGGED(victim, MOB_WIMPY) && (ch != victim))
//	        do_flee(victim, "", 0, 0);
//	}
//	if (!IS_NPC(victim) && GET_WIMP_LEV(victim) && (victim != ch) &&
//	    GET_POS(victim) >= POS_FIGHTING && GET_HIT(victim) < GET_WIMP_LEV(victim)) {
//	    send_to_char("You wimp out, and attempt to flee!\r\n", victim);
//	    ...do_retreat or do_flee...
//	}
//
// SPECIAL(fighter) reaches damage() through do_headbutt/do_bash with subcmd 1
// (src/spec_procs.c:509-535) — the native mob-skill seam, not a command — and
// that seam printed the wounded-band line but neither wimpy block, so a victim
// of a mob special never wimped out and the flee's draws never reached the
// shared stream.
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

	// 20 hit points is under a quarter of 100 (the wounded band) and under the
	// wimp level of 30, and the victim is fighting, so both wimpy blocks apply.
	victim.SetMaxHP(100)
	victim.SetHP(20)
	victim.WimpLevel = 30
	victim.SetPosition(combat.PosFighting)

	w.emitMobSkillSurvival(attacker, victim, 10, combat.PosStanding)

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
	victim.SetHP(60) // above the wimp level below, still inside the wounded band
	victim.WimpLevel = 30
	victim.SetPosition(combat.PosFighting)

	w.emitMobSkillSurvival(attacker, victim, 10, combat.PosStanding)

	if got := strings.Join(lines, ""); strings.Contains(got, "You wimp out") {
		t.Fatalf("victim lines = %q, want no wimpy line above the wimp level", got)
	}
	if fled {
		t.Fatal("fled above the wimp level")
	}
}
