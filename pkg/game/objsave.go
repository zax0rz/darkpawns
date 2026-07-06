// Ported from src/objsave.c
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// --------------------------------------------------------------------------
// C WEAR_* position mapping — matches the 0-based array indices in C
// char_data.equipment[] (structs.h WEAR_* constants 0–20).
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
		18: SlotAblegs,
		19: SlotFace,
		20: SlotHover,
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
		SlotAblegs:  18,
		SlotFace:    19,
		SlotHover:   20,
	}
	c, ok := m[s]
	return c, ok
}

// cWearPosCanWearFlag maps C WEAR_* index to the ITEM_WEAR_* bit required.
func cWearPosCanWearFlag(cPos int) int {
	m := map[int]int{
		0:  1 << 15, // ITEM_WEAR_LIGHT
		1:  1 << 1,  // ITEM_WEAR_FINGER
		2:  1 << 1,  // ITEM_WEAR_FINGER (alt)
		3:  1 << 2,  // ITEM_WEAR_NECK
		4:  1 << 2,  // ITEM_WEAR_NECK (alt)
		5:  1 << 3,  // ITEM_WEAR_BODY
		6:  1 << 4,  // ITEM_WEAR_HEAD
		7:  1 << 5,  // ITEM_WEAR_LEGS
		8:  1 << 6,  // ITEM_WEAR_FEET
		9:  1 << 7,  // ITEM_WEAR_HANDS
		10: 1 << 8,  // ITEM_WEAR_ARMS
		11: 1 << 9,  // ITEM_WEAR_SHIELD
		12: 1 << 10, // ITEM_WEAR_ABOUT
		13: 1 << 11, // ITEM_WEAR_WAIST
		14: 1 << 12, // ITEM_WEAR_WRIST
		15: 1 << 12, // ITEM_WEAR_WRIST (alt)
		16: 1 << 13, // ITEM_WEAR_WIELD
		17: 1 << 14, // ITEM_WEAR_HOLD
		18: 1 << 16, // ITEM_WEAR_ABLEGS
		19: 1 << 17, // ITEM_WEAR_FACE
		20: 1 << 18, // ITEM_WEAR_HOVER
	}
	return m[cPos]
}

// Flag constants matching ITEM_* from structs.h used for alignment checks.
// ExtraFlags[0] bits.
const (
	FlagAntiGood    = 1 << 2  // ITEM_ANTI_GOOD
	FlagAntiEvil    = 1 << 3  // ITEM_ANTI_EVIL
	FlagAntiNeutral = 1 << 11 // ITEM_ANTI_NEUTRAL
	FlagNoRent      = 1 << 5  // ITEM_NORENT
)

// Extended equipment slots (M4 additions).
const (
	SlotAblegs EquipmentSlot = 100 + iota
	SlotFace
	SlotHover
)

// NumWears is the number of equipment slots (0-based). Matches NUM_WEARS in C (used in loops).
const NumWears = 21

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
// AutoEquip — matches the C auto_equip() logic.
// locate: C WEAR_* index + 1 (1 = worn at pos 0, 20 = worn at pos 19).
// ==========================================================================
func AutoEquip(p *Player, obj *ObjectInstance, locate int) {
	if locate <= 0 {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	cPos := locate - 1
	_, ok := CWearPosToSlot(cPos)
	if !ok {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (invalid pos)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	rf := cWearPosCanWearFlag(cPos)
	wf := obj.Prototype.WearFlags[0]
	wears := (wf & rf) != 0
	// Warriors can wield in hold slot.
	if cPos == 17 && !wears {
		if (wf&(1<<13)) != 0 && ItemType(obj.Prototype.TypeFlag) == ItemWeaponType {
			wears = true
		}
	}
	if !wears {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (cant wear)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	// Alignment restrictions.
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagAntiEvil != 0 && p.IsEvil()) ||
		(xf&FlagAntiGood != 0 && p.IsGood()) ||
		(xf&FlagAntiNeutral != 0 && p.IsNeutral()) {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (alignment)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	if err := p.Equipment.Equip(obj, p.Inventory); err != nil {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (equip failed)", "player", p.Name, "obj_vnum", obj.VNum, "original_err", err)
		}
	}
}

// ==========================================================================
// RestoreItemsFromSave — new function to create ObjectInstances from saved
// saveItemData, using prototype lookups. This closes the gap between
// saveDataToPlayer (which creates bare Inventory/Equipment) and the actual
// item restoration needed after load.
// ==========================================================================
func RestoreItemsFromSave(inv []SaveItemData, eq []SaveItemData, getProto func(vnum int) (*parser.Obj, bool)) ([]*ObjectInstance, map[int]*ObjectInstance) {
	invItems := make([]*ObjectInstance, 0, len(inv))
	for _, s := range inv {
		proto, ok := getProto(s.VNum)
		if !ok {
			slog.Warn("RestoreItemsFromSave: missing proto", "vnum", s.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if s.State != nil {
			for k, v := range s.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		invItems = append(invItems, obj)
	}

	eqItems := make(map[int]*ObjectInstance)
	for _, s := range eq {
		proto, ok := getProto(s.VNum)
		if !ok {
			slog.Warn("RestoreItemsFromSave: missing eq proto", "vnum", s.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if s.State != nil {
			for k, v := range s.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		eqItems[0] = obj // slot mapping handled by AutoEquip
	}

	return invItems, eqItems
}
