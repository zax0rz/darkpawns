package game

import (
	"strings"
)

// GetItemsInRoom returns a snapshot copy: callers previously iterated the
// live slice while writers mutated it under w.mu — a fatal concurrent map
// read (VULN-019) and a torn iteration (VULN-055).
func (w *World) GetItemsInRoom(roomVNum int) []*ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]*ObjectInstance(nil), w.roomItems[roomVNum]...)
}

// GetItemsInRoomI returns room items as []interface{} for spell layer access.
func (w *World) GetItemsInRoomI(roomVNum int) []interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	snapshot := append([]*ObjectInstance(nil), w.roomItems[roomVNum]...)
	result := make([]interface{}, len(snapshot))
	for i, item := range snapshot {
		result[i] = item
	}
	return result
}

// AddItemToRoom appends an item to a room's item list.
// MED-023: Now sets Location and RoomVNum on the object, matching MoveObjectToRoom behavior.
// Prefer MoveObjectToRoom for new code (it also handles detach from current location).
func (w *World) AddItemToRoom(item *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roomItems[roomVNum] = append(w.roomItems[roomVNum], item)
	item.SetRoomVNum(roomVNum)
}

// AddItemToRoomFront is the raw-placement twin of AddItemToRoom for callers
// that port C obj_to_room (handler.c:897-910), which prepends to the room's
// contents: zone O resets (db.c:2155), house loads (house.c:100), and mob
// drops driven by the action() command path (perform_drop, act.item.c:504).
func (w *World) AddItemToRoomFront(item *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := w.roomItems[roomVNum]
	w.roomItems[roomVNum] = append(items, nil)
	copy(w.roomItems[roomVNum][1:], items)
	w.roomItems[roomVNum][0] = item
	item.SetRoomVNum(roomVNum)
}

// extractObjectLocked removes an object (and its contained children) from the world.
// Caller MUST hold w.mu. — handler.c:1006-1025
func (w *World) extractObjectLocked(obj *ObjectInstance) {
	// Recursively extract contents first — handler.c:1020-1024
	children := obj.Contains
	obj.Contains = nil
	for _, child := range children {
		w.extractObjectLocked(child)
	}

	// Remove from room if applicable
	if obj.Location.Kind == ObjInRoom {
		w.removeItemFromRoomLocked(obj, obj.Location.RoomVNum)
	}

	// Remove from carrier (inventory) based on Location
	switch obj.Location.Kind {
	case ObjInInventory:
		switch obj.Location.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[obj.Location.PlayerName]; ok {
				p.Inventory.removeItem(obj)
				// C's extract_obj removes a carried object through
				// obj_from_char (handler.c:1016-1017), which sets PLR_CRASH
				// for non-NPCs (handler.c:596-598).
				p.MarkCrashNeeded()
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(obj.Location.MobID); m != nil {
				m.RemoveFromInventory(obj)
			}
		}
	case ObjEquipped:
		switch obj.Location.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[obj.Location.PlayerName]; ok && p.Equipment != nil {
				// UnequipItem transfers the object into inventory. Extraction
				// must remove that transferred reference too; otherwise an
				// equipped NORENT object survives Crash_rentsave in the saved
				// inventory even though its world registration was deleted.
				if p.Equipment.UnequipItem(obj, p.Inventory) && p.Inventory != nil {
					p.Inventory.removeItem(obj)
				}
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(obj.Location.MobID); m != nil {
				m.mu.Lock()
				m.unequipMobileLocked(w, int(obj.Location.Slot))
				m.mu.Unlock()
			}
		}
	}

	// Remove from container based on Location
	if obj.Location.Kind == ObjInContainer && obj.Location.ContainerObjID > 0 {
		if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok {
			container.RemoveFromContainer(obj)
		}
	}

	obj.Location = LocNowhere()
	delete(w.objectInstances, obj.ID)
}

// ExtractObject removes an object from the world entirely.
// Recursively extracts container contents before removing the object itself.
func (w *World) ExtractObject(obj *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.extractObjectLocked(obj)
}

// RemoveItemFromRoomI removes an item (passed as interface{}) from a room.
// Used by the spells layer to avoid importing game.ObjectInstance.
func (w *World) RemoveItemFromRoomI(item interface{}, roomVNum int) {
	if obj, ok := item.(*ObjectInstance); ok {
		w.RemoveItemFromRoom(obj, roomVNum)
	}
}

// RemoveItemFromRoom removes an item from a room.
func (w *World) RemoveItemFromRoom(item *ObjectInstance, roomVNum int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	items := w.roomItems[roomVNum]
	for i, it := range items {
		if it == item {
			w.roomItems[roomVNum] = append(items[:i], items[i+1:]...)
			return true
		}
	}
	return false
}

// removeItemFromRoomLocked removes an item from a room. Caller must hold w.mu.
func (w *World) removeItemFromRoomLocked(item *ObjectInstance, roomVNum int) bool {
	items := w.roomItems[roomVNum]
	for i, it := range items {
		if it == item {
			w.roomItems[roomVNum] = append(items[:i], items[i+1:]...)
			return true
		}
	}
	return false
}

// FindObjectByName searches all objects in the world by keyword name.
// Returns matching objects as []interface{} for spell system compatibility.
func (w *World) FindObjectByName(name string) []interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var results []interface{}
	name = strings.ToLower(name)

	for _, objs := range w.roomItems {
		for _, obj := range objs {
			if obj.Prototype != nil && strings.Contains(strings.ToLower(obj.Prototype.Keywords), name) {
				results = append(results, obj)
			}
		}
	}
	return results
}

// GetMobPrototype returns a mob prototype by VNum.
