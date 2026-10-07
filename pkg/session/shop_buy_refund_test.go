package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Claude review on #1830: the atomic spend precedes the item hand-over in the
// buy loop, so a failed AddItem must refund — otherwise a weight-rejected buy
// eats the player's gold. Drop-gold already refunds on failure.
//
// The rejection must happen INSIDE AddItem (weight gate) with IsFull() false,
// after the spend; and a control phase proves the buy path is live so the
// test cannot pass vacuously (the first draft never placed the keeper in the
// room and cmdBuy exited before touching gold at all).
func TestCmdBuyRefundsWhenInventoryAddFails(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Shop Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5001, ShortDesc: "the shopkeeper", Level: 5,
			HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{{
			VNum: 6001, Keywords: "rock", ShortDesc: "a heavy rock",
			LongDesc: "A heavy rock.", Cost: 10, Weight: 50,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, nil)

	if _, err := world.SpawnMob(5001, 1001); err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{
		VNum: 1, KeeperVNum: 5001,
		SellTypes: []int{6001}, BuyTypes: []int{},
		ProfitBuy: 1.0, ProfitSell: 1.0,
	})
	world.SetShopManager(sm)

	s := makeTestSession(t, m, "Buyer", 1001, true)
	s.player.SetGold(100)

	// Control: with carry weight unenforced the buy must succeed and spend.
	s.player.Inventory.MaxWeight = 0
	if err := cmdBuy(s, []string{"rock"}); err != nil {
		t.Fatalf("cmdBuy control: %v", err)
	}
	if got := s.player.GetGold(); got != 90 {
		t.Fatalf("control: gold = %d, want 90 — buy path not live (shop or keeper not found)", got)
	}

	// Rejection: cap carry weight below the already-carried rock; the count
	// gate stays open, so AddItem itself fails after the spend.
	s.player.Inventory.MaxWeight = 10
	if err := cmdBuy(s, []string{"rock"}); err != nil {
		t.Fatalf("cmdBuy rejection: %v", err)
	}
	if got := s.player.GetGold(); got != 90 {
		t.Fatalf("gold after weight-rejected buy = %d, want 90 (refund missing — the failed add ate the gold)", got)
	}
}

// A rejected hand-over must not leave a registered object behind: it would sit
// in the world registry with no location, visible to world-wide object scans.
func TestBuyRejectedAtWeightLeavesNoRegisteredObject(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Shop Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5001, ShortDesc: "the shopkeeper", Level: 5,
			HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{{
			VNum: 6001, Keywords: "rock", ShortDesc: "a heavy rock",
			LongDesc: "A heavy rock.", Cost: 10, Weight: 50,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, nil)
	if _, err := world.SpawnMob(5001, 1001); err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{VNum: 1, KeeperVNum: 5001, SellTypes: []int{6001}, BuyTypes: []int{}, ProfitBuy: 1.0, ProfitSell: 1.0})
	world.SetShopManager(sm)

	s := makeTestSession(t, m, "Buyer", 1001, true)
	s.player.SetGold(100)
	s.player.Inventory.MaxWeight = 10 // below one rock: AddItem fails after the spend
	before := len(world.GetAllObjects())
	if err := cmdBuy(s, []string{"rock"}); err != nil {
		t.Fatalf("cmdBuy: %v", err)
	}
	if got := s.player.GetGold(); got != 100 {
		t.Fatalf("precondition: gold = %d, want 100 (the buy must have been rejected and refunded)", got)
	}
	if after := len(world.GetAllObjects()); after != before {
		t.Fatalf("registry grew from %d to %d after a rejected buy: the unsold rock stayed registered", before, after)
	}
}
