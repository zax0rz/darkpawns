package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCanTakeObjGateMessages proves the three can_take_obj failure branches
// (act.item.c:158) each emit their canonical byte and return false:
// carry-count, carry-weight, and the ITEM_WEAR_TAKE flag.
func TestCanTakeObjGateMessages(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		for i := 0; i < ch.MaxCarryItems(); i++ {
			obj := newTransferItem(6000+i, "a pebble", "pebble", 1)
			obj.SetWeight(0)
			registerTransferObject(w, obj)
			if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
				t.Fatalf("fill inventory: %v", err)
			}
		}
		extra := newTransferItem(6999, "a coin", "coin", 1)
		extra.SetWeight(0)
		if w.canTakeObj(ch, extra) {
			t.Fatal("full inventory should reject via count gate")
		}
		if got := lastMsg(); !strings.Contains(got, "you can't carry that many items.") {
			t.Fatalf("count gate byte: got %q", got)
		}
	})

	t.Run("weight", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		obj := newTransferItem(6100, "a boulder", "boulder", 1)
		obj.SetWeight(ch.MaxCarryWeight() + 1)
		if w.canTakeObj(ch, obj) {
			t.Fatal("overweight object should reject via weight gate")
		}
		if got := lastMsg(); !strings.Contains(got, "you can't carry that much weight.") {
			t.Fatalf("weight gate byte: got %q", got)
		}
	})

	t.Run("take-flag", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		obj := newTransferItem(6200, "a fountain", "fountain", 0) // no ITEM_WEAR_TAKE
		obj.SetWeight(0)
		if w.canTakeObj(ch, obj) {
			t.Fatal("object without TAKE should reject")
		}
		if got := lastMsg(); !strings.Contains(got, "you can't take that!") {
			t.Fatalf("take-flag gate byte: got %q", got)
		}
	})
}

// newMoneyItem builds an ITEM_MONEY object carrying `amount` coins in value 0.
func newMoneyItem(vnum, amount int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: "a pile of gold coins",
		Keywords:  "coins gold",
		TypeFlag:  ITEM_MONEY,
		Values:    [4]int{amount, 0, 0, 0},
		WearFlags: [4]int{1},
	}, -1)
}

// TestGetCheckMoneyMultiAndSingle proves get_check_money (act.item.c:180): a
// pile >1 prints "There were N coins.", a single coin is silent, and both add
// the exact gold and consume the object.
func TestGetCheckMoneyMultiAndSingle(t *testing.T) {
	t.Run("multi", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		ch.SetGold(10)
		money := newMoneyItem(6300, 5)
		registerTransferObject(w, money)
		if err := w.MoveObjectToPlayerInventory(money, ch); err != nil {
			t.Fatalf("seat money: %v", err)
		}
		w.getCheckMoney(ch, money)
		if ch.GetGold() != 15 {
			t.Fatalf("gold after 5-coin pile: got %d want 15", ch.GetGold())
		}
		if got := lastMsg(); !strings.Contains(got, "There were 5 coins.") {
			t.Fatalf("multi-coin byte: got %q", got)
		}
	})

	t.Run("single", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		ch.SetGold(10)
		money := newMoneyItem(6301, 1)
		registerTransferObject(w, money)
		if err := w.MoveObjectToPlayerInventory(money, ch); err != nil {
			t.Fatalf("seat money: %v", err)
		}
		w.getCheckMoney(ch, money)
		if ch.GetGold() != 11 {
			t.Fatalf("gold after single coin: got %d want 11", ch.GetGold())
		}
		if got := lastMsg(); strings.Contains(got, "There were") {
			t.Fatalf("single coin must be silent, got %q", got)
		}
	})
}

// newInvContainer returns an open container already seated in ch's inventory.
func newInvContainer(t *testing.T, w *World, ch *Player, vnum int) *ObjectInstance {
	t.Helper()
	cont := NewObjectInstance(&parser.Obj{
		VNum: vnum, ShortDesc: "a leather sack", Keywords: "sack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{100, 0, -1, 0}, WearFlags: [4]int{1},
	}, -1)
	registerTransferObject(w, cont)
	if err := w.MoveObjectToPlayerInventory(cont, ch); err != nil {
		t.Fatalf("seat container: %v", err)
	}
	return cont
}

