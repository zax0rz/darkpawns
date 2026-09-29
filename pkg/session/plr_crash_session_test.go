package session

// plr_crash_session_test.go — R5h proofs for the session-layer PLR_CRASH
// sites (shop buy/sell/sell-all) and the WirePlayerSaver guard.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// recordingStore counts SavePlayer writes; every other method panics via
// the nil embedded interface, which no path in these tests reaches.
type recordingStore struct {
	db.GameStore
	saves int
}

func (r *recordingStore) SavePlayer(p *db.PlayerRecord) error {
	r.saves++
	return nil
}

type flagShopHarness struct {
	m *Manager
	w *game.World
	s *Session
}

func newFlagShopHarness(t *testing.T) *flagShopHarness {
	return newFlagShopHarnessWithStore(t, nil)
}

func newFlagShopHarnessWithStore(t *testing.T, store db.GameStore) *flagShopHarness {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Market"}},
		Objs:  []parser.Obj{{VNum: 7101, Keywords: "trinket", ShortDesc: "a trinket", TypeFlag: 1, WearFlags: [4]int{1}, Cost: 10}},
		Mobs:  []parser.Mob{{VNum: 2001, Keywords: "keeper", ShortDesc: "the keeper"}},
		Shops: []parser.ShopProto{{
			VNum: 1, KeeperVNum: 2001,
			Products:  []int{7101},
			BuyTypes:  []int{1},
			BuyProfit: 1.0, SellProfit: 1.0,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, store)
	m.WirePlayerSaver(w)
	s := makeTestSession(t, m, "Shopper", 1001, true)
	m.mu.Lock()
	m.sessions["Shopper"] = s
	m.mu.Unlock()
	s.player.Stats.Str = 15
	// The world must know the player: the ObjectLocation attach arms key
	// player inventory through w.players.
	if err := w.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return &flagShopHarness{m: m, w: w, s: s}
}

// TestShopBuySetsCrashFlag: C shopping_buy hands the object over with
// obj_to_char (shop.c:545).
func TestShopBuySetsCrashFlag(t *testing.T) {
	h := newFlagShopHarness(t)
	keeper, err := h.w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	keeper.Level = 30
	h.s.player.SetGold(1000)
	h.s.player.SetPlrFlag(game.PlrCrash, false)

	// cmdBuy's quantity loop buys while gold lasts; the fixture price is
	// cheap so at least one purchase happens.
	if err := cmdBuy(h.s, []string{"trinket"}); err != nil {
		t.Fatalf("cmdBuy: %v", err)
	}
	if _, ok := h.s.player.Inventory.FindItem("trinket"); !ok {
		t.Fatal("buy did not deliver the object")
	}
	if !h.s.player.NeedsCrashSave() {
		t.Fatal("shop buy did not set PLR_CRASH (C shop.c:545 obj_to_char)")
	}
}

// TestShopSellSetsCrashFlag: C shopping_sell takes the object with
// obj_from_char (shop.c:776) — for both sell and sell all.
func TestShopSellSetsCrashFlag(t *testing.T) {
	h := newFlagShopHarness(t)
	keeper, err := h.w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	keeper.Level = 30
	obj, err := h.w.SpawnObject(7101, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := h.w.MoveObjectToPlayerInventory(obj, h.s.player); err != nil {
		t.Fatalf("carry: %v", err)
	}
	h.s.player.SetPlrFlag(game.PlrCrash, false)

	if err := cmdSell(h.s, []string{"trinket"}); err != nil {
		t.Fatalf("cmdSell: %v", err)
	}
	if _, ok := h.s.player.Inventory.FindItem("trinket"); ok {
		t.Fatal("sell left the object in inventory")
	}
	if !h.s.player.NeedsCrashSave() {
		t.Fatal("shop sell did not set PLR_CRASH (C shop.c:776 obj_from_char)")
	}

	// sell all exercises the same C site per object.
	obj2, err := h.w.SpawnObject(7101, -1)
	if err != nil {
		t.Fatalf("SpawnObject 2: %v", err)
	}
	if err := h.w.MoveObjectToPlayerInventory(obj2, h.s.player); err != nil {
		t.Fatalf("carry 2: %v", err)
	}
	h.s.player.SetPlrFlag(game.PlrCrash, false)
	shop, keeperName := findShopKeeperInRoom(h.s)
	if shop == nil {
		t.Fatal("no shop in room")
	}
	if err := cmdSellAll(h.s, shop, keeperName); err != nil {
		t.Fatalf("cmdSellAll: %v", err)
	}
	if !h.s.player.NeedsCrashSave() {
		t.Fatal("shop sell all did not set PLR_CRASH (C shop.c:776 per object)")
	}
}

// TestWirePlayerSaverRejectsForeignPlayer: a save of a Player object that is
// not the session's player must be reported skipped and write nothing —
// the `set file` path edits a separate JSON-loaded Player under the same
// name, and a live-session write must not be reported as that edit's
// success.
func TestWirePlayerSaverRejectsForeignPlayer(t *testing.T) {
	store := &recordingStore{}
	h := newFlagShopHarnessWithStore(t, store)
	foreign := game.NewPlayer(42, "Shopper", 1001) // same name, other object

	// The production wiring must skip the foreign object: no write, and the
	// seam reports SaveSkipped.
	if res := h.w.SavePlayerRecord(foreign, "set file", game.LoadRoomNowhere, game.SaveCharOnly); res != game.SaveSkipped {
		t.Fatalf("foreign-player save result = %v, want SaveSkipped", res)
	}
	if store.saves != 0 {
		t.Fatalf("foreign-player save wrote %d records", store.saves)
	}
	// ...and still save the live session player through the same seam.
	if res := h.w.SavePlayerRecord(h.s.player, "live", game.LoadRoomNowhere, game.SaveCharOnly); res != game.SaveSucceeded {
		t.Fatalf("live-player save result = %v, want SaveSucceeded", res)
	}
	if store.saves != 1 {
		t.Fatalf("live-player save wrote %d records, want 1", store.saves)
	}
}
