package game

import (
	"strings"
)

func (w *World) GetItemsInRoom(roomVNum int) []*ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.roomItems[roomVNum]
}

// GetItemsInRoomI returns room items as []interface{} for spell layer access.
func (w *World) GetItemsInRoomI(roomVNum int) []interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	items := w.roomItems[roomVNum]
	result := make([]interface{}, len(items))
	for i, item := range items {
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

// extractObjectLocked removes an object (and its contained children) from the world.
// Caller MUST hold w.mu. — handler.c:1006-1025
func (w *World) extractObjectLocked(obj *ObjectInstance) {
	// Recursively extract contents first — handler.c:1020-1024
	for _, child := range obj.Contains {
		w.extractObjectLocked(child)
	}
	obj.Contains = obj.Contains[:0]

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
			}
		case OwnerMob:
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				m.RemoveFromInventory(obj)
			}
		}
	case ObjEquipped:
		if obj.Location.OwnerKind == OwnerPlayer {
			if p, ok := w.players[obj.Location.PlayerName]; ok && p.Equipment != nil {
				p.Equipment.UnequipItem(obj, p.Inventory)
			}
		} else if obj.Location.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				for pos, eqItem := range m.Equipment {
					if eqItem == obj {
						delete(m.Equipment, pos)
						m.RemoveFromInventory(obj)
						break
					}
				}
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
