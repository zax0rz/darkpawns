package session

// recite_flag_test.go — R5h proofs for the recite (carried scroll) flag
// site: C's mag_objectmagic consumes the item with extract_obj
// (spell_parser.c:697); a carried scroll leaves through obj_from_char
// (handler.c:1016-1017) and flags the player, a held one leaves through
// unequip_char and does not.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newReciteHarness builds a world with a scroll prototype and a playing
// session in it.
func newReciteHarness(t *testing.T) (*game.World, *Session) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Study"}},
		Objs: []parser.Obj{{
			VNum: 7201, Keywords: "scroll", ShortDesc: "a scroll",
			TypeFlag: 2, WearFlags: [4]int{(1 << 0) | (1 << 14)},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Reciter", 1001, true)
	if err := w.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, s
}

// TestReciteCarriedSetsCrashFlag: a scroll from the carrying list is
// consumed through extract_obj → obj_from_char — PLR_CRASH set.
func TestReciteCarriedSetsCrashFlag(t *testing.T) {
	w, s := newReciteHarness(t)
	scroll, err := w.SpawnObject(7201, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(scroll, s.player); err != nil {
		t.Fatalf("carry: %v", err)
	}
	s.player.SetPlrFlag(game.PlrCrash, false)

	if err := cmdRecite(s, []string{"scroll"}); err != nil {
		t.Fatalf("cmdRecite: %v", err)
	}
	if !s.player.NeedsCrashSave() {
		t.Fatal("reciting a carried scroll did not set PLR_CRASH (C extract_obj → obj_from_char)")
	}
}

// TestReciteHeldLeavesFlagClear: a held scroll leaves through unequip_char,
// which never sets PLR_CRASH.
func TestReciteHeldLeavesFlagClear(t *testing.T) {
	w, s := newReciteHarness(t)
	scroll, err := w.SpawnObject(7201, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(scroll, s.player); err != nil {
		t.Fatalf("seat: %v", err)
	}
	if err := w.MoveObject(scroll, game.LocEquippedPlayer(s.player.Name, game.SlotHold)); err != nil {
		t.Fatalf("hold: %v", err)
	}
	s.player.SetPlrFlag(game.PlrCrash, false)

	if err := cmdRecite(s, []string{"scroll"}); err != nil {
		t.Fatalf("cmdRecite: %v", err)
	}
	if s.player.NeedsCrashSave() {
		t.Fatal("reciting a held scroll set PLR_CRASH; C's unequip_char never flags")
	}
}
