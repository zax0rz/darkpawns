package game

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"encoding/json"
)
import "github.com/zax0rz/darkpawns/pkg/parser"

func (w *World) saveHouseControl() {
	data, err := json.MarshalIndent(w.HouseControl, "", "  ")
	if err != nil {
		BasicMudLog(fmt.Sprintf("Error marshaling house control: %v", err))
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(houseControlFilename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

// #nosec G306
	if err := os.WriteFile(houseControlFilename, data, 0600); err != nil {
		BasicMudLog(fmt.Sprintf("Error writing house control file: %v", err))
	}
}

// ---------------------------------------------------------------------------
// House object load/save (stubs — full implementation needs object persistence)
// ---------------------------------------------------------------------------

// houseLoad loads objects for a house from its save file into the room.
// In C: House_load() — reads .house file, calls Obj_from_store + obj_to_room.
// When ObjFromStore is wired, this will deserialize each binary record and
// place surviving objects in the room. Currently a no-op — the file is
// opened to verify it exists, and any future persistence wire-up will fill
// in the read loop via ObjFromStore.
func (w *World) houseLoad(vnum int) bool {
	realRoom := w.GetRoomInWorld(vnum)
	if realRoom == nil {
		return false
	}

	fname := HouseGetFilename(vnum)
	if fname == "" {
		return false
	}

// #nosec G304
	data, err := os.ReadFile(fname)
	if err != nil {
		// No file found — not necessarily an error
		return false
	}

	BasicMudLogf("House_load: reading %s for room %d", fname, vnum)

	var saveData houseSaveData
	if err := json.Unmarshal(data, &saveData); err != nil {
		slog.Error("houseLoad: failed to parse save file", "file", fname, "error", err)
		return false
	}

	// Build container map for nesting. We load items in two passes:
	// first pass creates all objects, second pass places them.
	objMap := make(map[int]*ObjectInstance) // keyed by slice index for container resolution
	for i := range saveData.Items {
		item := &saveData.Items[i]
		// Look up object prototype by vnum
		var proto *parser.Obj
		for i := range w.GetParsedWorld().Objs {
			if w.GetParsedWorld().Objs[i].VNum == item.VNum {
				proto = &w.GetParsedWorld().Objs[i]
				break
			}
		}
		obj := ObjFromStore(item, func(vnum int) (*parser.Obj, bool) {
			if proto != nil && proto.VNum == vnum {
				return proto, true
			}
			return nil, false
		})
		if obj == nil {
			slog.Warn("houseLoad: missing prototype", "vnum", item.VNum)
			continue
		}
		if IsUnrentable(obj) {
			continue
		}
		objMap[i] = obj
	}

	// Place objects in room. Items with container_id >= 0 go into containers.
	for i, obj := range objMap {
		item := &saveData.Items[i]
		if item.ContainerID >= 0 {
			// Find container by iterating — container was saved first (recursive)
			for _, candidate := range objMap {
				if candidate.ID == item.ContainerID {
				candidate.Contains = append(candidate.Contains, obj)
					break
				}
			}
			continue
		}
		w.AddItemToRoom(obj, vnum)
	}

	return true
}

// houseCrashsave saves a house's objects to its save file.
// In C: House_crashsave() — opens file, calls House_save (recursive),
// clears ROOM_HOUSE_CRASH flag, then restores container weights.
// When ObjToStore is wired, this writes every object in the room via
// HouseSaveObjects, then restores the container-weight adjustments the
// save process made.
func (w *World) houseCrashsave(vnum int) {
	realHouse := w.GetRoomInWorld(vnum)
	if realHouse == nil {
		return
	}

	fname := HouseGetFilename(vnum)
	if fname == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(fname)
	if err := os.MkdirAll(dir, 0750); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

// #nosec G304
	fp, err := os.Create(fname)
	if err != nil {
		BasicMudLog(fmt.Sprintf("SYSERR: Error saving house file #%d: %v", vnum, err))
		return
	}
	defer fp.Close()

	// Collect all objects in the room and serialize to JSON
	objects := w.GetItemsInRoom(vnum)
	var items []houseSaveItem
	for _, obj := range objects {
		w.collectHouseItems(obj, &items)
	}

	// Write JSON
	data := houseSaveData{RoomVNum: vnum, Items: items}
	enc := json.NewEncoder(fp)
	if err := enc.Encode(data); err != nil {
		BasicMudLog(fmt.Sprintf("SYSERR: Error encoding house #%d: %v", vnum, err))
		return
	}

	// Restore container weights that were adjusted during save (C: House_restore_weight)
	for _, obj := range objects {
		w.HouseRestoreWeight(obj)
	}

	// Clear the crash flag
	removeRoomFlag(realHouse, RoomFlagCrash)
}

// collectHouseItems recursively collects objects into the items slice.
// Replaces C's House_save. Adjusts container weights for storage.
func (w *World) collectHouseItems(obj *ObjectInstance, items *[]houseSaveItem) {
	if obj == nil {
		return
	}
	// Recurse into contents first
	for _, contained := range obj.Contains {
		w.collectHouseItems(contained, items)
	}
	// Add this object
	if item := ObjToStore(obj); item != nil {
		*items = append(*items, *item)
		// Decrement container weight for the saved child
		if obj.Location.Kind == ObjInContainer {
			if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok && obj.Prototype != nil {
				container.Prototype.Weight -= obj.Prototype.Weight
				if container.Prototype.Weight < 1 {
					container.Prototype.Weight = 1
				}
			}
		}
	}
}

// HouseRestoreWeight recursively restores container weights after a save
// operation adjusted them (C: House_restore_weight). Called once per room
// object after houseCrashsave finishes writing.
func (w *World) HouseRestoreWeight(obj *ObjectInstance) {
	if obj == nil {
		return
	}

	// Recurse into contents first
	for _, contained := range obj.Contains {
		w.HouseRestoreWeight(contained)
	}

	// Restore the weight adjustment
	if obj.Location.Kind == ObjInContainer {
		if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok && obj.Prototype != nil {
			container.Prototype.Weight += obj.Prototype.Weight
		}
	}
}

// houseDeleteFile removes a house's save file.
// In C: House_delete_file()
func houseDeleteFile(vnum int) {
	fname := HouseGetFilename(vnum)
	if fname == "" {
		return
	}
	if err := os.Remove(fname); err != nil && !os.IsNotExist(err) {
		BasicMudLog(fmt.Sprintf("Error deleting house file #%d: %v", vnum, err))
	}
}

// ---------------------------------------------------------------------------
// House_listrent — list objects stored in a house save file
// ---------------------------------------------------------------------------

// HouseListrent lists all objects in a house's save file.
// In C: House_listrent() — reads .house file, prints obj vnum/weight/name.
// When ObjFromStore is wired, each record will be deserialized and its
// name/vnum/value shown to the player. Currently reports the file exists
// and its size, since the object-persistence layer is not yet connected.
func (w *World) HouseSaveAll() {
	w.mu.RLock()
	control := w.HouseControl
	w.mu.RUnlock()

	for _, h := range control {
		realHouse := w.GetRoomInWorld(h.VNum)
		if realHouse == nil {
			continue
		}
		if roomHasFlagLocal(realHouse, RoomFlagCrash) {
			w.houseCrashsave(h.VNum)
		}
	}
}

// ---------------------------------------------------------------------------
// hcontrol command handlers (admin-only, LVL_IMPL / LVL_GRGOD level)
// ---------------------------------------------------------------------------

// HcontrolFormat is the usage string for hcontrol.
