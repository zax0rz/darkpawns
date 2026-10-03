// Lock ordering (MUST be maintained to prevent deadlocks):
//   w.mu → Equipment.mu → Inventory.mu
// World movement code owns location changes. Equipment/Inventory
// methods should not call back into World.

package game

import (
	"fmt"
	"log/slog"
)

// detachObjectLocked removes an object from its current location.
// Caller MUST hold w.mu.
// Returns the old location for reference.
func (w *World) detachObjectLocked(obj *ObjectInstance) (ObjectLocation, error) {
	old := obj.Location

	// Detach based on Location field
	switch old.Kind {
	case ObjNowhere:
		// Nothing location-based to detach

	case ObjInRoom:
		w.removeItemFromRoomLocked(obj, old.RoomVNum)
		// C obj_from_room does not alter room light (src/handler.c).
		obj.RoomVNum = -1

	case ObjInInventory:
		switch old.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[old.PlayerName]; ok {
				p.Inventory.removeItem(obj)
				// C obj_from_char sets PLR_CRASH after the removal
				// (handler.c:596-598).
				p.MarkCrashNeeded()
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(old.MobID); m != nil {
				m.RemoveFromInventory(obj)
			}
		}

	case ObjEquipped:
		switch old.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[old.PlayerName]; ok && p.Equipment != nil {
				// If a light source is unequipped, decrement room light
				if isLitLightSource(obj) {
					w.adjustRoomLight(p.RoomVNum, -1)
				}
				if err := p.Equipment.Unequip(old.Slot, p.Inventory); err != nil {
					slog.Warn("unequip failed in detachObject", "player", p.Name, "slot", old.Slot, "error", err)
				} else {
					// unequip adds to inventory; this move owns the next destination.
					// C unequip_char returns the object without obj_to_char (:754-783).
					p.Inventory.removeItem(obj)
				}
				// No PLR_CRASH here: C's unequip_char only clears the eq
				// link (handler.c:754-783); the flag is obj_to_char's /
				// obj_from_char's alone. An object that continues into the
				// inventory is flagged by the attach arm below (C's
				// obj_to_char, handler.c:569-571); corpse and extract paths
				// never reach this arm's inventory (fight.c:406-407 uses
				// obj_to_obj, extract_obj uses unequip_char only,
				// handler.c:1010-1012).
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(old.MobID); m != nil {
				m.mu.Lock()
				m.unequipMobileLocked(w, int(old.Slot))
				m.mu.Unlock()
			}
		}

	case ObjInContainer:
		if old.ContainerObjID > 0 {
			if container, ok := w.objectInstances[old.ContainerObjID]; ok {
				container.RemoveFromContainer(obj)
			}
		}

	case ObjInShop:
		// for now just clear old fields
	}

	return old, nil
}

