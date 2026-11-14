package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C shopping_buy stops when the next matching object is not same_obj as the
// last purchase and tells the buyer "I only have %d to sell you."
// (src/shop.c:541-560). The oracle give-object fixture can deliver only one
// stock group in the shipped world (the second G line's object never reaches
// the keeper's list), so the loop's identity stop is proved at the session
// boundary with two same-keyword groups.
func TestCmdBuyStopsOnSecondStockGroup(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Market", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant",
			Level: 10, Gold: 1000, HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
			Int: 11, Wis: 11,
		}},
		Objs: []parser.Obj{
			{VNum: 6001, Keywords: "apple red", ShortDesc: "a red apple", TypeFlag: int(game.ItemOther), Cost: 10, Weight: 1},
			{VNum: 6004, Keywords: "apple green", ShortDesc: "a green apple", TypeFlag: int(game.ItemOther), Cost: 20, Weight: 1},
		},
		Shops: []parser.ShopProto{{
			VNum: 77, KeeperVNum: 5001, Rooms: []int{1001}, Products: nil,
			BuyTypes: []int{int(game.ItemOther)}, BuyProfit: 1, SellProfit: 1,
			OpenHour1: 0, CloseHour1: 28, OpenHour2: 0, CloseHour2: 28,
			Messages: [7]string{"%s No such item.", "%s You don't have that.", "%s I don't buy that.", "%s I can't afford that!", "%s You're too poor!", "%s That'll be %d coins, thanks.", "%s I'll give you %d coins for that."},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	keeper, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	keeper.SetGold(1000)
	addShopKeeperStock(t, w, keeper, 6001)
	addShopKeeperStock(t, w, keeper, 6004)

	buyer := addShopOutputSession(t, m, "GroupBuyer")
	buyer.player.SetGold(1000)
	buyer.player.Inventory.MaxWeight = 100
	oldRoller := combat.GetRoller()
	combat.SetRoller(shopStockRoller{})
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	if err := cmdBuy(buyer, []string{"2", "apple"}); err != nil {
		t.Fatalf("cmdBuy two apples: %v", err)
	}
	out := renderedOutput(buyer)
	if !strings.Contains(out, "I only have 1 to sell you.") {
		t.Fatalf("buy 2 across two identity groups = %q, want C's one-group stop", out)
	}
	if got := len(buyer.player.Inventory.FindItems("apple")); got != 1 {
		t.Fatalf("bought across two groups = %d items, want 1", got)
	}
}
