package combat

// PL-005: C's attack loop re-reads FIGHTING(ch) on every swing
// (src/fight.c:2017-2024). damage() can clear it MID-LOOP — the mercy stop
// for an NPC attacker on a level-<=5 PC victim (fight.c:1491-1494) and the
// !AWAKE victim stop (fight.c:1630-1632) — after which C's hit() never runs.
// A multi-attack round that kept swinging through the stale pair killed
// characters C leaves incapacitated (the second swing read the prone-victim
// multiplier, fight.c:1854-1855) and diverged the shared RNG stream.

import (
	"testing"
)

// pl005Pair enrolls a level-11 NPC (2 attacks per round) against a PC at
// victimHP 20. The scripted roller answers 20 to every draw: natural-20
// auto-hits, no random bonus attack (20 >= level 11), fixed damage of
// damroll 10 + dice 20 = ~30, enough to stun the victim without killing.
func pl005Pair(t *testing.T, ce *CombatEngine, victimLevel int) (*mockCombatant, *mockCombatant) {
	t.Helper()
	mob := &mockCombatant{
		name: "Zoomaster", npc: true, room: 1, level: 11,
		hp: 200, maxHP: 200, position: PosStanding,
		thac0: 1, hitroll: 20, damroll: 10, str: 11,
		damageRoll: DiceRoll{Num: 1, Sides: 4},
	}
	victim := &mockCombatant{
		name: "Deadwar", npc: false, room: 1, level: victimLevel,
		hp: 20, maxHP: 20, position: PosStanding, ac: 10,
	}
	if err := ce.StartCombat(mob, victim); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	return mob, victim
}

func woundedBand(pos int) bool {
	return pos == PosStunned || pos == PosIncap || pos == PosMortally
}

// TestMercyStopEndsMultiAttackRound: a level-11 NPC (2 attacks per round)
// stuns a level-1 PC with its first swing. The mercy stop must end the round
// — no stale-pair second swing — and the !AWAKE stop must remove the victim
// from the round list, so later rounds draw nothing: the victim stays
// incapacitated with unchanged HP, exactly C's outcome.
func TestMercyStopEndsMultiAttackRound(t *testing.T) {
	WithRoller(NewScriptedRoller([]int{20}), func() {
		// Package callbacks can leak from earlier tests (a GetSkill of 100
		// makes every scripted roll a parry and cuts the round to one swing).
		// Pin them to a no-skill stub so the mob really gets both attacks.
		old := GetCallbacks()
		SetCallbacks(&GameCallbacks{GetSkill: func(Combatant, int) int { return 0 }})
		t.Cleanup(func() { SetCallbacks(old) })
		ce := NewCombatEngine()
		mob, victim := pl005Pair(t, ce, 1)

		ce.PerformRound()

		if !woundedBand(victim.GetPosition()) {
			t.Fatalf("victim position after the stun round = %d, want the wounded band (C leaves a level-1 victim incapacitated)", victim.GetPosition())
		}
		if victim.GetFightingBody() != nil {
			t.Fatal("victim still fighting after !AWAKE (fight.c:1630-1632)")
		}
		if mob.GetFightingBody() != nil {
			t.Fatal("mob still fighting after the mercy stop (fight.c:1491-1494)")
		}

		hpAfterStun := victim.GetHP()
		posAfterStun := victim.GetPosition()
		ce.PerformRound()
		ce.PerformRound()
		if got := victim.GetHP(); got != hpAfterStun {
			t.Fatalf("victim HP changed after the fight ended: %d -> %d (stale-pair swing or stood-up victim swung)", hpAfterStun, got)
		}
		if got := victim.GetPosition(); got != posAfterStun {
			t.Fatalf("victim position drifted after the fight ended: %d -> %d (round-runner stand-up must not raise a !AWAKE body)", posAfterStun, got)
		}
	})
}

// TestUnawakeVictimStopKeepsAttacker: without the mercy gate (victim level 7),
// only the victim stops (fight.c:1630-1632) — C's attacker keeps its FIGHTING
// reference and finishes the victim off on the next round (fight.c:2017-2024
// hits FIGHTING(ch), which still names the prone victim, at the prone
// multiplier fight.c:1854-1855).
func TestUnawakeVictimStopKeepsAttacker(t *testing.T) {
	WithRoller(NewScriptedRoller([]int{20}), func() {
		ce := NewCombatEngine()
		mob, victim := pl005Pair(t, ce, 7)

		ce.PerformRound()

		if !woundedBand(victim.GetPosition()) && victim.GetPosition() != PosDead {
			t.Fatalf("victim position after the stun round = %d, want the wounded band or dead", victim.GetPosition())
		}
		if victim.GetFightingBody() != nil {
			t.Fatal("victim still fighting after !AWAKE (fight.c:1630-1632)")
		}
		if mob.GetFightingBody() != victim {
			t.Fatal("attacker must keep FIGHTING (C finishes an un-awake level>5 victim off)")
		}
		if victim.GetPosition() != PosDead {
			hpAfterStun := victim.GetHP()
			ce.PerformRound()
			if victim.GetHP() >= hpAfterStun && victim.GetPosition() != PosDead {
				t.Fatal("attacker did not finish the prone victim on the next round (fight.c:1854-1855 prone multiplier)")
			}
		}
	})
}
