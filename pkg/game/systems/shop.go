// Package systems manages game world systems including shops.
package systems

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Shop represents a shopkeeper NPC that can buy, sell, repair, and identify items.
type Shop struct {
	mu sync.RWMutex

	// Shop identification
	ID       int
	VNum     int // VNum of the NPC that runs this shop
	Name     string
	RoomVNum int // Room where the shop is located

	// Item types bought/sold
	ItemTypes []int // Object type flags that this shop deals in
	BuyTypes  []int // Specific types the shop will buy (if empty, uses ItemTypes)

	// Price multipliers (as percentages)
	BuyMultiplier  int // Percentage of base cost the shop pays for items (e.g., 50 = 50%)
	SellMultiplier int // Percentage of base cost the shop charges (e.g., 150 = 150%)

	// Repair and identification
	RepairSkill   int // 0-100 skill level for repairing items
	IdentifySkill int // 0-100 skill level for identifying items
	RepairCost    int // Base cost per repair point
	IdentifyCost  int // Base cost per identification

	// Shop inventory
	Inventory []common.ObjectInstance
	MaxItems  int // Maximum items shop can stock

	// Restocking
	RestockInterval int // How often to restock (in game ticks)
	LastRestock     int // Last restock tick
	RestockPercent  int // Percentage chance to restock each item

	// Business hours (optional - for future use)
	OpenHour  int // 0-23
	CloseHour int // 0-23
}

// NewShop creates a new shop instance.
func NewShop(id, vnum int, name string, roomVNum int) *Shop {
	return &Shop{
		ID:              id,
		VNum:            vnum,
		Name:            name,
		RoomVNum:        roomVNum,
		ItemTypes:       make([]int, 0),
		BuyTypes:        make([]int, 0),
		BuyMultiplier:   50,  // Default: pays 50% of item value
		SellMultiplier:  150, // Default: sells at 150% of item value
		RepairSkill:     75,  // Default: 75% repair skill
		IdentifySkill:   90,  // Default: 90% identify skill
		RepairCost:      10,  // Default: 10 gold per repair point
		IdentifyCost:    5,   // Default: 5 gold per identification
		Inventory:       make([]common.ObjectInstance, 0),
		MaxItems:        50,  // Default: max 50 items in stock
		RestockInterval: 100, // Default: restock every 100 game ticks
		RestockPercent:  30,  // Default: 30% chance to restock each item
		OpenHour:        0,   // Always open by default
		CloseHour:       23,
	}
}

// CanBuyType checks if the shop buys items of the given type.
func (s *Shop) CanBuyType(itemType int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If BuyTypes is specified, use it; otherwise use ItemTypes
	typesToCheck := s.BuyTypes
	if len(typesToCheck) == 0 {
		typesToCheck = s.ItemTypes
	}

	for _, t := range typesToCheck {
		if t == itemType {
			return true
		}
	}
	return false
}

// CanSellType checks if the shop sells items of the given type.
func (s *Shop) CanSellType(itemType int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.ItemTypes {
		if t == itemType {
			return true
		}
	}
	return false
}

// CalculateBuyPrice calculates how much the shop will pay for an item.
func (s *Shop) CalculateBuyPrice(item common.ObjectInstance) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	baseCost := item.GetCost()
	// Apply buy multiplier (percentage)
	price := (baseCost * s.BuyMultiplier) / 100

	// Minimum price of 1 gold
	if price < 1 {
		price = 1
	}

	return price
}

// CalculateSellPrice calculates how much the shop charges for an item.
func (s *Shop) CalculateSellPrice(item common.ObjectInstance) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	baseCost := item.GetCost()
	// Apply sell multiplier (percentage)
	price := (baseCost * s.SellMultiplier) / 100

	// Minimum price of 1 gold
	if price < 1 {
		price = 1
	}

	return price
}

