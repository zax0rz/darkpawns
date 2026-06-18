package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDonateTestWorld(t *testing.T) (*World, *Player, func() string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
			{VNum: DonationRoom1, Name: "Donation Room 1", Zone: 1},
			{VNum: DonationRoom2, Name: "Donation Room 2", Zone: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	lastMsg := func() string { s := out.String(); out.Reset(); return s }
	return w, ch, lastMsg
}

func newDonatableItem(vnum int, shortDesc, keywords string, cost int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		Cost:      cost,
	}, -1)
}

// TestPerformDispose verifies perform_drop()'s junk/donate mechanics —
// src/act.item.c:480-525.
func TestPerformDispose(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	// NODROP item: rejected, never leaves inventory.
	cursed := newDonatableItem(3000, "a cursed ring", "ring", 1000)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := ch.Inventory.AddItem(cursed); err != nil {
		t.Fatalf("AddItem cursed: %v", err)
	}
	if v := w.performDispose(ch, cursed, scmdJunk, "junk", 0); v != 0 {
		t.Errorf("NODROP junk: got value %d, want 0", v)
	}
	if msg := lastMsg(); !strings.Contains(msg, "it must be CURSED") {
		t.Errorf("NODROP junk message: got %q", msg)
	}
	found, _ := ch.Inventory.FindItem("ring")
	if found != cursed {
		t.Errorf("NODROP item should remain in inventory")
	}

	// Junk: value = clamp(cost>>4, 1, 200), item destroyed, message has VANISH.
	sword := newDonatableItem(3001, "a steel sword", "sword", 800)
	if err := ch.Inventory.AddItem(sword); err != nil {
		t.Fatalf("AddItem sword: %v", err)
	}
	gotValue := w.performDispose(ch, sword, scmdJunk, "junk", 0)
	wantValue := 800 >> 4
	if gotValue != wantValue {
		t.Errorf("junk value: got %d, want %d", gotValue, wantValue)
	}
	if msg := lastMsg(); !strings.Contains(msg, "You junk") || !strings.Contains(msg, "vanishes in a puff of smoke") {
		t.Errorf("junk message: got %q", msg)
	}
	if _, ok := ch.Inventory.FindItem("sword"); ok {
		t.Errorf("junked item should be removed from inventory")
	}

	// Junk value clamps: cheap item floors to 1, expensive item caps at 200.
	cheap := newDonatableItem(3002, "a pebble", "pebble", 1)
	if err := ch.Inventory.AddItem(cheap); err != nil {
		t.Fatalf("AddItem cheap: %v", err)
	}
	if v := w.performDispose(ch, cheap, scmdJunk, "junk", 0); v != 1 {
		t.Errorf("junk floor: got %d, want 1", v)
	}
	expensive := newDonatableItem(3003, "an artifact", "artifact", 100000)
	if err := ch.Inventory.AddItem(expensive); err != nil {
		t.Fatalf("AddItem expensive: %v", err)
	}
	if v := w.performDispose(ch, expensive, scmdJunk, "junk", 0); v != 200 {
		t.Errorf("junk cap: got %d, want 200", v)
	}

	// Donate: teleports to the destination room and broadcasts there, not
	// where the player is standing.
	shield := newDonatableItem(3004, "a wooden shield", "shield", 500)
	if err := ch.Inventory.AddItem(shield); err != nil {
		t.Fatalf("AddItem shield: %v", err)
	}
	if v := w.performDispose(ch, shield, scmdDonate, "donate", DonationRoom1); v != 0 {
		t.Errorf("donate value: got %d, want 0", v)
	}
	if msg := lastMsg(); !strings.Contains(msg, "You donate") || !strings.Contains(msg, "vanishes in a puff of smoke") {
		t.Errorf("donate message: got %q", msg)
	}
	items := w.GetItemsInRoom(DonationRoom1)
	foundInRoom := false
	for _, item := range items {
		if item == shield {
			foundInRoom = true
		}
	}
	if !foundInRoom {
		t.Errorf("donated item should land in the donation room")
	}

	// NODONATE item: forced to junk even on a "donate" call — no teleport,
	// and the item is destroyed instead of appearing in the donation room.
	tagged := newDonatableItem(3005, "a marked locket", "locket", 320)
	tagged.SetExtraFlag(0, extraFlagNoDonate)
	if err := ch.Inventory.AddItem(tagged); err != nil {
		t.Fatalf("AddItem tagged: %v", err)
	}
	gotValue = w.performDispose(ch, tagged, scmdDonate, "donate", DonationRoom2)
	if gotValue != 320>>4 {
		t.Errorf("NODONATE-forced-junk value: got %d, want %d", gotValue, 320>>4)
	}
	for _, item := range w.GetItemsInRoom(DonationRoom2) {
		if item == tagged {
			t.Errorf("NODONATE item should not land in the donation room")
		}
	}
}

