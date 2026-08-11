package systems

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newSellTestShop returns a shop that buys TypeFlag 1 items and starts solvent.
func newSellTestShop(manager *ShopManager) *Shop {
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1}
	return shop
}

// newSellTestItem returns an item worth 100 gold of TypeFlag 1, which the
// keeper buys for 50 (50% of cost, no CHA modifier).
func newSellTestItem() *game.ObjectInstance {
	return game.NewObjectInstance(&parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}, -1)
}

// TestProcessSell_ShopCannotPay_ItemStaysWithPlayer verifies that when the
// keeper lacks gold for an item, the sale is rejected without removing the item
// from the player's inventory (DP-1224).
func TestProcessSell_ShopCannotPay_ItemStaysWithPlayer(t *testing.T) {
	manager := NewShopManager()
	shop := newSellTestShop(manager)
	shop.Gold = 40 // Buy price is 50; keeper can't afford it.

	player := &game.Player{Name: "Test Player", Gold: 100}
	player.Inventory = game.NewInventory()
	item := newSellTestItem()
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item to player: %v", err)
	}

	success, message := manager.processSell(shop, player, item)
	if success {
		t.Fatalf("sell succeeded, want rejection: %s", message)
	}
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("item count in player inventory = %d, want 1", player.Inventory.GetItemCount())
	}
	if player.Gold != 100 {
		t.Errorf("player gold = %d, want 100", player.Gold)
	}
	if shop.Gold != 40 {
		t.Errorf("shop gold = %d, want 40", shop.Gold)
	}
}

// TestProcessSell_ShopFull_ItemStaysWithPlayer verifies that when the shop is
// at MaxItems, the sale is rejected with the item still in the player's
// inventory and no gold moved on either side (DP-1224).
func TestProcessSell_ShopFull_ItemStaysWithPlayer(t *testing.T) {
	manager := NewShopManager()
	shop := newSellTestShop(manager)
	shop.MaxItems = 1
	shop.Gold = 10000
	shop.AddItem(newSellTestItem()) // fills the shop

	player := &game.Player{Name: "Test Player", Gold: 100}
	player.Inventory = game.NewInventory()
	item := newSellTestItem()
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item to player: %v", err)
	}

	success, message := manager.processSell(shop, player, item)
	if success {
		t.Fatalf("sell succeeded, want rejection: %s", message)
	}
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("item count in player inventory = %d, want 1", player.Inventory.GetItemCount())
	}
	if player.Gold != 100 {
		t.Errorf("player gold = %d, want 100", player.Gold)
	}
	if shop.Gold != 10000 {
		t.Errorf("shop gold = %d, want 10000", shop.Gold)
	}
}

// TestProcessSell_Success verifies the happy path still works after the
// check-first reorder: item moves to the shop, the player is paid, and the
// keeper gold is reduced (DP-1224).
func TestProcessSell_Success(t *testing.T) {
	manager := NewShopManager()
	shop := newSellTestShop(manager)
	shop.Gold = 10000

	player := &game.Player{Name: "Test Player", Gold: 100}
	player.Inventory = game.NewInventory()
	item := newSellTestItem()
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item to player: %v", err)
	}

	success, message := manager.processSell(shop, player, item)
	if !success {
		t.Fatalf("sell failed: %s", message)
	}
	if player.Inventory.GetItemCount() != 0 {
		t.Errorf("item count in player inventory = %d, want 0", player.Inventory.GetItemCount())
	}
	if len(shop.GetInventory()) != 1 {
		t.Errorf("shop inventory size = %d, want 1", len(shop.GetInventory()))
	}
	if player.Gold != 150 { // 100 + 50 buy price
		t.Errorf("player gold = %d, want 150", player.Gold)
	}
	if shop.Gold != 9950 { // 10000 - 50 buy price
		t.Errorf("shop gold = %d, want 9950", shop.Gold)
	}
}
