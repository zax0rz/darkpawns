package game

import (
	"strconv"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Inventory represents a player's inventory.
type Inventory struct {
	mu    sync.RWMutex
	Items []*ObjectInstance
	// Capacity is the item count limit (CAN_CARRY_N).
	Capacity int
	// MaxWeight is the carry weight limit (CAN_CARRY_W). 0 = unenforced.
	MaxWeight int
}

// NewInventory creates a new inventory with default capacity.
func NewInventory() *Inventory {
	return &Inventory{
		Items:    make([]*ObjectInstance, 0),
		Capacity: 20, // Default base capacity
	}
}

// addItem adds an item to the inventory (internal — no lock, caller must hold appropriate locks).
// Checks carry weight (CAN_CARRY_W) before item count (CAN_CARRY_N), matching
// the C CAN_GET_OBJ macro order (utils.h:543-545) (DP-1038).
func (inv *Inventory) addItem(item *ObjectInstance) error {
	if inv.MaxWeight > 0 {
		// Compute weight inline — caller already holds the lock, so we can't
		// call the locking GetWeight() (re-entrant RLock under Lock deadlocks).
		weight := 0
		for _, o := range inv.Items {
			weight += o.GetTotalWeight()
		}
		if weight+item.GetTotalWeight() > inv.MaxWeight {
			return ErrInventoryTooHeavy
		}
	}
	if len(inv.Items) >= inv.Capacity {
		return ErrInventoryFull
	}
	inv.Items = append(inv.Items, item)
	return nil
}

// removeItem removes an item from the inventory by reference (internal — no lock).
func (inv *Inventory) removeItem(item *ObjectInstance) bool {
	for i, invItem := range inv.Items {
		if invItem == item {
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			return true
		}
	}
	return false
}

// removeItemByVNum removes an item from the inventory by VNum (internal — no lock).
func (inv *Inventory) removeItemByVNum(vnum int) (*ObjectInstance, bool) {
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

// GetCapacity returns the inventory capacity.
func (inv *Inventory) GetCapacity() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.Capacity
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

// SetCapacity sets the inventory capacity and max carry weight based on the
// character's strength, dexterity, and level.
//
//	CAN_CARRY_N(ch) = 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)  (utils.h:449)
//	CAN_CARRY_W(ch) = str_app[STRENGTH_APPLY_INDEX(ch)].carry_w       (utils.h:448)
//
// Source: utils.h:448-449 (DP-1038 — weight now enforced).
func (inv *Inventory) SetCapacity(str, dex, level int) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	// >> 1 means divide by 2 (integer division)
	inv.Capacity = 5 + (dex >> 1) + (level >> 1)
	inv.MaxWeight = combat.CarryWeight(str)
}

// clear removes all items from inventory (internal — no lock).
func (inv *Inventory) clear() {
	inv.Items = make([]*ObjectInstance, 0)
}

// Deprecated: Use MoveObject helpers instead. These exported wrappers exist for
// cross-package callers (db, systems) that cannot access unexported methods.

func (inv *Inventory) AddItem(item *ObjectInstance) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	return inv.addItem(item)
}

// RestoreItem adds an item to the inventory ignoring the capacity limit. It is
// for loading a player's saved inventory, where the items were already
// legitimately owned and must not be silently dropped if the current capacity
// (which depends on strength and may have changed) is smaller than the save.
// Returns true if the add pushed the inventory over capacity, so the caller can
// surface it; the item is added either way.
func (inv *Inventory) RestoreItem(item *ObjectInstance) (overCapacity bool) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	over := len(inv.Items) >= inv.Capacity
	inv.Items = append(inv.Items, item)
	return over
}

func (inv *Inventory) RemoveItem(item *ObjectInstance) bool {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	return inv.removeItem(item)
}

func (inv *Inventory) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.clear()
}

// ==========================================================================
// Handler.c utility functions — on-demand helpers
// ==========================================================================

// GetNumber parses a leading number from a dot-separated string.
// C: int get_number(char **name) — returns the number before a '.' separator.
// If no dot is found, returns 1.
// Used by get_char_room, get_obj_in_list_vis etc. for "2.guard" → 2 syntax.
func GetNumber(name *string) int {
	dot := strings.IndexByte(*name, '.')
	if dot < 0 {
		return 1
	}
	numStr := (*name)[:dot]
	*name = (*name)[dot+1:]
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return 0
		}
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

// GetObjNum finds an object instance by its prototype rnum.
// C: struct obj_data *get_obj_num(int nr) — linear search of object_list.
// Go: search World's objectInstances by prototype index.
func (w *World) GetObjNum(rnum int) *ObjectInstance {
	for _, obj := range w.objectInstances {
		if obj.Prototype != nil && obj.Prototype.VNum == rnum {
			return obj
		}
	}
	return nil
}

// GetCharNum finds a mob instance by its prototype rnum.
// C: struct char_data *get_char_num(int nr) — linear search of character_list.
func (w *World) GetCharNum(rnum int) *MobInstance {
	for _, mob := range w.activeMobs {
		if mob.Prototype != nil && mob.Prototype.VNum == rnum {
			return mob
		}
	}
	return nil
}

// RemoveFollower removes a specific player from someone's follower list.
// C: void remove_follower(struct char_data *ch) — removes ch from master's list.
// Unlike StopFollower (which makes you stop following), this removes a specific
// person who is following you.
func (w *World) RemoveFollower(ch *Player) {
	if ch == nil || ch.GetFollowing() == "" {
		return
	}
	_, _ = w.GetPlayer(ch.GetFollowing()) // verify leader exists
	// The follower removes itself from the leader's perspective
	ch.SetFollowing("")
	ch.RemoveAffectBit(affCharm)
	ch.RemoveAffectBit(affGroup)
}
