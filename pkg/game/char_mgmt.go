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
			removed := p.Equipment.Unequip(SlotLight, p.Inventory)
			_ = removed
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

			// Unequip light
			if p.Equipment != nil {
				item, ok := p.Equipment.GetItemInSlot(SlotLight)
				if ok && item != nil {
					p.Equipment.Unequip(SlotLight, p.Inventory)
				}
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