// TestPerformDisposeGold verifies perform_drop_gold()'s junk/donate
// mechanics — src/act.item.c:430-474.
func TestPerformDisposeGold(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	ch.SetGold(100)
	w.performDisposeGold(ch, 0, scmdJunk, 0)
	if msg := lastMsg(); !strings.Contains(msg, "jolly funny") {
		t.Errorf("non-positive amount: got %q", msg)
	}
	if ch.GetGold() != 100 {
		t.Errorf("gold must be untouched: got %d", ch.GetGold())
	}

	w.performDisposeGold(ch, 1000, scmdJunk, 0)
	if msg := lastMsg(); !strings.Contains(msg, "don't have that many coins") {
		t.Errorf("insufficient gold: got %q", msg)
	}
	if ch.GetGold() != 100 {
		t.Errorf("gold must be untouched: got %d", ch.GetGold())
	}

	// Junk gold: vanishes silently, no object created anywhere.
	w.performDisposeGold(ch, 40, scmdJunk, 0)
	if ch.GetGold() != 60 {
		t.Errorf("gold after junk: got %d, want 60", ch.GetGold())
	}
	if msg := lastMsg(); !strings.Contains(msg, "disappears in a puff of smoke") {
		t.Errorf("junk gold message: got %q", msg)
	}

	// Donate gold: a money pile appears in the donation room.
	w.performDisposeGold(ch, 25, scmdDonate, DonationRoom1)
	if ch.GetGold() != 35 {
		t.Errorf("gold after donate: got %d, want 35", ch.GetGold())
	}
	if msg := lastMsg(); !strings.Contains(msg, "throw some gold") {
		t.Errorf("donate gold message: got %q", msg)
	}
	foundPile := false
	for _, item := range w.GetItemsInRoom(DonationRoom1) {
		if item.CustomData != nil && item.CustomData["is_money"] == true {
			foundPile = true
		}
	}
	if !foundPile {
		t.Errorf("expected a money pile in the donation room after donating gold")
	}
}

// TestDoJunk verifies the junk command end to end — src/act.item.c
// ACMD(do_drop) with subcmd SCMD_JUNK.
func TestDoJunk(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	w.DoJunk(ch, "")
	if msg := lastMsg(); !strings.Contains(msg, "What do you want to junk") {
		t.Errorf("empty arg: got %q", msg)
	}

	w.DoJunk(ch, "all")
	if msg := lastMsg(); !strings.Contains(msg, "Go to the dump") {
		t.Errorf("bare all: got %q", msg)
	}

	w.DoJunk(ch, "nonexistent")
	if msg := lastMsg(); !strings.Contains(msg, "don't seem to have") {
		t.Errorf("not found: got %q", msg)
	}

	// Single item junk grants exp (min(10, value)).
	dagger := newDonatableItem(3010, "a rusty dagger", "dagger", 160) // value = 10
	if err := ch.Inventory.AddItem(dagger); err != nil {
		t.Fatalf("AddItem dagger: %v", err)
	}
	w.DoJunk(ch, "dagger")
	if msg := lastMsg(); !strings.Contains(msg, "rewarded by the gods") {
		t.Errorf("junk reward message: got %q", msg)
	}

	// NODROP item junked: no destruction, no reward.
	cursed := newDonatableItem(3011, "a cursed amulet", "amulet", 5000)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := ch.Inventory.AddItem(cursed); err != nil {
		t.Fatalf("AddItem cursed: %v", err)
	}
	w.DoJunk(ch, "amulet")
	if msg := lastMsg(); strings.Contains(msg, "rewarded by the gods") {
		t.Errorf("NODROP junk should not reward: got %q", msg)
	}
	if _, ok := ch.Inventory.FindItem("amulet"); !ok {
		t.Errorf("NODROP item should remain in inventory after a failed junk")
	}

	// all.dot junks every match and accumulates the reward, capped at 10.
	for i := 0; i < 3; i++ {
		coin := newDonatableItem(3020+i, "a brass token", "token", 800) // value 50 each
		if err := ch.Inventory.AddItem(coin); err != nil {
			t.Fatalf("AddItem token %d: %v", i, err)
		}
	}
	expBefore := ch.Exp
	w.DoJunk(ch, "all.token")
	if msg := lastMsg(); !strings.Contains(msg, "rewarded by the gods") {
		t.Errorf("all.dot junk reward message: got %q", msg)
	}
	if ch.Exp-expBefore > 10 {
		t.Errorf("junk reward should cap at 10 exp per command: got %d", ch.Exp-expBefore)
	}
	if _, ok := ch.Inventory.FindItem("token"); ok {
		t.Errorf("all.token should have junked every matching item")
	}

	// Coins: junked silently, no exp reward path.
	ch.SetGold(50)
	expBefore = ch.Exp
	w.DoJunk(ch, "10 coins")
	if ch.GetGold() != 40 {
		t.Errorf("gold after junking coins: got %d, want 40", ch.GetGold())
	}
	if ch.Exp != expBefore {
		t.Errorf("junking coins should never grant exp")
	}
}

