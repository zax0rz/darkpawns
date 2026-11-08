package game

import (
	"fmt"
	"sort"
	"strings"
)

// SpawnObjectToPlayerInventoryI mirrors read_object() followed by obj_to_char()
// for the spell layer without exposing concrete game types there.
func (w *World) SpawnObjectToPlayerInventoryI(vnum int, player any) (any, error) {
	p, ok := player.(*Player)
	if !ok || p == nil {
		return nil, fmt.Errorf("player is required")
	}
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		return nil, err
	}
	if err := w.PlaceObjectInPlayerInventory(obj, p); err != nil {
		w.ExtractObject(obj, -1)
		return nil, fmt.Errorf("place spawned object in player inventory: %w", err)
	}
	return obj, nil
}

// SpawnObjectToRoomForSpell mirrors read_object() followed by obj_to_room()
// for spell code without exposing the concrete game object type there.
func (w *World) SpawnObjectToRoomForSpell(vnum, roomVNum int) (any, error) {
	obj, err := w.SpawnObject(vnum, roomVNum)
	if err != nil {
		return nil, err
	}
	w.AddItemToRoom(obj, roomVNum)
	return obj, nil
}

// GetItemsInRoom returns a snapshot copy: callers previously iterated the
// live slice while writers mutated it under w.mu — a fatal concurrent map
// read (VULN-019) and a torn iteration (VULN-055).
func (w *World) GetItemsInRoom(roomVNum int) []*ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]*ObjectInstance(nil), w.roomItems[roomVNum]...)
}

// GetItemsInRoomI returns room items as []interface{} for spell layer access.
func (w *World) GetItemsInRoomI(roomVNum int) []any {
	w.mu.RLock()
	defer w.mu.RUnlock()
	snapshot := append([]*ObjectInstance(nil), w.roomItems[roomVNum]...)
	result := make([]any, len(snapshot))
	for i, item := range snapshot {
		result[i] = item
	}
	return result
}

