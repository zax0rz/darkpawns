package spells

import (
	"strings"
	"testing"
)

// DP-1446 witness: mag_points' lay-hands victim line
// ("Your wounds start to heal beneath $n's hands!", src/magic.c:1821) is a
// C act(..., TRUE, ch, 0, victim, TO_VICT), so the victim renders the
// healer through PERS. The port hand-pasted the real healer name; the
// bridged call instead routes through the game's SpellActToVictim bridge.
type layHandsBridgeWorld struct {
	act []string
}

func (w *layHandsBridgeWorld) SpellActToVictim(caster, victim interface{}, format string) bool {
	w.act = append(w.act, format)
	return true
}

func TestMagPointsLayHandsRoutesVictimLineThroughBridge(t *testing.T) {
	world := &layHandsBridgeWorld{}
	caster := &mockSpellsChar{name: "Paladin", hp: 100, maxHP: 100, roomVNum: 8105, class: 3}
	victim := &mockSpellsChar{name: "Buddy", hp: 1, maxHP: 100, roomVNum: 8105, class: 3}

	MagPoints(20, caster, victim, SpellLayHands, int(SaveSpell), world)

	if len(world.act) != 1 {
		t.Fatalf("bridge acts = %q, want the single TO_VICT lay-hands act", world.act)
	}
	got := world.act[0]
	if !strings.Contains(got, "$n") {
		t.Fatalf("bridged line lost its $n: %q", got)
	}
	if !strings.Contains(got, "heal beneath") {
		t.Fatalf("bridged line = %q, want the lay-hands act text", got)
	}
}
