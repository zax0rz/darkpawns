package game

// houses.go — Go port of src/house.c
// Player-house system: creation, ownership, guest lists, keys, rent storage.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
// TODO: Implement proper lookup via player database or in-memory cache.
var getPlayerNameByID func(id int64) string

// getPlayerIDByName looks up a player's numeric ID by their name.
// In the C code this was get_id_by_name(name).
// TODO: Implement proper lookup via player database or in-memory cache.
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

// ObjFromStore reconstructs an object from serialized data.
// Stub — returns nil. Replace with actual object deserialization when the
// object persistence system is implemented (C: Obj_from_store()).
func ObjFromStore(data interface{}) interface{} {
	return nil
}

// ObjToStore serializes an object to the given file.
// Stub — returns false. Replace with actual serialization (C: Obj_to_store()).
func ObjToStore(obj interface{}, fp *os.File) bool {
	return false
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
func (w *World) saveHouseControl() {
	data, err := json.MarshalIndent(w.HouseControl, "", "  ")
	if err != nil {
		BasicMudLog(fmt.Sprintf("Error marshaling house control: %v", err))
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(houseControlFilename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

	if err := os.WriteFile(houseControlFilename, data, 0644); err != nil {
		BasicMudLog(fmt.Sprintf("Error writing house control file: %v", err))
	}
}

// ---------------------------------------------------------------------------
// House object load/save (stubs — full implementation needs object persistence)
// ---------------------------------------------------------------------------

// houseLoad loads objects for a house from its save file into the room.
// In C: House_load() — reads .house file, calls Obj_from_store + obj_to_room.
func (w *World) houseLoad(vnum int) bool {
	// TODO: Implement full object loading when ObjFromStore is wired.
	// For now this is a no-op. The C version:
	//   - reads binary obj_file_elem structs from "house/<vnum>.house"
	//   - calls Obj_from_store() for each
	//   - skips unrentable objects (extract_obj)
	//   - adds rest to room via obj_to_room()
	return true
}

// houseCrashsave saves a house's objects to its save file.
// In C: House_crashsave() — opens file, calls House_save, clears crash flag.
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		BasicMudLog(fmt.Sprintf("Error creating house directory: %v", err))
		return
	}

	// Clear the crash flag
	removeRoomFlag(realHouse, RoomFlagCrash)
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
func (w *World) HouseListrent(ch *Player, vnum int) {
	fname := HouseGetFilename(vnum)
	if fname == "" {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}

	// Check if file exists
	data, err := os.ReadFile(fname)
	if err != nil {
		if os.IsNotExist(err) {
			sendToChar(ch, fmt.Sprintf("No objects on file for house #%d.\r\n", vnum))
		} else {
			sendToChar(ch, "Error reading house file.\r\n")
		}
		return
	}

	// Stub: minimal output
	if len(data) == 0 {
		sendToChar(ch, fmt.Sprintf("No objects on file for house #%d.\r\n", vnum))
		return
	}

	sendToChar(ch, fmt.Sprintf("Objects stored for house #%d (stub - full listing requires ObjFromStore):\r\n", vnum))
	sendToChar(ch, fmt.Sprintf("  File size: %d bytes\r\n", len(data)))
}

// ---------------------------------------------------------------------------
// House_can_enter — check if a player may enter a house
// ---------------------------------------------------------------------------

// HouseCanEnter checks if a character can enter the given house.
// In C: House_can_enter() — GRGOD+ always allowed; private houses check owner + guests.
func HouseCanEnter(ch *Player, houseVNum int, control []HouseControl) bool {
	// GRGOD+ always allowed
	if ch.GetLevel() >= LVL_GRGOD {
		return true
	}

	i := findHouse(control, houseVNum)
	if i < 0 {
		return true // house doesn't exist — allow
	}

	h := control[i]
	switch h.Mode {
	case HousePriv:
		if int64(ch.GetID()) == h.Owner {
			return true
		}
		for j := 0; j < h.NumOfGuests; j++ {
			if int64(ch.GetID()) == h.Guests[j] {
				return true
			}
		}
		return false
	}

	return true
}

// ---------------------------------------------------------------------------
// House_save_all — crash-save all houses flagged for save
// ---------------------------------------------------------------------------

// HouseSaveAll crash-saves all houses that have the crash flag set.
// In C: House_save_all() — iterates houses, checks ROOM_HOUSE_CRASH flag.
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
var HcontrolFormat = "Usage: hcontrol build <house vnum> <exit direction> <player name>\r\n" +
	"       hcontrol destroy <house vnum>\r\n" +
	"       hcontrol pay <house vnum>\r\n" +
	"       hcontrol show\r\n" +
	"       hcontrol key <house vnum> <key vnum>\r\n"

// HcontrolListHouses lists all defined houses.
// In C: hcontrol_list_houses()
func (w *World) HcontrolListHouses(ch *Player) {
	w.mu.RLock()
	control := w.HouseControl
	w.mu.RUnlock()

	if len(control) == 0 {
		sendToChar(ch, "No houses have been defined.\r\n")
		return
	}

	var b strings.Builder
	b.WriteString("Address  Atrium  Build Date  Guests  Owner        Last Paymt Key\r\n")
	b.WriteString("-------  ------  ----------  ------  ------------ ---------- ---\r\n")

	for _, h := range control {
		// Skip houses whose owner no longer exists
		ownerName := ""
		if getPlayerNameByID != nil {
			ownerName = getPlayerNameByID(h.Owner)
		}
		if ownerName == "" {
			continue
		}

		builtOn := "Unknown"
		if h.BuiltOn != 0 {
			builtOn = time.Unix(h.BuiltOn, 0).Format("Jan 2 2006")
		}

		lastPay := "None"
		if h.LastPayment != 0 {
			lastPay = time.Unix(h.LastPayment, 0).Format("Jan 2 2006")
		}

		b.WriteString(fmt.Sprintf("%7d %7d  %-10s    %2d    %-12s %-10s %d\r\n",
			h.VNum, h.Atrium, builtOn, h.NumOfGuests,
			toTitle(toLower(ownerName)), lastPay, h.Key))

		if h.NumOfGuests > 0 {
			b.WriteString("     Guests: ")
			for j := 0; j < h.NumOfGuests; j++ {
				gName := ""
				if getPlayerNameByID != nil {
					gName = getPlayerNameByID(h.Guests[j])
				}
				if gName == "" {
					gName = "<UNDEF>"
				}
				b.WriteString(toTitle(toLower(gName)))
				b.WriteString(" ")
			}
			b.WriteString("\r\n")
		}
	}

	sendToChar(ch, b.String())
}

// HcontrolBuildHouse creates a new house.
// In C: hcontrol_build_house()
func (w *World) HcontrolBuildHouse(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 3 {
		sendToChar(ch, HcontrolFormat)
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.HouseControl) >= MaxHouses {
		sendToChar(ch, "Max houses already defined.\r\n")
		return
	}

	// First arg: house vnum
	virtHouse, err := parseInt(args[0])
	if err != nil {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}
	realHouse := w.GetRoomInWorld(virtHouse)
	if realHouse == nil {
		sendToChar(ch, "No such room exists.\r\n")
		return
	}
	if findHouse(w.HouseControl, virtHouse) >= 0 {
		sendToChar(ch, "House already exists.\r\n")
		return
	}

	// Second arg: exit direction
	dirName := strings.ToLower(args[1])
	exitNum := -1
	for i, d := range dirs {
		if d == dirName || strings.HasPrefix(d, dirName) {
			exitNum = i
			break
		}
	}
	if exitNum < 0 {
		sendToChar(ch, fmt.Sprintf("'%s' is not a valid direction.\r\n", dirName))
		return
	}

	destVNum := toRoom(realHouse, exitNum)
	if destVNum < 0 {
		sendToChar(ch, fmt.Sprintf("There is no exit %s from room %d.\r\n", dirName, virtHouse))
		return
	}

	destRoom := w.GetRoomInWorld(destVNum)
	if destRoom == nil {
		sendToChar(ch, "Destination room does not exist.\r\n")
		return
	}

	// Check that the return path exists (two-way door)
	revDest := toRoom(destRoom, revDir[exitNum])
	if revDest != virtHouse {
		sendToChar(ch, "A house's exit must be a two-way door.\r\n")
		return
	}

	// Third arg: player name
	if getPlayerIDByName == nil {
		sendToChar(ch, "Player lookup not available.\r\n")
		return
	}
	owner := getPlayerIDByName(args[2])
	if owner < 0 {
		sendToChar(ch, fmt.Sprintf("Unknown player '%s'.\r\n", args[2]))
		return
	}

	now := time.Now().Unix()
	tempHouse := HouseControl{
		VNum:        virtHouse,
		Atrium:      destVNum,
		ExitNum:     exitNum,
		BuiltOn:     now,
		LastPayment: 0,
		Owner:       owner,
		NumOfGuests: 0,
		Key:         -1, // NOTHING
	}

	w.HouseControl = append(w.HouseControl, tempHouse)

	setRoomFlag(realHouse, RoomFlagHouse)
	setRoomFlag(realHouse, RoomFlagPriv)
	setRoomFlag(destRoom, RoomFlagAtrium)

	sendToChar(ch, "House built.  Mazel tov!\r\n")
	w.saveHouseControl()
}

// HcontrolDestroyHouse deletes a house.
// In C: hcontrol_destroy_house()
func (w *World) HcontrolDestroyHouse(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, HcontrolFormat)
		return
	}

	vnum, err := parseInt(args[0])
	if err != nil {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		sendToChar(ch, "Unknown house.\r\n")
		return
	}

	h := w.HouseControl[i]

	// Clear atrium flag
	realAtrium := w.GetRoomInWorld(h.Atrium)
	if realAtrium != nil {
		removeRoomFlag(realAtrium, RoomFlagAtrium)
	}

	// Clear house flags
	realHouse := w.GetRoomInWorld(h.VNum)
	if realHouse != nil {
		removeRoomFlag(realHouse, RoomFlagHouse)
		removeRoomFlag(realHouse, RoomFlagCrash)
	}

	// Delete house file
	houseDeleteFile(h.VNum)

	// Remove from slice
	w.HouseControl = append(w.HouseControl[:i], w.HouseControl[i+1:]...)

	sendToChar(ch, "House deleted.\r\n")
	w.saveHouseControl()

	// Re-set atrium flags on remaining houses that may share this atrium
	for j := range w.HouseControl {
		ra := w.GetRoomInWorld(w.HouseControl[j].Atrium)
		if ra != nil {
			setRoomFlag(ra, RoomFlagAtrium)
		}
	}
}

