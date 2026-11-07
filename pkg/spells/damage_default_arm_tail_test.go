package spells

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestSpellDamageRunsDamageDefaultArm pins DP-1460. C's damage() runs its
// default arm for EVERY attack type — melee, a skill, a mob special, a spell
// (src/fight.c:1587-1614) — which prints "That really did HURT!" to a victim
// whose blow exceeded a quarter of its maximum hit points and takes the
// unconditional number(0,2) scream draw. The spell seam omitted the arm, so the
// draw was missing too; every later draw in the round then shifted, and the
// next perform_violence attack counts (src/fight.c:1904-1940) came out wrong —
// three attacks for one combatant in C against one in Go.
func TestSpellDamageRunsDamageDefaultArm(t *testing.T) {
	// The victim's maximum hit points are small so the spell's damage exceeds a
	// quarter of them, which is the arm's own gate.
	caster := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Caster", level: 31, class: 0, roomVNum: 100,
		position: int(PosFighting), hp: 200, maxHP: 200,
	}}
	victim := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Victim", level: 10, class: 0, roomVNum: 100,
		position: int(PosStanding), hp: 50, maxHP: 50,
	}}

	combat.WithRoller(combat.NewScriptedRoller([]int{4}), func() {
		if !CallMagic(caster, victim, nil, SpellMagicMissile, 31, CastScroll, nil) {
			t.Fatal("scroll cast did not resolve")
		}
	})

	if victim.GetHP() == 50 {
		t.Fatal("the spell dealt no damage; the fixture no longer reaches the arm")
	}
	found := false
	for _, message := range victim.messages {
		if strings.Contains(message, "That really did HURT!") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("victim did not receive damage()'s default arm; messages = %q", victim.messages)
	}
}