// AddItemToRoom places an item at the head of a room's C-ordered contents.
// MED-023: Now sets Location and RoomVNum on the object, matching MoveObjectToRoom behavior.
// Prefer MoveObjectToRoom for new code (it also handles detach from current location).
func (w *World) AddItemToRoom(item *ObjectInstance, roomVNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := w.roomItems[roomVNum]
	w.roomItems[roomVNum] = append(items, nil)
	copy(w.roomItems[roomVNum][1:], items)
	w.roomItems[roomVNum][0] = item
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
				p.Inventory.RemoveItem(obj)
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
				// extract_obj unequips a worn object through unequip_char
				// (src/handler.c:1010-1012), dropping a lit light from the
				// owner's room (src/handler.c:779-782).
				litLight := isLitLightSource(obj)
				room := p.GetRoom()
				// UnequipItem transfers the object into inventory. Extraction
				// must remove that transferred reference too; otherwise an
				// equipped NORENT object survives Crash_rentsave in the saved
				// inventory even though its world registration was deleted.
				if p.Equipment.UnequipItem(obj, p.Inventory) && p.Inventory != nil {
					p.Inventory.RemoveItem(obj)
				}
				if litLight && room >= 0 {
					w.adjustRoomLight(room, -1)
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
func (w *World) RemoveItemFromRoomI(item any, roomVNum int) {
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
// It preserves C object_list order (newest first) and exact isname token
// matching, after spell_locate_object's fname() reduction. C references:
// src/spells.c:361-365, src/handler.c:68-95, src/db.c:1892-1893,1950-1951.
// Returns matching objects as []interface{} for spell system compatibility.
func (w *World) FindObjectByName(name string) []any {
	name = CFName(name)
	if name == "" {
		return nil
	}
	objects := w.GetAllObjects()
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID > objects[j].ID })
	results := make([]any, 0, len(objects))
	for _, obj := range objects {
		if obj != nil && cIsName(name, obj.GetKeywords()) {
			results = append(results, obj)
		}
	}
	return results
}

// LocateObjectResults formats spell_locate_object's object_list matches.
// The bool is true when C's ITEM_NOLOCATE gate aborts the entire spell.
func (w *World) LocateObjectResults(name string, viewer any, limit int, unseenOwnerText string) ([]string, bool) {
	observer, _ := viewer.(Actor)
	results := make([]string, 0)
	if limit <= 0 {
		return results, false
	}
	for _, candidate := range w.FindObjectByName(name) {
		obj, ok := candidate.(*ObjectInstance)
		if !ok || obj == nil {
			continue
		}
		if obj.HasExtraFlag(0, 25) { // ITEM_NOLOCATE, src/structs.h:493.
			return nil, true
		}
		results = append(results, w.locateObjectDescription(obj, observer, unseenOwnerText))
		limit--
		if limit == 0 {
			break
		}
	}
	return results, false
}

func (w *World) locateObjectDescription(obj *ObjectInstance, observer Actor, unseenOwnerText string) string {
	description := obj.GetShortDesc()
	var text string
	switch loc := obj.Location; loc.Kind {
	case ObjInRoom:
		if room := w.GetRoomInWorld(loc.RoomVNum); room != nil {
			text = fmt.Sprintf("%s is in %s.\n\r", description, room.Name)
		}
	case ObjInInventory, ObjEquipped:
		owner := w.locateObjectOwner(loc)
		if owner != nil {
			if observer == nil || !canSeeForPers(observer, owner) {
				// C leaves the global buf untouched in the failed CAN_SEE arm;
				// say_spell's syllable buffer remains in place (spells.c:377-391).
				return capLocateObjectText(unseenOwnerText)
			}
			verb := "being carried by"
			if loc.Kind == ObjEquipped {
				verb = "being worn by"
			}
			text = fmt.Sprintf("%s is %s %s.\n\r", description, verb, persName(owner, observer))
		}
	case ObjInContainer:
		if container := w.locateObjectByID(loc.ContainerObjID); container != nil {
			text = fmt.Sprintf("%s is in %s.\n\r", description, container.GetShortDesc())
		}
	}
	if text == "" {
		text = fmt.Sprintf("%s's location is uncertain.\n\r", description)
	}
	return capLocateObjectText(text)
}

func capLocateObjectText(text string) string {
	if len(text) > 0 && text[0] >= 'a' && text[0] <= 'z' {
		return string(text[0]-('a'-'A')) + text[1:]
	}
	return text
}

func (w *World) locateObjectOwner(loc ObjectLocation) Actor {
	switch loc.OwnerKind {
	case OwnerPlayer:
		if owner, ok := w.GetPlayer(loc.PlayerName); ok {
			return owner
		}
	case OwnerMob:
		if owner, ok := w.GetMobByID(loc.MobID); ok {
			return owner
		}
	}
	return nil
}

func (w *World) locateObjectByID(id int) *ObjectInstance {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.objectInstances[id]
}

// CFName is the single Go port of C's fname() (src/handler.c:68-79). C copies
// the leading run of alphabetic bytes into a static holder[30] and stops at the
// first non-letter, so "black-banded" -> "black", "D'haran" -> "D", "chai-dao"
// -> "chai", and a leading non-letter (or empty input) -> "". Every caller of
// fname shares this one implementation: OBJN/objName (src/utils.h:557-558),
// act's $F (src/comm.c:2454), door names in movement/open-close/look/bash
// (src/act.movement.c:329,522, src/act.offensive.c:847, src/act.informative.c:867,878),
// the look-in container header (src/act.informative.c:942), shop refusals
// (src/shop.c:521,528), and spell_locate_object (src/spells.c:361).
//
// holder[30]: C terminates at holder[30], one past the array, so a first
// keyword of 30 or more letters writes out of bounds. No object or door keyword
// in lib/world has a first letter-run that long (verified by a lib/world scan),
// so the overflow is not emulated. Companion: cIsName.
func CFName(name string) string {
	end := 0
	for end < len(name) && isASCIIAlpha(name[end]) {
		end++
	}
	return name[:end]
}

// cIsName is the single Go port of C's isname() (src/handler.c:81-111). Every
// space-or-non-alpha-separated letter-run in nameList is compared to the whole
// name case-insensitively: isname matches a WHOLE keyword word, never a prefix,
// so cIsName("d", "door") is false and cIsName("door", "door") is true. This is
// the matcher for find_door, do_steal's worn-item scan, janitor, turn_undead,
// mirror, horn and moon_gate. The abbreviation matcher for generic_find /
// get_obj_in_list_vis sites is isnameWithAbbrevs (C isname_with_abbrevs).
func cIsName(name, nameList string) bool {
	if name == "" {
		return false
	}
	for offset := 0; offset < len(nameList); {
		for offset < len(nameList) && !isASCIIAlpha(nameList[offset]) {
			offset++
		}
		start := offset
		for offset < len(nameList) && isASCIIAlpha(nameList[offset]) {
			offset++
		}
		if offset > start && strings.EqualFold(name, nameList[start:offset]) {
			return true
		}
	}
	return false
}

// GetMobPrototype returns a mob prototype by VNum.
