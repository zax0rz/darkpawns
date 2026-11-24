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
		// Look up object prototype by vnum (parse-time VNum index; same
		// first-match pointer the per-item scan returned)
		var proto *parser.Obj
		if parsed := w.GetParsedWorld(); parsed != nil {
			proto, _ = parsed.ObjByVnum(item.VNum)
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
