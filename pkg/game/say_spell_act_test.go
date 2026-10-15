package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// say_spell renders each listener's line through perform_act
// (src/spell_parser.c:325-343): PERS hides a caster the listener cannot see,
// the target gets its own TO_VICT line, and a sleeping listener hears nothing.
// The old path pasted the caster's real name for everyone.
func TestSaySpellRendersThroughPerformAct(t *testing.T) {
	t.Run("visible caster, class split", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		caster.Class = ClassMageUser
		local.Class = ClassMageUser
		spells.SaySpell(caster, spells.SpellArmor, nil, nil, w)
		if got, want := outputOf(out, local.Name), "Actor utters the words, 'holy ward'.\r\n"; got != want {
			t.Fatalf("same-class listener = %q, want %q", got, want)
		}
	})
	t.Run("invisible caster is someone", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		caster.Class = ClassMageUser
		local.Class = ClassMageUser
		caster.SetAffect(affInvisible, true)
		spells.SaySpell(caster, spells.SpellArmor, nil, nil, w)
		if got, want := outputOf(out, local.Name), "Someone utters the words, 'holy ward'.\r\n"; got != want {
			t.Fatalf("listener of an invisible caster = %q, want %q", got, want)
		}
	})
	t.Run("blind listener sees someone", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		caster.Class = ClassMageUser
		local.Class = ClassMageUser
		local.SetAffect(affBlind, true)
		spells.SaySpell(caster, spells.SpellArmor, nil, nil, w)
		if got, want := outputOf(out, local.Name), "Someone utters the words, 'holy ward'.\r\n"; got != want {
			t.Fatalf("blind listener = %q, want %q", got, want)
		}
	})
	t.Run("sleeping listener hears nothing", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		local.SetPosition(combat.PosSleeping)
		spells.SaySpell(caster, spells.SpellArmor, nil, nil, w)
		if got := outputOf(out, local.Name); got != "" {
			t.Fatalf("sleeping listener = %q, want nothing (C: !AWAKE)", got)
		}
	})
	t.Run("target gets its own line, not the room line", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		caster.Class = ClassMageUser
		local.Class = ClassMageUser
		spells.SaySpell(caster, spells.SpellArmor, local, nil, w)
		if got, want := outputOf(out, local.Name), "Actor stares at you and utters the words, 'holy ward'.\r\n"; got != want {
			t.Fatalf("target = %q, want %q", got, want)
		}
	})
}
