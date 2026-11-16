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
	if cPos < 0 || cPos >= int(SlotMax) {
		return 0, false
	}
	return EquipmentSlot(cPos), true
}

func SlotToCWearPos(s EquipmentSlot) (int, bool) {
	if s < 0 || s >= SlotMax {
		return 0, false
	}
	return int(s), true
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
