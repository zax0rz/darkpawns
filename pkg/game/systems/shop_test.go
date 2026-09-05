package systems

import (
	"os"
	"sync"
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

	// Test buy price (50% of cost, no CHA modifier)
	buyPrice := shop.CalculateBuyPrice(item, 0)
	expectedBuyPrice := 50 // 100 * 50 / 100
	if buyPrice != expectedBuyPrice {
		t.Errorf("Expected buy price %d, got %d", expectedBuyPrice, buyPrice)
	}

	// Test sell price (150% of cost, no CHA modifier)
	sellPrice := shop.CalculateSellPrice(item, 0)
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

// TestShopTransactionRaceOnCha verifies that reading player.Stats.Cha during a
// shop transaction is synchronized with concurrent stat updates (DP-563).
func TestShopTransactionRaceOnCha(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1}

	player := &game.Player{Name: "Test Player", Gold: 10000}
	player.Inventory = game.NewInventory()

	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}
	for i := 0; i < 10; i++ {
		shop.AddItem(game.NewObjectInstance(proto, -1))
	}
	_ = player.Inventory.AddItem(game.NewObjectInstance(proto, -1))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			items := shop.GetInventory()
			if len(items) > 0 {
				_, _ = manager.ProcessTransaction(shop, player, items[0], true)
			}
		}(i)
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			player.Lock()
			player.Stats.Cha = id%25 + 1
			player.Unlock()
		}(i)
	}
	wg.Wait()
}

// TestShopTransactionConcurrentDeadlock simulates concurrent buy and sell transactions
// to verify that no deadlocks occur under race conditions.
func TestShopTransactionConcurrentDeadlock(t *testing.T) {
	manager := NewShopManager()

	// Create a shop
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1}

	// Create a player
	player := &game.Player{
		Name: "Test Player",
		Gold: 10000,
	}
	player.Inventory = game.NewInventory()

	// Create item prototype
	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}

	// Add some items to shop inventory
	for i := 0; i < 10; i++ {
		item := game.NewObjectInstance(proto, -1)
		shop.AddItem(item)
	}

	// Run concurrent buys and sells
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				// Buy
				items := shop.GetInventory()
				if len(items) > 0 {
					_, _ = manager.ProcessTransaction(shop, player, items[0], true)
				}
			} else {
				// Sell
				player.Lock()
				items := player.Inventory.FindItems("")
				player.Unlock()
				if len(items) > 0 {
					_, _ = manager.ProcessTransaction(shop, player, items[0], false)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestShopConcurrentSell_NoOverdraw ensures two simultaneous sells cannot
// drain more gold than the shop has. With Gold equal to one item's buy price,
// exactly one sale succeeds and total player gold grows by exactly that price.
func TestShopConcurrentSell_NoOverdraw(t *testing.T) {
	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1}

	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}

	// Buy price is 50% of cost = 50. Give the shop exactly enough for one item.
	shop.Gold = 50

	playerA := &game.Player{Name: "A", Gold: 0}
	playerA.Inventory = game.NewInventory()
	itemA := game.NewObjectInstance(proto, -1)
	_ = playerA.Inventory.AddItem(itemA)

	playerB := &game.Player{Name: "B", Gold: 0}
	playerB.Inventory = game.NewInventory()
	itemB := game.NewObjectInstance(proto, -1)
	_ = playerB.Inventory.AddItem(itemB)

	var wg sync.WaitGroup
	var okA, okB bool
	var mu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		success, _ := manager.ProcessTransaction(shop, playerA, itemA, false)
		mu.Lock()
		okA = success
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		success, _ := manager.ProcessTransaction(shop, playerB, itemB, false)
		mu.Lock()
		okB = success
		mu.Unlock()
	}()
	wg.Wait()

	if okA && okB {
		t.Error("both sells succeeded, but shop only had gold for one")
	}
	if !okA && !okB {
		t.Error("neither sell succeeded, expected exactly one")
	}

	totalPlayerGold := playerA.Gold + playerB.Gold
	if totalPlayerGold != 50 {
		t.Errorf("total player gold = %d, want 50", totalPlayerGold)
	}
	if shop.Gold != 0 {
		t.Errorf("shop.Gold = %d, want 0 after one successful purchase", shop.Gold)
	}
}

