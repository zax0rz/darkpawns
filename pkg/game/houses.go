package game

// houses.go — Go port of src/house.c
// Player-house system: creation, ownership, guest lists, keys, rent storage.

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// House constants — from src/house.h
const (
	MaxHouses = 100
	MaxGuests = 50
	HousePriv = 0 // HOUSE_PRIVATE

	// Room flag names used by the house system
	RoomFlagHouse  = "house"       // ROOM_HOUSE
	RoomFlagAtrium = "atrium"      // ROOM_ATRIUM
	RoomFlagCrash  = "house_crash" // ROOM_HOUSE_CRASH — set when contents changed
	RoomFlagPriv   = "private"     // ROOM_PRIVATE — set on house interior

	// Admin level for house commands
	LVL_GRGOD = 51 // Greater God level — full house access
)

// HouseControl mirrors the C struct house_control_rec.
type HouseControl struct {
	VNum        int       `json:"vnum"`          // vnum of this house
	Atrium      int       `json:"atrium"`        // vnum of atrium
	ExitNum     int       `json:"exit_num"`      // direction of house's exit
	BuiltOn     int64     `json:"built_on"`      // unix timestamp when built
	Mode        int       `json:"mode"`          // ownership mode (HOUSE_PRIVATE)
	Owner       int64     `json:"owner"`         // idnum of owner (long → int64)
	NumOfGuests int       `json:"num_of_guests"` // how many guests
	Guests      [50]int64 `json:"guests"`        // idnums of guests
	LastPayment int64     `json:"last_payment"`  // unix timestamp of last payment
	Key         int       `json:"key"`           // vnum of key object
	Spare1      int64     `json:"spare1"`
	Spare2      int64     `json:"spare2"`
	Spare3      int64     `json:"spare3"`
	Spare4      int64     `json:"spare4"`
	Spare5      int64     `json:"spare5"`
	Spare6      int64     `json:"spare6"`
	Spare7      int64     `json:"spare7"`
}

// houseControlFilename is the path to the house control JSON file.
var houseControlFilename = "house/house_control.json"

// ---------------------------------------------------------------------------
// Player ID ↔ Name lookup (stubs — replace with DB/player-name-cache as needed)
// ---------------------------------------------------------------------------

// getPlayerNameByID looks up a player's name by their numeric ID.
// In the C code this was get_name_by_id(id).
// Implemented as a dependency-injected function pointer. Register via
// RegisterHousePlayerLookup() during server initialization, backing the
// lookup with a player database or in-memory player name cache indexed by ID.
var getPlayerNameByID func(id int64) string

// getPlayerIDByName looks up a player's numeric ID by their name.
// In the C code this was get_id_by_name(name).
// Implemented as a dependency-injected function pointer. Register via
// RegisterHousePlayerLookup() during server initialization, backing the
// lookup with a player database or in-memory player ID cache indexed by name.
var getPlayerIDByName func(name string) int64

// RegisterHousePlayerLookup sets the player-name lookup functions used by
// the house system. Call during server initialization.
func RegisterHousePlayerLookup(nameByID func(int64) string, idByName func(string) int64) {
	getPlayerNameByID = nameByID
	getPlayerIDByName = idByName
}

// ---------------------------------------------------------------------------
// File helpers
// ---------------------------------------------------------------------------

// HouseGetFilename returns the file path for a house's contents given its vnum.
// In C: House_get_filename() — writes "house/%d.house" to a buffer.
func HouseGetFilename(vnum int) string {
	if vnum < 0 {
		return ""
	}
	return filepath.Join("house", fmt.Sprintf("%d.house", vnum))
}

// ---------------------------------------------------------------------------
// Obj_from_store / Obj_to_store stubs
// ---------------------------------------------------------------------------

// houseSaveItem represents a single object in a house save file.
// Uses JSON instead of C's binary obj_file_elem for readability and simplicity.
type houseSaveItem struct {
	VNum        int                    `json:"vnum"`
	ContainerID int                    `json:"container_id,omitempty"`
	State       map[string]interface{} `json:"state,omitempty"`
}

