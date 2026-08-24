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
// (act.item.c:737): actor "Ok.", recipient "$n gives you N gold coins.", the
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
	if got := msgOf(msgs, "Giver"); !strings.Contains(got, "Ok.") {
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
		heavy := newTransferItem(8200, "an anvil", "anvil", 1)
		heavy.SetWeight(vict.Inventory.GetCapacity()*10 + 1)
		registerTransferObject(w, heavy)
		if err := w.MoveObjectToPlayerInventory(heavy, ch); err != nil {
			t.Fatalf("seat anvil: %v", err)
		}
		w.performGive(ch, vict, heavy)
		if got := msgOf(msgs, "Giver"); !strings.Contains(got, "can't carry that much weight.") {
			t.Fatalf("weight byte: got %q", got)
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
