package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C's read_mobile(nr, REAL) rolls the mobile's hit dice and then its gold
// variance (src/db.c:1748-1773). do_checkload and check_load call it for a
// throwaway instance, so a caller that reads the prototype instead loses those
// draws and desynchronizes the shared stream (R3a). The checkload-depth DRAW row
// is the incident: `checkload mob 16303` drew Dice(1,5) + Number(0,1) +
// Number(1,20) in C and nothing in Go.
func TestReadDisposableMobRollsReadMobileDraws(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room A", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      200,
			ShortDesc: "a test mob",
			HP:        parser.DiceRoll{Num: 1, Sides: 5, Plus: 20},
			Gold:      100,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	const seed = 1
	dprng.ResetStream(seed)
	mob, ok := w.ReadDisposableMob(200)
	if !ok || mob == nil {
		t.Fatalf("ReadDisposableMob(200) = %v, %v; want an instance", mob, ok)
	}

	// Reference: C read_mobile's own draw order — hit dice, then the gold sign
	// draw and the percentage draw, which both branches of the +/- run.
	ref := dprng.New(seed)
	hp := ref.Dice(1, 5) + 20
	ref.Number(0, 1)
	ref.Number(1, 20)
	if got, want := mob.MaxHP, hp; got != want {
		t.Errorf("read mob MaxHP = %d, want %d", got, want)
	}
	if got, want := dprng.Next(), ref.Next(); got != want {
		t.Errorf("stream after ReadDisposableMob = %d, want %d (hit dice + two gold draws)", got, want)
	}

	// An unknown vnum is C's real_mobile() < 0: no read, no draws.
	dprng.ResetStream(seed)
	if _, ok := w.ReadDisposableMob(999999); ok {
		t.Fatal("ReadDisposableMob(999999) reported an instance")
	}
	untouched := dprng.New(seed)
	if got, want := dprng.Next(), untouched.Next(); got != want {
		t.Errorf("stream after a missing mob = %d, want untouched %d", got, want)
	}
}

// C's read_object(nr, REAL) applies init_rare to an ITEM_RARE prototype, which
// burns dice(1,100) for every affected slot with a non-zero location and one
// number(0,1) when that roll lands (src/db.c:1899-1925, :1928-1962). check_load
// reads an object for every zone O/E/G command, so the variance must happen even
// though only the load percentage and short description are kept (R3a).
func TestReadDisposableObjectRollsRareVariance(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room A", Zone: 1}},
		Objs: []parser.Obj{{
			VNum:       300,
			ShortDesc:  "a rare sword",
			ExtraFlags: [4]int{FlagItemRare},
			Affects: []parser.ObjAffect{
				{Location: 0, Modifier: 3},  // no location: no draw
				{Location: 19, Modifier: 1}, // APPLY_DAMROLL: dice(1,100) + sign
			},
		}, {
			VNum:      301,
			ShortDesc: "a plain sword",
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	const seed = 3
	dprng.ResetStream(seed)
	obj, ok := w.ReadDisposableObject(300)
	if !ok || obj == nil {
		t.Fatalf("ReadDisposableObject(300) = %v, %v; want an instance", obj, ok)
	}

	ref := dprng.New(seed)
	if ref.Number(1, 100) <= 20 {
		ref.Number(0, 1)
	}
	if got, want := dprng.Next(), ref.Next(); got != want {
		t.Errorf("stream after a rare read = %d, want %d (dice(1,100), sign only on a hit)", got, want)
	}

	// A non-rare object draws nothing (db.c:1960 guards init_rare).
	dprng.ResetStream(seed)
	if _, ok := w.ReadDisposableObject(301); !ok {
		t.Fatal("ReadDisposableObject(301) reported no instance")
	}
	plain := dprng.New(seed)
	if got, want := dprng.Next(), plain.Next(); got != want {
		t.Errorf("stream after a non-rare read = %d, want untouched %d", got, want)
	}
}
