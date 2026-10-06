package combat

import "testing"

// Value-backed and non-comparable adapters must be refused before map access.
type (
	valueBody struct{ *mockCombatant }
	sliceBody struct {
		*mockCombatant
		values []int
	}
)

func TestCombatBodyRejectsInvalidReferences(t *testing.T) {
	var typedNil *mockCombatant
	valid := &mockCombatant{name: "valid", position: PosStanding, hp: 100}
	for _, bad := range []Combatant{nil, typedNil, valueBody{valid}, sliceBody{valid, []int{1}}} {
		ce := NewCombatEngine()
		for _, entry := range []func(Combatant, Combatant) error{ce.StartCombat, ce.StartCombatAfterDamage, ce.PerformInitialAttack, ce.PerformUnenrolledInitialAttack} {
			if err := entry(bad, valid); err == nil {
				t.Fatalf("invalid %T attacker accepted", bad)
			}
			if err := entry(valid, bad); err == nil {
				t.Fatalf("invalid %T defender accepted", bad)
			}
		}
		ce.MarkParried(bad, "parry")
		ce.StopCombat(bad)
		if ce.IsFighting(bad) {
			t.Fatal("invalid body is fighting")
		}
		if _, ok := ce.GetCombatTarget(bad); ok {
			t.Fatal("invalid body acquired target")
		}
		if len(ce.combatPairs) != 0 || len(ce.combatOrder) != 0 || len(ce.parried) != 0 {
			t.Fatal("invalid body mutated engine")
		}
	}
}

func TestCombatBodyParriedIsolation(t *testing.T) {
	first := &mockCombatant{name: "a guard"}
	second := &mockCombatant{name: "a guard"}
	ce := NewCombatEngine()
	ce.MarkParried(first, "parry")
	if got := ce.consumeParried(second); got != "" {
		t.Fatalf("other same-description body consumed %q", got)
	}
	if got := ce.consumeParried(first); got != "parry" {
		t.Fatalf("actual body lost parry: %q", got)
	}
}
