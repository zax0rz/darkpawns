package game

import (
	"strings"
	"testing"
)

// TestDonateAppearLinesCapitalized pins C act()'s CAP(lbuf) behavior
// (comm.c:2477): the donation-room appear lines start with the substituted
// $p short description, and act() uppercases the assembled message's first
// letter, so "a loaf of bread" renders "A loaf of bread suddenly appears
// in a puff a smoke!". The oracle proved the item line live (donate-routing,
// seed 4 exposed the divergence); the gold line has no mortal gold vehicle
// and is unit-proven here.
func TestDonateAppearLinesCapitalized(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	ch.Inventory.AddItem(newDonatableItem(9101, "a loaf of bread", "loaf bread", 3))
	w.performDispose(ch, ch.Inventory.Items[0], scmdDonate, "donate", DonationRoom1)
	if msg := lastMsg(); !strings.Contains(msg, "You donate a loaf of bread.") {
		t.Fatalf("donate actor line: got %q", msg)
	}

	// Listen from the donation room for the appear line.
	listener := NewPlayer(2, "Listener", DonationRoom1)
	if err := w.AddPlayer(listener); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	ch.Inventory.AddItem(newDonatableItem(9102, "a crusty baguette", "crusty baguette", 4))
	w.performDispose(ch, ch.Inventory.Items[0], scmdDonate, "donate", DonationRoom1)
	if got := lastMsg(); !strings.Contains(got, "A crusty baguette suddenly appears in a puff a smoke!") {
		t.Fatalf("item appear line must be capitalized: got %q", got)
	}

	ch.SetGold(30)
	w.performDisposeGold(ch, 25, scmdDonate, DonationRoom1)
	if got := lastMsg(); !strings.Contains(got, "A little pile of gold coins suddenly appears in a puff of orange smoke!") {
		t.Fatalf("gold appear line must be capitalized: got %q", got)
	}
}
