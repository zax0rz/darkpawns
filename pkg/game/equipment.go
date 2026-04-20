package game

import (
	"fmt"
	"strings"
	"sync"
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
	// Dual equipment slots (M4)
	SlotFingerR
	SlotFingerL
	SlotNeck1
	SlotNeck2
	SlotWristR
	SlotWristL
	// Shield slot (M2/M3)
	SlotShield
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
	// Dual equipment slots (M4)
	case SlotFingerR:
		return "finger right"
	case SlotFingerL:
		return "finger left"
	case SlotNeck1:
		return "neck 1"
	case SlotNeck2:
		return "neck 2"
	case SlotWristR:
		return "wrist right"
	case SlotWristL:
		return "wrist left"
	// Shield slot (M2/M3)
	case SlotShield:
		return "shield"
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
	// Dual equipment slots (M4)
	case "finger right", "finger_r":
		return SlotFingerR, true
	case "finger left", "finger_l":
		return SlotFingerL, true
	case "neck 1", "neck1":
		return SlotNeck1, true
	case "neck 2", "neck2":
		return SlotNeck2, true
	case "wrist right", "wrist_r":
		return SlotWristR, true
	case "wrist left", "wrist_l":
		return SlotWristL, true
	// Shield slot (M2/M3)
	case "shield":
		return SlotShield, true
	default:
		return SlotMax, false
	}
}

// Equipment represents a player's equipped items.
type Equipment struct {
	mu    sync.RWMutex
	Slots map[EquipmentSlot]*ObjectInstance
}

// NewEquipment creates a new empty equipment set.
func NewEquipment() *Equipment {
	return &Equipment{
		Slots: make(map[EquipmentSlot]*ObjectInstance),
	}
}

// Equip attempts to equip an item in the appropriate slot(s).
func (eq *Equipment) Equip(item *ObjectInstance, inv *Inventory) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	// Check if item can be equipped
	wearFlags := eq.getWearFlags(item)
	if len(wearFlags) == 0 {
		return fmt.Errorf("item cannot be equipped")
	}

	// Handle dual equipment slots (M4)
	// When equipping a ring/neck/wrist item, prefer the right/first slot;
	// use the left/second if already occupied.
	// Source: structs.h:391-405 - players have dual slots for rings, necks, wrists
	
	// Group dual slots
	dualSlotGroups := map[EquipmentSlot][]EquipmentSlot{
		SlotFingerR: {SlotFingerR, SlotFingerL},
		SlotFingerL: {SlotFingerR, SlotFingerL},
		SlotNeck1:   {SlotNeck1, SlotNeck2},
		SlotNeck2:   {SlotNeck1, SlotNeck2},
		SlotWristR:  {SlotWristR, SlotWristL},
		SlotWristL:  {SlotWristR, SlotWristL},
	}
	
	for _, slot := range wearFlags {
		// Check if this slot is part of a dual slot group
		if group, isDual := dualSlotGroups[slot]; isDual {
			// Try each slot in the group in order
			for _, trySlot := range group {
				if _, occupied := eq.Slots[trySlot]; !occupied {
					// Found empty slot
					item.EquippedOn = eq
					item.EquipPosition = int(trySlot)
					eq.Slots[trySlot] = item
					return nil
				}
			}
			// All slots in group are occupied, unequip from first slot
			if err := eq.unequip(group[0], inv); err != nil {
				return fmt.Errorf("cannot unequip existing %s: %v", group[0], err)
			}
			// Now equip in first slot
			item.EquippedOn = eq
			item.EquipPosition = int(group[0])
			eq.Slots[group[0]] = item
			return nil
		}
		
		// Non-dual slot
		if _, ok := eq.Slots[slot]; ok {
			// Unequip existing item first
			if err := eq.unequip(slot, inv); err != nil {
				return fmt.Errorf("cannot unequip existing %s: %v", slot, err)
			}
		}
		// Set equipment state
		item.EquippedOn = eq
		item.EquipPosition = int(slot)
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

	// Clear equipment state
	item.EquippedOn = nil
	item.EquipPosition = -1

	// Try to add to inventory
	if err := inv.AddItem(item); err != nil {
		return fmt.Errorf("inventory full, cannot unequip")
	}

	delete(eq.Slots, slot)
	return nil
}

