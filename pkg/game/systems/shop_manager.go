// Package systems manages game world systems including shops.
package systems

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ShopManager manages all shops in the game world.
type ShopManager struct {
	mu     sync.RWMutex
	shops  map[int]*Shop // key: shop ID
	nextID int

	// NPC VNum to Shop ID mapping
	npcToShop  map[int]int   // key: NPC VNum, value: shop ID
	roomToShop map[int][]int // key: room VNum, value: list of shop IDs
}

// NewShopManager creates a new shop manager.
func NewShopManager() *ShopManager {
	return &ShopManager{
		shops:      make(map[int]*Shop),
		npcToShop:  make(map[int]int),
		roomToShop: make(map[int][]int),
		nextID:     1,
	}
}

// CreateShopConcrete creates a new shop and adds it to the manager.
func (sm *ShopManager) CreateShopConcrete(vnum int, name string, roomVNum int) *Shop {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shop := NewShop(sm.nextID, vnum, name, roomVNum)
	sm.shops[shop.ID] = shop
	sm.npcToShop[vnum] = shop.ID

	// Add to room mapping
	sm.roomToShop[roomVNum] = append(sm.roomToShop[roomVNum], shop.ID)

	sm.nextID++
	return shop
}

// GetShopConcrete returns a shop by ID.
func (sm *ShopManager) GetShopConcrete(id int) (*Shop, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shop, exists := sm.shops[id]
	return shop, exists
}

// GetShopByNPCConcrete returns a shop by NPC VNum.
func (sm *ShopManager) GetShopByNPCConcrete(vnum int) (*Shop, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shopID, exists := sm.npcToShop[vnum]
	if !exists {
		return nil, false
	}

	shop, exists := sm.shops[shopID]
	return shop, exists
}

// GetShopsInRoomConcrete returns all shops in a room.
func (sm *ShopManager) GetShopsInRoomConcrete(roomVNum int) []*Shop {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shopIDs, exists := sm.roomToShop[roomVNum]
	if !exists {
		return []*Shop{}
	}

	shops := make([]*Shop, 0, len(shopIDs))
	for _, id := range shopIDs {
		if shop, exists := sm.shops[id]; exists {
			shops = append(shops, shop)
		}
	}
	return shops
}

// RemoveShop removes a shop from the manager.
func (sm *ShopManager) RemoveShop(id int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shop, exists := sm.shops[id]
	if !exists {
		return false
	}

	// Remove from NPC mapping
	delete(sm.npcToShop, shop.VNum)

	// Remove from room mapping
	if shopIDs, exists := sm.roomToShop[shop.RoomVNum]; exists {
		for i, shopID := range shopIDs {
			if shopID == id {
				sm.roomToShop[shop.RoomVNum] = append(shopIDs[:i], shopIDs[i+1:]...)
				break
			}
		}
		// Clean up empty room entries
		if len(sm.roomToShop[shop.RoomVNum]) == 0 {
			delete(sm.roomToShop, shop.RoomVNum)
		}
	}

	// Remove from shops map
	delete(sm.shops, id)
	return true
}

// GetAllShops returns all shops.
func (sm *ShopManager) GetAllShops() []*Shop {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shops := make([]*Shop, 0, len(sm.shops))
	for _, shop := range sm.shops {
		shops = append(shops, shop)
	}
	return shops
}

// ProcessTransaction handles a buy/sell transaction between a player and a shop.
func (sm *ShopManager) ProcessTransaction(shop *Shop, player *game.Player, item common.ObjectInstance, isBuy bool) (bool, string) {
	if shop == nil || player == nil || item == nil {
		return false, "Invalid transaction parameters."
	}

	// Check if shop is open (simplified - always open for now)
	// In a real implementation, we'd check game time

	if isBuy {
		// Player buying from shop
		return sm.processBuy(shop, player, item)
	}
	// Player selling to shop
	return sm.processSell(shop, player, item)
}

