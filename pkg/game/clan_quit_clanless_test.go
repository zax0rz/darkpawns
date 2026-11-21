package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1610, approved divergence (Zach, 2026-11-21: "keep go"). C's
// do_clan_quit has no membership check for mortals: find_clan_by_id returns
// -1, it writes clan[-1] out of bounds (src/clan.c:623-638), then prints
// "You've quit your clan." (src/clan.c:642-647). The port keeps its refusal,
// which C itself uses in do_clan_members (src/clan.c:587), and never touches
// the clan table.
func TestClanQuitClanlessKeepsApprovedRefusal(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.Clans = NewClanManager()
	clan := &Clan{Name: "Wolves", Ranks: 2}
	w.Clans.AddClan(clan)
	before := len(w.Clans.Clans)

	var out strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == "Loner" {
			out.Write(msg)
		}
	}
	loner := &Player{Name: "Loner", Level: 10, RoomVNum: 1001, ClanID: 0}
	if err := w.AddPlayer(loner); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.ExecClanCommand(loner, "quit")

	if got := out.String(); !strings.Contains(got, "You aren't in a clan!\r\n") || strings.Contains(got, "quit your clan") {
		t.Fatalf("clanless clan quit = %q, want the approved refusal and not C's quit message", got)
	}
	if after := len(w.Clans.Clans); after != before {
		t.Fatalf("clanless clan quit changed the clan table: %d clans -> %d", before, after)
	}
}
