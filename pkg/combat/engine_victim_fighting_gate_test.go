package combat

import "testing"

// src/fight.c:1443-1445: damage() calls set_fighting only for a new victim.
// Exercise the shared damage point on both hit and miss, then a second swing
// after the victim sits during combat. Eager Go enrollment must not erase the
// distinction between the initial transition and an already-fighting victim.
func TestVictimStandRequiresNewCombatOnHitAndMiss(t *testing.T) {
	original := GetCallbacks()
	SetCallbacks(defaultCombatCallbacks())
	t.Cleanup(func() { SetCallbacks(original) })
	for _, hit := range []bool{false, true} {
		name := "miss"
		roll := 1
		if hit {
			name, roll = "hit", 20
		}
		for _, fighting := range []string{"", "Hero", "SomeoneElse"} {
			t.Run(name+"/fighting="+fighting, func(t *testing.T) {
				attacker := &mockCombatant{name: "Hero", room: 1, position: PosStanding, hp: 1000, maxHP: 1000, level: 10, thac0: 10}
				defender := &mockCombatant{name: "Victim", npc: true, room: 1, position: PosSitting, fighting: fighting, hp: 1000, maxHP: 1000, level: 10}
				ce := NewCombatEngine()
				if err := ce.StartCombat(attacker, defender); err != nil {
					t.Fatal(err)
				}
				WithRoller(NewScriptedRoller([]int{roll, 1, 1, 1}), func() {
					if err := ce.PerformInitialAttack(attacker, defender); err != nil {
						t.Fatal(err)
					}
				})
				want := PosSitting
				if fighting == "" {
					want = PosFighting
				}
				if got := defender.GetPosition(); got != want {
					t.Fatalf("position after %s = %d, want %d (initial fighting %q)", name, got, want, fighting)
				}
				defender.SetPosition(PosSitting)
				WithRoller(NewScriptedRoller([]int{roll, 1, 1, 1}), func() {
					if err := ce.PerformInitialAttack(attacker, defender); err != nil {
						t.Fatal(err)
					}
				})
				if got := defender.GetPosition(); got != PosSitting {
					t.Fatalf("second %s stood an already-fighting victim: %d", name, got)
				}
			})
		}
	}
}
