// Ported from src/objsave.c
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

// --------------------------------------------------------------------------
// C WEAR_* position mapping — matches the 0-based array indices in C
// char_data.equipment[] (structs.h WEAR_* constants 0–21).
// --------------------------------------------------------------------------

func CWearPosToSlot(cPos int) (EquipmentSlot, bool) {
	m := map[int]EquipmentSlot{
		0:  SlotLight,
		1:  SlotFingerR,
		2:  SlotFingerL,
		3:  SlotNeck1,
		4:  SlotNeck2,
		5:  SlotBody,
		6:  SlotHead,
		7:  SlotLegs,
		8:  SlotFeet,
		9:  SlotHands,
		10: SlotArms,
		11: SlotShield,
		12: SlotAbout,
		13: SlotWaist,
		14: SlotWristR,
		15: SlotWristL,
		16: SlotWield,
		17: SlotHold,
		18: SlotThrow,
		19: SlotAblegs,
		20: SlotFace,
		21: SlotHover,
	}
	s, ok := m[cPos]
	return s, ok
}

func SlotToCWearPos(s EquipmentSlot) (int, bool) {
	m := map[EquipmentSlot]int{
		SlotLight:   0,
		SlotFingerR: 1,
		SlotFingerL: 2,
		SlotNeck1:   3,
		SlotNeck2:   4,
		SlotBody:    5,
		SlotHead:    6,
		SlotLegs:    7,
		SlotFeet:    8,
		SlotHands:   9,
		SlotArms:    10,
		SlotShield:  11,
		SlotAbout:   12,
		SlotWaist:   13,
		SlotWristR:  14,
		SlotWristL:  15,
		SlotWield:   16,
		SlotHold:    17,
		SlotThrow:   18,
		SlotAblegs:  19,
		SlotFace:    20,
		SlotHover:   21,
	}
	c, ok := m[s]
	return c, ok
}

// Flag constants matching ITEM_* from structs.h used for alignment checks.
// ExtraFlags[0] bits.
const (
	FlagAntiGood    = 1 << 9  // ITEM_ANTI_GOOD
	FlagAntiEvil    = 1 << 10 // ITEM_ANTI_EVIL
	FlagAntiNeutral = 1 << 11 // ITEM_ANTI_NEUTRAL
	FlagNoRent      = 1 << 2  // ITEM_NORENT
)

// NumWears is the number of equipment slots (0-based). Matches NUM_WEARS in C (used in loops).
const NumWears = 22

// MaxBagRow is the max nesting depth for container loading (matching C's MAX_BAG_ROW = 5).
const MaxBagRow = 5

// ==========================================================================
// IsUnrentable — ported from C Crash_is_unrentable()
// Returns true if the object cannot be stored in rent/crash:
//   - ITEM_NORENT flag set
//   - load < 0 (virtual/negative vnum)
//   - type == ITEM_KEY
//
// Kept because house item persistence still uses it to filter stored objects.
// ==========================================================================
func IsUnrentable(obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return true
	}
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagNoRent) != 0 || obj.VNum < 0 || ItemType(obj.Prototype.TypeFlag) == ItemKey {
		return true
	}
	return false
}

// ==========================================================================
