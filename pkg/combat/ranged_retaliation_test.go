package combat

import "testing"

func TestRangedRetaliationEnrollmentAndProtections(t *testing.T) {
	for _, protect := range []bool{false, true} {
		for _, position := range []int{PosSleeping, PosStanding} {
			t.Run(string(rune('a'+position))+map[bool]string{false: "open", true: "protected"}[protect], func(t *testing.T) {
				oldCB, oldRoller := GetCallbacks(), GetRoller()
				t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
				attacker := &mockCombatant{name: "Mob", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14, damageRoll: DiceRoll{Num: 1, Sides: 2}}
				victim := &mockCombatant{name: "Shooter", room: 1, level: 20, hp: 1000, maxHP: 1000, position: position, dex: 14, str: 14}
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
				if err := ce.PerformRangedRetaliation(attacker, victim); err != nil {
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

func TestRangedRetaliationWoundedMobAndJailOrder(t *testing.T) {
	oldCB, oldRoller := GetCallbacks(), GetRoller()
	t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
	for _, jail := range []bool{false, true} {
		attacker := &mockCombatant{name: "Mob", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: PosStunned, dex: 14, str: 14}
		if jail {
			attacker.position = PosStanding
		}
		victim := &mockCombatant{name: "Shooter", room: 1, level: 20, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14}
		SetRoller(fixedRoller{number: 10})
		SetCallbacks(&GameCallbacks{MobHasJailGuardSpec: func(Combatant) bool { return jail }, JailGuardSubdue: func(a, b Combatant) bool {
			if attacker.fighting != nil || victim.fighting != nil {
				t.Fatal("jail redirect saw premature enrollment")
			}
			return true
		}})
		ce := NewCombatEngine()
		ce.MessageFunc = func(Combatant, Combatant, int, int) bool { return true }
		if err := ce.PerformRangedRetaliation(attacker, victim); err != nil {
			t.Fatal(err)
		}
		if jail {
			if len(ce.combatOrder) != 0 {
				t.Fatal("jail subdue enrolled combat")
			}
			continue
		}
		if attacker.fighting != nil || victim.fighting != attacker || len(ce.combatOrder) != 1 || ce.combatOrder[0] != victim || ce.findFightingTarget(victim) != attacker {
			t.Fatal("wounded mob retaliation lost awake victim's next turn")
		}
	}
}

func TestRangedRetaliationReadsSleepingPostureBeforeDamage(t *testing.T) {
	oldCB, oldRoller := GetCallbacks(), GetRoller()
	t.Cleanup(func() { SetCallbacks(oldCB); SetRoller(oldRoller) })
	SetCallbacks(&GameCallbacks{})
	deltas := map[int]int{}
	for _, position := range []int{PosStanding, PosSleeping} {
		attacker := &mockCombatant{name: "Mob", npc: true, room: 1, level: 10, hp: 1000, maxHP: 1000, position: PosStanding, dex: 14, str: 14, damageRoll: DiceRoll{Num: 1, Sides: 2, Plus: 10}}
		victim := &mockCombatant{name: "Shooter", room: 1, level: 20, hp: 1000, maxHP: 1000, position: position, ac: 100, dex: 14, str: 14}
		SetRoller(fixedRoller{number: 20}) // Natural 20 hits either posture.
		ce := NewCombatEngine()
		ce.MessageFunc = func(Combatant, Combatant, int, int) bool { return true }
		if err := ce.PerformRangedRetaliation(attacker, victim); err != nil {
			t.Fatal(err)
		}
		deltas[position] = 1000 - victim.hp
	}
	if deltas[PosStanding] <= 0 || deltas[PosSleeping] != 2*deltas[PosStanding] {
		t.Fatalf("pre-fight posture damage standing/sleeping=%d/%d", deltas[PosStanding], deltas[PosSleeping])
	}
}
