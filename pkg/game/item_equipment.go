package game

import (
	"fmt"
	"strings"
)

// findEqPos finds the equipment position for an object
func findEqPos(obj *ObjectInstance, arg string) int {
	if arg != "" {
		if pos, ok := eqPosKeywords[strings.ToLower(arg)]; ok {
			return pos
		}
		return -1
	}

	// Auto-detect
	if canWearObject(obj, eqWearFingerR) {
		return eqWearFingerR
	}
	if canWearObject(obj, eqWearNeck1) {
		return eqWearNeck1
	}
	if canWearObject(obj, eqWearBody) {
		return eqWearBody
	}
	if canWearObject(obj, eqWearHead) {
		return eqWearHead
	}
	if canWearObject(obj, eqWearLegs) {
		return eqWearLegs
	}
	if canWearObject(obj, eqWearFeet) {
		return eqWearFeet
	}
	if canWearObject(obj, eqWearHands) {
		return eqWearHands
	}
	if canWearObject(obj, eqWearArms) {
		return eqWearArms
	}
	if canWearObject(obj, eqWearShield) {
		return eqWearShield
	}
	if canWearObject(obj, eqWearAbout) {
		return eqWearAbout
	}
	if canWearObject(obj, eqWearWaist) {
		return eqWearWaist
	}
	if canWearObject(obj, eqWearWristR) {
		return eqWearWristR
	}
	if canWearObject(obj, eqWearAblegs) {
		return eqWearAblegs
	}
	if canWearObject(obj, eqWearFace) {
		return eqWearFace
	}
	if canWearObject(obj, eqWearHover) {
		return eqWearHover
	}
	if canWearObject(obj, eqWearWield) {
		return eqWearWield
	}
	if canWearObject(obj, eqWearHold) {
		return eqWearHold
	}
	return -1
}

// wearMessage sends the wear message for an equipment position
func (w *World) wearMessage(ch *Player, obj *ObjectInstance, where int) {
	if where < 0 || where >= len(wearMessages) {
		return
	}
	msg := wearMessages[where]
	// Room message (TO_ROOM)
	w.actToRoom(ch, msg[0], obj, nil)
	// Character message (TO_CHAR)
	w.actToChar(ch, msg[1], obj, nil)
}

// performWear equips an item at a given position
func (w *World) performWear(ch *Player, obj *ObjectInstance, where int) {
	if !canWearObject(obj, where) || where == eqWearLight {
		w.actToChar(ch, "You can't wear $p there.", obj, nil)
		return
	}

	// For finger, neck, wrist: try secondary if primary full
	if where == eqWearFingerR {
		// Check if slot is occupied, try other finger
		if w.IsEquipped(ch, eqWearFingerR) {
			where = eqWearFingerL
		}
	}
	if where == eqWearNeck1 {
		if w.IsEquipped(ch, eqWearNeck1) {
			where = eqWearNeck2
		}
	}
	if where == eqWearWristR {
		if w.IsEquipped(ch, eqWearWristR) {
			where = eqWearWristL
		}
	}

	if w.IsEquipped(ch, where) {
		if where >= 0 && where < len(alreadyWearing) {
			ch.SendMessage(alreadyWearing[where])
		}
		return
	}

	// Wielding checks
	if where == eqWearWield {
		if !canWearObject(obj, eqWearWield) {
			ch.SendMessage("You can't wield that.\r\n")
			return
		}
		if obj.GetWeight() > 50 { // simplified str_app check
			ch.SendMessage("It is too heavy for you to use.\r\n")
			return
		}
		// Check for two-handed
		if objHasFlag(obj, 1<<3) && (w.IsEquipped(ch, eqWearHold) || w.IsEquipped(ch, eqWearShield)) {
			ch.SendMessage("Both hands must be free to wield that.\r\n")
			return
		}
	} else if where == eqWearHold || where == eqWearShield {
		if w.IsEquipped(ch, eqWearWield) {
			// Check if wielded weapon is two-handed
			wpn := w.GetEquipped(ch, eqWearWield)
			if wpn != nil && objHasFlag(wpn, 1<<3) {
				ch.SendMessage("Both your hands are occupied with your weapon at the moment.\r\n")
				return
			}
		}
	}

	// Remove from inventory and equip
	ch.Inventory.removeItem(obj)
	w.EquipItem(ch, obj, where)
	w.wearMessage(ch, obj, where)
}

