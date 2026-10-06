package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type reviewBodyWorld struct {
	starts  int
	leaders map[combat.Combatant]combat.Combatant
}

func (w *reviewBodyWorld) StartCombat(a, b combat.Combatant) error           { w.starts++; return nil }
func (w *reviewBodyWorld) FollowingBody(b combat.Combatant) combat.Combatant { return w.leaders[b] }
func reviewBody(name string) *breathCombatant {
	return &breathCombatant{mockSpellsChar: mockSpellsChar{name: name, npc: true, level: 40, roomVNum: 100, position: int(PosStanding), hp: 100, maxHP: 100, inGroup: true}}
}

func TestCombatBodyBreathDuplicateEnrollment(t *testing.T) {
	a, b := reviewBody("dragon"), reviewBody("dragon")
	w := &reviewBodyWorld{}
	inflictDamage(a, b, 0, SpellFrostBreath, w)
	if w.starts != 1 {
		t.Fatalf("engine enrollments=%d want1", w.starts)
	}
	if a.GetFightingBody() != b || b.GetFightingBody() != a {
		t.Fatal("actual breath bodies not retained")
	}
}

func TestCombatBodyGroupDuplicateLeaders(t *testing.T) {
	a, b := reviewBody("leader"), reviewBody("leader")
	x, y := reviewBody("follower"), reviewBody("follower")
	w := &reviewBodyWorld{leaders: map[combat.Combatant]combat.Combatant{x: a, y: b}}
	if areGrouped(x, y, w) || areGrouped(a, b, w) {
		t.Fatal("separate duplicate leaders conflated")
	}
	if !areGrouped(a, x, w) || !areGrouped(x, a, w) {
		t.Fatal("actual leader/follower group lost")
	}
	w.leaders[y] = a
	if !areGrouped(x, y, w) {
		t.Fatal("actual shared leader group lost")
	}
}