// CalculateRepairCost calculates the cost to repair an item.
func (s *Shop) CalculateRepairCost(item common.ObjectInstance, damage int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Base cost per damage point
	cost := damage * s.RepairCost

	// Apply item value modifier (more valuable items cost more to repair)
	baseCost := item.GetCost()
	if baseCost > 1000 {
		cost = (cost * baseCost) / 1000
	}

	// Minimum cost of 1 gold
	if cost < 1 {
		cost = 1
	}

	return cost
}

// CalculateIdentifyCost calculates the cost to identify an item.
func (s *Shop) CalculateIdentifyCost(item common.ObjectInstance) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Base identification cost
	cost := s.IdentifyCost

	// Apply item value modifier (more valuable items cost more to identify)
	baseCost := item.GetCost()
	if baseCost > 1000 {
		cost = (cost * baseCost) / 1000
	}

	// Minimum cost of 1 gold
	if cost < 1 {
		cost = 1
	}

	return cost
}

// AddItem adds an item to the shop's inventory.
func (s *Shop) AddItem(item common.ObjectInstance) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Inventory) >= s.MaxItems {
		return false
	}

	// Set item location to shop
	item.SetRoomVNum(-1)
	item.SetCarrier(s)
	s.Inventory = append(s.Inventory, item)
	return true
}

// RemoveItem removes an item from the shop's inventory.
func (s *Shop) RemoveItem(item common.ObjectInstance) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, shopItem := range s.Inventory {
		if shopItem == item {
			s.Inventory = append(s.Inventory[:i], s.Inventory[i+1:]...)
			item.SetCarrier(nil)
			return true
		}
	}
	return false
}

// FindItem finds an item in the shop's inventory by name.
func (s *Shop) FindItem(name string) (common.ObjectInstance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lowerName := strings.ToLower(name)
	for _, item := range s.Inventory {
		// Check keywords
		keywords := strings.ToLower(item.GetKeywords())
		if strings.Contains(keywords, lowerName) {
			return item, true
		}
		// Check short description
		shortDesc := strings.ToLower(item.GetShortDesc())
		if strings.Contains(shortDesc, lowerName) {
			return item, true
		}
	}
	return nil, false
}

// FindItemsByType finds all items of a specific type in the shop's inventory.
func (s *Shop) FindItemsByType(itemType int) []common.ObjectInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []common.ObjectInstance
	for _, item := range s.Inventory {
		if item.GetTypeFlag() == itemType {
			matches = append(matches, item)
		}
	}
	return matches
}

// GetInventory returns a copy of the shop's inventory.
func (s *Shop) GetInventory() []common.ObjectInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inventory := make([]common.ObjectInstance, len(s.Inventory))
	copy(inventory, s.Inventory)
	return inventory
}

// IsOpen checks if the shop is open based on game time.
func (s *Shop) IsOpen(currentHour int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.OpenHour <= s.CloseHour {
		// Normal hours (e.g., 9 AM to 5 PM)
		return currentHour >= s.OpenHour && currentHour < s.CloseHour
	}
	// Overnight hours (e.g., 8 PM to 4 AM)
	return currentHour >= s.OpenHour || currentHour < s.CloseHour
}

// Restock attempts to restock the shop's inventory.
func (s *Shop) Restock(prototypes []*parser.Obj, currentTick int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if it's time to restock
	if currentTick-s.LastRestock < s.RestockInterval {
		return 0
	}

	s.LastRestock = currentTick
	restocked := 0

	// Try to add new items
	for _, proto := range prototypes {
		// Check if this item type is sold by this shop
		if !s.CanSellType(proto.TypeFlag) {
			continue
		}

		// Check restock chance
		// In a real implementation, we'd use random number
		// For now, we'll just add if we have space
		if len(s.Inventory) < s.MaxItems {
			item := game.NewObjectInstance(proto, -1)
			item.Carrier = s
			s.Inventory = append(s.Inventory, item)
			restocked++
		}
	}

	return restocked
}

// String returns a string representation of the shop.
func (s *Shop) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("Shop %d: %s (VNum: %d, Room: %d, Items: %d/%d)",
		s.ID, s.Name, s.VNum, s.RoomVNum, len(s.Inventory), s.MaxItems)
}
