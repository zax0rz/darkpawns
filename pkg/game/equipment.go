package game

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// EquipmentSlot represents where an item can be equipped.
type EquipmentSlot int

const (
	SlotHead EquipmentSlot = iota
	SlotBody
	SlotArms
	SlotHands
	SlotLegs
	SlotFeet
	SlotWield
	SlotHold
	SlotLight
	SlotNeck
	SlotAbout
	SlotWaist
	SlotWrist
	SlotFinger
	SlotEar
	SlotShoulder
	SlotBack
	SlotMax // Sentinel value
)

// String returns the name of the equipment slot.
func (s EquipmentSlot) String() string {
	switch s {
	case SlotHead:
		return "head"
	case SlotBody:
		return "body"
	case SlotArms:
		return "arms"
	case SlotHands:
		return "hands"
	case SlotLegs:
		return "legs"
	case SlotFeet:
		return "feet"
	case SlotWield:
		return "wield"
	case SlotHold:
		return "hold"
	case SlotLight:
		return "light"
	case SlotNeck:
		return "neck"
	case SlotAbout:
		return "about"
	case SlotWaist:
		return "waist"
	case SlotWrist:
		return "wrist"
	case SlotFinger:
		return "finger"
	case SlotEar:
		return "ear"
	case SlotShoulder:
		return "shoulder"
	case SlotBack:
		return "back"
	default:
		return "unknown"
	}
}

// ParseEquipmentSlot parses a string into an EquipmentSlot.
func ParseEquipmentSlot(s string) (EquipmentSlot, bool) {
	switch strings.ToLower(s) {
	case "head":
		return SlotHead, true
	case "body":
		return SlotBody, true
	case "arms":
		return SlotArms, true
	case "hands":
		return SlotHands, true
	case "legs":
		return SlotLegs, true
	case "feet":
		return SlotFeet, true
	case "wield":
		return SlotWield, true
	case "hold":
		return SlotHold, true
	case "light":
		return SlotLight, true
	case "neck":
		return SlotNeck, true
	case "about":
		return SlotAbout, true
	case "waist":
		return SlotWaist, true
	case "wrist":
		return SlotWrist, true
	case "finger":
		return SlotFinger, true
	case "ear":
		return SlotEar, true
	case "shoulder":
		return SlotShoulder, true
	case "back":
		return SlotBack, true
	default:
		return SlotMax, false
	}
}

// Equipment represents a player's equipped items.
type Equipment struct {
	mu    sync.RWMutex
	Slots map[EquipmentSlot]*parser.Obj
}

// NewEquipment creates a new empty equipment set.
func NewEquipment() *Equipment {
	return &Equipment{
		Slots: make(map[EquipmentSlot]*parser.Obj),
	}
}

// Equip attempts to equip an item in the appropriate slot(s).
func (eq *Equipment) Equip(item *parser.Obj, inv *Inventory) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	// Check if item can be equipped
	wearFlags := eq.getWearFlags(item)
	if len(wearFlags) == 0 {
		return fmt.Errorf("item cannot be equipped")
	}

	// For now, equip in first available slot
	// In a full implementation, we'd check for multiple slots (like finger)
	for _, slot := range wearFlags {
		if existing, ok := eq.Slots[slot]; ok {
			// Unequip existing item first
			if err := eq.unequip(slot, inv); err != nil {
				return fmt.Errorf("cannot unequip existing %s: %v", slot, err)
			}
		}
		eq.Slots[slot] = item
		return nil
	}

	return fmt.Errorf("no suitable slot found for item")
}

// Unequip removes an item from a slot and returns it to inventory.
func (eq *Equipment) Unequip(slot EquipmentSlot, inv *Inventory) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return eq.unequip(slot, inv)
}

// unequip is the internal implementation without locking.
func (eq *Equipment) unequip(slot EquipmentSlot, inv *Inventory) error {
	item, ok := eq.Slots[slot]
	if !ok {
		return fmt.Errorf("slot %s is empty", slot)
	}

	// Try to add to inventory
	if err := inv.AddItem(item); err != nil {
		return fmt.Errorf("inventory full, cannot unequip")
	}

	delete(eq.Slots, slot)
	return nil
}

// UnequipItem removes a specific item from equipment.
func (eq *Equipment) UnequipItem(item *parser.Obj, inv *Inventory) bool {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	for slot, eqItem := range eq.Slots {
		if eqItem == item {
			if err := eq.unequip(slot, inv); err == nil {
				return true
			}
			return false
		}
	}
	return false
}