// TestDoDonate verifies the donate command end to end — src/act.item.c
// ACMD(do_drop) with subcmd SCMD_DONATE, including the dice-roll quirk
// where a fraction of donations secretly destroy the item instead.
func TestDoDonate(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	w.DoDonate(ch, "")
	// The dice roll happens before argument parsing in C too, so an empty
	// arg with both donation rooms present always reaches the empty-arg
	// message (the dice can only reject when a room is missing).
	if msg := lastMsg(); !strings.Contains(msg, "What do you want to donate") {
		t.Errorf("empty arg: got %q", msg)
	}

	w.DoDonate(ch, "all")
	if msg := lastMsg(); !strings.Contains(msg, "Go do the donation room") {
		t.Errorf("bare all: got %q", msg)
	}

	// A NODONATE item is destroyed regardless of which way the dice land:
	// either the roll picks junk directly, or it picks a donate room and
	// performDispose's per-item override converts it to junk anyway. Loop
	// enough times to exercise all four dice outcomes.
	for i := 0; i < 20; i++ {
		locket := newDonatableItem(3030, "a marked locket", "locket", 320)
		locket.SetExtraFlag(0, extraFlagNoDonate)
		if err := ch.Inventory.AddItem(locket); err != nil {
			t.Fatalf("AddItem locket: %v", err)
		}
		expBefore := ch.Exp
		w.DoDonate(ch, "locket")
		if msg := lastMsg(); !strings.Contains(msg, "You donate") {
			t.Errorf("NODONATE donate message: got %q", msg)
		}
		if _, ok := ch.Inventory.FindItem("locket"); ok {
			t.Errorf("NODONATE item should always leave the inventory")
		}
		for _, vnum := range []int{DonationRoom1, DonationRoom2} {
			for _, item := range w.GetItemsInRoom(vnum) {
				if item == locket {
					t.Errorf("NODONATE item should never land in a donation room")
				}
			}
		}
		if ch.Exp != expBefore {
			t.Errorf("donate must never grant exp, even when the dice secretly junks the item")
		}
	}

	// A normal item either teleports to one of the two donation rooms or is
	// destroyed by the 25% dice outcome — never left in inventory, never
	// rewarded with exp. Loop to get probabilistic coverage of all branches.
	sawTeleport := false
	for i := 0; i < 40; i++ {
		trinket := newDonatableItem(3031, "a tin trinket", "trinket", 400)
		if err := ch.Inventory.AddItem(trinket); err != nil {
			t.Fatalf("AddItem trinket: %v", err)
		}
		expBefore := ch.Exp
		w.DoDonate(ch, "trinket")
		if _, ok := ch.Inventory.FindItem("trinket"); ok {
			t.Fatalf("donated item should always leave the inventory")
		}
		if ch.Exp != expBefore {
			t.Errorf("donate must never grant exp")
		}
		for _, vnum := range []int{DonationRoom1, DonationRoom2} {
			items := w.GetItemsInRoom(vnum)
			for _, item := range items {
				if item == trinket {
					sawTeleport = true
				}
			}
		}
	}
	if !sawTeleport {
		t.Errorf("expected at least one of 40 donates to land in a donation room")
	}

	// Donating coins: gold always deducted; sometimes a pile appears in a
	// donation room (donate-success), sometimes it's silently destroyed
	// (the 25% junk outcome) — either way no exp and no rejection.
	ch.SetGold(1000)
	for i := 0; i < 20; i++ {
		before := ch.GetGold()
		w.DoDonate(ch, "5 coins")
		if ch.GetGold() != before-5 {
			t.Fatalf("gold after donating coins: got %d, want %d", ch.GetGold(), before-5)
		}
	}

	// Missing donation rooms: the dice roll's case 1/2/3 branches reject the
	// whole command up front. Run enough trials that at least one non-junk
	// roll (75% per trial) is observed.
	w2, ch2, lastMsg2 := newDonateTestWorldWithoutDonationRooms(t)
	rejected := false
	for i := 0; i < 30; i++ {
		item := newDonatableItem(3032, "a tin trinket", "trinket", 400)
		if err := ch2.Inventory.AddItem(item); err != nil {
			t.Fatalf("AddItem trinket: %v", err)
		}
		w2.DoDonate(ch2, "trinket")
		if strings.Contains(lastMsg2(), "can't donate anything right now") {
			rejected = true
		}
		// Clean up whatever happened to the item so the next trial starts fresh.
		ch2.Inventory.removeItem(item)
	}
	if !rejected {
		t.Errorf("expected at least one of 30 donates to hit the NOWHERE rejection with no donation rooms loaded")
	}
}

func newDonateTestWorldWithoutDonationRooms(t *testing.T) (*World, *Player, func() string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	lastMsg := func() string { s := out.String(); out.Reset(); return s }
	return w, ch, lastMsg
}

// TestMakeCorpseNoDonate verifies corpses get ITEM_NODONATE — src/fight.c:388.
func TestMakeCorpseNoDonate(t *testing.T) {
	w, _, _ := newDonateTestWorld(t)
	corpse := w.makeCorpse("a goblin", 1, nil, nil, 1001, 0, 0, true)
	if !corpse.HasExtraFlag(0, extraFlagNoDonate) {
		t.Errorf("corpse should be flagged NODONATE")
	}
}