// HcontrolPayHouse records a payment for a house.
// In C: hcontrol_pay_house()
func (w *World) HcontrolPayHouse(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, HcontrolFormat)
		return
	}

	vnum, err := parseInt(args[0])
	if err != nil {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		sendToChar(ch, "Unknown house.\r\n")
		return
	}

	chName := ch.GetName()
	MudLog(fmt.Sprintf("Payment for house %d collected by %s.", vnum, chName), 0, LVL_IMMORT, true)

	w.HouseControl[i].LastPayment = time.Now().Unix()
	w.saveHouseControl()
	sendToChar(ch, "Payment recorded.\r\n")
}

// HcontrolSetKey sets the key vnum for a house.
// In C: hcontrol_set_key()
func (w *World) HcontrolSetKey(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 2 {
		sendToChar(ch, HcontrolFormat)
		return
	}

	vnum, err := parseInt(args[0])
	if err != nil {
		sendToChar(ch, "Invalid house vnum.\r\n")
		return
	}

	keyVNum, err := parseInt(args[1])
	if err != nil {
		sendToChar(ch, "Invalid key vnum.\r\n")
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		sendToChar(ch, "That house doesn't exist!\r\n")
		return
	}

	// Validate key object exists
	if _, ok := w.GetObjPrototype(keyVNum); !ok {
		sendToChar(ch, "That object doesn't exist!\r\n")
		return
	}

	w.HouseControl[i].Key = keyVNum
	w.saveHouseControl()
	sendToChar(ch, "House key set.\r\n")
}

