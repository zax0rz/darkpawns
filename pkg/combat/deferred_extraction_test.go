package combat

import "testing"

// The deferred-extraction window (src/handler.c:1194-1209): extract_char only
// flags a killed body, which stays in the room until extract_pending_chars at
// the next heartbeat (src/comm.c:805, handler.c:1221-1254). Every combatant
// whose turn comes later in the same violence pass keeps swinging at the
// flagged corpse: hit() consumes its to-hit and damage draws, then damage()
// refuses the body after its position gate (src/fight.c:1319-1326). die_with_
// killer stops only the victim and the killer (src/fight.c:612-616); everyone
// else's stop waits for extract_char_final's sweep (src/handler.c:1147-1151).
//
// R5h: with the corpse-swing branch reverted, the survivor's turn consumes
// only its attack-count draw and this test fails on the consumption count;
// with the old RetireCombatant sweep, the survivor is stopped outright and the
// window-reference assertion fails too.
func TestSurvivorSwingsAtFlaggedCorpseDraws(t *testing.T) {
	oldCB, oldRoller := GetCallbacks(), GetRoller()
	t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
	// A sparse callback set keeps damage() on the engine's own death path
	// (performOneHit → handleDeath); defaultCombatCallbacks routes the kill
	// through fight_core's legacy die handler instead.
	SetCallbacks(&GameCallbacks{})
	// Values wrap around; 5 keeps NPC attack counts at the level-derived base
	// (number(0,900) < level is false for level 1), 20 forces the to-hit, and
	// 9 feeds the damage dice.
	roller := NewScriptedRoller([]int{5, 20, 9, 9, 9, 9, 9, 9, 5, 20, 9, 9, 5, 20, 9, 9, 5, 20, 9, 9})
	SetRoller(roller)

	ce := NewCombatEngine()
	ce.BroadcastFunc = func(room int, msg string, excluded []Combatant) {}

	killer := &mockCombatant{
		name: "Killer", npc: true, room: 100, level: 1,
		hp: 100, maxHP: 100, position: PosStanding, ac: 100, str: 16,
		hitroll: 20, damroll: 50,
		damageRoll: DiceRoll{Num: 5, Sides: 20},
	}
	victim := &mockCombatant{
		name: "Victim", npc: true, room: 100, level: 1,
		hp: 1, maxHP: 1, position: PosStanding, ac: 100, str: 11,
		damageRoll: DiceRoll{Num: 1, Sides: 2},
	}
	survivor := &mockCombatant{
		name: "Survivor", npc: true, room: 100, level: 1,
		hp: 100, maxHP: 100, position: PosStanding,
		damageRoll: DiceRoll{Num: 1, Sides: 2},
	}

	// Enrollment order fixes the round order at [killer, victim, survivor]:
	// set_fighting prepends, so the survivor is enrolled first and the killer
	// last (src/fight.c:207-225).
	if err := ce.StartCombat(survivor, victim); err != nil {
		t.Fatal(err)
	}
	if err := ce.StartCombat(killer, victim); err != nil {
		t.Fatal(err)
	}

	ce.PerformRound()

	if victim.GetHP() > 0 || victim.GetPosition() != PosDead {
		t.Fatalf("victim survived the round: hp=%d pos=%d", victim.GetHP(), victim.GetPosition())
	}
	// die_with_killer stops the killer (fight.c:615-616) but leaves the
	// survivor's FIGHTING on the flagged body for the window.
	if killer.GetFightingBody() != nil {
		t.Fatal("killer still fighting after its victim died")
	}
	if survivor.GetFightingBody() != victim {
		t.Fatal("survivor lost the deferred-extraction window reference")
	}

	// The survivor's next turn swings at the flagged corpse: the attack-count
	// draw plus, per swing, hit()'s to-hit and damage draws (fight.c:1815,
	// 1843-1851) — all consumed before damage()'s refusal (fight.c:1319-1326).
	before := roller.Index
	ce.PerformRound()
	consumed := roller.Index - before
	if consumed != 3 {
		t.Fatalf("survivor corpse turn consumed %d draws, want 3 (attack count, to-hit, damage)", consumed)
	}
	if victim.GetHP() < -1 && victim.hp != victim.GetHP() {
		t.Fatalf("corpse took damage during the window: %d", victim.GetHP())
	}
}
