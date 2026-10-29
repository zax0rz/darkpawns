package combat

import "testing"

// TestPerformOneHit_MissRunsDamageTail — C's miss path is not a short circuit:
// hit() calls damage(ch, victim, 0, w_type), and damage() runs its whole tail
// anyway (src/fight.c:1568-1614) — update_pos, the position-state bytes, the
// !AWAKE stop_fighting, and the default arm's bleed check, which is where a
// MOB_WIMPY victim flees (fight.c:1588-1601). The port returned from the miss
// branch before that tail, so a missed swing neither moved a wimpy victim nor
// burned do_flee's six direction rolls (R1/R3).
func TestPerformOneHit_MissRunsDamageTail(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	fled := false
	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(Combatant) (wType, damDice, damSize int, isBlessed bool) { return 0, 0, 0, false }
	cb.HasMobFlag = func(_ Combatant, flag string) bool { return flag == "MOB_WIMPY" }
	cb.DoFlee = func(Combatant) { fled = true }
	SetCallbacks(cb)

	engine := NewCombatEngine()
	defer engine.Stop()
	engine.MessageFunc = func(Combatant, Combatant, int, int) bool { return true }
	engine.BroadcastFunc = func(int, string, []Combatant) {}

	attacker := &mockCombatant{
		name: "Attacker", npc: true, room: 100, level: 8,
		hp: 100, maxHP: 100, ac: 0, thac0: 1,
		damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
	}
	// HP below max/4 is the bleed gate damage() tests before the wimpy flee.
	defender := &mockCombatant{
		name: "Wimpy Wolf", npc: true, room: 100, level: 8,
		hp: 10, maxHP: 100, ac: 0, thac0: 20,
		damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
	}

	// Natural 1 against an awake defender always misses.
	WithRoller(NewScriptedRoller([]int{1}), func() {
		engine.performOneHit(&CombatPair{Attacker: attacker, Defender: defender})
	})

	if !fled {
		t.Fatal("a miss did not run damage()'s default arm: the victim's MOB_WIMPY " +
			"flee (and do_flee's six direction draws) never ran (src/fight.c:1588-1601)")
	}
}