// Hcontrol is the dispatcher for the hcontrol command.
// In C: ACMD(do_hcontrol)
func (w *World) Hcontrol(ch *Player, argument string) {
	args := strings.Fields(argument)
	if len(args) < 1 {
		sendToChar(ch, HcontrolFormat)
		return
	}

	subCmd := strings.ToLower(args[0])
	rest := ""
	if len(args) > 1 {
		rest = strings.Join(args[1:], " ")
	}

	switch {
	case isAbbrev(subCmd, "build"):
		w.HcontrolBuildHouse(ch, rest)
	case isAbbrev(subCmd, "destroy"):
		w.HcontrolDestroyHouse(ch, rest)
	case isAbbrev(subCmd, "pay"):
		w.HcontrolPayHouse(ch, rest)
	case isAbbrev(subCmd, "show"):
		w.HcontrolListHouses(ch)
	case isAbbrev(subCmd, "key"):
		w.HcontrolSetKey(ch, rest)
	default:
		sendToChar(ch, HcontrolFormat)
	}
}

// ---------------------------------------------------------------------------
// do_house — player-facing house command for guest management
// ---------------------------------------------------------------------------

// DoHouse handles the "house" command for guest management and ownership transfer.
// In C: ACMD(do_house)
func (w *World) DoHouse(ch *Player, argument string) {
	args := strings.Fields(argument)

	// Check we're in a house room
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil || !roomHasFlagLocal(room, RoomFlagHouse) {
		sendToChar(ch, "You must be in your house to set guests.\r\n")
		return
	}

	i := findHouse(w.HouseControl, room.VNum)
	if i < 0 {
		sendToChar(ch, "Um.. this house seems to be screwed up.\r\n")
		return
	}

	w.mu.RLock()
	h := w.HouseControl[i]
	w.mu.RUnlock()

	// Only owner (or immortals) can set guests
	if int64(ch.GetID()) != h.Owner && ch.GetLevel() < LVL_IMMORT {
		sendToChar(ch, "Only the primary owner can set guests.\r\n")
		return
	}

	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, "house guest <name>\r\nhouse transfer <name>\r\n")
		return
	}

	subCmd := strings.ToLower(args[0])

	switch {
	case isAbbrev(subCmd, "guest"):
		w.doHouseGuest(ch, i, args[1:])
	case isAbbrev(subCmd, "transfer"):
		w.doHouseTransfer(ch, i, args[1:])
	default:
		sendToChar(ch, "house guest <name>\r\nhouse transfer <name>\r\n")
	}
}

