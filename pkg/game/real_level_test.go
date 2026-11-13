package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRealLevel(t *testing.T) {
	w, mortal, imm, _, _ := newChannelWorld(t)
	mortal.SetLevel(5)
	imm.SetLevel(LVL_IMPL)

	mob := NewMob(&parser.Mob{VNum: 100, Keywords: "guard", ShortDesc: "a guard", Level: 10}, 1001)
	mob.world = w
	w.activeMobs[mob.GetID()] = mob

	if got := realLevel(mortal, w); got != 5 {
		t.Fatalf("unswitched player realLevel = %d, want 5", got)
	}
	if got := realLevel(mob, w); got != 10 {
		t.Fatalf("unswitched mob realLevel = %d, want 10", got)
	}

	// Mob switch: MobileDescriptorOwner maps mob to immortal.
	w.MobileDescriptorOwner = func(m *MobInstance) *Player {
		if m == mob {
			return imm
		}
		return nil
	}
	if got := realLevel(mob, w); got != LVL_IMPL {
		t.Fatalf("mob-switched realLevel = %d, want %d", got, LVL_IMPL)
	}

	// PC switch: SetSwitchedOriginal maps borrowed player to immortal.
	mortal.SetSwitchedOriginal(imm)
	if got := realLevel(mortal, w); got != LVL_IMPL {
		t.Fatalf("PC-switched realLevel = %d, want %d", got, LVL_IMPL)
	}

	// After return: link cleared, falls back to mortal's own level.
	mortal.SetSwitchedOriginal(nil)
	if got := realLevel(mortal, w); got != 5 {
		t.Fatalf("post-return player realLevel = %d, want 5", got)
	}
}

