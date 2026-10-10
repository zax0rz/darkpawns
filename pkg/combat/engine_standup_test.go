package combat

import (
	"testing"
)

// TestProcessCombatPair_MobStandupRoundDrawsFirst — DP-1215: the NPC's
// Number(0,900) attacks-count draw (fight.c:1917) runs BEFORE the stand-up and
// attack — it is no longer skipped. Previously the wait/stand early-return
// skipped GetAttacksPerRound, dropping the draw and shifting the stream.

// TestProcessCombatPair_MobAttacksOnStandupRound — DP-1215: a downed mob with
// wait stands AND attacks in the same round (C zeroes attacks only for
// GET_MOB_WAIT, which is never written in normal gameplay). The mob should
// reach PosFighting and produce an attack (a Number(1,20) to-hit draw).

// TestProcessCombatPair_ScrambleCapitalized — R1: C act() → CAP uppercases the
// first byte of the composed message. A mob named "a guard trainee" emits
// "A guard trainee scrambles to his feet!".
func TestProcessCombatPair_ScrambleCapitalized(t *testing.T) {
	origCB := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(origCB) })
	SetCallbacks(defaultCombatCallbacks())

	attacker := &mockCombatant{name: "a guard trainee", npc: true, room: 1, position: PosSitting, fighting: &mockCombatant{name: "Hero"}, hp: 100, maxHP: 100, level: 10, ac: 10, sex: 0}
	defender := &mockCombatant{name: "Hero", room: 1, position: PosFighting, hp: 100, maxHP: 100, ac: 10}
	attacker.SetFightingBody(defender)

	ce := NewCombatEngine()
	var broadcasts []string
	ce.BroadcastFunc = func(_ int, msg string, _ []Combatant) { broadcasts = append(broadcasts, msg) }
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	// StartCombat stands combatants at entry (C set_fighting, fight.c:223);
	// re-down the attacker to model a mid-fight bash.
	attacker.SetPosition(PosSitting)

	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: attacker, Target: defender}])

	if len(broadcasts) == 0 {
		t.Fatal("expected scramble broadcast")
	}
	if want := "A guard trainee scrambles to his feet!"; broadcasts[0] != want {
		t.Errorf("scramble capitalization: got %q, want %q (R1 — C CAP)", broadcasts[0], want)
	}
}

// TestProcessCombatPair_PCStandup — DP-1215: C stands PCs too (fight.c:1990-
// 1998): !IS_NPC && GET_POS < POS_FIGHTING && !CHECK_WAIT (wait <= 1). A downed
// PC with wait ≤ 1 stands + gets "You drag yourself to your feet.\r\n". A downed
// PC with wait > 1 (CHECK_WAIT) stays sitting but STILL attacks (AWAKE gate).

// TestProcessCombatPair_PositionGateAWAKE — DP-1215: C gates the attack loop
// on AWAKE (GET_POS > POS_SLEEPING), NOT on POS_FIGHTING. A sitting attacker
// (PosSitting, awake) keeps swinging; a sleeping attacker (≤PosSleeping) stops.
