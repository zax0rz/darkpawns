package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShowAggrCreationOrder(t *testing.T) {
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 9, ShortDesc: "older"}, {VNum: 1, ShortDesc: "newer"}, {VNum: 3, ShortDesc: "ordinary"}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
	if err := cmdShow(s, []string{"aggr"}); err != nil {
		t.Fatal(err)
	}
	if got := drainShowReport(t, s); got != "" {
		t.Fatalf("empty report %q", got)
	}
	a, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatal(err)
	}
	a.SetMobFlag(19)
	b, err := w.SpawnMob(9, 1002)
	if err != nil {
		t.Fatal(err)
	}
	b.SetMobFlag(19)
	if _, err := w.SpawnMob(3, 1001); err != nil {
		t.Fatal(err)
	}
	if err := cmdShow(s, []string{"aggr"}); err != nil {
		t.Fatal(err)
	}
	if got, want := drainShowReport(t, s), "9 older\r\n1 newer\r\n"; got != want {
		t.Fatalf("creation order %q want %q", got, want)
	}
	// Room arrival order must not replace the global read_mobile order.
	if err := w.CharTransfer("newer", true, 1002); err != nil {
		t.Fatal(err)
	}
	if err := cmdShow(s, []string{"aggr"}); err != nil {
		t.Fatal(err)
	}
	if got, want := drainShowReport(t, s), "9 older\r\n1 newer\r\n"; got != want {
		t.Fatalf("after transfer %q want %q", got, want)
	}
	b.ClearMobFlag(19)
	if err := cmdShow(s, []string{"aggr"}); err != nil {
		t.Fatal(err)
	}
	if got := drainShowReport(t, s); got != "1 newer\r\n" {
		t.Fatalf("one report %q", got)
	}
}
