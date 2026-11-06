package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestActToRoomHideInvisibleSkipsBlindObserver pins DP-1481 shape 1: C's item
// and OLC acts pass hide_invisible = TRUE (src/act.item.c:279 gets, :1727 stops
// using; src/olc.c:275,514 OLC presence), so a receiver who cannot see the
// actor is skipped entirely. The actToRoom adapter lost that flag; this proves
// it is honoured, and that the FALSE sites (C's money drop, light and zap)
// still deliver.
func TestActToRoomHideInvisibleSkipsBlindObserver(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room"}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	actor := NewPlayer(1, "Actor", 1001)
	sighted := NewPlayer(2, "Sighted", 1001)
	blind := NewPlayer(3, "Blind", 1001)
	for _, p := range []*Player{actor, sighted, blind} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}
	blind.SetAffect(affBlind, true)

	var sightedOut, blindOut strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		switch name {
		case "Sighted":
			sightedOut.Write(msg)
		case "Blind":
			blindOut.Write(msg)
		}
	}

	// C: src/act.item.c:279 act("$n gets $p.", TRUE, ...) — the blind observer is skipped.
	w.actToRoom(actor, "$n gets $p.", nil, nil, true)
	if !strings.Contains(sightedOut.String(), "Actor gets") {
		t.Errorf("sighted observer missed a hide_invisible act: %q", sightedOut.String())
	}
	if got := blindOut.String(); got != "" {
		t.Errorf("blind observer received a hide_invisible act: %q", got)
	}

	// Control: C's FALSE sites (src/act.item.c:460 money drop, src/handler.c:705
	// zap) still deliver to the blind observer.
	sightedOut.Reset()
	blindOut.Reset()
	w.actToRoom(actor, "$n drops some coins.", nil, nil, false)
	if !strings.Contains(blindOut.String(), "drops some coins") {
		t.Errorf("hide=false act did not reach the blind observer: %q", blindOut.String())
	}
}
