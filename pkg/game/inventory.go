package game

import (
	"strings"
	"sync"
)

// Inventory represents a player's inventory.
type Inventory struct {
	mu    sync.RWMutex
	Items []*ObjectInstance
	// Capacity is based on strength (default 20 + strength * 5)
	Capacity int
}

// NewInventory creates a new inventory with default capacity.
func NewInventory() *Inventory {
	return &Inventory{
		Items:    make([]*ObjectInstance, 0),
		Capacity: 20, // Default base capacity
	}
}

// AddItem adds an item to the inventory.
func (inv *Inventory) AddItem(item *ObjectInstance) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	if len(inv.Items) >= inv.Capacity {
		return ErrInventoryFull
	}
	inv.Items = append(inv.Items, item)
	return nil
}

// RemoveItem removes an item from the inventory by reference.
func (inv *Inventory) RemoveItem(item *ObjectInstance) bool {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	for i, invItem := range inv.Items {
		if invItem == item {
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveItemByVNum removes an item from the inventory by VNum.
func (inv *Inventory) RemoveItemByVNum(vnum int) (*ObjectInstance, bool) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	for i, item := range inv.Items {
		if item.VNum == vnum {
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			return item, true
		}
	}
	return nil, false
}

// FindItem finds an item by name (case-insensitive partial match).
func (inv *Inventory) FindItem(name string) (*ObjectInstance, bool) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	lowerName := strings.ToLower(name)
	for _, item := range inv.Items {
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

// FindItems finds all items matching a name (case-insensitive partial match).
// If name is empty string, returns all items.
func (inv *Inventory) FindItems(name string) []*ObjectInstance {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	if name == "" {
		// Return a copy of all items
		allItems := make([]*ObjectInstance, len(inv.Items))
		copy(allItems, inv.Items)
		return allItems
	}

	lowerName := strings.ToLower(name)
	var matches []*ObjectInstance
	for _, item := range inv.Items {
		// Check keywords
		keywords := strings.ToLower(item.GetKeywords())
		if strings.Contains(keywords, lowerName) {
			matches = append(matches, item)
			continue
		}
		// Check short description
		shortDesc := strings.ToLower(item.GetShortDesc())
		if strings.Contains(shortDesc, lowerName) {
			matches = append(matches, item)
		}
	}
	return matches
}

// GetItemCount returns the number of items in inventory.
func (inv *Inventory) GetItemCount() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.Items)
}

// IsFull returns true if inventory is at capacity.
func (inv *Inventory) IsFull() bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.Items) >= inv.Capacity
}

// GetWeight returns the total weight of items in inventory.
func (inv *Inventory) GetWeight() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, item := range inv.Items {
		total += item.GetTotalWeight()
	}
	return total
}

// SetCapacity sets the inventory capacity based on dexterity and level.
// Formula: CAN_CARRY_N(ch) = 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)
// Source: utils.h:448-449
// Note: Weight tracking (CAN_CARRY_W) requires str_app table — implement item count limit only for now
func (inv *Inventory) SetCapacity(dex int, level int) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	// >> 1 means divide by 2 (integer division)
	inv.Capacity = 5 + (dex / 2) + (level / 2)
}

// Clear removes all items from inventory.
func (inv *Inventory) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.Items = make([]*ObjectInstance, 0)
}
