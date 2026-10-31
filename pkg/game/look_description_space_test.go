package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// do_description appends a space to the room description when the room has extra
// descriptions, and sends it as one write (src/act.informative.c:2773-2804):
//
//	if (num_extras == 0) { send_to_char(description, ch); return; }
//	sprintf(buf2, "%s ", description);
//	...
//	send_to_char(buf2, ch);
//
// Because the description ends with a newline, that space is the first byte of
// the *next* line, and look_at_room's next write is the CCGRN bracket of the
// content lists. The port carried the space forward as a line prefix but let the
// bare bracket writes pass it by, so it landed on the first character-list entry
// instead — the wrong byte stream, and invisible in a room without extras.
func TestRoomLookDescriptionTrailingSpacePrecedesTheBrackets(t *testing.T) {
	newRoom := func(t *testing.T) (*World, *Player) {
		t.Helper()
		w, err := NewWorld(&parser.World{
			Rooms: []parser.Room{{
				VNum: 1001, Name: "Sign Room",
				Description: "A dusty room.\n",
				ExtraDescs:  []parser.ExtraDesc{{Keywords: "sign", Description: "It points north."}},
			}},
			Mobs: []parser.Mob{{
				VNum: 99100, Keywords: "guard", ShortDesc: "a guard",
				Level: 3, Position: 8, DefaultPos: 8,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(w.StopAITicker)
		ch := NewPlayer(1, "Viewer", 1001)
		ch.SetAutoExit(false)
		if err := w.AddPlayer(ch); err != nil {
			t.Fatal(err)
		}
		return w, ch
	}

	t.Run("complete", func(t *testing.T) {
		w, ch := newRoom(t)
		ch.SetPlrFlag(PrfColor1, true)
		ch.SetPlrFlag(PrfColor2, true)
		if _, err := w.SpawnMob(99100, 1001); err != nil {
			t.Fatal(err)
		}

		var bares []string
		var literals []string
		for _, message := range w.DoLookRoomAt(ch, 1001, true).Messages {
			if message.Bare {
				bares = append(bares, message.Format)
			} else if message.Literal {
				literals = append(literals, message.Format)
			}
		}

		// C writes the space, then the green bracket: " " + KGRN.
		if len(bares) == 0 || bares[0] != " \x1b[32m" {
			t.Fatalf("first bare write = %q, want C's space before the object-list bracket", bares)
		}
		if want := []string{" \x1b[32m", "\x1b[33m", "\x1b[0m"}; strings.Join(bares, "|") != strings.Join(want, "|") {
			t.Fatalf("bare writes = %q, want %q", bares, want)
		}
		// The space must not have survived onto a list entry.
		for _, line := range literals {
			if strings.HasPrefix(line, " ") && !strings.HasPrefix(line, " A dusty room.") {
				t.Fatalf("line %q starts with the description's trailing space; C puts it before the bracket", line)
			}
		}
	})

	t.Run("colour off", func(t *testing.T) {
		w, ch := newRoom(t)
		if _, err := w.SpawnMob(99100, 1001); err != nil {
			t.Fatal(err)
		}

		// C's send_to_char("") writes nothing, so the space stands alone and the
		// list lines stay clean.
		var bares []string
		for _, message := range w.DoLookRoomAt(ch, 1001, true).Messages {
			if message.Bare {
				bares = append(bares, message.Format)
			}
			if message.Literal && strings.HasPrefix(message.Format, " ") {
				t.Fatalf("colour off: line %q carries the description's space", message.Format)
			}
		}
		if len(bares) != 1 || bares[0] != " " {
			t.Fatalf("colour-off bare writes = %q, want C's lone space", bares)
		}
	})

	// With autoexit on, C's next write after the description is the auto-exit
	// line, so that is where the space lands.
	t.Run("autoexit on", func(t *testing.T) {
		w, ch := newRoom(t)
		ch.SetAutoExit(true)
		ch.SetPlrFlag(PrfColor1, true)
		ch.SetPlrFlag(PrfColor2, true)
		if _, err := w.SpawnMob(99100, 1001); err != nil {
			t.Fatal(err)
		}

		for _, message := range w.DoLookRoomAt(ch, 1001, true).Messages {
			// The space precedes the exit line's own CCCYN, so it is the first byte.
			if message.Literal && strings.HasPrefix(message.Format, " ") && strings.Contains(message.Format, "[ Exits:") {
				return
			}
		}
		t.Fatal("autoexit on: no line carries the description's trailing space")
	})

	t.Run("no extra descriptions", func(t *testing.T) {
		w, ch := newRoom(t)
		ch.SetPlrFlag(PrfColor1, true)
		ch.SetPlrFlag(PrfColor2, true)
		room, ok := w.GetRoom(1001)
		if !ok {
			t.Fatal("room missing")
		}
		room.ExtraDescs = nil
		if _, err := w.SpawnMob(99100, 1001); err != nil {
			t.Fatal(err)
		}

		// num_extras == 0 sends the description alone: no space, no prefix.
		for _, message := range w.DoLookRoomAt(ch, 1001, true).Messages {
			if message.Bare && strings.HasPrefix(message.Format, " ") {
				t.Fatalf("bare write %q carries a space with no extra descriptions", message.Format)
			}
			if message.Literal && strings.HasPrefix(message.Format, " ") {
				t.Fatalf("line %q carries a space with no extra descriptions", message.Format)
			}
		}
	})
}
