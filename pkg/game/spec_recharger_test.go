package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSpecRecharger verifies the recharger spec proc trades carried gold for
// wand/staff charges, matching src/new_cmds2.c SPECIAL(recharger). C's effect
// on a successful recharge is two independent ops: current charges (value 2)
// increments by one, max charges (value 1) decrements by one — a quirky
// trade where current can end up exceeding max, as the recharger's own "list"
// flavor text describes ("an item could have 1 charge maximum, with 2 charges
// remaining"). The Go port instead set both value(1) and value(2) to the same
// (maxCharges-1), discarding the player's existing charges instead of adding
// one — an economy/usability bug, not a faithfully-ported quirk.
func TestSpecRecharger(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Recharge Shop", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 8, Keywords: "recharger", ShortDesc: "the recharger"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	proto, _ := w.GetMobPrototype(8)
	recharger := NewMob(proto, 1001)

	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	newWand := func(maxCharges, currCharges int) *ObjectInstance {
		wandProto := &parser.Obj{
			VNum:      3000,
			ShortDesc: "a recharge wand",
			Keywords:  "wand",
			TypeFlag:  ITEM_WAND,
			Values:    [4]int{5, maxCharges, currCharges, 0}, // spell level 5
		}
		return NewObjectInstance(wandProto, -1)
	}

	// A blank or missing name follows C's ordinary missing-object branch.
	specRecharger(w, ch, recharger, "recharge", "")
	if msg := lastMsg(); msg != "The recharger tells you, 'You don't have that!'\r\nThe recharger whaps you upside the head.\r\n" {
		t.Errorf("missing item: got %q", msg)
	}

	// Not a wand/staff: rejected.
	notAWand := NewObjectInstance(&parser.Obj{VNum: 3001, ShortDesc: "a rock", Keywords: "rock", TypeFlag: ITEM_LIGHT}, -1)
	if err := ch.Inventory.AddItem(notAWand); err != nil {
		t.Fatalf("AddItem notAWand: %v", err)
	}
	ch.SetGold(10000)
	specRecharger(w, ch, recharger, "recharge", "rock")
	if msg := lastMsg(); msg != "The recharger tells you, 'Ummm... does that look like a wand or staff to you?'\r\nThe recharger shakes his head in disgust.\r\n" {
		t.Errorf("non-wand item: got %q", msg)
	}
	ch.Inventory.RemoveItem(notAWand)

	// Insufficient gold: rejected, item untouched.
	wand := newWand(10, 5)
	if err := ch.Inventory.AddItem(wand); err != nil {
		t.Fatalf("AddItem wand: %v", err)
	}
	ch.SetGold(0)
	specRecharger(w, ch, recharger, "recharge", "wand")
	if msg := lastMsg(); msg != "The recharger tells you, 'You don't have enough gold!'\r\n" {
		t.Errorf("insufficient gold: got %q", msg)
	}
	if wand.GetValue(1) != 10 || wand.GetValue(2) != 5 {
		t.Fatalf("insufficient gold must not alter charges: max=%d curr=%d", wand.GetValue(1), wand.GetValue(2))
	}

	// Successful recharge: max=10,curr=5 -> max=9,curr=6 (current goes UP by
	// one, max goes DOWN by one — independently, not collapsed to one value).
	ch.SetGold(500) // cost = spellLvl(5)*100 = 500
	specRecharger(w, ch, recharger, "recharge", "wand")
	if wand.GetValue(1) != 9 || wand.GetValue(2) != 6 {
		t.Fatalf("after recharge: max=%d curr=%d, want max=9 curr=6", wand.GetValue(1), wand.GetValue(2))
	}
	if ch.GetGold() != 0 {
		t.Errorf("gold after recharge: got %d, want 0", ch.GetGold())
	}
	if msg := lastMsg(); msg != "The recharger waves his hands around and chants strange things.\r\nThe item now has 6 charges remaining.\r\n" {
		t.Errorf("successful recharge: got %q", msg)
	}

	// The quirky C overflow: recharging at max=2,curr=1 yields max=1,curr=2 —
	// current exceeding max is allowed, matching the recharger's own flavor
	// text, not blocked as "worn out". Use a distinct keyword ("rod") so
	// FindItem can't match the earlier "wand" item still in inventory.
	overflowWand := NewObjectInstance(&parser.Obj{VNum: 3002, ShortDesc: "overflow rod", Keywords: "rod", TypeFlag: ITEM_WAND, Values: [4]int{5, 2, 1, 0}}, -1)
	if err := ch.Inventory.AddItem(overflowWand); err != nil {
		t.Fatalf("AddItem overflowWand: %v", err)
	}
	ch.SetGold(500)
	specRecharger(w, ch, recharger, "recharge", "rod")
	if overflowWand.GetValue(1) != 1 || overflowWand.GetValue(2) != 2 {
		t.Fatalf("overflow recharge: max=%d curr=%d, want max=1 curr=2 (current > max is the C quirk)", overflowWand.GetValue(1), overflowWand.GetValue(2))
	}
	lastMsg()

	// Already fully charged: rejected.
	fullWand := NewObjectInstance(&parser.Obj{VNum: 3003, ShortDesc: "full staff", Keywords: "staff", TypeFlag: ITEM_STAFF, Values: [4]int{5, 10, 10, 0}}, -1)
	if err := ch.Inventory.AddItem(fullWand); err != nil {
		t.Fatalf("AddItem fullWand: %v", err)
	}
	ch.SetGold(500)
	specRecharger(w, ch, recharger, "recharge", "staff")
	if msg := lastMsg(); msg != "The recharger tells you, 'The item does not need recharging, stop wasting my time!'\r\n" {
		t.Errorf("fully charged item: got %q", msg)
	}
	if fullWand.GetValue(1) != 10 || fullWand.GetValue(2) != 10 {
		t.Fatalf("fully charged item must not change: max=%d curr=%d", fullWand.GetValue(1), fullWand.GetValue(2))
	}
}

func TestSpecRechargerListHelpExactAudience(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Recharge Shop", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 8, Keywords: "recharger", ShortDesc: "the recharger"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	outputs := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { outputs[name] += string(msg) }
	actor := NewPlayer(1, "Tester", 1001)
	observer := NewPlayer(2, "Observer", 1001)
	for _, player := range []*Player{actor, observer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}
	proto, _ := w.GetMobPrototype(8)
	recharger := NewMob(proto, 1001)

	roomLines := "The recharger sighs loudly.\r\n" +
		"The recharger says 'I recharge wands and staves, of course!'\r\n" +
		"The recharger says 'My price is 100 coins per spell level per charge..'\r\n" +
		"The recharger says 'So a wand casting heal at level 25 would cost 2500 coins per charge.'\r\n" +
		"The recharger says 'If it had 10 max charges at first, it would have 9 max after charging.\r\n" +
		"The recharger says 'That means that an item could have 1 charge maximum, with 2 charges remaining (after a recharge)'\r\n" +
		"The recharger says 'To recharge an item type: recharge <staff or wand>.'\r\n"
	for _, cmd := range []string{"list", "help"} {
		outputs = map[string]string{}
		if !specRecharger(w, actor, recharger, cmd, "") {
			t.Fatalf("%s was not handled", cmd)
		}
		if got, want := outputs[actor.Name], roomLines+"The recharger stares expectantly at you.\r\n"; got != want {
			t.Errorf("%s actor output = %q, want %q", cmd, got, want)
		}
		if got, want := outputs[observer.Name], roomLines+"The recharger stares expectantly at Tester.\r\n"; got != want {
			t.Errorf("%s observer output = %q, want %q", cmd, got, want)
		}
	}
}
