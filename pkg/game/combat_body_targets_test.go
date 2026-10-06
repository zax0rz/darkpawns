package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCombatBodyDefaultTargets(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 3001, Keywords: "guard", ShortDesc: "a guard", Level: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	first, err := w.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	actor := NewPlayer(1, "Actor", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatal(err)
	}
	t.Run("player-default", func(t *testing.T) {
		// Each body must remain selected regardless of room map iteration order.
		for i := 0; i < 100; i++ {
			for _, want := range []*MobInstance{first, second} {
				actor.SetFightingBody(want)
				got, ok := w.ResolveFightingTarget(actor)
				if !ok || got.Combatant != want {
					t.Fatalf("default target=%p/%t, want selected body %p", got.Combatant, ok, want)
				}
			}
		}
	})
	t.Run("mobile-default", func(t *testing.T) {
		// The separate scripting Target field must not override FIGHTING.
		first.SetTarget(second)
		first.SetFightingBody(actor)
		if got := mobFightingTarget(w, first); got != actor {
			t.Fatalf("mobile default=%p, want stored fighting body %p", got, actor)
		}
	})
}

func TestCombatBodyWeaponCallback(t *testing.T) {
	w, armed, _ := zoneArmedMob(t)
	bare, err := w.SpawnMobQuiet(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	cb := w.WireCombatCallbacks()
	for i := 0; i < 100; i++ {
		kind, _, _, blessed := cb.GetWeaponInfo(armed)
		if kind != 3 || !blessed {
			t.Fatalf("armed duplicate weapon=%d/%t, want slash/blessed", kind, blessed)
		}
		kind, _, _, blessed = cb.GetWeaponInfo(bare)
		if kind != 0 || blessed {
			t.Fatalf("bare duplicate inherited weapon=%d/%t", kind, blessed)
		}
	}
}
