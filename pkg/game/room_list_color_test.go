package game

import (
	"strings"
	"testing"
)

// C's look_at_room brackets the two room-content lists once each
// (src/act.informative.c:823-833):
//
//	send_to_char(CCGRN(ch, C_NRM), ch);   /* once, before the object list   */
//	list_obj_to_char(world[ch->in_room].contents, ch, 20, FALSE);
//	send_to_char(CCYEL(ch, C_NRM), ch);   /* once, before the character list */
//	list_char_to_char(world[ch->in_room].people, ch);
//	send_to_char(CCNRM(ch, C_NRM), ch);   /* once, after both               */
//
// The three are direct writes with no line ending, so the entries carry no
// colour of their own. The port wrapped every entry instead, which closed the
// colour per line and never emitted the trailing reset.
func TestRoomLookBracketsTheContentListsOnce(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	corpse := w.makeCorpse("a goblin", 1, nil, nil, 1001, 0, 0, true)
	w.AddItemToRoom(corpse, 1001)

	// The list escapes are gated on C_NRM, which both colour bits satisfy.
	ch.SetPlrFlag(PrfColor1, true)
	ch.SetPlrFlag(PrfColor2, true)

	result := w.DoLookRoomAt(ch, 1001, true)

	var bares []string
	for _, message := range result.Messages {
		switch {
		case message.Bare:
			bares = append(bares, message.Format)
		}
	}

	want := []string{"\x1b[32m", "\x1b[33m", "\x1b[0m"}
	if strings.Join(bares, "|") != strings.Join(want, "|") {
		t.Fatalf("bare writes = %q, want C's three list brackets %q", bares, want)
	}

	// Everything the two lists contribute must be escape-free: the bracketing
	// is the only colour they carry. The room name keeps its own CCCYN/CCNRM
	// pair (src/act.informative.c:806-810), so only the messages from the first
	// bracket onward are checked.
	seenBracket := false
	for _, message := range result.Messages {
		if message.Bare {
			seenBracket = true
			continue
		}
		if seenBracket && message.Literal && strings.Contains(message.Format, "\x1b") {
			t.Fatalf("list line %q carries its own colour; C brackets the list, not the entry", message.Format)
		}
	}

	// Below C_NRM every macro expands to the empty string, so no bare write is
	// queued at all — the same look at `color off` or `color sparse` is plain.
	ch.SetPlrFlag(PrfColor1, false)
	ch.SetPlrFlag(PrfColor2, false)
	for _, message := range w.DoLookRoomAt(ch, 1001, true).Messages {
		if message.Bare {
			t.Fatalf("bare write %q at colour level 0; C's macros are empty below C_NRM", message.Format)
		}
	}
}
