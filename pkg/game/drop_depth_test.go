package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPerformDropGoldSuccess proves perform_drop_gold's success branch
// (act.item.c:456): it prints "You drop some gold.", deducts the coins, and
// sets a wait state.
func TestPerformDropGoldSuccess(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	ch.SetGold(100)
	w.performDropGold(ch, 10)
	if ch.GetGold() != 90 {
		t.Fatalf("gold after dropping 10: got %d want 90", ch.GetGold())
	}
	if got := lastMsg(); !strings.Contains(got, "You drop some gold.") {
		t.Fatalf("gold-drop actor byte: got %q", got)
	}
	if ch.GetWaitState() <= 0 {
		t.Fatalf("dropping gold should set a wait state (coin-bomb guard)")
	}
}

// TestDropFiresOnDropRoomScript proves drop runs the room's ondrop trigger
// after a successful drop (act.item.c:495), and only when the room carries
// RS_ONDROP (1<<3).
func TestDropFiresOnDropRoomScript(t *testing.T) {
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
		ch := NewPlayer(1, "Dropper", 1001)
		if err := w.AddPlayer(ch); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		gem := newTransferItem(7100, "a gem", "gem", 1)
		gem.SetWeight(0)
		registerTransferObject(w, gem)
		if err := w.MoveObjectToPlayerInventory(gem, ch); err != nil {
			t.Fatalf("seat gem: %v", err)
		}

		events := make([]string, 0, 1)
		prev := ScriptEngine
		ScriptEngine = movementTriggerRecorder{events: &events}
		t.Cleanup(func() { ScriptEngine = prev })

		w.DoDrop(ch, "gem")
		if _, ok := ch.Inventory.FindItem("gem"); ok {
			t.Fatal("gem should have been dropped")
		}
		return events
	}

	t.Run("fires-with-flag", func(t *testing.T) {
		if got := run(t, 1<<3); len(got) != 1 || got[0] != "room.lua:ondrop" {
			t.Fatalf("ondrop events = %v, want [room.lua:ondrop]", got)
		}
	})
	t.Run("silent-without-flag", func(t *testing.T) {
		if got := run(t, 0); len(got) != 0 {
			t.Fatalf("no ondrop without RS_ONDROP; got %v", got)
		}
	})
}