// IsEquipped checks if a character has something equipped in a slot (0-based eq pos)
func (w *World) IsEquipped(ch *Player, slot int) bool {
	if ch.Equipment == nil {
		return false
	}
	_, found := ch.Equipment.GetItemInSlot(EquipmentSlot(slot))
	return found
}

// GetEquipped returns the item in a given slot
func (w *World) GetEquipped(ch *Player, slot int) *ObjectInstance {
	if ch.Equipment == nil {
		return nil
	}
	item, found := ch.Equipment.GetItemInSlot(EquipmentSlot(slot))
	if !found {
		return nil
	}
	return item
}

// EquipItem equips an item at the given slot
func (w *World) EquipItem(ch *Player, obj *ObjectInstance, slot int) {
	if ch.Equipment == nil {
		return
	}
	ch.Equipment.Equip(obj, ch.Inventory)
}

// UnequipItem removes an item from a slot
func (w *World) UnequipItem(ch *Player, slot int) {
	if ch.Equipment == nil {
		return
	}
	ch.Equipment.Unequip(EquipmentSlot(slot), ch.Inventory)
}

// doWear handles the wear command
func (w *World) doWear(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		ch.SendMessage("Wear what?\r\n")
		return true
	}

	dotmode := findAllDots(arg1)

	if arg2 != "" && dotmode != findIndiv {
		ch.SendMessage("You can't specify the same body location for more than one item!\r\n")
		return true
	}

	if dotmode == findAll {
		items := ch.Inventory.Items
		if len(items) == 0 {
			ch.SendMessage("You don't seem to have anything to wear.\r\n")
			return true
		}
		for _, obj := range items {
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if where := findEqPos(obj, ""); where >= 0 {
				w.performWear(ch, obj, where)
			} else {
				w.actToChar(ch, "You can't wear $p.", obj, nil)
			}
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("Wear all of what?\r\n")
			return true
		}
		found := false
		items := ch.Inventory.Items
		for _, obj := range items {
			if isname(keyword, obj.GetKeywords()) {
				if where := findEqPos(obj, ""); where >= 0 {
					w.performWear(ch, obj, where)
					found = true
				} else {
					w.actToChar(ch, "You can't wear $p.", obj, nil)
				}
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	if where := findEqPos(obj, arg2); where >= 0 {
		w.performWear(ch, obj, where)
	} else if arg2 == "" {
		w.actToChar(ch, "You can't wear $p.", obj, nil)
	}
	return true
}

// performRemove removes an item from an equipment slot
func (w *World) performRemove(ch *Player, pos int) {
	if ch.Equipment == nil {
		return
	}
	obj, found := ch.Equipment.GetItemInSlot(EquipmentSlot(pos))
	if !found {
		return
	}
	if objHasFlag(obj, 1<<0) {
		w.actToChar(ch, "You can't remove $p, it must be CURSED!", obj, nil)
		return
	}
	if len(ch.Inventory.Items) >= ch.Inventory.Capacity {
		w.actToChar(ch, "$p: you can't carry that many items!", obj, nil)
		return
	}

	w.UnequipItem(ch, pos)

	w.actToChar(ch, "You stop using $p.", obj, nil)
	w.actToRoom(ch, "$n stops using $p.", obj, nil)
}

// doRemove handles the remove command
func (w *World) doRemove(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Remove what?\r\n")
		return true
	}
	arg1 := parts[0]

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		found := false
		for i := 0; i < eqWearMax; i++ {
			if w.IsEquipped(ch, i) {
				w.performRemove(ch, i)
				found = true
			}
		}
		if !found {
			ch.SendMessage("You're not using anything.\r\n")
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("Remove all of what?\r\n")
			return true
		}
		found := false
		for i := 0; i < eqWearMax; i++ {
			if w.IsEquipped(ch, i) {
				obj, _ := ch.Equipment.GetItemInSlot(EquipmentSlot(i))
				if obj != nil && isname(keyword, obj.GetKeywords()) {
					w.performRemove(ch, i)
					found = true
				}
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to be using any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual remove
	for i := 0; i < eqWearMax; i++ {
		if w.IsEquipped(ch, i) {
			obj, _ := ch.Equipment.GetItemInSlot(EquipmentSlot(i))
			if obj != nil && isname(arg1, obj.GetKeywords()) {
				w.performRemove(ch, i)
				return true
			}
		}
	}
	ch.SendMessage(fmt.Sprintf("You don't seem to be using %s %s.\r\n", an(arg1), arg1))
	return true
}
