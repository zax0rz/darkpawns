package combat

import "testing"

// TestStartCombat_StandsSleepingDefender pins C set_fighting's POS_FIGHTING
// (fight.c:223): engaging a sleeping victim (POS_SLEEPING > POS_STUNNED, so it
// passes the fight.c:1443 gate) stands BOTH sides at entry. The AWAKE attack
// gate and CalculateHitChance's awake-defender AC both read the defender's
// position, so a sleeping victim that is attacked retaliates on the next
// violence round instead of being stopped.
func TestStartCombat_StandsSleepingDefender(t *testing.T) {
	attacker := &mockCombatant{name: "Hero", npc: false, room: 1, position: PosStanding, fighting: "", hp: 100, maxHP: 100, level: 10, ac: 10, thac0: 10}
	defender := &mockCombatant{name: "Orc", npc: true, room: 1, position: PosSleeping, hp: 100, maxHP: 100, ac: 10}

	ce := NewCombatEngine()
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	if defender.GetPosition() != PosFighting {
		t.Errorf("sleeping defender must stand at entry (C set_fighting, fight.c:223): pos %d, want %d", defender.GetPosition(), PosFighting)
	}
	if attacker.GetPosition() != PosFighting {
		t.Errorf("attacker must stand at entry: pos %d", attacker.GetPosition())
	}
}

// TestStartCombat_DoesNotStandStunnedDefender pins C's position gate
// (fight.c:1443: GET_POS(victim) > POS_STUNNED): a mortally-wounded, incap, or
// stunned victim is NOT stood by set_fighting, so attacking it must leave its
// prone position intact rather than popping it up to POS_FIGHTING. Regression
// guard for the round-9 defender-stand (#648), which stood it unconditionally.
func TestStartCombat_DoesNotStandStunnedDefender(t *testing.T) {
	for _, pos := range []int{PosMortally, PosIncap, PosStunned} {
		attacker := &mockCombatant{name: "Hero", npc: false, room: 1, position: PosStanding, fighting: "", hp: 100, maxHP: 100, level: 10, ac: 10, thac0: 10}
		defender := &mockCombatant{name: "Dyingmob", npc: true, room: 1, position: pos, hp: 1, maxHP: 100, ac: 10}

		ce := NewCombatEngine()
		if err := ce.StartCombat(attacker, defender); err != nil {
			t.Fatalf("StartCombat(pos=%d): %v", pos, err)
		}
		if defender.GetPosition() != pos {
			t.Errorf("sub-stunned defender (pos %d) must not be stood at entry (fight.c:1443): got pos %d", pos, defender.GetPosition())
		}
	}
}
