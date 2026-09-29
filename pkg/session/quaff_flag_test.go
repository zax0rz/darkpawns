package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestQuaffSetsCrashFlag: C's mag_objectmagic consumes a carried potion via
// extract_obj → obj_from_char (spell_parser.c:697, handler.c:1016-1017) —
// PLR_CRASH set. A held potion leaves via unequip_char and does not flag.
func TestQuaffSetsCrashFlag(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Lab"}},
		Objs:  []parser.Obj{{VNum: 7101, Keywords: "potion", ShortDesc: "a potion", TypeFlag: 10, WearFlags: [4]int{1}, Values: [4]int{1, 0, 0, 0}}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Quaffer", 1001, true)
	if err := w.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	obj, err := w.SpawnObject(7101, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, s.player); err != nil {
		t.Fatalf("carry: %v", err)
	}
	s.player.SetPlrFlag(game.PlrCrash, false)

	if err := cmdQuaff(s, []string{"potion"}); err != nil {
		t.Fatalf("cmdQuaff: %v", err)
	}
	if !s.player.NeedsCrashSave() {
		t.Fatal("quaffing a carried potion did not set PLR_CRASH (C extract_obj → obj_from_char)")
	}
}