func TestVisibilitySitesUseRealLevel(t *testing.T) {
	t.Run("Act PERS for PC-switched and mob-switched observer", func(t *testing.T) {
		w, mortal, imm, _, out := newChannelWorld(t)
		mortal.SetLevel(5)
		imm.SetLevel(LVL_IMPL)
		invisActor := channelPlayer(t, w, 4, "Shadow", 1001)
		invisActor.SetLevel(LVL_GRGOD)
		invisActor.SetInvisLevel(LVL_GRGOD)

		// Before switch, mortal cannot see wizinvis actor.
		Act(w, false, invisActor, nil, nil, nil, "$n waves.", "", ToRoom)
		if got, want := outputOf(out, mortal.Name), "Someone waves.\r\n"; got != want {
			t.Fatalf("unswitched mortal saw %q, want %q", got, want)
		}
		out[mortal.Name].Reset()

		// PC-switched mortal uses immortal's real level.
		mortal.SetSwitchedOriginal(imm)
		Act(w, false, invisActor, nil, nil, nil, "$n waves.", "", ToRoom)
		if got, want := outputOf(out, mortal.Name), "Shadow waves.\r\n"; got != want {
			t.Fatalf("PC-switched mortal saw %q, want %q", got, want)
		}
		out[mortal.Name].Reset()

		// After return, mortal sees Someone again.
		mortal.SetSwitchedOriginal(nil)
		Act(w, false, invisActor, nil, nil, nil, "$n waves.", "", ToRoom)
		if got, want := outputOf(out, mortal.Name), "Someone waves.\r\n"; got != want {
			t.Fatalf("returned mortal saw %q, want %q", got, want)
		}

		// Mob-switched observer routes through MobileDescriptorOwner.
		mob := NewMob(&parser.Mob{VNum: 101, Keywords: "imp", ShortDesc: "an imp", Level: 3}, 1001)
		mob.world = w
		w.activeMobs[mob.GetID()] = mob
		w.MobileDescriptorOwner = func(m *MobInstance) *Player {
			if m == mob {
				return imm
			}
			return nil
		}
		if !CanSee(mob, invisActor) {
			t.Fatal("mob-switched immortal should see wizinvis actor below real level")
		}
	})

	t.Run("combatCanSee", func(t *testing.T) {
		w, mortal, imm, _, _ := newChannelWorld(t)
		mortal.SetLevel(5)
		imm.SetLevel(LVL_IMPL)
		invisTarget := channelPlayer(t, w, 4, "Shadow", 1001)
		invisTarget.SetLevel(LVL_GRGOD)
		invisTarget.SetInvisLevel(LVL_GRGOD)

		if w.combatCanSee(mortal, invisTarget) {
			t.Fatal("unswitched mortal should not combatCanSee wizinvis target")
		}
		mortal.SetSwitchedOriginal(imm)
		if !w.combatCanSee(mortal, invisTarget) {
			t.Fatal("PC-switched immortal should combatCanSee wizinvis target")
		}
		mortal.SetSwitchedOriginal(nil)
		if w.combatCanSee(mortal, invisTarget) {
			t.Fatal("returned mortal should not combatCanSee wizinvis target")
		}
	})

	t.Run("tattooCanSee", func(t *testing.T) {
		w, mortal, imm, _, _ := newChannelWorld(t)
		mortal.SetLevel(5)
		imm.SetLevel(LVL_IMPL)
		invisCaster := channelPlayer(t, w, 4, "Shadow", 1001)
		invisCaster.SetLevel(LVL_GRGOD)
		invisCaster.SetInvisLevel(LVL_GRGOD)

		if w.tattooCanSee(mortal, invisCaster) {
			t.Fatal("unswitched mortal should not tattooCanSee wizinvis caster")
		}
		mortal.SetSwitchedOriginal(imm)
		if !w.tattooCanSee(mortal, invisCaster) {
			t.Fatal("PC-switched immortal should tattooCanSee wizinvis caster")
		}
		mortal.SetSwitchedOriginal(nil)
	})

	t.Run("listsHiddenBody", func(t *testing.T) {
		w, mortal, imm, _, _ := newChannelWorld(t)
		mortal.SetLevel(5)
		mortal.SetHolyLight(true)
		imm.SetLevel(LVL_IMPL)
		hiddenPeer := channelPlayer(t, w, 4, "Shadow", 1001)
		hiddenPeer.SetLevel(LVL_GRGOD)
		hiddenPeer.SetInvisLevel(LVL_GRGOD)
		hiddenPeer.SetAffect(affHide, true)

		if listsHiddenBody(mortal, hiddenPeer) {
			t.Fatal("unswitched mortal should not list hidden wizinvis peer")
		}
		mortal.SetSwitchedOriginal(imm)
		if !listsHiddenBody(mortal, hiddenPeer) {
			t.Fatal("PC-switched immortal with holylight should list hidden wizinvis peer")
		}
		mortal.SetSwitchedOriginal(nil)
		if listsHiddenBody(mortal, hiddenPeer) {
			t.Fatal("returned mortal should not list hidden wizinvis peer")
		}
	})
}

// A subject without wizinvis passes C's GET_REAL_LEVEL(sub) >= GET_INVIS_LEV(obj)
// test for every observer, since levels are never negative. The visibility
// sites skip the switch lookup then, so an ordinary mob looking at an ordinary
// player never scans the session table under the manager lock.
func TestVisibilitySkipsSwitchLookupWithoutInvisLevel(t *testing.T) {
	w, mortal, _, _, _ := newChannelWorld(t)
	mortal.SetLevel(5)
	mob := NewMob(&parser.Mob{VNum: 100, Keywords: "guard", ShortDesc: "a guard", Level: 10}, 1001)
	mob.world = w
	w.activeMobs[mob.GetID()] = mob
	lookups := 0
	w.MobileDescriptorOwner = func(*MobInstance) *Player { lookups++; return nil }

	_ = canSeeWithHide(mob, mortal, false)
	_ = w.combatCanSee(mob, mortal)
	if lookups != 0 {
		t.Fatalf("switch lookups for a subject with no invis level = %d, want 0", lookups)
	}

	mortal.SetInvisLevel(LVL_IMMORT)
	_ = canSeeWithHide(mob, mortal, false)
	if lookups == 0 {
		t.Fatal("a wizinvis subject must still resolve the observer's real level")
	}
}
