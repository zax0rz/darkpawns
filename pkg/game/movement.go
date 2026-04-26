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
		obj.RoomVNum = -1

	case ObjInInventory:
		if old.OwnerKind == OwnerPlayer {
			if p, ok := w.players[old.PlayerName]; ok {
				p.Inventory.removeItem(obj)
			}
		} else if old.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[old.MobID]; ok {
				m.RemoveFromInventory(obj)
			}
		}

	case ObjEquipped:
		if old.OwnerKind == OwnerPlayer {
			if p, ok := w.players[old.PlayerName]; ok && p.Equipment != nil {
				p.Equipment.unequip(old.Slot, p.Inventory)
			}
		} else if old.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[old.MobID]; ok {
				delete(m.Equipment, int(old.Slot))
				m.AddToInventory(obj) // return to inventory on unequip
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
		obj.RoomVNum = dst.RoomVNum

	case ObjInInventory:
		if dst.OwnerKind == OwnerPlayer {
			if p, ok := w.players[dst.PlayerName]; ok {
				if err := p.Inventory.addItem(obj); err != nil {
					return fmt.Errorf("attach to player %s inventory: %w", dst.PlayerName, err)
				}
			}
		} else if dst.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[dst.MobID]; ok {
				m.AddToInventory(obj)
			}
		}

	case ObjEquipped:
		if dst.OwnerKind == OwnerPlayer {
			if p, ok := w.players[dst.PlayerName]; ok && p.Equipment != nil {
				// Remove from inventory first if it's there
				p.Inventory.removeItem(obj)
				if err := p.Equipment.equip(obj, p.Inventory); err != nil {
					return fmt.Errorf("equip on player %s: %w", dst.PlayerName, err)
				}
			}
		} else if dst.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[dst.MobID]; ok {
				m.AddToInventory(obj)
				if m.Equipment != nil {
					m.Equipment[int(dst.Slot)] = obj
					// Remove from inventory since it's now equipped
					m.RemoveFromInventory(obj)
				}
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

// MoveObject moves an object from its current location to a new one.
// This is the centralized movement function. All object location changes
// should go through this to maintain invariant consistency.
func (w *World) MoveObject(obj *ObjectInstance, dst ObjectLocation) error {
	if err := dst.Validate(); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Detach from current location
	if _, err := w.detachObjectLocked(obj); err != nil {
		return fmt.Errorf("detach failed: %w", err)
	}

	// Attach to new location
	if err := w.attachObjectLocked(obj, dst); err != nil {
		// Best-effort re-attach to old Location on failure
		if rollbackErr := w.attachObjectLocked(obj, obj.Location); rollbackErr != nil {
			slog.Error("move object rollback failed",
				"obj_id", obj.ID, "obj_vnum", obj.VNum,
				"target", dst.Kind,
				"error", err,
				"rollback_error", rollbackErr,
			)
			obj.Location = LocNowhere()
		}
		return fmt.Errorf("attach failed: %w", err)
	}

	obj.Location = dst
	return nil
}

// --- Ergonomic helpers ---

func (w *World) MoveObjectToRoom(obj *ObjectInstance, roomVNum int) error {
	return w.MoveObject(obj, LocRoom(roomVNum))
}

func (w *World) MoveObjectToPlayerInventory(obj *ObjectInstance, p *Player) error {
	return w.MoveObject(obj, LocInventoryPlayer(p.Name))
}

func (w *World) MoveObjectToMobInventory(obj *ObjectInstance, m *MobInstance) error {
	return w.MoveObject(obj, LocInventoryMob(m.GetID()))
}

func (w *World) MoveObjectToContainer(obj, container *ObjectInstance) error {
	return w.MoveObject(obj, LocContainer(container.ID))
}

func (w *World) MoveObjectToNowhere(obj *ObjectInstance) error {
	return w.MoveObject(obj, LocNowhere())
}
