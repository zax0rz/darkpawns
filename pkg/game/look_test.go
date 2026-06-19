package game

import (
	"strings"
	"testing"
)

// TestListObjToCharCorpse verifies listObjToChar doesn't panic on a
// Prototype-nil object (a corpse) and renders its Runtime short description
// instead of crashing or showing a blank line.
func TestListObjToCharCorpse(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	corpse := w.makeCorpse("a goblin", 1, nil, nil, 1001, 0, 0, true)
	w.AddItemToRoom(corpse, 1001)

	room := w.GetRoomInWorld(1001)
	if room == nil {
		t.Fatalf("expected room 1001 to exist")
	}

	w.listObjToChar(room, ch) // must not panic on the Prototype-nil corpse
	if msg := lastMsg(); !strings.Contains(msg, "the corpse of a goblin") {
		t.Errorf("expected corpse short desc in room listing, got %q", msg)
	}
}