// TestGetFromInvContainerBypassesCanTake proves perform_get_from_container
// (act.item.c:197): mode==FIND_OBJ_INV skips can_take_obj entirely, so a
// no-TAKE object still leaves your own container, whereas the room path is
// gated by the TAKE flag.
func TestGetFromInvContainerBypassesCanTake(t *testing.T) {
	t.Run("inv-bypass", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		cont := newInvContainer(t, w, ch, 6400)
		noTake := newTransferItem(6401, "a heavy idol", "idol", 0) // no TAKE
		noTake.SetWeight(0)
		registerTransferObject(w, noTake)
		if err := w.MoveObjectToContainer(noTake, cont); err != nil {
			t.Fatalf("seat idol: %v", err)
		}
		w.performGetFromContainer(ch, noTake, cont, findObjInv)
		if _, ok := ch.Inventory.FindItem("idol"); !ok {
			t.Fatal("FIND_OBJ_INV get should bypass TAKE gate and move the idol")
		}
		if got := lastMsg(); strings.Contains(got, "you can't take that!") {
			t.Fatalf("inv path must not run the TAKE gate, got %q", got)
		}
	})

	t.Run("room-path-gated", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		cont := newInvContainer(t, w, ch, 6410)
		noTake := newTransferItem(6411, "a heavy idol", "idol", 0)
		noTake.SetWeight(0)
		registerTransferObject(w, noTake)
		if err := w.MoveObjectToContainer(noTake, cont); err != nil {
			t.Fatalf("seat idol: %v", err)
		}
		w.performGetFromContainer(ch, noTake, cont, findObjRoom)
		if _, ok := ch.Inventory.FindItem("idol"); ok {
			t.Fatal("FIND_OBJ_ROOM get of a no-TAKE object must be rejected")
		}
		if got := lastMsg(); !strings.Contains(got, "you can't take that!") {
			t.Fatalf("room path should run the TAKE gate, got %q", got)
		}
	})
}

// TestGetFromInvContainerArmsFull proves the inv path's distinct full-arms
// byte (act.item.c:202): even bypassing can_take_obj, a full inventory blocks
// with "you can't hold any more items." rather than the can_take_obj wording.
func TestGetFromInvContainerArmsFull(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	cont := newInvContainer(t, w, ch, 6500)
	item := newTransferItem(6501, "a gem", "gem", 1)
	item.SetWeight(0)
	registerTransferObject(w, item)
	if err := w.MoveObjectToContainer(item, cont); err != nil {
		t.Fatalf("seat gem: %v", err)
	}
	// Fill remaining inventory slots to the carry cap (the container already
	// occupies one slot).
	for i := len(ch.Inventory.Items); i < ch.MaxCarryItems(); i++ {
		pebble := newTransferItem(6600+i, "a pebble", "pebble", 1)
		pebble.SetWeight(0)
		registerTransferObject(w, pebble)
		if err := w.MoveObjectToPlayerInventory(pebble, ch); err != nil {
			t.Fatalf("fill inventory: %v", err)
		}
	}
	w.performGetFromContainer(ch, item, cont, findObjInv)
	if got := lastMsg(); !strings.Contains(got, "you can't hold any more items.") {
		t.Fatalf("inv arms-full byte: got %q", got)
	}
}

// TestGetFiresOnGetRoomScript proves get runs the room's onget trigger after a
// successful pickup (act.item.c:283), and only when the room carries RS_ONGET
// (1<<4). Uses the shared movementTriggerRecorder as a no-op script engine.
func TestGetFiresOnGetRoomScript(t *testing.T) {
	run := func(t *testing.T, scriptFuncs int) []string {
		t.Helper()
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{
			{VNum: 1001, Name: "Vault", ScriptName: "room.lua", ScriptFunctions: scriptFuncs},
		}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		w.MessageSink = func(string, []byte) {}
		ch := NewPlayer(1, "Taker", 1001)
		if err := w.AddPlayer(ch); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		gem := newTransferItem(7000, "a gem", "gem", 1)
		gem.SetWeight(0)
		w.AddItemToRoom(gem, 1001)

		events := make([]string, 0, 1)
		prev := ScriptEngine
		ScriptEngine = movementTriggerRecorder{events: &events}
		t.Cleanup(func() { ScriptEngine = prev })

		w.DoGet(ch, "gem")
		if _, ok := ch.Inventory.FindItem("gem"); !ok {
			t.Fatal("gem should have been picked up")
		}
		return events
	}

	t.Run("fires-with-flag", func(t *testing.T) {
		if got := run(t, 1<<4); len(got) != 1 || got[0] != "room.lua:onget" {
			t.Fatalf("onget trigger events = %v, want [room.lua:onget]", got)
		}
	})

	t.Run("silent-without-flag", func(t *testing.T) {
		if got := run(t, 0); len(got) != 0 {
			t.Fatalf("no onget without RS_ONGET; got %v", got)
		}
	})
}