// attachObjectLocked adds an object to a destination location.
// Caller MUST hold w.mu.
func (w *World) attachObjectLocked(obj *ObjectInstance, dst ObjectLocation) error {
	switch dst.Kind {
	case ObjNowhere:
		return nil

	case ObjInRoom:
		w.roomItems[dst.RoomVNum] = append(w.roomItems[dst.RoomVNum], obj)
		// C obj_to_room does not alter room light (src/handler.c:897-913).
		obj.RoomVNum = dst.RoomVNum

	case ObjInInventory:
		switch dst.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[dst.PlayerName]; ok {
				if err := p.Inventory.addItem(obj); err != nil {
					return fmt.Errorf("attach to player %s inventory: %w", dst.PlayerName, err)
				}
				// C obj_to_char sets PLR_CRASH after the attach
				// (handler.c:569-571).
				p.MarkCrashNeeded()
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(dst.MobID); m != nil {
				m.AddToInventory(obj)
			}
		}

	case ObjEquipped:
		switch dst.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[dst.PlayerName]; ok && p.Equipment != nil {
				// If a light source is equipped, increment room light
				if isLitLightSource(obj) {
					w.adjustRoomLight(p.RoomVNum, 1)
				}
				// Remove from inventory first if it's there
				p.Inventory.removeItem(obj)
				if err := p.Equipment.Equip(obj, p.Inventory); err != nil {
					return fmt.Errorf("equip on player %s: %w", dst.PlayerName, err)
				}
			}
		case OwnerMob:
			if m := w.mobileObjectOwnerLocked(dst.MobID); m != nil {
				m.mu.Lock()
				_, err := m.equipMobileLocked(w, obj, int(dst.Slot))
				m.mu.Unlock()
				return err
			}
		}

	case ObjInContainer:
		if dst.ContainerObjID > 0 {
			// Prevent container cycles: A contains B contains A
			if container, ok := w.objectInstances[dst.ContainerObjID]; ok {
				current := container
				depth := 0
				for current != nil && depth < 10 {
					if current.ID == obj.ID {
						return fmt.Errorf("container cycle detected: object %d would contain itself", obj.ID)
					}
					if current.Location.Kind == ObjInContainer && current.Location.ContainerObjID > 0 {
						if parent, ok := w.objectInstances[current.Location.ContainerObjID]; ok {
							current = parent
						} else {
							break
						}
					} else {
						break
					}
					depth++
				}

				if !container.AddToContainer(obj) {
					return fmt.Errorf("container %d cannot hold object", dst.ContainerObjID)
				}
			} else {
				return fmt.Errorf("container object %d not found", dst.ContainerObjID)
			}
		}

	case ObjInShop:
	}

	return nil
}

// moveObjectLocked moves an object while the world lock is held.
func (w *World) moveObjectLocked(obj *ObjectInstance, dst ObjectLocation) error {
	// Detach from current location
	if _, err := w.detachObjectLocked(obj); err != nil {
		return fmt.Errorf("detach failed: %w", err)
	}

	// Attach to new location
	if err := w.attachObjectLocked(obj, dst); err != nil {
		// Best-effort re-attach to old Location on failure
		if rollbackErr := w.attachObjectLocked(obj, obj.Location); rollbackErr != nil {
			slog.Error(
				"move object rollback failed — object stranded",
				"obj_id", obj.ID, "obj_vnum", obj.VNum,
				"target", dst.Kind,
				"error", err,
				"rollback_error", rollbackErr,
			)
			obj.Location = LocNowhere()
			return fmt.Errorf("attach failed: %w; rollback also failed: %w", err, rollbackErr)
		}
		return fmt.Errorf("attach failed: %w", err)
	}

	obj.Location = dst
	return nil
}

// MoveObject moves an object from its current location to a new one.
// This is the centralized movement function. All object location changes
// should go through this to maintain invariant consistency.
func (w *World) MoveObject(obj *ObjectInstance, dst ObjectLocation) error {
	if dst.Kind == ObjEquipped && dst.OwnerKind == OwnerMob {
		w.mu.Lock()
		m := w.mobileObjectOwnerLocked(dst.MobID)
		if m == nil {
			w.mu.Unlock()
			return fmt.Errorf("mobile owner not found")
		}
		_, err := w.detachObjectLocked(obj)
		if err != nil {
			w.mu.Unlock()
			return err
		}
		obj.Location = LocNowhere()
		m.mu.Lock()
		effects, err := m.equipMobileLocked(w, obj, int(dst.Slot))
		m.mu.Unlock()
		w.mu.Unlock()
		if err == nil {
			w.finishMobileEquipment(m, obj, effects)
		}
		return err
	}
	if err := dst.Validate(); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.moveObjectLocked(obj, dst)
}

// MoveObjectToRoomFront mirrors C obj_to_room, which prepends to the room's
// object list. It is used by C paths whose output can expose room list order.
func (w *World) MoveObjectToRoomFront(obj *ObjectInstance, roomVNum int) error {
	dst := LocRoom(roomVNum)
	if err := dst.Validate(); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.moveObjectLocked(obj, dst); err != nil {
		return err
	}

	items := w.roomItems[roomVNum]
	for i, item := range items {
		if item != obj {
			continue
		}
		copy(items[1:i+1], items[:i])
		items[0] = obj
		return nil
	}
	return fmt.Errorf("moved object %d missing from room %d", obj.ID, roomVNum)
}