// processBuy handles a player buying an item from a shop.
func (sm *ShopManager) processBuy(shop *Shop, player *game.Player, item common.ObjectInstance) (bool, string) {
	// Check if item is in shop inventory
	if !shop.RemoveItem(item) {
		return false, "That item is not for sale."
	}

	// Calculate price
	price := shop.CalculateSellPrice(item)

	// Check if player has enough gold
	player.Lock()
	if player.Gold < price {
		player.Unlock()
		// Put item back in shop
		shop.AddItem(item)
		return false, fmt.Sprintf("You need %d gold to buy that.", price)
	}

	// Check if player has inventory space
	if player.Inventory.IsFull() {
		player.Unlock()
		// Put item back in shop
		shop.AddItem(item)
		return false, "Your inventory is full."
	}

	// Transfer gold
	player.Gold -= price

	// Transfer item to player
	if g, ok := item.(*game.ObjectInstance); ok { g.Location = game.LocNowhere() }
	// Type assert to *game.ObjectInstance for Inventory methods
	gameItem, ok := item.(*game.ObjectInstance)
	if !ok {
		// This shouldn't happen, but handle it gracefully
		player.Gold += price
		player.Unlock()
		shop.AddItem(item)
		return false, "Internal error: item type mismatch"
	}
	if err := player.Inventory.AddItem(gameItem); err != nil {
		// Failed to add to inventory, refund and return item
		player.Gold += price
		player.Unlock()
		shop.AddItem(item)
		return false, fmt.Sprintf("Failed to add item to inventory: %v", err)
	}
	player.Unlock()

	return true, fmt.Sprintf("You buy %s for %d gold.", item.GetShortDesc(), price)
}