// UnequipItem removes a specific item from equipment.
func (eq *Equipment) UnequipItem(item *ObjectInstance, inv *Inventory) bool {
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
func (eq *Equipment) GetItemInSlot(slot EquipmentSlot) (*ObjectInstance, bool) {
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
		for _, affect := range item.GetAffects() {
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
		// Check if item is armor (type 9 is ITEM_ARMOR in CircleMUD)
		if item.GetTypeFlag() == 9 {
			// Values[0] is AC for armor
			ac += item.Prototype.Values[0]
		}
	}
	return ac
}

// GetWeaponDamage returns weapon damage dice if a weapon is equipped.
func (eq *Equipment) GetWeaponDamage() (numDice, diceType int) {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	if weapon, ok := eq.Slots[SlotWield]; ok {
		// Check if item is a weapon (type 5 is ITEM_WEAPON in CircleMUD)
		if weapon.GetTypeFlag() == 5 {
			// Values[1] is number of dice, Values[2] is dice type
			return weapon.Prototype.Values[1], weapon.Prototype.Values[2]
		}
	}
	return 1, 4 // Default bare-handed damage
}

// getWearFlags returns which equipment slots an item can be worn in.
// Maps ITEM_WEAR_* flags from structs.h:446-462 to EquipmentSlot
func (eq *Equipment) getWearFlags(item *ObjectInstance) []EquipmentSlot {
	var slots []EquipmentSlot
	
	// Check each wear flag position
	for i, flag := range item.Prototype.WearFlags {
		if flag == 0 {
			continue
		}
		
		// Convert Dark Pawns wear flags to our EquipmentSlot
		// Source: structs.h:446-462 ITEM_WEAR_* constants
		switch i {
		case 0: // Primary wear flags (bits 0-15)
			// ITEM_WEAR_TAKE (bit 0) = item can be picked up, NOT an equip slot
			// Do NOT map to SlotHold for bit 0
			
			if flag&(1<<1) != 0 { // ITEM_WEAR_FINGER (bit 1)
				// Can be worn on finger - map to both finger slots
				slots = append(slots, SlotFingerR, SlotFingerL)
			}
			if flag&(1<<2) != 0 { // ITEM_WEAR_NECK (bit 2)
				// Can be worn around neck - map to both neck slots
				slots = append(slots, SlotNeck1, SlotNeck2)
			}
			if flag&(1<<3) != 0 { // ITEM_WEAR_BODY (bit 3)
				slots = append(slots, SlotBody)
			}
			if flag&(1<<4) != 0 { // ITEM_WEAR_HEAD (bit 4)
				slots = append(slots, SlotHead)
			}
			if flag&(1<<5) != 0 { // ITEM_WEAR_LEGS (bit 5)
				slots = append(slots, SlotLegs)
			}
			if flag&(1<<6) != 0 { // ITEM_WEAR_FEET (bit 6)
				slots = append(slots, SlotFeet)
			}
			if flag&(1<<7) != 0 { // ITEM_WEAR_HANDS (bit 7)
				slots = append(slots, SlotHands)
			}
			if flag&(1<<8) != 0 { // ITEM_WEAR_ARMS (bit 8)
				slots = append(slots, SlotArms)
			}
			if flag&(1<<9) != 0 { // ITEM_WEAR_SHIELD (bit 9)
				// Shield should map to SlotShield, not SlotHold
				slots = append(slots, SlotShield)
			}
			if flag&(1<<10) != 0 { // ITEM_WEAR_ABOUT (bit 10)
				slots = append(slots, SlotAbout)
			}
			if flag&(1<<11) != 0 { // ITEM_WEAR_WAIST (bit 11)
				slots = append(slots, SlotWaist)
			}
			if flag&(1<<12) != 0 { // ITEM_WEAR_WRIST (bit 12)
				// Can be worn on wrist - map to both wrist slots
				slots = append(slots, SlotWristR, SlotWristL)
			}
			if flag&(1<<13) != 0 { // ITEM_WEAR_WIELD (bit 13)
				slots = append(slots, SlotWield)
			}
			if flag&(1<<14) != 0 { // ITEM_WEAR_HOLD (bit 14)
				slots = append(slots, SlotHold)
			}
			if flag&(1<<15) != 0 { // ITEM_WEAR_THROW (bit 15)
				// Can be thrown - not an equip slot
			}
		case 1: // Secondary wear flags (bits 16-31)
			if flag&(1<<0) != 0 { // ITEM_WEAR_ABLEGS (bit 16)
				// Can be worn about legs
				slots = append(slots, SlotLegs) // Approximate
			}
			if flag&(1<<1) != 0 { // ITEM_WEAR_FACE (bit 17)
				// Can be worn as a mask
				slots = append(slots, SlotHead) // Approximate
			}
			if flag&(1<<2) != 0 { // ITEM_WEAR_HOVER (bit 18)
				// Hovers near head
				slots = append(slots, SlotHead) // Approximate
			}
			// Note: ITEM_WEAR_LIGHT is not in Dark Pawns structs.h
			// Light items might be handled differently
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
func (eq *Equipment) GetEquippedItems() map[EquipmentSlot]*ObjectInstance {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[EquipmentSlot]*ObjectInstance)
	for k, v := range eq.Slots {
		result[k] = v
	}
	return result
}