// houseSaveData is the top-level structure for a house save file.
type houseSaveData struct {
	RoomVNum int             `json:"room_vnum"`
	Items    []houseSaveItem `json:"items"`
}

// ObjFromStore reconstructs a game object from a houseSaveItem.
// Ported from C Obj_from_store() — creates ObjectInstance from prototype.
// The data parameter must be a *houseSaveItem.
// Returns nil if the prototype can't be found.
func ObjFromStore(data *houseSaveItem, getProto func(vnum int) (*parser.Obj, bool)) *ObjectInstance {
	if data == nil {
		return nil
	}
	proto, ok := getProto(data.VNum)
	if !ok {
		return nil
	}
	obj := NewObjectInstance(proto, -1)
	if data.State != nil {
		obj.CustomData = make(map[string]interface{}, len(data.State))
		for k, v := range data.State {
			obj.CustomData[k] = v
		}
		obj.MigrateCustomData()
	}
	return obj
}

// ObjToStore converts an ObjectInstance to a houseSaveItem.
// Ported from C Obj_to_store() — serializes object for house save.
// Returns the save item, or nil if the object is invalid.
func ObjToStore(obj *ObjectInstance) *houseSaveItem {
	if obj == nil || obj.Prototype == nil {
		return nil
	}
	item := &houseSaveItem{
		VNum:        obj.Prototype.VNum,
		ContainerID: -1,
	}
	if obj.Location.Kind == ObjInContainer {
		item.ContainerID = obj.Location.ContainerObjID
	}
	if state := obj.GetSaveState(); state != nil {
		item.State = state
	}
	return item
}

// ---------------------------------------------------------------------------
// Room flag helpers (wrappers around parser.Room flag string slices)
// ---------------------------------------------------------------------------

// setRoomFlag adds a flag string to a room's flag list if not already present.
func setRoomFlag(room *parser.Room, flag string) {
	for _, f := range room.Flags {
		if f == flag {
			return
		}
	}
	room.Flags = append(room.Flags, flag)
}

// removeRoomFlag removes a flag string from a room's flag list.
func removeRoomFlag(room *parser.Room, flag string) {
	for i, f := range room.Flags {
		if f == flag {
			room.Flags = append(room.Flags[:i], room.Flags[i+1:]...)
			return
		}
	}
}

// roomHasFlagLocal checks if a room has a specific flag string.
// This is a standalone version (World.roomHasFlag is defined in limits.go).
func roomHasFlagLocal(room *parser.Room, flag string) bool {
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// toRoom returns the destination room vnum for a given exit direction from a room.
// In C: TOROOM(room, dir) — world[room].dir_option[dir]->to_room
func toRoom(room *parser.Room, dir int) int {
	if room == nil || dir < 0 || dir >= len(dirs) {
		return -1 // NOWHERE
	}
	dirName := dirs[dir]
	exit, ok := room.Exits[dirName]
	if !ok {
		return -1
	}
	return exit.ToRoom
}

// ---------------------------------------------------------------------------
// Text helpers (avoids deprecated strings.Title)
// ---------------------------------------------------------------------------

// toTitle capitalizes the first rune of s.
func toTitle(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// toLower lowercases all runes in s.
func toLower(s string) string {
	return strings.ToLower(s)
}

// ---------------------------------------------------------------------------
// House control slice management
// ---------------------------------------------------------------------------

// findHouse returns the index of the house control record for vnum, or -1.
// In C: find_house()
func findHouse(control []HouseControl, vnum int) int {
	for i, h := range control {
		if h.VNum == vnum {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// House_boot — load control records from JSON
// ---------------------------------------------------------------------------

// HouseBoot loads house control records from JSON, validates them, and sets
// room flags. Called during game initialization (boot_db equivalent).
// In C: House_boot() — loads house_control from binary, validates, sets flags.
var HcontrolFormat = "Usage: hcontrol build <house vnum> <exit direction> <player name>\r\n" +
	"       hcontrol destroy <house vnum>\r\n" +
	"       hcontrol pay <house vnum>\r\n" +
	"       hcontrol show\r\n" +
	"       hcontrol key <house vnum> <key vnum>\r\n"

// HcontrolListHouses lists all defined houses.
// In C: hcontrol_list_houses()
