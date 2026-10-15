package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// palmPeekWorld builds a bare two-room world with an immortal observer whose
// peek class gate passes and whose room holds one spawned trainee.
func palmPeekWorld(t *testing.T) (*World, *Player, *MobInstance) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 99061, Keywords: "guard trainee", ShortDesc: "a guard trainee",
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Peeker", 1001)
	p.SetLevel(LVL_IMMORT)
	p.SetClass(ClassThief)
	p.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(99061, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosStanding)
	return w, p, mob
}

// TestPalmEntryDrawPrecedesEveryGate pins DP-1436 (R3): do_palm consumes
// number(1,101) before its no-argument return and before the item and weight
// checks (src/new_cmds.c:2110-2122), so every refusal still burns the draw.
func TestPalmEntryDrawPrecedesEveryGate(t *testing.T) {
	w, p, _ := palmPeekWorld(t)
	seed := uint32(163)

	for _, tc := range []struct {
		name    string
		objName string
		want    string
	}{
		{name: "no argument", objName: "", want: "Palm what?\r\n"},
		{name: "nothing matches", objName: "nothing", want: "You don't see that here.\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dprng.ResetStream(seed)
			_ = dprng.Number(1, 101) // reference: the entry draw
			wantNext := dprng.Number(1, 101)

			dprng.ResetStream(seed)
			result := DoPalm(p, tc.objName, w)
			if result.MessageToCh != tc.want {
				t.Fatalf("palm bytes = %q, want %q", result.MessageToCh, tc.want)
			}
			if gotNext := dprng.Number(1, 101); gotNext != wantNext {
				t.Fatalf("next draw after palm = %d, want %d (the entry draw was skipped or doubled)", gotNext, wantNext)
			}
		})
	}
}

// TestPeekImmortalConsumesPercentDraw pins DP-1437 (R3): for a valid non-self
// target, do_peek consumes number(1,101) before its immortal-level test
// (src/act.other.c:1688-1692), so an immortal burns the same draw a mortal
// would. improve_skill's number(1,200) follows on the success path, so the
// whole command consumes exactly two draws here (bare mob target, no
// inventory-probe draws).
func TestPeekImmortalConsumesPercentDraw(t *testing.T) {
	w, p, mob := palmPeekWorld(t)
	seed := uint32(163)

	dprng.ResetStream(seed)
	_ = dprng.Number(1, 101) // peek's percent draw
	_ = dprng.Number(1, 200) // improve_skill's gate draw
	wantNext := dprng.Number(1, 101)

	dprng.ResetStream(seed)
	w.doPeek(p, nil, "peek", "trainee")
	if p.GetRoom() != 1001 || mob.GetRoom() != 1001 {
		t.Fatal("fixture moved")
	}
	if gotNext := dprng.Number(1, 101); gotNext != wantNext {
		t.Fatalf("next draw after immortal peek = %d, want %d (the percent draw was skipped or the tail drew differently)", gotNext, wantNext)
	}
}
