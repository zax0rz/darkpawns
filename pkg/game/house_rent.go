package game

import (
	"fmt"
	"os"
	"encoding/json"
)

func (w *World) HouseListrent(ch *Player, vnum int) {
	fname := HouseGetFilename(vnum)
	if fname == "" {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}

	// Check if file exists
// #nosec G304
	data, err := os.ReadFile(fname)
	if err != nil {
		if os.IsNotExist(err) {
			sendToChar(ch, fmt.Sprintf("No objects on file for house #%d.\r\n", vnum))
		} else {
			sendToChar(ch, "Error reading house file.\r\n")
		}
		return
	}

	if len(data) == 0 {
		sendToChar(ch, fmt.Sprintf("No objects on file for house #%d.\r\n", vnum))
		return
	}

	// When ObjFromStore is wired, loop over binary records in data,
	// call ObjFromStore for each, and format a table of vnum/weight/name.
	// The C equivalent:
	//   while (!feof(fl)) {
	//     fread(&object, sizeof(struct obj_file_elem), 1, fl);
	//     if ((obj = Obj_from_store(object)) != NULL) {
	//       sprintf(buf, "%s [%5d] (%.2fau) %s\r\n",
	//           buf, GET_OBJ_VNUM(obj), GET_OBJ_COST(obj),
	//           obj->short_description);
	//       free_obj(obj);
	//     }
	//   }

	sendToChar(ch, fmt.Sprintf("Objects stored for house #%d:\r\n", vnum))
	var saveData houseSaveData
	if err := json.Unmarshal(data, &saveData); err != nil {
		sendToChar(ch, "Error reading house file.\r\n")
		return
	}
	for _, item := range saveData.Items {
		var name string
		for i := range w.GetParsedWorld().Objs {
			if w.GetParsedWorld().Objs[i].VNum == item.VNum {
				name = w.GetParsedWorld().Objs[i].ShortDesc
				break
			}
		}
		if name == "" {
			name = "unknown item"
		}
		sendToChar(ch, fmt.Sprintf("  [%5d] %s\r\n", item.VNum, name))
	}
}
// House_can_enter — check if a player may enter a house
// ---------------------------------------------------------------------------

// HouseCanEnter checks if a character can enter the given house.
// In C: House_can_enter() — GRGOD+ always allowed; private houses check owner + guests.
