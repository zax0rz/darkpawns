package game

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func (w *World) saveHouseControl() {
	data, err := json.MarshalIndent(w.HouseControl, "", "  ")
	if err != nil {
		BasicMudLog(fmt.Sprintf("Error marshaling house control: %v", err))
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(houseControlFilename)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

	if err := os.WriteFile(houseControlFilename, data, 0o600); err != nil {
		BasicMudLog(fmt.Sprintf("Error writing house control file: %v", err))
	}
}

// ---------------------------------------------------------------------------
// House object load/save (stubs — full implementation needs object persistence)
// ---------------------------------------------------------------------------

// houseLoad loads objects for a house from its save file into the room.
// C src/house.c:87-101 reads in file order, extracts unrentable objects,
// and puts every survivor on the room floor. Saved container_index metadata
// is retained in the format but never used to re-nest loaded objects.
func (w *World) houseLoad(vnum int) bool {
	realRoom := w.GetRoomInWorld(vnum)
	if realRoom == nil {
		return false
	}

	fname := HouseGetFilename(vnum)
	if fname == "" {
		return false
	}

	data, err := os.ReadFile(filepath.Clean(fname))
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
		w.registerExistingObject(obj)
		if IsUnrentable(obj) {
			w.ExtractObject(obj, vnum)
			continue
		}
		// obj_to_room prepends each record, so the floor list reverses file order.
		w.AddItemToRoomFront(obj, vnum)
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

	fp, err := os.Create(filepath.Clean(fname))
	if err != nil {
		BasicMudLog(fmt.Sprintf("SYSERR: Error saving house file #%d: %v", vnum, err))
		return
	}
	defer func() { _ = fp.Close() }()

	// Collect all objects in the room, contents-first (C's House_save order,
	// house.c:112-120), and serialize with container references by index.
	objects := w.GetItemsInRoom(vnum)
	var flat []*ObjectInstance
	index := make(map[*ObjectInstance]int)
	containerOf := make(map[*ObjectInstance]int)
	for _, obj := range objects {
		w.collectHouseItems(obj, &flat, index, containerOf)
	}
	items := make([]houseSaveItem, len(flat))
	for i, obj := range flat {
		item := ObjToStore(obj)
		if item == nil {
			// Unreachable for a live object (its prototype is non-nil),
			// but one record per flat entry keeps every container_index
			// below aligned; vnum 0 matches no prototype and is skipped
			// on load.
			item = &houseSaveItem{VNum: 0}
		}
		if ci, ok := containerOf[obj]; ok {
			item.ContainerIndex = &ci
		}
		items[i] = *item
	}

	// Write JSON
	data := houseSaveData{RoomVNum: vnum, Items: items}
	enc := json.NewEncoder(fp)
	if err := enc.Encode(data); err != nil {
		BasicMudLog(fmt.Sprintf("SYSERR: Error encoding house #%d: %v", vnum, err))
		return
	}

	// Clear the crash flag
	removeRoomFlag(realHouse, RoomFlagCrash)
}

// collectHouseItems recursively flattens objects contents-first — C
// House_save recurses into contains before writing the object itself
// (house.c:112-120) — recording each object's position in the flat slice and
// the position of its container, so the save file can express nesting by
// index (DP-1401).
func (w *World) collectHouseItems(obj *ObjectInstance, flat *[]*ObjectInstance, index map[*ObjectInstance]int, containerOf map[*ObjectInstance]int) {
	if obj == nil {
		return
	}
	// Recurse into contents first
	for _, contained := range obj.Contains {
		w.collectHouseItems(contained, flat, index, containerOf)
	}
	// Add this object
	index[obj] = len(*flat)
	*flat = append(*flat, obj)
	for _, contained := range obj.Contains {
		containerOf[contained] = index[obj]
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

// ---------------------------------------------------------------------------
// hcontrol command handlers (admin-only, LVL_IMPL / LVL_GRGOD level)
// ---------------------------------------------------------------------------

// HcontrolFormat is the usage string for hcontrol.
