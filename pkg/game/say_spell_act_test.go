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

func TestSpellActToVictimRendersCasterThroughPers(t *testing.T) {
	w, caster, target, _, out := newChannelWorld(t)
	caster.SetAffect(affInvisible, true)

	if !w.SpellActToVictim(caster, target, "Isn't $n just such a nice fellow?") {
		t.Fatal("spell act did not reach the live caster and victim")
	}
	if got, want := outputOf(out, target.Name), "Isn't someone just such a nice fellow?\r\n"; got != want {
		t.Fatalf("victim spell act = %q, want %q", got, want)
	}
}

func TestSpellAddFollowerRendersCFollowerAudiences(t *testing.T) {
	w, leader, follower, _, out := newChannelWorld(t)
	observer := channelPlayer(t, w, 4, "Observer", 1001)

	if !w.AddFollowerForSpell(follower, leader) {
		t.Fatal("spell follower link was not applied to live players")
	}
	if w.FollowingBody(follower) != leader {
		t.Fatal("spell follower link does not point to the caster")
	}
	if got, want := outputOf(out, follower.Name), "You now follow Actor.\r\n"; got != want {
		t.Fatalf("follower output = %q, want %q", got, want)
	}
	if got, want := outputOf(out, leader.Name), "Local starts following you.\r\n"; got != want {
		t.Fatalf("leader output = %q, want %q", got, want)
	}
	if got, want := outputOf(out, observer.Name), "Local starts to follow Actor.\r\n"; got != want {
		t.Fatalf("room output = %q, want %q", got, want)
	}
}

// mag_points' heal line is act(..., TRUE, ch, 0, victim, TO_NOTVICT)
// (src/magic.c:1769): a listener who cannot see the caster hears nothing, and
// $N is PERS(victim). The old path pasted the victim's name for everyone.
func TestMendRoomLineRendersThroughAct(t *testing.T) {
	t.Run("seeing listener", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		target := channelPlayer(t, w, 4, "Patient", 1001)
		spells.MagPoints(10, caster, target, spells.SpellCureLight, 0, w)
		if got, want := outputOf(out, local.Name), "Patient's wounds glow and mend themselves.\r\n"; got != want {
			t.Fatalf("listener = %q, want %q", got, want)
		}
	})
	t.Run("blind listener hears nothing", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		target := channelPlayer(t, w, 4, "Patient", 1001)
		local.SetAffect(affBlind, true)
		spells.MagPoints(10, caster, target, spells.SpellCureLight, 0, w)
		if got := outputOf(out, local.Name); got != "" {
			t.Fatalf("blind listener = %q, want nothing (C: hide_invisible)", got)
		}
	})
	t.Run("invisible patient is someone", func(t *testing.T) {
		w, caster, local, _, out := newChannelWorld(t)
		target := channelPlayer(t, w, 4, "Patient", 1001)
		target.SetAffect(affInvisible, true)
		spells.MagPoints(10, caster, target, spells.SpellCureLight, 0, w)
		if got, want := outputOf(out, local.Name), "Someone's wounds glow and mend themselves.\r\n"; got != want {
			t.Fatalf("listener = %q, want %q", got, want)
		}
	})
}
