package combat

import "testing"

// TestStartCombat_DoesNotStandDefenderAtEntry pins the C set_fighting timing:
// the victim's POS_FIGHTING is set inside damage() (fight.c:1443-1445), which
// runs AFTER the opener's to-hit decision — NOT at combat entry. So StartCombat
// must leave the defender's position untouched; a sleeping victim stays asleep
// through the opener's to-hit (so it is auto-hit, AWAKE(victim) being false,
// like C) and is stood at the damage point (performOneHit). The attacker, which
// is always standing when it initiates, does stand at entry.
func TestStartCombat_DoesNotStandDefenderAtEntry(t *testing.T) {
	for _, pos := range []int{PosSleeping, PosResting, PosSitting, PosStanding} {
		attacker := &mockCombatant{name: "Hero", npc: false, room: 1, position: PosStanding, fighting: "", hp: 100, maxHP: 100, level: 10, ac: 10, thac0: 10}
		defender := &mockCombatant{name: "Orc", npc: true, room: 1, position: pos, hp: 100, maxHP: 100, ac: 10}

		ce := NewCombatEngine()
		if err := ce.StartCombat(attacker, defender); err != nil {
			t.Fatalf("StartCombat(pos=%d): %v", pos, err)
		}
		if defender.GetPosition() != pos {
			t.Errorf("defender position must be untouched at entry (C set_fighting is in damage(), fight.c:1443): pos %d became %d", pos, defender.GetPosition())
		}
		if defender.GetFighting() != "Hero" {
			t.Errorf("defender must be enrolled as fighting at entry: GetFighting()=%q", defender.GetFighting())
		}
		if attacker.GetPosition() != PosFighting {
			t.Errorf("attacker must stand at entry: pos %d", attacker.GetPosition())
		}
	}
}
