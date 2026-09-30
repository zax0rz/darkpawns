package combat

import "testing"

// The brace at src/fight.c:1442 ends the attacker's position gate.
// Victim enrollment at :1443 is independent and still requires its own gate.
func TestDamageVictimEnrollmentIndependentOfAttackerPosition(t *testing.T) {
	for _, seam := range []string{"plain", "full"} {
		t.Run(seam, func(t *testing.T) {
			old := GetCallbacks()
			t.Cleanup(func() { SetCallbacks(old) })
			SetCallbacks(&GameCallbacks{})
			attacker := &mockCombatant{name: "Attacker", hp: 100, maxHP: 100, position: PosStunned, level: 10}
			victim := &mockCombatant{name: "Victim", hp: 500, maxHP: 500, position: PosStanding, level: 10, npc: true}
			if seam == "plain" {
				EnterDamageFighting(attacker, victim)
			} else {
				TakeDamageAfterGate(attacker, victim, 0, -1, nil)
			}
			if attacker.GetFighting() != "" || attacker.GetPosition() != PosStunned {
				t.Fatal("stunned attacker was enrolled or stood")
			}
			if victim.GetFighting() != attacker.GetName() || victim.GetPosition() != PosFighting {
				t.Fatal("fresh victim lost its independent C enrollment")
			}
			engine := NewCombatEngine()
			if err := engine.StartCombatAfterDamage(attacker, victim); err != nil {
				t.Fatal(err)
			}
			if len(engine.combatOrder) != 1 || engine.combatOrder[0] != victim {
				t.Fatal("post-damage engine did not enroll only the fresh victim")
			}
			if attacker.GetFighting() != "" || attacker.GetPosition() != PosStunned {
				t.Fatal("post-damage engine re-enrolled the stunned attacker")
			}
		})
	}
}
