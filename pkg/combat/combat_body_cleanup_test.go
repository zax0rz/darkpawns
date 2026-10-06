package combat

import "testing"

func cleanupFighter(name string) *mockCombatant {
	return &mockCombatant{name: name, room: 1, hp: 100, maxHP: 100, level: 10, position: PosStanding}
}

func TestCombatBodyStopOneWay(t *testing.T) {
	ce := NewCombatEngine()
	a, b := cleanupFighter("A"), cleanupFighter("B")
	if err := ce.StartCombat(a, b); err != nil {
		t.Fatal(err)
	}
	ce.StopCombat(b)
	if b.GetFightingBody() != nil || a.GetFightingBody() != b {
		t.Fatal("ordinary stop erased the other body's fight")
	}
	if ce.IsFighting(b) {
		t.Fatal("stopped body still reports FIGHTING")
	}
}

func TestCombatBodyStopRetarget(t *testing.T) {
	for _, fled := range []bool{false, true} {
		t.Run(map[bool]string{false: "dead", true: "fled"}[fled], func(t *testing.T) {
			ce := NewCombatEngine()
			a, old, first, last := cleanupFighter("A"), cleanupFighter("old"), cleanupFighter("Guard"), cleanupFighter("Guard")
			for _, p := range [][2]*mockCombatant{{a, old}, {first, a}, {last, a}} {
				if err := ce.StartCombat(p[0], p[1]); err != nil {
					t.Fatal(err)
				}
			}
			if fled {
				old.room = 2
			} else {
				old.position = PosDead
				old.hp = -11
			}
			ce.StopCombat(a)
			if a.GetFightingBody() != last {
				t.Fatalf("retarget=%p want newest eligible duplicate %p", a.GetFightingBody(), last)
			}
			if len(ce.combatOrder) != 4 || ce.combatOrder[0] != last {
				t.Fatal("retarget changed combat-list order")
			}
			if ce.combatPairs[CombatPairKey{Attacker: a, Target: last}] == nil {
				t.Fatal("retarget left an obsolete pair")
			}
		})
	}
}

func TestCombatBodyRetirementIsolation(t *testing.T) {
	ce := NewCombatEngine()
	one, two, p, q := cleanupFighter("Guard"), cleanupFighter("Guard"), cleanupFighter("P"), cleanupFighter("Q")
	for _, pair := range [][2]*mockCombatant{{one, p}, {two, q}} {
		if err := ce.StartCombat(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	retire, ok := interface{}(ce).(interface{ RetireCombatant(Combatant) })
	if !ok {
		t.Fatal("missing full body retirement boundary")
	}
	retire.RetireCombatant(one)
	if one.GetFightingBody() != nil || p.GetFightingBody() != nil {
		t.Fatal("retired references retained")
	}
	if two.GetFightingBody() != q || q.GetFightingBody() != two || len(ce.combatOrder) != 2 || ce.combatOrder[0] != q {
		t.Fatal("retirement disturbed duplicate survivor/order")
	}
}
