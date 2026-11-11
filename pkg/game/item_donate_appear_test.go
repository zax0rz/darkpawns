package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
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
	w.performDispose(ch, ch.Inventory.Snapshot()[0], scmdDonate, "donate", DonationRoom1)
	if msg := lastMsg(); !strings.Contains(msg, "You donate a loaf of bread.") {
		t.Fatalf("donate actor line: got %q", msg)
	}

	// Listen from the donation room for the appear line.
	listener := NewPlayer(2, "Listener", DonationRoom1)
	if err := w.AddPlayer(listener); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	ch.Inventory.AddItem(newDonatableItem(9102, "a crusty baguette", "crusty baguette", 4))
	w.performDispose(ch, ch.Inventory.Snapshot()[0], scmdDonate, "donate", DonationRoom1)
	if got := lastMsg(); !strings.Contains(got, "A crusty baguette suddenly appears in a puff a smoke!") {
		t.Fatalf("item appear line must be capitalized: got %q", got)
	}

	ch.SetGold(30)
	w.performDisposeGold(ch, 25, scmdDonate, DonationRoom1)
	if got := lastMsg(); !strings.Contains(got, "A little pile of gold coins suddenly appears in a puff of orange smoke!") {
		t.Fatalf("gold appear line must be capitalized: got %q", got)
	}
}

func TestDonateArrivalUsesActObjectVisibilityAndSleepAudience(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

	visible := NewPlayer(2, "Visible", DonationRoom1)
	visible.SetAffect(affDetectInvisible, true)
	hidden := NewPlayer(3, "Hidden", DonationRoom1)
	sleeper := NewPlayer(4, "Sleeper", DonationRoom1)
	sleeper.SetAffect(affDetectInvisible, true)
	sleeper.SetPosition(combat.PosSleeping)
	for _, observer := range []*Player{visible, hidden, sleeper} {
		if err := w.AddPlayer(observer); err != nil {
			t.Fatalf("AddPlayer(%s): %v", observer.GetName(), err)
		}
	}

	obj := newDonatableItem(9110, "a hidden silver ring", "silver ring", 10)
	obj.SetExtraFlag(0, extraFlagInvisible)
	if err := ch.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	w.performDispose(ch, obj, scmdDonate, "donate", DonationRoom1)

	if got, want := messages[visible.GetName()], "A hidden silver ring suddenly appears in a puff a smoke!\r\n"; got != want {
		t.Errorf("visible observer output = %q, want %q", got, want)
	}
	if got, want := messages[hidden.GetName()], "Something suddenly appears in a puff a smoke!\r\n"; got != want {
		t.Errorf("observer unable to see object output = %q, want %q", got, want)
	}
	if got := messages[sleeper.GetName()]; got != "" {
		t.Errorf("sleeping observer output = %q, want no message", got)
	}
}

func TestDonateGoldArrivalUsesActDarknessVisibility(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	if !w.SetRoomFlagBit(DonationRoom1, 0) {
		t.Fatal("SetRoomFlagBit(DonationRoom1, ROOM_DARK) failed")
	}
	if !w.IsRoomDark(DonationRoom1) {
		t.Fatal("precondition: donation room must be dark")
	}

	hidden := NewPlayer(2, "Darkobserver", DonationRoom1)
	if err := w.AddPlayer(hidden); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var hiddenOutput strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == hidden.GetName() {
			hiddenOutput.Write(msg)
		}
	}

	ch.SetGold(25)
	w.performDisposeGold(ch, 25, scmdDonate, DonationRoom1)
	if got, want := hiddenOutput.String(), "Something suddenly appears in a puff of orange smoke!\r\n"; got != want {
		t.Fatalf("dark donation-room observer output = %q, want %q", got, want)
	}
}
