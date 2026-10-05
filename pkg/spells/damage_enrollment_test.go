package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSpellDamagePreservesWoundedVictimStop(t *testing.T) {
	for _, wounded := range []bool{false, true} {
		name := "fresh"
		hp := 500
		if wounded {
			name = "wounded"
			hp = 10
		}
		t.Run(name, func(t *testing.T) {
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			combat.SetCallbacks(&combat.GameCallbacks{})
			ch := &spellCombatant{name: "Caster", level: 10, hp: 100, maxHP: 100, pos: combat.PosStanding}
			victim := &spellCombatant{name: "Victim", npc: true, level: 10, hp: hp, maxHP: 500, pos: combat.PosStanding}
			if !inflictDamage(ch, victim, 16, 5, nil) {
				t.Fatal("damage did not execute")
			}
			if ch.GetFighting() != victim.name {
				t.Fatal("C damage() did not enroll caster")
			}
			if wounded {
				if victim.GetFighting() != "" || victim.pos != combat.PosMortally {
					t.Fatal("spell re-enrolled unconscious victim")
				}
			} else if victim.GetFighting() != ch.name {
				t.Fatal("spell did not enroll fresh victim")
			}
		})
	}
}
