package systems

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeShopItem is a test helper that builds a minimal tradeable item instance.
func makeShopItem(vnum int, typeFlag int) *game.ObjectInstance {
	proto := &parser.Obj{
		VNum:      vnum,
		Keywords:  "test item",
		ShortDesc: "a test item",
		LongDesc:  "A test item lies here.",
		Cost:      100,
		TypeFlag:  typeFlag,
	}
	return game.NewObjectInstance(proto, -1)
}

// TestProcessRepair_HappyPath covers ProcessRepair charging the player and
// returning the item (shop_manager.go:283 — previously 0% coverage).
func TestProcessRepair_HappyPath(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Smithy", 3001)

	player := &game.Player{Name: "Buyer", Gold: 1000}
	player.Inventory = game.NewInventory()

	item := makeShopItem(1001, 5) // weapon
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item: %v", err)
	}

	// damage=10, repair cost = damage * shop.RepairCost (default 10) = 100.
	ok, msg := manager.ProcessRepair(shop, player, item, 10)
	if !ok {
		t.Fatalf("ProcessRepair failed: %s", msg)
	}
	if player.Gold != 900 {
		t.Errorf("gold = %d, want 900 after 100-gold repair", player.Gold)
	}
	// Item is returned to the player.
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("inventory count = %d, want 1 (item returned after repair)", player.Inventory.GetItemCount())
	}
}

// TestProcessRepair_NotEnoughGold covers the insufficient-funds branch:
// the player keeps their item and is not charged.
func TestProcessRepair_NotEnoughGold(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Smithy", 3001)

	player := &game.Player{Name: "Poor", Gold: 10}
	player.Inventory = game.NewInventory()

	item := makeShopItem(1001, 5)
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item: %v", err)
	}

	ok, _ := manager.ProcessRepair(shop, player, item, 10) // costs 100
	if ok {
		t.Error("repair should fail: player cannot afford 100 gold")
	}
	if player.Gold != 10 {
		t.Errorf("gold = %d, want 10 (should not be charged on failure)", player.Gold)
	}
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("inventory count = %d, want 1 (item must be returned)", player.Inventory.GetItemCount())
	}
}

// TestProcessIdentify_HappyPath covers ProcessIdentify charging the player
// and returning the item (shop_manager.go:335 — previously 0% coverage).
func TestProcessIdentify_HappyPath(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Sage", 3001)

	player := &game.Player{Name: "Buyer", Gold: 100}
	player.Inventory = game.NewInventory()

	item := makeShopItem(1001, 5)
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item: %v", err)
	}

	// Default identify cost is 5.
	ok, msg := manager.ProcessIdentify(shop, player, item)
	if !ok {
		t.Fatalf("ProcessIdentify failed: %s", msg)
	}
	if player.Gold != 95 {
		t.Errorf("gold = %d, want 95 after 5-gold identify", player.Gold)
	}
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("inventory count = %d, want 1 (item returned after identify)", player.Inventory.GetItemCount())
	}
}

// TestProcessIdentify_NotEnoughGold covers the insufficient-funds branch.
func TestProcessIdentify_NotEnoughGold(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Sage", 3001)

	player := &game.Player{Name: "Poor", Gold: 2}
	player.Inventory = game.NewInventory()

	item := makeShopItem(1001, 5)
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item: %v", err)
	}

	ok, _ := manager.ProcessIdentify(shop, player, item) // costs 5
	if ok {
		t.Error("identify should fail: player cannot afford 5 gold")
	}
	if player.Gold != 2 {
		t.Errorf("gold = %d, want 2 (should not be charged on failure)", player.Gold)
	}
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("inventory count = %d, want 1 (item must be returned)", player.Inventory.GetItemCount())
	}
}

