package game

// char_mgmt.go — character lifecycle and helper functions
//
// Ported from CircleMUD handler.c and utils.c.

import (
	"log/slog"
)

// CircleMUD constants used for character management.
const (
	plrExtractBit     = 21  // PLR_EXTRACT from structs.h (bit 21)
	roomNowhere       = -1  // NOWHERE from structs.h
	itemLightTypeFlag = 1   // ITEM_LIGHT from structs.h (type 1)
)

// ---------------------------------------------------------------------------
// has_light — handler.c:823
// ---------------------------------------------------------------------------

// HasLight checks if the player has a working light source equipped.
// Source: src/handler.c has_light() lines 823-835.
// A light source has TypeFlag==ITEM_LIGHT and Values[1] > 0 (hours remaining).
func (p *Player) HasLight() bool {
	if p.Equipment == nil {
		return false
	}
	item, ok := p.Equipment.GetItemInSlot(SlotLight)
	if !ok || item == nil || item.Prototype == nil {
		return false
	}
	if item.Prototype.TypeFlag == itemLightTypeFlag {
		return item.Prototype.Values[1] > 0
	}
	return false
}

// ---------------------------------------------------------------------------
// update_char_objects — handler.c:1016
// ---------------------------------------------------------------------------

// UpdateCharObjects processes light source timers each pulse.
// Source: src/handler.c update_char_objects() lines 1016-1042.
func (p *Player) UpdateCharObjects() {
	if p.Equipment == nil {
		return
	}
	item, ok := p.Equipment.GetItemInSlot(SlotLight)
	if !ok || item == nil || item.Prototype == nil {
		return
	}
	if item.Prototype.TypeFlag != itemLightTypeFlag {
		return
	}
	if item.Prototype.Values[1] > 0 {
		item.Prototype.Values[1]--
		if item.Prototype.Values[1] == 1 {
			p.SendMessage("Your light source flickers and sputters.\r\n")
		} else if item.Prototype.Values[1] <= 0 {
			p.SendMessage("Your light source has gone out.\r\n")
			if err := p.Equipment.Unequip(SlotLight, p.Inventory); err != nil {
				slog.Warn("light unequip failed", "player", p.Name, "error", err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// update_char_objects_ar — handler.c:1047
// ---------------------------------------------------------------------------

// UpdateCharObjectsAR processes light source timers in the anti-regen form.
// Source: src/handler.c update_char_objects_ar() lines 1047-1080.
// In the original C, update_char_objects_ar behaves exactly like
// update_char_objects but is called during anti-regen checks.
// (The "ar" suffix denotes "anti-regen" mode.)
func (p *Player) UpdateCharObjectsAR() {
	p.UpdateCharObjects()
}

// ---------------------------------------------------------------------------
// extract_char — handler.c:1194
// ---------------------------------------------------------------------------

// ExtractChar flags a character for extraction (removal from game).
// Source: src/handler.c extract_char() lines 1194-1221.
// The character is saved and then removed from the world on the next tick
// by ExtractPendingChars.
func ExtractChar(p *Player) {
	p.Flags |= 1 << uint(plrExtractBit)
}

// ExtractPendingChars processes all characters marked for extraction.
// Source: src/handler.c extract_pending_chars() lines 1221-1265.
// Must be called each heartbeat tick after event processing.
func (w *World) ExtractPendingChars() {
	w.mu.Lock()
	defer w.mu.Unlock()

	extractMask := uint64(1 << uint(plrExtractBit))
	for name, p := range w.players {
		if p.Flags&extractMask != 0 {
			slog.Debug("extracting player", "name", name)

			roomVNum := p.RoomVNum

			// Unequip ALL equipment, dropping each item to the room floor.
			// Source: handler.c extract_pending_chars — obj_to_room for every item.
			if p.Equipment != nil {
				p.Equipment.mu.Lock()
				for slot, item := range p.Equipment.Slots {
					if isLitLightSource(item) {
						w.adjustRoomLight(roomVNum, -1)
					}
					delete(p.Equipment.Slots, slot)
					if roomVNum >= 0 {
						w.roomItems[roomVNum] = append(w.roomItems[roomVNum], item)
						item.Location = LocRoom(roomVNum)
						item.RoomVNum = roomVNum
						if isLitLightSource(item) {
							w.adjustRoomLight(roomVNum, 1)
						}
					} else {
						item.Location = LocNowhere()
						item.RoomVNum = -1
					}
				}
				p.Equipment.mu.Unlock()
			}

			// Drop all carried inventory items to the room floor.
			if p.Inventory != nil {
				p.Inventory.mu.Lock()
				for _, item := range p.Inventory.Items {
					if roomVNum >= 0 {
						w.roomItems[roomVNum] = append(w.roomItems[roomVNum], item)
						item.Location = LocRoom(roomVNum)
						item.RoomVNum = roomVNum
					} else {
						item.Location = LocNowhere()
						item.RoomVNum = -1
					}
				}
				p.Inventory.Items = p.Inventory.Items[:0]
				p.Inventory.mu.Unlock()
			}

			// Stop fighting
			p.Fighting = ""

			// Move to nowhere
			p.RoomVNum = roomNowhere

			// Remove from world
			delete(w.players, name)

			// Save to disk
			_ = SavePlayer(p)

			// Clear flag
			p.Flags &^= extractMask

			slog.Debug("player extracted", "name", name)
		}
	}

	// Extract mobs flagged for deferred removal (MOB_EXTRACT / bit 25).
	// Source: handler.c extract_pending_chars mob loop.
	for id, m := range w.activeMobs {
		m.mu.Lock()
		if m.Flags&(1<<uint(MobFlagExtract)) == 0 {
			m.mu.Unlock()
			continue
		}
		mobRoom := m.RoomVNum

		// Drop equipment to room floor.
		for slot, item := range m.Equipment {
			if isLitLightSource(item) && mobRoom >= 0 {
				w.adjustRoomLight(mobRoom, -1)
			}
			delete(m.Equipment, slot)
			if mobRoom >= 0 {
				w.roomItems[mobRoom] = append(w.roomItems[mobRoom], item)
				item.Location = LocRoom(mobRoom)
				item.RoomVNum = mobRoom
				if isLitLightSource(item) {
					w.adjustRoomLight(mobRoom, 1)
				}
			} else {
				item.Location = LocNowhere()
				item.RoomVNum = -1
			}
		}

		// Drop inventory to room floor.
		for _, item := range m.Inventory {
			if mobRoom >= 0 {
				w.roomItems[mobRoom] = append(w.roomItems[mobRoom], item)
				item.Location = LocRoom(mobRoom)
				item.RoomVNum = mobRoom
			} else {
				item.Location = LocNowhere()
				item.RoomVNum = -1
			}
		}
		m.Inventory = m.Inventory[:0]
		m.mu.Unlock()

		slog.Debug("mob extracted", "id", id)
		delete(w.activeMobs, id)
	}
}

// ---------------------------------------------------------------------------
// update_object — handler.c:1006
// ---------------------------------------------------------------------------

// UpdateObject decrements an object's timer by `use` ticks.
// Source: src/handler.c update_object() lines 1006-1014.
func UpdateObject(obj *ObjectInstance, use int) {
	if obj == nil {
		return
	}
	timer := obj.GetTimer()
	if timer > 0 {
		obj.SetTimer(timer - use)
	}
	for _, c := range obj.Contains {
		UpdateObject(c, use)
	}
}
