package combat

import "testing"

// TestStartCombat_StandsSleepingDefender pins C set_fighting's unconditional
// POS_FIGHTING (fight.c:223): engaging a sleeping victim stands BOTH sides at
// entry. The AWAKE attack gate and CalculateHitChance's awake-defender AC
// both read the defender's position, so a sleeping victim that is attacked
// retaliates on the next violence round instead of being stopped.
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
