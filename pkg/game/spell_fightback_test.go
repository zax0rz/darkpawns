package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSpellDamageEnrollsPlayerVictimInCombat pins DP-1481 shape 2. C's
// damage() calls set_fighting on both sides before applying the amount
// (src/fight.c:1403-1452), and set_fighting is what puts them on combat_list
// for violence_update; a damage spell that only copies the FIGHTING fields
// leaves the fight unattached, so the victim never takes a turn. The staff,
// wand, scroll and directly cast spell seams all reach damage() this way.
func TestSpellDamageEnrollsPlayerVictimInCombat(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Arena"}}})
	if err != nil {
		t.Fatal(err)
	}
	ce := combat.NewCombatEngine()
	w.SetCombatEngine(ce)
	old := combat.GetCallbacks()
	t.Cleanup(func() {
		combat.SetCallbacks(old)
		ce.Stop()
		w.StopAITicker()
	})
	combat.SetCallbacks(w.WireCombatCallbacks())

	caster := NewPlayer(1, "Caster", 1001)
	caster.SetLevel(30)
	victim := NewPlayer(2, "Victim", 1001)
	victim.SetLevel(30)
	for _, p := range []*Player{caster, victim} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}

	if !w.DoSpellDamage(caster, victim, 10, "burning hands") {
		t.Fatal("spell damage was not applied")
	}
	if got := victim.GetFightingBody(); got != combat.Combatant(caster) {
		t.Fatalf("victim's FIGHTING field = %v, want the caster", got)
	}
	if !ce.Enrolled(victim) {
		t.Error("victim is not enrolled in the fight engine: no round can ever run")
	}
}
