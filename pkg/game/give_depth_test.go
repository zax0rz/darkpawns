package game

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newGiveTestWorld builds a world with a giver and a recipient in one room,
// capturing each player's output separately.
func newGiveTestWorld(t *testing.T) (*World, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	msgs := map[string]*strings.Builder{}
	w.MessageSink = func(name string, m []byte) {
		if msgs[name] == nil {
			msgs[name] = &strings.Builder{}
		}
		msgs[name].Write(m)
	}
	ch := NewPlayer(1, "Giver", 1001)
	vict := NewPlayer(2, "Taker", 1001)
	for _, p := range []*Player{ch, vict} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}
	return w, ch, vict, msgs
}

func msgOf(msgs map[string]*strings.Builder, name string) string {
	if b := msgs[name]; b != nil {
		return b.String()
	}
	return ""
}

// TestPerformGiveGoldSuccess proves perform_give_gold's success branch
// (act.item.c:737): actor "Okay.\r\n", recipient "$n gives you N gold coins.", the
// exact gold transfer — and that it no longer deadlocks on the player mutex.
func TestPerformGiveGoldSuccess(t *testing.T) {
	w, ch, vict, msgs := newGiveTestWorld(t)
	ch.SetGold(100)
	vict.SetGold(5)

	done := make(chan struct{})
	go func() { w.performGiveGold(ch, vict, 10); close(done) }()
	select {
	case <-done:
	case <-timeoutAfter():
		t.Fatal("performGiveGold deadlocked")
	}

	if ch.GetGold() != 90 || vict.GetGold() != 15 {
		t.Fatalf("gold transfer: giver=%d taker=%d, want 90/15", ch.GetGold(), vict.GetGold())
	}
	if got := msgOf(msgs, "Giver"); !strings.Contains(got, "Okay.\r\n") {
		t.Fatalf("giver byte: got %q", got)
	}
	if got := msgOf(msgs, "Taker"); !strings.Contains(got, "Giver gives you 10 gold coins.") {
		t.Fatalf("recipient byte: got %q", got)
	}
}

// TestPerformGiveHandsFullAndWeight proves perform_give's recipient gates
// (act.item.c:692,696): a full-handed recipient and an over-weight recipient
// each reject with their canonical byte.
func TestPerformGiveHandsFullAndWeight(t *testing.T) {
	t.Run("hands-full", func(t *testing.T) {
		w, ch, vict, msgs := newGiveTestWorld(t)
		gift := newTransferItem(8000, "a gift", "gift", 1)
		gift.SetWeight(0)
		registerTransferObject(w, gift)
		if err := w.MoveObjectToPlayerInventory(gift, ch); err != nil {
			t.Fatalf("seat gift: %v", err)
		}
		for i := 0; i < vict.Inventory.GetCapacity(); i++ {
			p := newTransferItem(8100+i, "a pebble", "pebble", 1)
			p.SetWeight(0)
			registerTransferObject(w, p)
			if err := w.MoveObjectToPlayerInventory(p, vict); err != nil {
				t.Fatalf("fill recipient: %v", err)
			}
		}
		w.performGive(ch, vict, gift)
		if got := msgOf(msgs, "Giver"); !strings.Contains(got, "seems to have") || !strings.Contains(got, "hands full") {
			t.Fatalf("hands-full byte: got %q", got)
		}
		if _, ok := vict.Inventory.FindItem("gift"); ok {
			t.Fatal("gift should not transfer to a full-handed recipient")
		}
	})

	t.Run("weight", func(t *testing.T) {
		w, ch, vict, msgs := newGiveTestWorld(t)
		// A strong giver can hold an item heavier than the recipient's limit.
		ch.Stats.Str = 25
		ch.CopyBaseAttributes()
		heavy := newTransferItem(8200, "an anvil", "anvil", 1)
		heavy.SetWeight(vict.MaxCarryWeight() + 1)
		registerTransferObject(w, heavy)
		if err := w.MoveObjectToPlayerInventory(heavy, ch); err != nil {
			t.Fatalf("seat anvil: %v", err)
		}
		w.performGive(ch, vict, heavy)
		if got := msgOf(msgs, "Giver"); !strings.Contains(got, "can't carry that much weight.") {
			t.Fatalf("weight byte: got %q", got)
		}
	})

	// A weak recipient's limit is str_app[STRENGTH_APPLY_INDEX].carry_w
	// (src/utils.h:448), not a multiple of the item-count capacity (DP-1516
	// item 1: STR 3 -> 10, so an 8-weight item on top of 3 carried is refused at
	// the weight gate, not later with "can't carry any more").
	t.Run("weak recipient uses str_app carry_w", func(t *testing.T) {
		w, ch, vict, msgs := newGiveTestWorld(t)
		vict.Stats.Str = 3
		vict.Stats.StrAdd = 0
		vict.CopyBaseAttributes()
		if got := vict.MaxCarryWeight(); got != 10 {
			t.Fatalf("MaxCarryWeight(3) = %d, want C's 10 (str_app[3])", got)
		}
		carried := newTransferItem(8300, "a rock", "rock", 1)
		carried.SetWeight(3)
		registerTransferObject(w, carried)
		if err := w.MoveObjectToPlayerInventory(carried, vict); err != nil {
			t.Fatalf("seat rock: %v", err)
		}
		sword := newTransferItem(8301, "a sword", "sword", 1)
		sword.SetWeight(8)
		registerTransferObject(w, sword)
		if err := w.MoveObjectToPlayerInventory(sword, ch); err != nil {
			t.Fatalf("seat sword: %v", err)
		}
		w.performGive(ch, vict, sword)
		if got := msgOf(msgs, "Giver"); !strings.Contains(got, "can't carry that much weight.") {
			t.Fatalf("weak-recipient weight byte: got %q", got)
		}
		if strings.Contains(msgOf(msgs, "Giver"), "can't carry any more") {
			t.Fatal("weak recipient hit the move-failure arm instead of the weight gate")
		}
	})
}

func timeoutAfter() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		// 2s is ample for a non-blocking call; a deadlock never closes.
		time.Sleep(2 * time.Second)
		close(ch)
	}()
	return ch
}
