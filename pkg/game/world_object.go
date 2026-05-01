package game

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

// Deprecated: AddItemToRoom only appends to roomItems without setting Location or RoomVNum.
// Use MoveObjectToRoom instead, which properly handles detach/attach and location tracking.
func (w *World) AddItemToRoom(item *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roomItems[roomVNum] = append(w.roomItems[roomVNum], item)
}

// ExtractObject removes an object from the world entirely.
// Removes from room, carrier, container, and the global instance map.
func (w *World) ExtractObject(obj *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Remove from room (inline to avoid lock reentrancy — RemoveItemFromRoom also locks)
	w.removeItemFromRoomLocked(obj, roomVNum)

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
		// Actually unequip the item before extraction
		if obj.Location.OwnerKind == OwnerPlayer {
			if p, ok := w.players[obj.Location.PlayerName]; ok && p.Equipment != nil {
				p.Equipment.UnequipItem(obj, p.Inventory)
			}
		} else if obj.Location.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				// Mobs use int-keyed equipment map
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

	// Remove from global instance map
	delete(w.objectInstances, obj.ID)
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

// GetMobPrototype returns a mob prototype by VNum.
