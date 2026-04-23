package systems

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestNewShop tests creating a new shop.
func TestNewShop(t *testing.T) {
	shop := NewShop(1, 1001, "Test Shop", 3001)

	if shop.ID != 1 {
		t.Errorf("Expected shop ID 1, got %d", shop.ID)
	}

	if shop.VNum != 1001 {
		t.Errorf("Expected shop VNum 1001, got %d", shop.VNum)
	}

	if shop.Name != "Test Shop" {
		t.Errorf("Expected shop name 'Test Shop', got '%s'", shop.Name)
	}

	if shop.RoomVNum != 3001 {
		t.Errorf("Expected shop room VNum 3001, got %d", shop.RoomVNum)
	}

	// Check default values
	if shop.BuyMultiplier != 50 {
		t.Errorf("Expected default buy multiplier 50, got %d", shop.BuyMultiplier)
	}

	if shop.SellMultiplier != 150 {
		t.Errorf("Expected default sell multiplier 150, got %d", shop.SellMultiplier)
	}

	if shop.MaxItems != 50 {
		t.Errorf("Expected default max items 50, got %d", shop.MaxItems)
	}
}

// TestShopAddRemoveItem tests adding and removing items from shop inventory.
func TestShopAddRemoveItem(t *testing.T) {
	shop := NewShop(1, 1001, "Test Shop", 3001)

	// Create a test item
	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test sword",
		ShortDesc: "a test sword",
		LongDesc:  "A test sword lies here.",
		Cost:      100,
		TypeFlag:  5, // Weapon type
	}

	item := game.NewObjectInstance(proto, -1)

	// Test adding item
	if !shop.AddItem(item) {
		t.Error("Failed to add item to shop")
	}

	if len(shop.GetInventory()) != 1 {
		t.Errorf("Expected 1 item in inventory, got %d", len(shop.GetInventory()))
	}

	// Test finding item
	foundItem, found := shop.FindItem("sword")
	if !found {
		t.Error("Failed to find item by name")
	}

	if foundItem != item {
		t.Error("Found item doesn't match added item")
	}

	// Test removing item
	if !shop.RemoveItem(item) {
		t.Error("Failed to remove item from shop")
	}

	if len(shop.GetInventory()) != 0 {
		t.Errorf("Expected 0 items in inventory after removal, got %d", len(shop.GetInventory()))
	}
}

// TestShopPriceCalculations tests price calculation methods.
func TestShopPriceCalculations(t *testing.T) {
	shop := NewShop(1, 1001, "Test Shop", 3001)

	// Create a test item with cost 100
	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}

	item := game.NewObjectInstance(proto, -1)

	// Test buy price (50% of cost)
	buyPrice := shop.CalculateBuyPrice(item)
	expectedBuyPrice := 50 // 100 * 50 / 100
	if buyPrice != expectedBuyPrice {
		t.Errorf("Expected buy price %d, got %d", expectedBuyPrice, buyPrice)
	}

	// Test sell price (150% of cost)
	sellPrice := shop.CalculateSellPrice(item)
	expectedSellPrice := 150 // 100 * 150 / 100
	if sellPrice != expectedSellPrice {
		t.Errorf("Expected sell price %d, got %d", expectedSellPrice, sellPrice)
	}

	// Test repair cost
	repairCost := shop.CalculateRepairCost(item, 10)
	expectedRepairCost := 100 // 10 * 10 (damage * repair cost)
	if repairCost != expectedRepairCost {
		t.Errorf("Expected repair cost %d, got %d", expectedRepairCost, repairCost)
	}

	// Test identify cost
	identifyCost := shop.CalculateIdentifyCost(item)
	expectedIdentifyCost := 5 // base identify cost
	if identifyCost != expectedIdentifyCost {
		t.Errorf("Expected identify cost %d, got %d", expectedIdentifyCost, identifyCost)
	}
}

