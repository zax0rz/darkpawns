package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// SelectLoginRoom — C's CON_MENU '1' load-room selection (interpreter.c:2191-2210)
// ---------------------------------------------------------------------------

func loginRoomTestWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8008, Name: "Temple Annex", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func TestSelectLoginRoom(t *testing.T) {
	tests := []struct {
		name     string
		loadRoom int
		level    int
		frozen   bool
		want     int
	}{
		{"mortal saved room wins", 8008, 1, false, 8008},
		{"immortal saved room wins", 8008, LVL_IMPL, false, 8008},
		{"mortal fallback", LoadRoomNowhere, 1, false, MortalStartRoom},
		{"exact LVL_IMMORT fallback", LoadRoomNowhere, LVL_IMMORT, false, ImmortStartRoom},
		{"above LVL_IMMORT fallback", LoadRoomNowhere, LVL_IMMORT + 1, false, ImmortStartRoom},
		{"invalid vnum mortal fallback", 99999, 1, false, MortalStartRoom},
		{"invalid vnum immortal fallback", 99999, LVL_IMPL, false, ImmortStartRoom},
		{"frozen overrides saved room", 8008, LVL_IMPL, true, FrozenStartRoom},
		{"frozen overrides fallback", LoadRoomNowhere, 1, true, FrozenStartRoom},
	}
	w := loginRoomTestWorld(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlayer(1, "Selector", MortalStartRoom)
			p.Level = tt.level
			p.SetLoadRoom(tt.loadRoom)
			if tt.frozen {
				p.Flags |= 1 << uint(PlrFrozen)
			}
			if got := w.SelectLoginRoom(p); got != tt.want {
				t.Errorf("SelectLoginRoom = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DoQuit's load-room boundary (act.other.c:167-169)
// ---------------------------------------------------------------------------

func TestDoQuitRecordsLoadRoomThroughLVLIMMORT(t *testing.T) {
	tests := []struct {
		name         string
		level        int
		quitRoom     int
		wantLoadRoom int
	}{
		// GET_LEVEL(ch) <= LVL_IMMORT records the quit room.
		{"exact LVL_IMMORT records quit room", LVL_IMMORT, 8008, 8008},
		// One level above, the load room keeps its previous value.
		{"above LVL_IMMORT keeps load room", LVL_IMMORT + 1, 8008, LoadRoomNowhere},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := loginRoomTestWorld(t)
			p := NewPlayer(1, "Quitter", tt.quitRoom)
			p.Level = tt.level
			p.SetLoadRoom(LoadRoomNowhere)
			if out := w.DoQuit(p, false); out != QuitLogoutKeepEQ {
				t.Fatalf("DoQuit = %v, want QuitLogoutKeepEQ", out)
			}
			if got := p.GetLoadRoom(); got != tt.wantLoadRoom {
				t.Errorf("load room after quit = %d, want %d", got, tt.wantLoadRoom)
			}
		})
	}
}
