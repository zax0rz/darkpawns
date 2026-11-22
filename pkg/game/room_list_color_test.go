package game

import (
	"strings"
	"testing"
)

type roomLookDelivery struct {
	bare bool
	text string
}

func renderRoomLook(w *World, ch *Player) []roomLookDelivery {
	events := make([]roomLookDelivery, 0)
	w.MessageSink = func(_ string, msg []byte) {
		events = append(events, roomLookDelivery{text: string(msg)})
	}
	w.LiteralMessageSink = func(_ string, msg []byte) {
		events = append(events, roomLookDelivery{bare: true, text: string(msg)})
	}
	w.RenderObservationMessages(w.DoLookRoomAt(ch, 1001, true))
	return events
}

// C's look_at_room brackets the two room-content lists once each
// (src/act.informative.c:823-837):
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
	corpse := w.makeCorpse("a goblin", 1, nil, nil, 1001, 0, 0, true, 50)
	w.AddItemToRoom(corpse, 1001)

	// The list escapes are gated on C_NRM, which both colour bits satisfy.
	ch.SetPlrFlag(PrfColor1, true)
	ch.SetPlrFlag(PrfColor2, true)

	events := renderRoomLook(w, ch)
	var bares []string
	greenIndex, resetIndex := -1, -1
	for i, event := range events {
		if event.bare {
			bares = append(bares, event.text)
			switch event.text {
			case "\x1b[32m":
				greenIndex = i
			case "\x1b[0m":
				resetIndex = i
			}
		}
	}

	want := []string{"\x1b[32m", "\x1b[33m", "\x1b[0m"}
	if strings.Join(bares, "|") != strings.Join(want, "|") {
		t.Fatalf("bare writes = %q, want C's three list brackets %q", bares, want)
	}

	if greenIndex < 0 || resetIndex <= greenIndex {
		t.Fatalf("delivered writes omitted C's list brackets: %+v", events)
	}
	for _, event := range events[greenIndex+1 : resetIndex] {
		if !event.bare && strings.Contains(event.text, "\x1b") {
			t.Fatalf("delivered list line %q carries its own colour; C brackets the list, not the entry", event.text)
		}
	}

	// Below C_NRM every macro expands to the empty string, so no bare write is
	// delivered — the same look at `color off` or `color sparse` is plain.
	ch.SetPlrFlag(PrfColor1, false)
	ch.SetPlrFlag(PrfColor2, false)
	for _, event := range renderRoomLook(w, ch) {
		if event.bare {
			t.Fatalf("bare write %q at colour level 0; C's macros are empty below C_NRM", event.text)
		}
	}
}