// --- Ergonomic helpers ---

func (w *World) MoveObjectToRoom(obj *ObjectInstance, roomVNum int) error {
	return w.MoveObject(obj, LocRoom(roomVNum))
}

func (w *World) MoveObjectToPlayerInventory(obj *ObjectInstance, p *Player) error {
	return w.MoveObject(obj, LocInventoryPlayer(p.Name))
}

// PlaceWizardLoadedObjectInInventory mirrors C obj_to_char() for the immortal
// load command, which deliberately bypasses mortal carry limits. C prepends
// each loaded object to ch->carrying, so preserve that visible list order.
func (w *World) PlaceWizardLoadedObjectInInventory(obj *ObjectInstance, p *Player) error {
	if obj == nil || p == nil || p.Inventory == nil {
		return fmt.Errorf("object and player inventory are required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if registered, ok := w.players[p.Name]; !ok || registered != p {
		return fmt.Errorf("player %s is not in the world", p.Name)
	}
	if _, err := w.detachObjectLocked(obj); err != nil {
		return fmt.Errorf("detach loaded object: %w", err)
	}
	p.Inventory.mu.Lock()
	p.Inventory.Items = append(p.Inventory.Items, nil)
	copy(p.Inventory.Items[1:], p.Inventory.Items[:len(p.Inventory.Items)-1])
	p.Inventory.Items[0] = obj
	p.Inventory.mu.Unlock()
	obj.RoomVNum = -1
	obj.Location = LocInventoryPlayer(p.Name)
	return nil
}

func (w *World) MoveObjectToMobInventory(obj *ObjectInstance, m *MobInstance) error {
	return w.MoveObject(obj, LocInventoryMob(m.GetID()))
}

// MoveObjectToMobInventoryFront mirrors C obj_to_char, which prepends to the
// mob's carrying list. It is used by C paths whose inventory order is visible.
func (w *World) MoveObjectToMobInventoryFront(obj *ObjectInstance, m *MobInstance) error {
	dst := LocInventoryMob(m.GetID())
	if err := dst.Validate(); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.moveObjectLocked(obj, dst); err != nil {
		return err
	}

	for i, item := range m.Inventory {
		if item != obj {
			continue
		}
		copy(m.Inventory[1:i+1], m.Inventory[:i])
		m.Inventory[0] = obj
		return nil
	}
	return fmt.Errorf("moved object %d missing from mob %d inventory", obj.ID, m.GetID())
}

func (w *World) MoveObjectToContainer(obj, container *ObjectInstance) error {
	return w.MoveObject(obj, LocContainer(container.ID))
}

func (w *World) MoveObjectToNowhere(obj *ObjectInstance) error {
	return w.MoveObject(obj, LocNowhere())
}

// putResetObject is reset_zone's obj_to_obj path for a newly read object.
// Unlike the player put command, C does not require ITEM_CONTAINER here
// (db.c:2167-2184; handler.c:939-954). Ownership stays in the canonical registry.
func (w *World) putResetObject(obj, target *ObjectInstance) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if obj == target {
		return nil
	} // C logs and leaves the new object floating.
	if w.objectInstances[obj.ID] != obj || w.objectInstances[target.ID] != target {
		return fmt.Errorf("reset object or target is no longer live")
	}
	if obj.Location.Kind != ObjNowhere {
		return fmt.Errorf("reset object is not floating")
	}
	target.Contains = append([]*ObjectInstance{obj}, target.Contains...)
	obj.Location = LocContainer(target.ID)
	return nil
}

// equipResetObject is the same floating-object boundary used by Lua and wear.
func (w *World) equipResetObject(obj *ObjectInstance, mob *MobInstance, position int) error {
	return w.EquipMobileObject(mob, obj, position)
}