// TestShopTypeChecking tests item type checking methods.
func TestShopTypeChecking(t *testing.T) {
	shop := NewShop(1, 1001, "Test Shop", 3001)

	// Add some item types the shop deals in
	shop.ItemTypes = []int{1, 2, 3} // Container, weapon, armor
	shop.BuyTypes = []int{1, 2}     // Only buys containers and weapons

	// Test CanSellType
	if !shop.CanSellType(1) {
		t.Error("Shop should sell type 1 (container)")
	}

	if !shop.CanSellType(2) {
		t.Error("Shop should sell type 2 (weapon)")
	}

	if shop.CanSellType(4) {
		t.Error("Shop should not sell type 4 (not in ItemTypes)")
	}

	// Test CanBuyType
	if !shop.CanBuyType(1) {
		t.Error("Shop should buy type 1 (container)")
	}

	if !shop.CanBuyType(2) {
		t.Error("Shop should buy type 2 (weapon)")
	}

	if shop.CanBuyType(3) {
		t.Error("Shop should not buy type 3 (armor, not in BuyTypes)")
	}

	// Test with empty BuyTypes (should use ItemTypes)
	shop.BuyTypes = []int{}
	if !shop.CanBuyType(3) {
		t.Error("With empty BuyTypes, shop should buy type 3 (in ItemTypes)")
	}
}

// TestShopManager tests the shop manager.
func TestShopManager(t *testing.T) {
	manager := NewShopManager()

	// Test creating a shop
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	if shop == nil {
		t.Fatal("Failed to create shop")
	}

	// Test getting shop by ID
	retrievedShop, found := manager.GetShopConcrete(shop.ID)
	if !found {
		t.Error("Failed to get shop by ID")
	}

	if retrievedShop != shop {
		t.Error("Retrieved shop doesn't match created shop")
	}

	// Test getting shop by NPC VNum
	npcShop, found := manager.GetShopByNPCConcrete(1001)
	if !found {
		t.Error("Failed to get shop by NPC VNum")
	}

	if npcShop != shop {
		t.Error("Shop retrieved by NPC VNum doesn't match created shop")
	}

	// Test getting shops in room
	shopsInRoom := manager.GetShopsInRoomConcrete(3001)
	if len(shopsInRoom) != 1 {
		t.Errorf("Expected 1 shop in room, got %d", len(shopsInRoom))
	}

	if shopsInRoom[0] != shop {
		t.Error("Shop in room doesn't match created shop")
	}

	// Test removing shop
	if !manager.RemoveShop(shop.ID) {
		t.Error("Failed to remove shop")
	}

	_, found = manager.GetShopConcrete(shop.ID)
	if found {
		t.Error("Shop should have been removed")
	}

	// Test getting all shops
	manager.CreateShopConcrete(1002, "Shop 2", 3002)
	manager.CreateShopConcrete(1003, "Shop 3", 3003)

	allShops := manager.GetAllShops()
	if len(allShops) != 2 {
		t.Errorf("Expected 2 shops total, got %d", len(allShops))
	}
}

// TestShopTransaction tests buy/sell transactions.
func TestShopTransaction(t *testing.T) {
	manager := NewShopManager()

	// Create a shop
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1} // Shop deals in type 1 items

	// Create a player
	player := &game.Player{
		Name: "Test Player",
		Gold: 1000,
	}
	player.Inventory = game.NewInventory()

	// Create an item prototype
	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}

	// Create an item instance
	item := game.NewObjectInstance(proto, -1)

	// Add item to shop inventory
	shop.AddItem(item)

	// Test buying from shop
	success, message := manager.ProcessTransaction(shop, player, item, true)
	if !success {
		t.Errorf("Buy transaction failed: %s", message)
	}

	// Player should have less gold
	if player.Gold != 850 { // 1000 - 150 (sell price)
		t.Errorf("Expected player gold 850 after purchase, got %d", player.Gold)
	}

	// Player should have the item
	if player.Inventory.GetItemCount() != 1 {
		t.Errorf("Expected 1 item in player inventory, got %d", player.Inventory.GetItemCount())
	}

	// Shop should not have the item
	if len(shop.GetInventory()) != 0 {
		t.Errorf("Expected 0 items in shop inventory after sale, got %d", len(shop.GetInventory()))
	}

	// Test selling back to shop
	success, message = manager.ProcessTransaction(shop, player, item, false)
	if !success {
		t.Errorf("Sell transaction failed: %s", message)
	}

	// Player should have more gold (but less than original due to buy/sell spread)
	if player.Gold != 900 { // 850 + 50 (buy price)
		t.Errorf("Expected player gold 900 after selling back, got %d", player.Gold)
	}

	// Player should not have the item
	if player.Inventory.GetItemCount() != 0 {
		t.Errorf("Expected 0 items in player inventory after selling, got %d", player.Inventory.GetItemCount())
	}

	// Shop should have the item again
	if len(shop.GetInventory()) != 1 {
		t.Errorf("Expected 1 item in shop inventory after buying back, got %d", len(shop.GetInventory()))
	}
}
