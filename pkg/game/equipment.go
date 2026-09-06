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

// Extended equipment positions are real C WEAR_* slots. They sit outside
// the legacy Go slot range so the existing public slot values remain stable.
const (
	SlotThrow EquipmentSlot = 100 + iota
	SlotAblegs
	SlotFace
	SlotHover
)

// equipmentSlotNames is the canonical player-facing name for each slot.
var equipmentSlotNames = map[EquipmentSlot]string{
	SlotHead:     "head",
	SlotBody:     "body",
	SlotArms:     "arms",
	SlotHands:    "hands",
	SlotLegs:     "legs",
	SlotFeet:     "feet",
	SlotWield:    "wield",
	SlotHold:     "hold",
	SlotLight:    "light",
	SlotNeck:     "neck",
	SlotAbout:    "about",
	SlotWaist:    "waist",
	SlotWrist:    "wrist",
	SlotFinger:   "finger",
	SlotEar:      "ear",
	SlotShoulder: "shoulder",
	SlotBack:     "back",
	SlotFingerR:  "finger right",
	SlotFingerL:  "finger left",
	SlotNeck1:    "neck 1",
	SlotNeck2:    "neck 2",
	SlotWristR:   "wrist right",
	SlotWristL:   "wrist left",
	SlotShield:   "shield",
	SlotThrow:    "throw",
	SlotAblegs:   "ablegs",
	SlotFace:     "face",
	SlotHover:    "hover",
}

var equipmentSlotInputs = map[string]EquipmentSlot{
	"head": SlotHead, "body": SlotBody, "arms": SlotArms, "hands": SlotHands,
	"legs": SlotLegs, "feet": SlotFeet, "wield": SlotWield, "hold": SlotHold,
	"light": SlotLight, "neck": SlotNeck, "about": SlotAbout, "waist": SlotWaist,
	"wrist": SlotWrist, "finger": SlotFinger, "ear": SlotEar, "shoulder": SlotShoulder,
	"back": SlotBack, "finger right": SlotFingerR, "finger_r": SlotFingerR,
	"finger left": SlotFingerL, "finger_l": SlotFingerL, "neck 1": SlotNeck1,
	"neck1": SlotNeck1, "neck 2": SlotNeck2, "neck2": SlotNeck2,
	"wrist right": SlotWristR, "wrist_r": SlotWristR, "wrist left": SlotWristL,
	"wrist_l": SlotWristL, "shield": SlotShield, "throw": SlotThrow,
	"ablegs": SlotAblegs, "face": SlotFace, "hover": SlotHover,
}

// String returns the name of the equipment slot.
func (s EquipmentSlot) String() string {
	if name, ok := equipmentSlotNames[s]; ok {
		return name
	}
	return "unknown"
}

// ParseEquipmentSlot parses a string into an EquipmentSlot.
func ParseEquipmentSlot(s string) (EquipmentSlot, bool) {
	if slot, ok := equipmentSlotInputs[strings.ToLower(s)]; ok {
		return slot, true
	}
	return SlotMax, false
}

// Equipment represents a player's equipped items.
type Equipment struct {
	mu        sync.RWMutex
	Slots     map[EquipmentSlot]*ObjectInstance
	OwnerName string
}

// NewEquipment creates a new empty equipment set.
func NewEquipment() *Equipment {
	return &Equipment{
		Slots: make(map[EquipmentSlot]*ObjectInstance),
	}
}

// SetSlot places an item in one exact equipment slot without deriving a slot
// from its wear flags. Callers remain responsible for moving the item out of
// its previous location and updating its ObjectLocation.
func (eq *Equipment) SetSlot(slot EquipmentSlot, item *ObjectInstance) error {
	if item == nil {
		return fmt.Errorf("cannot equip a nil item")
	}

	eq.mu.Lock()
	defer eq.mu.Unlock()
	if eq.Slots == nil {
		eq.Slots = make(map[EquipmentSlot]*ObjectInstance)
	}
	if _, occupied := eq.Slots[slot]; occupied {
		return fmt.Errorf("slot %s is already occupied", slot)
	}
	eq.Slots[slot] = item
	return nil
}