// processSell handles a player selling an item to a shop.
func (sm *ShopManager) processSell(shop *Shop, player *game.Player, item common.ObjectInstance) (bool, string) {
	// Check if shop buys this type of item
	if !shop.CanBuyType(item.GetTypeFlag()) {
		return false, "The shopkeeper isn't interested in that type of item."
	}

	// Check if item is in player's inventory
	// Type assert to *game.ObjectInstance for Inventory methods
	gameItem, ok := item.(*game.ObjectInstance)
	if !ok {
		return false, "Internal error: item type mismatch"
	}
	if !player.Inventory.RemoveItem(gameItem) {
		return false, "You don't have that item."
	}

	// Calculate price
	price := shop.CalculateBuyPrice(item)

	// Check if shop has enough gold (shops have unlimited gold for now)
	// In a real implementation, shops would have limited funds

	// Check if shop has inventory space
	if len(shop.GetInventory()) >= shop.MaxItems {
		// Return item to player
		if err := player.Inventory.AddItem(gameItem); err != nil {
			slog.Error("shop sell rollback: inventory full, failed to restore item", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
		}
		return false, "The shop's inventory is full."
	}

	// Transfer gold
	player.Lock()
	player.Gold += price
	player.Unlock()

	// Transfer item to shop
	if g, ok := item.(*game.ObjectInstance); ok { g.Location = game.LocNowhere() }
	if !shop.AddItem(item) {
		// Failed to add to shop, refund and return item
		player.Lock()
		player.Gold -= price
		player.Unlock()
		if err := player.Inventory.AddItem(gameItem); err != nil {
			slog.Error("shop sell rollback: failed shop add, failed to restore item", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
		}
		return false, "Failed to add item to shop inventory."
	}

	return true, fmt.Sprintf("You sell %s for %d gold.", item.GetShortDesc(), price)
}

// ProcessRepair handles repairing an item at a shop.
func (sm *ShopManager) ProcessRepair(shop *Shop, player *game.Player, item common.ObjectInstance, damage int) (bool, string) {
	if shop == nil || player == nil || item == nil {
		return false, "Invalid repair parameters."
	}

	// Check if item is in player's inventory or equipment
	// Type assert to *game.ObjectInstance for Inventory methods
	gameItem, ok := item.(*game.ObjectInstance)
	if !ok {
		return false, "Internal error: item type mismatch"
	}
	// For now, we'll assume it's in inventory
	if !player.Inventory.RemoveItem(gameItem) {
		// Check equipment
		// In a real implementation, we'd check equipment too
		return false, "You don't have that item."
	}

	// Calculate cost
	cost := shop.CalculateRepairCost(item, damage)

	// Check if player has enough gold
	player.Lock()
	if player.Gold < cost {
		player.Unlock()
		// Return item to player
		if err := player.Inventory.AddItem(gameItem); err != nil {
			slog.Error("shop operation: failed to restore item to player inventory", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
		}
		return false, fmt.Sprintf("You need %d gold to repair that.", cost)
	}

	// Check repair skill success
	// In a real implementation, we'd use shop.RepairSkill
	// For now, assume 100% success

	// Charge player
	player.Gold -= cost
	player.Unlock()

	// Repair item (in a real implementation, we'd update item condition)
	// For now, we just return the item

	// Return item to player
	if err := player.Inventory.AddItem(gameItem); err != nil {
		slog.Error("shop operation: failed to restore item to player inventory", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
	}

	return true, fmt.Sprintf("You repair %s for %d gold.", item.GetShortDesc(), cost)
}

// ProcessIdentify handles identifying an item at a shop.
func (sm *ShopManager) ProcessIdentify(shop *Shop, player *game.Player, item common.ObjectInstance) (bool, string) {
	if shop == nil || player == nil || item == nil {
		return false, "Invalid identify parameters."
	}

	// Check if item is in player's inventory
	// Type assert to *game.ObjectInstance for Inventory methods
	gameItem, ok := item.(*game.ObjectInstance)
	if !ok {
		return false, "Internal error: item type mismatch"
	}
	if !player.Inventory.RemoveItem(gameItem) {
		return false, "You don't have that item."
	}

	// Calculate cost
	cost := shop.CalculateIdentifyCost(item)

	// Check if player has enough gold
	player.Lock()
	if player.Gold < cost {
		player.Unlock()
		// Return item to player
		if err := player.Inventory.AddItem(gameItem); err != nil {
			slog.Error("shop operation: failed to restore item to player inventory", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
		}
		return false, fmt.Sprintf("You need %d gold to identify that.", cost)
	}

	// Check identify skill success
	// In a real implementation, we'd use shop.IdentifySkill
	// For now, assume 100% success

	// Charge player
	player.Gold -= cost
	player.Unlock()

	// Identify item (in a real implementation, we'd reveal hidden properties)
	// For now, we just return the item

	// Return item to player
	if err := player.Inventory.AddItem(gameItem); err != nil {
		slog.Error("shop operation: failed to restore item to player inventory", "player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
	}

	return true, fmt.Sprintf("You identify %s for %d gold.", item.GetShortDesc(), cost)
}

// RestockAll restocks all shops.
func (sm *ShopManager) RestockAll(prototypes []*parser.Obj, currentTick int) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	totalRestocked := 0
	for _, shop := range sm.shops {
		restocked := shop.Restock(prototypes, currentTick)
		totalRestocked += restocked
	}
	return totalRestocked
}

const shopsFile = "./data/shops.json"

// saveShopData is a JSON-serializable snapshot of a Shop.
type saveShopData struct {
	ID              int    `json:"id"`
	VNum            int    `json:"vnum"`
	Name            string `json:"name"`
	RoomVNum        int    `json:"room_vnum"`
	ItemTypes       []int  `json:"item_types"`
	BuyTypes        []int  `json:"buy_types"`
	BuyMultiplier   int    `json:"buy_multiplier"`
	SellMultiplier  int    `json:"sell_multiplier"`
	RepairSkill     int    `json:"repair_skill"`
	IdentifySkill   int    `json:"identify_skill"`
	RepairCost      int    `json:"repair_cost"`
	IdentifyCost    int    `json:"identify_cost"`
	MaxItems        int    `json:"max_items"`
	RestockInterval int    `json:"restock_interval"`
	RestockPercent  int    `json:"restock_percent"`
	OpenHour        int    `json:"open_hour"`
	CloseHour       int    `json:"close_hour"`
}

// saveShopsData is the top-level JSON structure for shop persistence.
type saveShopsData struct {
	NextID int            `json:"next_id"`
	Shops  []saveShopData `json:"shops"`
}

// SaveShops persists all shops to ./data/shops.json.
func (sm *ShopManager) SaveShops() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data := saveShopsData{
		NextID: sm.nextID,
		Shops:  make([]saveShopData, 0, len(sm.shops)),
	}

	for _, shop := range sm.shops {
		shop.mu.RLock()
		sd := saveShopData{
			ID:              shop.ID,
			VNum:            shop.VNum,
			Name:            shop.Name,
			RoomVNum:        shop.RoomVNum,
			ItemTypes:       append([]int(nil), shop.ItemTypes...),
			BuyTypes:        append([]int(nil), shop.BuyTypes...),
			BuyMultiplier:   shop.BuyMultiplier,
			SellMultiplier:  shop.SellMultiplier,
			RepairSkill:     shop.RepairSkill,
			IdentifySkill:   shop.IdentifySkill,
			RepairCost:      shop.RepairCost,
			IdentifyCost:    shop.IdentifyCost,
			MaxItems:        shop.MaxItems,
			RestockInterval: shop.RestockInterval,
			RestockPercent:  shop.RestockPercent,
			OpenHour:        shop.OpenHour,
			CloseHour:       shop.CloseHour,
		}
		shop.mu.RUnlock()
		data.Shops = append(data.Shops, sd)
	}

	if err := os.MkdirAll(filepath.Dir(shopsFile), 0750); err != nil {
		return fmt.Errorf("create shops dir: %w", err)
	}

	f, err := os.Create(filepath.Clean(shopsFile))
	if err != nil {
		return fmt.Errorf("create shops file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode shops: %w", err)
	}

	slog.Debug("Shops saved", "path", shopsFile, "count", len(data.Shops))
	return nil
}

// LoadShops restores shops from ./data/shops.json.
// If the file does not exist, returns nil (first boot).
func (sm *ShopManager) LoadShops() error {
	f, err := os.Open(filepath.Clean(shopsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open shops file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var data saveShopsData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return fmt.Errorf("decode shops: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.nextID = data.NextID
	if sm.nextID < 1 {
		sm.nextID = 1
	}

	for _, sd := range data.Shops {
		shop := NewShop(sd.ID, sd.VNum, sd.Name, sd.RoomVNum)
		shop.ItemTypes = append([]int(nil), sd.ItemTypes...)
		shop.BuyTypes = append([]int(nil), sd.BuyTypes...)
		shop.BuyMultiplier = sd.BuyMultiplier
		shop.SellMultiplier = sd.SellMultiplier
		shop.RepairSkill = sd.RepairSkill
		shop.IdentifySkill = sd.IdentifySkill
		shop.RepairCost = sd.RepairCost
		shop.IdentifyCost = sd.IdentifyCost
		shop.MaxItems = sd.MaxItems
		shop.RestockInterval = sd.RestockInterval
		shop.RestockPercent = sd.RestockPercent
		shop.OpenHour = sd.OpenHour
		shop.CloseHour = sd.CloseHour

		sm.shops[shop.ID] = shop
		sm.npcToShop[shop.VNum] = shop.ID
		sm.roomToShop[shop.RoomVNum] = append(sm.roomToShop[shop.RoomVNum], shop.ID)
	}

	slog.Info("Shops loaded", "path", shopsFile, "count", len(data.Shops))
	return nil
}

// Implement common.ShopManager interface

// CreateShop implements common.ShopManager.CreateShop.
func (sm *ShopManager) CreateShop(vnum int, name string, roomVNum int) interface{} {
	return sm.CreateShopConcrete(vnum, name, roomVNum)
}

// GetShop implements common.ShopManager.GetShop.
func (sm *ShopManager) GetShop(id int) (interface{}, bool) {
	return sm.GetShopConcrete(id)
}

// GetShopByNPC implements common.ShopManager.GetShopByNPC.
func (sm *ShopManager) GetShopByNPC(vnum int) (interface{}, bool) {
	return sm.GetShopByNPCConcrete(vnum)
}

// GetShopsInRoom implements common.ShopManager.GetShopsInRoom.
func (sm *ShopManager) GetShopsInRoom(roomVNum int) []interface{} {
	shops := sm.GetShopsInRoomConcrete(roomVNum)
	result := make([]interface{}, len(shops))
	for i, shop := range shops {
		result[i] = shop
	}
	return result
}