// TestShopPersistence tests saving and loading shop state.
func TestShopPersistence(t *testing.T) {
	// Clean up shops file if it exists
	_ = os.Remove(shopsFilePath)
	defer func() { _ = os.Remove(shopsFilePath) }()

	manager := NewShopManager()
	shop := manager.CreateShopConcrete(1001, "Test Shop", 3001)
	shop.ItemTypes = []int{1}
	shop.Gold = 5000
	shop.BankAccount = 12000
	shop.WithWho = NotradeGood | NotradeEvil

	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}
	item := game.NewObjectInstance(proto, -1)
	shop.AddItem(item)

	// Save shops
	err := manager.SaveShops()
	if err != nil {
		t.Fatalf("Failed to save shops: %v", err)
	}

	// Create a new manager and load
	manager2 := NewShopManager()
	mockGetProto := func(vnum int) (*parser.Obj, bool) {
		if vnum == 1001 {
			return proto, true
		}
		return nil, false
	}

	err = manager2.LoadShops(mockGetProto)
	if err != nil {
		t.Fatalf("Failed to load shops: %v", err)
	}

	loadedShop, ok := manager2.GetShopConcrete(shop.ID)
	if !ok {
		t.Fatal("Failed to get loaded shop")
	}

	if loadedShop.Name != "Test Shop" {
		t.Errorf("Expected shop name 'Test Shop', got '%s'", loadedShop.Name)
	}
	if loadedShop.Gold != 5000 {
		t.Errorf("Expected shop gold 5000, got %d", loadedShop.Gold)
	}
	if loadedShop.BankAccount != 12000 {
		t.Errorf("Expected shop BankAccount 12000, got %d", loadedShop.BankAccount)
	}
	if loadedShop.WithWho != NotradeGood|NotradeEvil {
		t.Errorf("Expected WithWho %d, got %d", NotradeGood|NotradeEvil, loadedShop.WithWho)
	}
	if len(loadedShop.GetInventory()) != 1 {
		t.Errorf("Expected 1 item in loaded shop inventory, got %d", len(loadedShop.GetInventory()))
	}
}

// TestLoadShops_ClearsExistingState ensures a reload replaces stale manager
// state rather than merging with it.
func TestLoadShops_ClearsExistingState(t *testing.T) {
	_ = os.Remove(shopsFilePath)
	defer func() { _ = os.Remove(shopsFilePath) }()

	proto := &parser.Obj{
		VNum:      1001,
		Keywords:  "test item",
		ShortDesc: "a test item",
		Cost:      100,
		TypeFlag:  1,
	}
	mockGetProto := func(vnum int) (*parser.Obj, bool) {
		if vnum == 1001 {
			return proto, true
		}
		return nil, false
	}

	// First manager: save one shop.
	manager1 := NewShopManager()
	shop1 := manager1.CreateShopConcrete(1001, "Saved Shop", 3001)
	shop1.ItemTypes = []int{1}
	if err := manager1.SaveShops(); err != nil {
		t.Fatalf("save shops: %v", err)
	}

	// Second manager: pre-populate with a different shop, then load saved data.
	manager2 := NewShopManager()
	stale := manager2.CreateShopConcrete(9999, "Stale Shop", 3999)
	stale.ItemTypes = []int{1}

	if err := manager2.LoadShops(mockGetProto); err != nil {
		t.Fatalf("load shops: %v", err)
	}

	// Stale shop must be gone.
	if _, ok := manager2.GetShopByNPCConcrete(9999); ok {
		t.Error("stale NPC mapping should have been removed")
	}
	foundStaleRoom := false
	for _, s := range manager2.GetShopsInRoomConcrete(3999) {
		if s.VNum == 9999 {
			foundStaleRoom = true
			break
		}
	}
	if foundStaleRoom {
		t.Error("stale room mapping should have been removed")
	}

	// Saved shop must be present.
	loaded, ok := manager2.GetShopByNPCConcrete(1001)
	if !ok {
		t.Fatal("saved shop should be present after load")
	}
	if loaded.Name != "Saved Shop" {
		t.Errorf("loaded shop name = %q, want Saved Shop", loaded.Name)
	}
}

func TestResolveShopsFile_DataDirOverride(t *testing.T) {
	t.Setenv("DARKPAWNS_DATA_DIR", "/srv/dp-data")
	if got, want := resolveShopsFile(), "/srv/dp-data/shops.json"; got != want {
		t.Fatalf("override: got %q, want %q", got, want)
	}
}

func TestResolveShopsFile_DefaultRelative(t *testing.T) {
	if got, want := resolveShopsFile(), "./data/shops.json"; got != want {
		t.Fatalf("default: got %q, want %q", got, want)
	}
}