// Equip attempts to equip an item in the appropriate slot(s).
func (eq *Equipment) Equip(item *ObjectInstance, inv *Inventory) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return eq.equip(item, inv)
}

// EquipForPlayer equips an item with anti-alignment and anti-class validation.
// Returns (zapped bool, err error). If zapped is true the item stays in inventory.
// Source: handler.c equip_char() lines 701-720 (DP-369)
func (eq *Equipment) EquipForPlayer(item *ObjectInstance, inv *Inventory, alignment int, class int) (bool, error) {
	if item != nil && item.Prototype != nil {
		xf := item.Prototype.ExtraFlags[0]
		isEvil := alignment <= -350
		isGood := alignment >= 350
		isNeutral := !isEvil && !isGood
		if (xf&FlagAntiEvil != 0 && isEvil) ||
			(xf&FlagAntiGood != 0 && isGood) ||
			(xf&FlagAntiNeutral != 0 && isNeutral) {
			return true, nil
		}
		// Detect weapon type and shield status
		// Source: src/act.item.c:1600 — wear() function
		isSlash := false
		isShieldItem := false

		if item.Prototype != nil {
			if item.Prototype.TypeFlag == int(ItemWeaponType) {
				isSlash = item.Prototype.Values[3] == 3 // TYPE_SLASH - TYPE_HIT
			}
			for _, flag := range item.Prototype.WearFlags {
				if flag == 9 { // ITEM_WEAR_SHIELD = bit 9
					isShieldItem = true
					break
				}
			}
		}

		if InvalidClass(class, uint32(xf), isSlash, isShieldItem) { // #nosec G115 -- xf is a non-negative Diku extra-flags bitmask that fits in uint32
			return true, nil
		}
	}
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return false, eq.equip(item, inv)
}

// equip is the internal implementation without locking.
func (eq *Equipment) equip(item *ObjectInstance, inv *Inventory) error {
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

	//nolint:staticcheck // Loop structure supports future multi-slot item handling
	for _, slot := range wearFlags {
		// Check if this slot is part of a dual slot group
		if group, isDual := dualSlotGroups[slot]; isDual {
			// Try each slot in the group in order
			for _, trySlot := range group {
				if _, occupied := eq.Slots[trySlot]; !occupied {
					// Found empty slot
					item.Location = LocEquippedPlayer(eq.OwnerName, trySlot)
					eq.Slots[trySlot] = item
					return nil
				}
			}
			// All slots in group are occupied, unequip from first slot
			if err := eq.unequip(group[0], inv); err != nil {
				return fmt.Errorf("cannot unequip existing %s: %v", group[0], err)
			}
			// Now equip in first slot
			item.Location = LocEquippedPlayer(eq.OwnerName, group[0])
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
		item.Location = LocEquippedPlayer(eq.OwnerName, slot)
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

	// Try to add to inventory
	if err := inv.addItem(item); err != nil {
		return fmt.Errorf("inventory full, cannot unequip")
	}

	item.Location = LocInventoryPlayer(eq.OwnerName)

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
	// Bare hands use number(0, level/3) in the combat formula, not weapon dice.
	return 0, 0
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
			// ITEM_WEAR_THROW (bit 15) is not selected by C find_eq_pos.
		case 1: // Secondary wear flags (bits 16-31)
			if flag&(1<<0) != 0 { // ITEM_WEAR_ABLEGS (bit 16)
				slots = append(slots, SlotAblegs)
			}
			if flag&(1<<1) != 0 { // ITEM_WEAR_FACE (bit 17)
				slots = append(slots, SlotFace)
			}
			if flag&(1<<2) != 0 { // ITEM_WEAR_HOVER (bit 18)
				slots = append(slots, SlotHover)
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

// extractAll removes every equipped item without moving it through inventory.
// It is reserved for terminal character-state transitions such as death and an
// unsafe REALLYQUIT, where the objects are destroyed rather than carried.
func (eq *Equipment) extractAll() []*ObjectInstance {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	items := make([]*ObjectInstance, 0, len(eq.Slots))
	for _, item := range eq.Slots {
		if item != nil {
			items = append(items, item)
		}
	}
	eq.Slots = make(map[EquipmentSlot]*ObjectInstance)
	return items
}