// TestShopRestock covers Shop.Restock: items are added up to MaxItems and
// keeper gold is replenished from the bank (shop.go:393 — previously 0%).
func TestShopRestock(t *testing.T) {
	shop := NewShop(1, 1001, "General Store", 3001)
	shop.ItemTypes = []int{1}
	shop.MaxItems = 5
	shop.RestockInterval = 100
	shop.Gold = 10
	shop.BankAccount = 500

	proto := &parser.Obj{
		VNum:     1001,
		Keywords: "bread loaf",
		Cost:     5,
		TypeFlag: 1,
	}

	// First restock: tick has advanced past interval.
	restocked := shop.Restock([]*parser.Obj{proto}, 200)
	if restocked != 1 {
		t.Errorf("restocked = %d, want 1", restocked)
	}
	if len(shop.GetInventory()) != 1 {
		t.Errorf("inventory = %d items, want 1", len(shop.GetInventory()))
	}
	// Keeper gold replenished from bank (src/shop.c:808).
	if shop.Gold != 500 {
		t.Errorf("gold = %d, want 500 (replenished from BankAccount)", shop.Gold)
	}

	// Second restock before interval elapses: no-op.
	restocked = shop.Restock([]*parser.Obj{proto}, 250)
	if restocked != 0 {
		t.Errorf("restocked = %d, want 0 (interval not elapsed)", restocked)
	}
}

// TestShopIsOpen covers both normal and overnight opening-hour windows
// (shop.go:340 — previously 0%).
func TestShopIsOpen(t *testing.T) {
	shop := NewShop(1, 1001, "Store", 3001)

	// Normal hours: 6..22.
	shop.OpenHour = 6
	shop.CloseHour = 22
	if !shop.IsOpen(12) {
		t.Error("shop should be open at noon (6-22)")
	}
	if shop.IsOpen(23) {
		t.Error("shop should be closed at 23 (6-22)")
	}

	// Overnight hours: 20..4.
	shop.OpenHour = 20
	shop.CloseHour = 4
	if !shop.IsOpen(21) {
		t.Error("overnight shop should be open at 21 (20-4)")
	}
	if !shop.IsOpen(2) {
		t.Error("overnight shop should be open at 02:00 (20-4)")
	}
	if shop.IsOpen(10) {
		t.Error("overnight shop should be closed at 10 (20-4)")
	}
}

// TestShopGoldMethods exercises CanAffordToBuy and DeductGold directly
// (shop.go:354,361 — previously 0%). DeductGold must clamp at 0.
func TestShopGoldMethods(t *testing.T) {
	shop := NewShop(1, 1001, "Store", 3001)
	shop.Gold = 100

	if !shop.CanAffordToBuy(100) {
		t.Error("CanAffordToBuy(100) should be true when Gold == 100")
	}
	if shop.CanAffordToBuy(101) {
		t.Error("CanAffordToBuy(101) should be false when Gold == 100")
	}

	shop.DeductGold(30)
	if shop.Gold != 70 {
		t.Errorf("gold = %d after deducting 30, want 70", shop.Gold)
	}

	// Deducting more than available clamps to 0, not negative.
	shop.DeductGold(1000)
	if shop.Gold != 0 {
		t.Errorf("gold = %d after overdraw, want 0 (clamped)", shop.Gold)
	}
}

// TestShopRestockAll covers ShopManager.RestockAll across multiple shops
// (shop_manager.go:384 — previously 0%).
func TestShopRestockAll(t *testing.T) {
	manager := NewShopManager()
	shop1 := manager.CreateShopConcrete(1001, "Store A", 3001)
	shop1.ItemTypes = []int{1}
	shop1.RestockInterval = 1
	shop2 := manager.CreateShopConcrete(1002, "Store B", 3002)
	shop2.ItemTypes = []int{1}
	shop2.RestockInterval = 1

	proto := &parser.Obj{VNum: 1001, Cost: 5, TypeFlag: 1}

	total := manager.RestockAll([]*parser.Obj{proto}, 100)
	if total != 2 {
		t.Errorf("RestockAll returned %d, want 2 (one per shop)", total)
	}
}