// doHouseGuest handles "house guest" subcommand: list, add, or remove guests.
func (w *World) doHouseGuest(ch *Player, houseIdx int, args []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	h := &w.HouseControl[houseIdx]

	// If no args, list guests
	if len(args) < 1 || args[0] == "" {
		// Clean up guests whose players no longer exist
		cleaned := false
		j := 0
		for j < h.NumOfGuests {
			gName := ""
			if getPlayerNameByID != nil {
				gName = getPlayerNameByID(h.Guests[j])
			}
			if gName == "" {
				// Shift array left
				for k := j; k < h.NumOfGuests-1; k++ {
					h.Guests[k] = h.Guests[k+1]
				}
				h.NumOfGuests--
				cleaned = true
			} else {
				j++
			}
		}
		if cleaned {
			w.saveHouseControl()
		}

		sendToChar(ch, "Guests of your house:\r\n")
		if h.NumOfGuests == 0 {
			sendToChar(ch, "  None.\r\n")
			return
		}
		for j := 0; j < h.NumOfGuests; j++ {
			gName := ""
			if getPlayerNameByID != nil {
				gName = getPlayerNameByID(h.Guests[j])
			}
			if gName != "" {
				sendToChar(ch, toTitle(toLower(gName))+"\r\n")
			}
		}
		return
	}

	// Have a name — add or remove guest
	if getPlayerIDByName == nil {
		sendToChar(ch, "Player lookup not available.\r\n")
		return
	}

	id := getPlayerIDByName(args[0])
	if id < 0 {
		sendToChar(ch, "No such player.\r\n")
		return
	}

	// Check if already a guest (toggle: if present, remove)
	for j := 0; j < h.NumOfGuests; j++ {
		if h.Guests[j] == id {
			// Remove guest
			for k := j; k < h.NumOfGuests-1; k++ {
				h.Guests[k] = h.Guests[k+1]
			}
			h.NumOfGuests--
			w.saveHouseControl()
			sendToChar(ch, "Guest deleted.\r\n")
			return
		}
	}

	// Add guest
	if h.NumOfGuests >= MaxGuests {
		sendToChar(ch, "You've already reached the maximum number of guests in your house!\r\n")
		return
	}
	h.Guests[h.NumOfGuests] = id
	h.NumOfGuests++
	w.saveHouseControl()
	sendToChar(ch, "Guest added.\r\n")
}

// doHouseTransfer handles "house transfer" subcommand: change ownership.
func (w *World) doHouseTransfer(ch *Player, houseIdx int, args []string) {
	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, "Transfer your house to whom?\r\n")
		return
	}

	if getPlayerIDByName == nil {
		sendToChar(ch, "Player lookup not available.\r\n")
		return
	}

	id := getPlayerIDByName(args[0])
	if id < 0 {
		sendToChar(ch, "No such player.\r\n")
		return
	}

	w.mu.Lock()
	w.HouseControl[houseIdx].Owner = id
	w.saveHouseControl()
	w.mu.Unlock()

	chName := ch.GetName()
	sendToChar(ch, "House transferred.\r\n")
	MudLog(fmt.Sprintf("%s transferred house to %s.", chName, toTitle(toLower(args[0]))),
		0, LVL_IMMORT, true)
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// parseInt parses an integer from a string.
func parseInt(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, err
	}
	return n, nil
}

// isAbbrev checks if arg is a case-insensitive abbreviation of name.
// In C: is_abbrev() — prefix match of length >= 1.
func isAbbrev(arg, name string) bool {
	if len(arg) == 0 || len(name) == 0 {
		return false
	}
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(arg))
}
