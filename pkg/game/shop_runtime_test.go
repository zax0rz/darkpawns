package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShopAccountPurchaseAndSaleBanking(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}},
		Mobs:  []parser.Mob{{VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	keeper, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	keeper.SetGold(5000)
	shops := NewShopManager()
	shops.AddShop(&Shop{VNum: 1, KeeperVNum: 5001, Flags: ShopFlagUsesBank, Bank: 3000})
	w.SetShopManager(shops)

	if !w.TryPayShopSale(1, keeper, 6000) {
		t.Fatal("sale covered by keeper cash and bank was rejected")
	}
	w.RebalanceShopBank(1, keeper)
	shop, coins, ok := w.SnapshotShopAccount(1, keeper)
	if !ok || coins != 2000 || shop.Bank != 0 {
		t.Fatalf("after bank-funded sale: found=%v coins=%d bank=%d, want true/2000/0", ok, coins, shop.Bank)
	}

	if w.TryPayShopSale(1, keeper, 2001) {
		t.Fatal("sale beyond remaining funds was accepted")
	}

	// A purchase above MAX_OUTSIDE_BANK leaves only that cap with the keeper.
	w.CreditShopPurchase(1, keeper, 14000)
	shop, coins, _ = w.SnapshotShopAccount(1, keeper)
	if coins != 15000 || shop.Bank != 1000 {
		t.Fatalf("after purchase overflow: coins=%d bank=%d, want 15000/1000", coins, shop.Bank)
	}
	if w.TryPayShopSale(1, keeper, 16001) {
		t.Fatal("sale beyond combined cash and bank was accepted")
	}
	shopAfter, coinsAfter, _ := w.SnapshotShopAccount(1, keeper)
	if coinsAfter != coins || shopAfter.Bank != shop.Bank {
		t.Fatalf("rejected sale changed account: coins=%d bank=%d, want %d/%d", coinsAfter, shopAfter.Bank, coins, shop.Bank)
	}
}

func TestReplacingShopPreservesRuntimeBank(t *testing.T) {
	shops := NewShopManager()
	shops.AddShop(&Shop{VNum: 1, Bank: 1234})
	shops.ReplaceShop(&Shop{VNum: 1, KeeperVNum: 5001})
	if got := shops.GetShopByVNum(1).Bank; got != 1234 {
		t.Fatalf("shop edit reset live bank to %d, want 1234", got)
	}
}
