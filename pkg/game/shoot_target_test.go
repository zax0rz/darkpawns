package game

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShootTargetCMatcherAndRoomOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 2001, Keywords: "guard hidden", ShortDesc: "a hidden guard", AffectFlags: []string{"INVISIBLE"}}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	first, err := w.SpawnMobQuiet(2001, 1002)
	if err != nil {
		t.Fatal(err)
	}
	pc := NewPlayer(1, "Guardplayer", 1002)
	if err := w.AddPlayer(pc); err != nil {
		t.Fatal(err)
	}
	last, err := w.SpawnMobQuiet(2001, 1002)
	if err != nil {
		t.Fatal(err)
	}
	viewer := NewPlayer(2, "Viewer", 1001)
	viewer.SetLevel(1)
	if canSee(viewer, asActor(first)) {
		t.Fatal("invisible fixture is visible")
	}
	for _, tc := range []struct {
		name string
		want combat.Combatant
	}{{"gu", last}, {"2.gu", pc}, {"3.gu", first}, {"hidden", last}, {"2.hidden", first}, {"nobody", last}, {"0.Guardplayer", last}, {"me", last}, {"self", last}, {"99.guard", last}, {"x.guard", last}, {"", last}} {
		if got := w.ResolveShootTarget(1002, tc.name); got != tc.want {
			t.Errorf("%q picked %v want %v", tc.name, got, tc.want)
		}
	}
	if got := w.ResolveShootTarget(1001, "guard"); got != nil {
		t.Fatal("empty room target", got)
	}
	// A real room move prepends the older body, rather than sorting its name or ID.
	if err := w.PlayerTransfer(pc, 1001); err != nil {
		t.Fatal(err)
	}
	if err := w.PlayerTransfer(pc, 1002); err != nil {
		t.Fatal(err)
	}
	if got := w.ResolveShootTarget(1002, "nobody"); got != pc {
		t.Fatal("fallback did not follow latest room arrival")
	}
	if got := w.ResolveShootTarget(1002, "2.gu"); got != last {
		t.Fatal("ordinal did not merge PCs and NPCs in room order")
	}
}

func TestShootTargetConcurrentMovement(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	p := NewPlayer(1, "Traveler", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			if err := w.PlayerTransfer(p, 1001+i%2); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			for room := 1001; room <= 1002; room++ {
				if got := w.ResolveShootTarget(room, "Traveler"); got != nil && got != p {
					t.Error("unexpected body", got)
					return
				}
			}
		}
	}()
	workers.Wait()
}
