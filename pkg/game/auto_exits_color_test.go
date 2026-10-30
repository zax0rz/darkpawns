package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func autoExitsTestRoom() *parser.Room {
	return &parser.Room{
		VNum: 1001, Name: "Hall",
		Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 1002},
			"east":  {Direction: "east", ToRoom: 1002, ExitInfo: parser.ExitClosed},
			"south": {Direction: "south", ToRoom: -1},
		},
	}
}

func newAutoExitsWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}, {VNum: 1002, Name: "Yard"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Viewer", 1001)
	return w, ch
}

// do_auto_exits appends a trailing space to every entry and wraps an imm-only
// closed door in CCRED(ch, C_CMP) "(" dir ")" CCCYN(ch, C_NRM)
// (src/act.informative.c:663-674), then brackets the whole list in
// CCCYN(ch, C_NRM) ... CCNRM(ch, C_NRM) with no space before the closing bracket
// (line 676). Because the space moved into the entries, the uncoloured bytes are
// unchanged — which is why no scenario had ever seen the missing red sandwich —
// and only `complete` shows it, since CCRED here is C_CMP while the list itself
// is C_NRM. Expected bytes use the literal codes from src/screen.h:34-41
// (R1/R5e/R5h).
func TestAutoExitsColorLevelsMatchC(t *testing.T) {
	for _, test := range []struct {
		name     string
		level    string
		color1   bool
		color2   bool
		immortal bool
		want     string
	}{
		{name: "mortal color off", want: "[ Exits: north ]"},
		{name: "mortal sparse", level: "sparse", color1: true, want: "[ Exits: north ]"},
		{
			name: "mortal normal", level: "normal", color2: true,
			want: "\x1b[36m[ Exits: north ]\x1b[0m",
		},
		{
			name: "mortal complete", level: "complete", color1: true, color2: true,
			want: "\x1b[36m[ Exits: north ]\x1b[0m",
		},
		{name: "immortal color off", immortal: true, want: "[ Exits: north (east) ]"},
		// C_CMP needs both bits, so at normal the closed door is listed without
		// the red sandwich — but C re-emits CCCYN after the ")" whenever the door
		// is imm-only (line 673), so that escape is present either way.
		{
			name: "immortal normal", level: "normal", color2: true, immortal: true,
			want: "\x1b[36m[ Exits: north (east)\x1b[36m ]\x1b[0m",
		},
		{
			name: "immortal complete", level: "complete", color1: true, color2: true, immortal: true,
			want: "\x1b[36m[ Exits: north \x1b[31m(east)\x1b[36m ]\x1b[0m",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, ch := newAutoExitsWorld(t)
			ch.SetPlrFlag(PrfColor1, test.color1)
			ch.SetPlrFlag(PrfColor2, test.color2)
			if test.immortal {
				ch.SetLevel(LVL_IMMORT)
			}
			if got := w.autoExitsText(ch, autoExitsTestRoom()); got != test.want {
				t.Fatalf("autoExitsText() = %q, want %q", got, test.want)
			}
		})
	}
}

// A room with nothing visible prints C's literal "None! " fallback, trailing
// space included, so the bracket closes at the same column (src/act.informative.c:677).
func TestAutoExitsEmptyListKeepsCTrailingSpace(t *testing.T) {
	w, ch := newAutoExitsWorld(t)
	ch.SetPlrFlag(PrfColor2, true)
	room := &parser.Room{VNum: 1001, Name: "Void", Exits: map[string]parser.Exit{
		"north": {Direction: "north", ToRoom: -1},
		"east":  {Direction: "east", ToRoom: 1002, ExitInfo: parser.ExitClosed},
	}}
	if got, want := w.autoExitsText(ch, room), "\x1b[36m[ Exits: None! ]\x1b[0m"; got != want {
		t.Fatalf("empty auto-exits = %q, want %q", got, want)
	}
}
