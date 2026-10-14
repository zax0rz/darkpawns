package combat

import "testing"

// DP-1438: a first hit() enrols inside damage(), not before it. C hit()
// consumes number(1,20) — and the damage dice on a hit — before damage()'s
// protection block reads either FIGHTING field; set_fighting for both sides
// happens at the damage point and only when the damage is not refused
// (src/fight.c:1318-1368, 1404-1445). R5h: against the old
// StartCombat+PerformInitialAttack shape these assertions fail — enrolment
// precedes the gate, and a command preflight can skip the draws entirely.
func TestDeferredInitialAttackEnrollmentAndProtections(t *testing.T) {
	for _, protect := range []bool{false, true} {
		for _, position := range []int{PosSleeping, PosStanding} {
			t.Run(string(rune('a'+position))+map[bool]string{false: "open", true: "protected"}[protect], func(t *testing.T) {
				oldCB, oldRoller := GetCallbacks(), GetRoller()
				t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
				attacker := &mockCombatant{name: "Hero", room: 1, level: 20, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14, damageRoll: DiceRoll{Num: 1, Sides: 2}}
				victim := &mockCombatant{name: "Mob", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: position, dex: 14, str: 14}
				cb := &GameCallbacks{DamageRefused: func(a, b Combatant) bool {
					if a.GetFighting() != "" || b.GetFighting() != "" {
						t.Fatal("protection read eager enrollment")
					}
					if b.GetPosition() != position {
						t.Fatal("victim stood before protection")
					}
					return protect
				}}
				SetCallbacks(cb)
				roller := NewScriptedRoller([]int{10, 2, 2, 2, 2})
				SetRoller(roller)
				ce := NewCombatEngine()
				swings := 0
				ce.MessageFunc = func(a, b Combatant, dam, typ int) bool { swings++; return true }
				if err := ce.PerformDeferredInitialAttack(attacker, victim); err != nil {
					t.Fatal(err)
				}
				if roller.Index == 0 {
					t.Fatal("protected attack skipped hit draws")
				}
				if protect {
					if swings != 0 || len(ce.combatPairs) != 0 || len(ce.combatOrder) != 0 || attacker.GetFighting() != "" || victim.GetFighting() != "" {
						t.Fatal("protection created combat")
					}
					return
				}
				if swings != 1 || len(ce.combatPairs) != 1 || len(ce.combatOrder) != 2 || ce.combatOrder[0] != victim || ce.combatOrder[1] != attacker || attacker.GetFighting() != victim.name || victim.GetFighting() != attacker.name {
					t.Fatalf("direct swing/enrollment = %d/%d/%d, fighting %q/%q", swings, len(ce.combatPairs), len(ce.combatOrder), attacker.GetFighting(), victim.GetFighting())
				}
				if victim.position != PosFighting || attacker.position != PosFighting {
					t.Fatal("damage did not stand enrolled fighters")
				}
				if ce.findFightingTarget(victim) != attacker || ce.findFightingTarget(attacker) != victim {
					t.Fatal("next pulse lost a fighter")
				}
			})
		}
	}
}

// damage()'s `if (!(FIGHTING(ch))) set_fighting(ch, victim)` (fight.c:1408)
// keeps an already-fighting attacker on its current target; the deferred
// initial hit must swing without re-enrolling or aborting.
func TestDeferredInitialAttackKeepsExistingTarget(t *testing.T) {
	oldCB, oldRoller := GetCallbacks(), GetRoller()
	t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
	attacker := &mockCombatant{name: "Mob", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14, damageRoll: DiceRoll{Num: 1, Sides: 2}}
	other := &mockCombatant{name: "Rival", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14}
	victim := &mockCombatant{name: "Courier", npc: true, room: 1, level: 5, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14}
	SetCallbacks(&GameCallbacks{})
	SetRoller(fixedRoller{number: 20}) // Natural 20 hits.
	ce := NewCombatEngine()
	swings := 0
	ce.MessageFunc = func(a, b Combatant, dam, typ int) bool { swings++; return true }
	if err := ce.StartCombat(attacker, other); err != nil {
		t.Fatal(err)
	}
	if err := ce.PerformDeferredInitialAttack(attacker, victim); err != nil {
		t.Fatal(err)
	}
	if swings != 1 {
		t.Fatalf("already-fighting attacker swung %d times, want 1", swings)
	}
	if attacker.GetFighting() != other.name {
		t.Fatalf("attacker re-enrolled: fighting %q, want %q", attacker.GetFighting(), other.name)
	}
	if victim.GetFighting() != attacker.name {
		t.Fatal("victim not enrolled by the damage")
	}
}
