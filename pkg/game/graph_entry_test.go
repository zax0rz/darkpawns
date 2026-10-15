package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1438 hunter arm: hunt_victim's arrival attack is
// do_follow(ch, ...) then hit(ch, HUNTING(ch), TYPE_UNDEFINED)
// (src/graph.c:312, 340-345). The swing draws before damage()'s gates; a
// peaceful room refuses it and enrols nobody. The old path set both FIGHTING
// fields by hand and applied the placeholder fixed-damage Attack stub — zero
// draws, both enrolled, 10 flat damage — so both tests fail on that shape
// (R5h).
func huntSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 8192; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 20) == 1 {
			return seed
		}
	}
	t.Fatal("no seed found whose first number(1,20) is 1")
	return 0
}

func TestAttackHuntingPreyPeacefulRoomRefusesAndDraws(t *testing.T) {
	w, ce := newRealPulseWorld(t, true,
		parser.Mob{
			VNum: 9701, Keywords: "hound", ShortDesc: "the hound", Level: 20,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 200}, ActionFlags: []string{"sentinel"},
		},
	)
	hound, err := w.SpawnMob(9701, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	p := NewPlayer(1, "Quarry", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	hp := p.GetHP()

	seed := huntSeed(t)
	dprng.ResetStream(seed)
	w.attackHuntingPrey(hound, p)
	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	if want := dprng.Number(0, 999); gotNext != want {
		t.Fatalf("peaceful hunt strike changed the draw stream: next=%d, want %d (one number(1,20) consumed)", gotNext, want)
	}
	if ce.IsFighting(hound) || p.GetFightingBody() != nil {
		t.Fatal("peaceful hunt strike enrolled anyone")
	}
	if p.GetHP() != hp {
		t.Fatalf("peaceful hunt strike damaged the quarry: hp %d, want %d", p.GetHP(), hp)
	}
}

func TestAttackHuntingPreyNormalRoomSwingsAndEnrols(t *testing.T) {
	w, ce := newRealPulseWorld(t, false,
		parser.Mob{
			VNum: 9702, Keywords: "hound", ShortDesc: "the hound", Level: 20,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 200}, ActionFlags: []string{"sentinel"},
		},
	)
	hound, err := w.SpawnMob(9702, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	p := NewPlayer(1, "Quarry", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	seed := huntSeed(t)
	dprng.ResetStream(seed)
	w.attackHuntingPrey(hound, p)
	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	if want := dprng.Number(0, 999); gotNext != want {
		t.Fatalf("hunt strike changed the draw stream: next=%d, want %d", gotNext, want)
	}
	if !ce.IsFighting(hound) || p.GetFightingBody() != hound {
		t.Fatal("hunt strike failed to enrol the pair")
	}
}