// GetItemInSlot returns the item in a specific slot.
func (eq *Equipment) GetItemInSlot(slot EquipmentSlot) (*parser.Obj, bool) {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	item, ok := eq.Slots[slot]
	return item, ok
}

// GetEquipmentBonus calculates total bonus for a stat from equipped items.
func (eq *Equipment) GetEquipmentBonus(stat string) int {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	total := 0
	for _, item := range eq.Slots {
		for _, affect := range item.Affects {
			// This is a simplified version - in a full implementation,
			// we'd map affect.Location to specific stats
			if affect.Location == getStatLocation(stat) {
				total += affect.Modifier
			}
		}
	}
	return total
}

// GetArmorClass returns total AC bonus from equipped armor.
func (eq *Equipment) GetArmorClass() int {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	ac := 0
	for _, item := range eq.Slots {
		// Check if item is armor (type 2 is armor in CircleMUD)
		if item.TypeFlag == 2 {
			// Values[0] is AC for armor
			ac += item.Values[0]
		}
	}
	return ac
}

// GetWeaponDamage returns weapon damage dice if a weapon is equipped.
func (eq *Equipment) GetWeaponDamage() (numDice, diceType int) {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	if weapon, ok := eq.Slots[SlotWield]; ok {
		// Check if item is a weapon (type 1 is weapon in CircleMUD)
		if weapon.TypeFlag == 1 {
			// Values[1] is number of dice, Values[2] is dice type
			return weapon.Values[1], weapon.Values[2]
		}
	}
	return 1, 4 // Default bare-handed damage
}

// getWearFlags returns which equipment slots an item can be worn in.
func (eq *Equipment) getWearFlags(item *parser.Obj) []EquipmentSlot {
	var slots []EquipmentSlot
	
	// Check each wear flag position
	for i, flag := range item.WearFlags {
		if flag == 0 {
			continue
		}
		
		// Convert CircleMUD wear flags to our EquipmentSlot
		// This is a simplified mapping
		switch i {
		case 0: // Primary wear flags
			if flag&(1<<0) != 0 { // TAKE
				// Can be held
				slots = append(slots, SlotHold)
			}
			if flag&(1<<1) != 0 { // FINGER
				slots = append(slots, SlotFinger)
			}
			if flag&(1<<2) != 0 { // NECK
				slots = append(slots, SlotNeck)
			}
			if flag&(1<<3) != 0 { // BODY
				slots = append(slots, SlotBody)
			}
			if flag&(1<<4) != 0 { // HEAD
				slots = append(slots, SlotHead)
			}
			if flag&(1<<5) != 0 { // LEGS
				slots = append(slots, SlotLegs)
			}
			if flag&(1<<6) != 0 { // FEET
				slots = append(slots, SlotFeet)
			}
			if flag&(1<<7) != 0 { // HANDS
				slots = append(slots, SlotHands)
			}
			if flag&(1<<8) != 0 { // ARMS
				slots = append(slots, SlotArms)
			}
			if flag&(1<<9) != 0 { // SHIELD
				slots = append(slots, SlotHold) // Shield goes in hold slot
			}
			if flag&(1<<10) != 0 { // ABOUT
				slots = append(slots, SlotAbout)
			}
			if flag&(1<<11) != 0 { // WAIST
				slots = append(slots, SlotWaist)
			}
			if flag&(1<<12) != 0 { // WRIST
				slots = append(slots, SlotWrist)
			}
			if flag&(1<<13) != 0 { // WIELD
				slots = append(slots, SlotWield)
			}
			if flag&(1<<14) != 0 { // HOLD
				slots = append(slots, SlotHold)
			}
		case 1: // Secondary wear flags
			if flag&(1<<0) != 0 { // LIGHT
				slots = append(slots, SlotLight)
			}
		}
	}
	
	return slots
}

// getStatLocation maps stat names to affect locations.
// This is a simplified version - CircleMUD has specific location numbers.
func getStatLocation(stat string) int {
	switch strings.ToLower(stat) {
	case "strength":
		return 1
	case "dexterity":
		return 2
	case "constitution":
		return 3
	case "intelligence":
		return 4
	case "wisdom":
		return 5
	case "charisma":
		return 6
	case "hp":
		return 12
	case "mana":
		return 13
	case "move":
		return 14
	case "ac":
		return 17
	default:
		return 0
	}
}

// GetEquippedItems returns all equipped items.
func (eq *Equipment) GetEquippedItems() map[EquipmentSlot]*parser.Obj {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[EquipmentSlot]*parser.Obj)
	for k, v := range eq.Slots {
		result[k] = v
	}
	return result
}