package game

import (
	"regexp"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// do_description writes its buffer with one send_to_char
// (src/act.informative.c:2757-2803). writeRoomDescription frames it only
// where a framed line is the same bytes, and writes it verbatim everywhere
// else. Each room below is a shape the framed line changed: 15, 4 and 8 of
// the 8,218 indexed rooms carry the first three, and the parser cannot yet
// produce the fourth (readTildeString drops the empty last line).
//
// The events are one look's writes in delivery order. The bare ones are the
// literal sink's, which is where C's buffer must arrive untouched.

var ansiCSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func newDescriptionShapeWorld(t *testing.T, room parser.Room) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{room}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Viewer", room.VNum)
	ch.SetAutoExit(true)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatal(err)
	}
	return w, ch
}

// lookWrites renders one room look and returns every write in delivery order
// and the first literal write, which is do_description's whenever the look
// has one to make (the list brackets are literal writes too, but they come
// after it).
func lookWrites(w *World, ch *Player) (stream, literal string) {
	var all strings.Builder
	seen := false
	for _, event := range renderRoomLook(w, ch) {
		all.WriteString(event.text)
		if event.bare && !seen {
			literal, seen = event.text, true
		}
	}
	return all.String(), literal
}

func TestRoomLookDescriptionIsWrittenVerbatimWhereFramingChangedIt(t *testing.T) {
	const bold, normal = "\x1b[0;1m", "\x1b[0m" // VT_BOLDTEX, VT_NORMALT (vt100.h:43-44)
	cases := []struct {
		name string
		room parser.Room
		// colour sets both PRF_COLOR bits, so COLOR_LEV is 3 and C's
		// keyword wrap runs (src/act.informative.c:2785).
		colour      bool
		wantLiteral string // the bytes of C's one send_to_char
		wantStream  string // every write of the look, ANSI removed
	}{
		{
			name: "keyword is the description's last word",
			room: parser.Room{
				VNum: 1001, Name: "Southern Wall Road",
				Description: "The tavern hides the thieves guild.\n",
				ExtraDescs:  []parser.ExtraDesc{{Keywords: "guild", Description: "Everyone knows it."}},
			},
			colour: true,
			// fread_string keeps the last line as CRLF (src/db.c:2705-2717).
			// C's scan stops at that CR, so the reset follows it
			// (src/act.informative.c:2789-2796), and the space follows the LF.
			wantLiteral: "The tavern hides the thieves" + bold + " guild.\r" + normal + "\n ",
			wantStream:  "Southern Wall Road\r\nThe tavern hides the thieves guild.\r\n [ Exits: None! ]\r\n",
		},
		{
			name:        "no final line break",
			room:        parser.Room{VNum: 1001, Name: "Crypts", Description: "Empty"},
			wantLiteral: "Empty",
			wantStream:  "Crypts\r\nEmpty[ Exits: None! ]\r\n",
		},
		{
			// send_to_char does not capitalize; the framed act() did.
			name:        "first letter lowercase",
			room:        parser.Room{VNum: 1001, Name: "Stable", Description: "stable room.\n"},
			wantLiteral: "stable room.\r\n",
			wantStream:  "Stable\r\nstable room.\r\n[ Exits: None! ]\r\n",
		},
		{
			name:        "trailing blank line",
			room:        parser.Room{VNum: 1001, Name: "Hall", Description: "A line.\n\n"},
			wantLiteral: "A line.\r\n\r\n",
			wantStream:  "Hall\r\nA line.\r\n\r\n[ Exits: None! ]\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newDescriptionShapeWorld(t, tc.room)
			if tc.colour {
				ch.SetPlrFlag(PrfColor1, true)
				ch.SetPlrFlag(PrfColor2, true)
			}
			stream, literal := lookWrites(w, ch)
			if literal != tc.wantLiteral {
				t.Errorf("literal writes = %q, want C's buffer %q", literal, tc.wantLiteral)
			}
			if got := ansiCSI.ReplaceAllString(stream, ""); got != tc.wantStream {
				t.Errorf("look stream = %q, want %q", got, tc.wantStream)
			}
		})
	}
}
