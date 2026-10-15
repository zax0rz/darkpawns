package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1438 special callers: the no_get hand-strike (new_cmds2.c:561-569), the
// elements guardian's charmed-player hit (spec_procs3.c:1281) and the
// cityguard bribe refusal (mobprog.c:220-226) are all C hit() calls — the
// swing draws before damage()'s gates and enrols only on un-refused damage.
// The old plain enrolments drew nothing (no_get, the bribe) or enrolled
// eagerly (the guardian); every test here fails on those shapes (R5h).

func firstSwingMissSeed(t *testing.T) uint32 {
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

func replayOneSwing(seed uint32) int {
	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	return dprng.Number(0, 999)
}

func TestSpecNoGetHandStrikeSwings(t *testing.T) {
	for _, peaceful := range []bool{true, false} {
		w, ce := newRealPulseWorld(t, peaceful,
			parser.Mob{
				VNum: 9801, Keywords: "guard", ShortDesc: "the guard", Level: 20,
				HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 200}, ActionFlags: []string{"sentinel"},
			},
		)
		guard, err := w.SpawnMob(9801, 1001)
		if err != nil {
			t.Fatalf("SpawnMob: %v", err)
		}
		p := NewPlayer(1, "Picker", 1001)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		seed := firstSwingMissSeed(t)
		dprng.ResetStream(seed)
		if !specNoGet(w, p, guard, "get", "coin") {
			t.Fatal("specNoGet refused its command")
		}
		gotNext := dprng.Number(0, 999)
		if want := replayOneSwing(seed); gotNext != want {
			t.Fatalf("peaceful=%v: hand strike changed the draw stream: next=%d, want %d", peaceful, gotNext, want)
		}
		if peaceful && (ce.IsFighting(guard) || p.GetFightingBody() != nil) {
			t.Fatal("peaceful hand strike enrolled anyone")
		}
		if !peaceful && (!ce.IsFighting(guard) || p.GetFightingBody() != guard) {
			t.Fatal("open-room hand strike failed to enrol the pair")
		}
	}
}

func TestElementsGuardianPlayerHitSwings(t *testing.T) {
	for _, peaceful := range []bool{true, false} {
		w, ce := newRealPulseWorld(t, peaceful)
		charmed := NewPlayer(1, "Charmed", 1001)
		charmed.SetLevel(30)
		next := NewPlayer(2, "Bystander", 1001)
		next.SetLevel(30)
		if err := w.AddPlayer(charmed); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		if err := w.AddPlayer(next); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		seed := firstSwingMissSeed(t)
		dprng.ResetStream(seed)
		elementsGuardianPlayerHit(w, charmed, next)
		gotNext := dprng.Number(0, 999)
		if want := replayOneSwing(seed); gotNext != want {
			t.Fatalf("peaceful=%v: guardian strike changed the draw stream: next=%d, want %d", peaceful, gotNext, want)
		}
		if peaceful && (ce.IsFighting(charmed) || next.GetFightingBody() != nil) {
			t.Fatal("peaceful guardian strike enrolled anyone")
		}
		if !peaceful && (!ce.IsFighting(charmed) || next.GetFightingBody() != charmed) {
			t.Fatal("open-room guardian strike failed to enrol the pair")
		}
	}
}

func TestCityguardBribeRefusalSwings(t *testing.T) {
	for _, peaceful := range []bool{true, false} {
		w, ce := newRealPulseWorld(t, peaceful,
			parser.Mob{
				VNum: 2747, Keywords: "cityguard guard", ShortDesc: "the cityguard", Level: 33,
				HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 500},
			},
		)
		guard, err := w.SpawnMob(2747, 1001)
		if err != nil {
			t.Fatalf("SpawnMob: %v", err)
		}
		p := NewPlayer(1, "Briber", 1001)
		p.SetLevel(30)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		// The bribe arm draws number(0,2) first; pin a seed where it refuses
		// (roll != 0) and the swing then misses.
		var seed uint32
		for s := uint32(1); s < 8192; s++ {
			dprng.ResetStream(s)
			if dprng.Number(0, 2) == 0 {
				continue
			}
			dprng.ResetStream(s)
			dprng.Number(0, 2)
			if dprng.Number(1, 20) == 1 {
				seed = s
				break
			}
		}
		if seed == 0 {
			t.Fatal("no seed found with a refusing bribe and a missed swing")
		}

		dprng.ResetStream(seed)
		w.MpGive(guard, p, 10) // amount < 200 also forces the refusal arm
		gotNext := dprng.Number(0, 999)
		dprng.ResetStream(seed)
		dprng.Number(0, 2)
		dprng.Number(1, 20)
		wantNext := dprng.Number(0, 999)
		if gotNext != wantNext {
			t.Fatalf("peaceful=%v: bribe strike changed the draw stream: next=%d, want %d", peaceful, gotNext, wantNext)
		}
		if peaceful && (ce.IsFighting(guard) || p.GetFightingBody() != nil) {
			t.Fatal("peaceful bribe strike enrolled anyone")
		}
		if !peaceful && (!ce.IsFighting(guard) || p.GetFightingBody() != guard) {
			t.Fatal("open-room bribe strike failed to enrol the pair")
		}
	}
}
