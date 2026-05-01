package game

import (
	"fmt"
	"os"
	"encoding/json"
)

func (w *World) HouseBoot() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.HouseControl = nil

	data, err := os.ReadFile(houseControlFilename)
	if err != nil {
		if os.IsNotExist(err) {
			BasicMudLog("House control file does not exist.")
		} else {
			BasicMudLog(fmt.Sprintf("Error reading house control file: %v", err))
		}
		w.HouseControl = make([]HouseControl, 0)
		return
	}

	var houseControl []HouseControl
	if err := json.Unmarshal(data, &houseControl); err != nil {
		BasicMudLog(fmt.Sprintf("Error parsing house control file: %v", err))
		w.HouseControl = make([]HouseControl, 0)
		return
	}

	// Validate and filter records
	var valid []HouseControl
	for _, h := range houseControl {
		// Owner must exist
		if getPlayerNameByID != nil && getPlayerNameByID(h.Owner) == "" {
			continue
		}

		// House room must exist
		realHouse := w.GetRoomInWorld(h.VNum)
		if realHouse == nil {
			continue
		}

		// Not already a house
		if findHouse(valid, h.VNum) >= 0 {
			continue
		}

		// Atrium must exist
		realAtrium := w.GetRoomInWorld(h.Atrium)
		if realAtrium == nil {
			continue
		}

		// Exit number must be valid
		if h.ExitNum < 0 || h.ExitNum >= len(dirs) {
			continue
		}

		// TOROOM must match atrium
		if toRoom(realHouse, h.ExitNum) != h.Atrium {
			continue
		}

		valid = append(valid, h)

		// Set room flags
		setRoomFlag(realHouse, RoomFlagHouse)
		setRoomFlag(realAtrium, RoomFlagAtrium)

		// Load house contents
		w.houseLoad(h.VNum)
	}

	w.HouseControl = valid
	w.saveHouseControl()
}

// ---------------------------------------------------------------------------
// House control file I/O
// ---------------------------------------------------------------------------

// saveHouseControl writes the house control records to JSON.
// In C: House_save_control() — fwrite binary records.
