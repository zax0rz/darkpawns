package game

import (
	"fmt"
	"strings"
)

// DOOR_IS_OPENABLE for containers
func doorIsOpenable(obj *ObjectInstance) bool {
	return obj.GetTypeFlag() == ITEM_CONTAINER && contIsCloseable(obj)
}

// doOpen handles opening containers
func (w *World) doOpen(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Open what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Look for container in inventory or room
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if !doorIsOpenable(obj) {
		w.actToChar(ch, "You can't open $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("It's already open!\r\n")
		return true
	}
	if contIsLocked(obj) {
		ch.SendMessage("It seems to be locked.\r\n")
		return true
	}

	contSetClosed(obj, false)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n opens $p.", obj, nil)
	return true
}

// doClose handles closing containers
func (w *World) doClose(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Close what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if !doorIsOpenable(obj) {
		w.actToChar(ch, "You can't close $p!", obj, nil)
		return true
	}
	if contIsClosed(obj) {
		ch.SendMessage("It's already closed!\r\n")
		return true
	}

	contSetClosed(obj, true)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n closes $p.", obj, nil)
	return true
}

// doUnlock handles unlocking containers
func (w *World) doUnlock(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Unlock what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if obj.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "That's not a container.", nil, nil)
		return true
	}
	if !contIsCloseable(obj) {
		w.actToChar(ch, "You can't unlock $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("It's not closed.\r\n")
		return true
	}
	if !contIsLocked(obj) {
		ch.SendMessage("It's not locked.\r\n")
		return true
	}

	// Check for key in inventory
	keyVNum := obj.Prototype.Values[contKey]
	if keyVNum > 0 {
		hasKey := false
		for _, inv := range ch.Inventory.Items {
			if inv.GetTypeFlag() == ITEM_KEY && inv.GetVNum() == keyVNum {
				hasKey = true
				break
			}
		}
		if !hasKey {
			w.actToChar(ch, "You don't have the key for $p.", obj, nil)
			return true
		}
	}

	if contIsLocked(obj) {
		contSetLocked(obj, false)
	}

	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n unlocks $p.", obj, nil)
	return true
}

// doLock handles locking containers
func (w *World) doLock(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Lock what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if obj.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "That's not a container.", nil, nil)
		return true
	}
	if !contIsCloseable(obj) {
		w.actToChar(ch, "You can't lock $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("You'd better close it first.\r\n")
		return true
	}
	if contIsLocked(obj) {
		ch.SendMessage("It's already locked!\r\n")
		return true
	}

	// Check for key in inventory
	keyVNum := obj.Prototype.Values[contKey]
	if keyVNum > 0 {
		hasKey := false
		for _, inv := range ch.Inventory.Items {
			if inv.GetTypeFlag() == ITEM_KEY && inv.GetVNum() == keyVNum {
				hasKey = true
				break
			}
		}
		if !hasKey {
			w.actToChar(ch, "You don't have the key for $p.", obj, nil)
			return true
		}
	}

	contSetLocked(obj, true)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n locks $p.", obj, nil)
	return true
}
