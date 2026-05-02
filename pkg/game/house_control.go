package game

import (
	"fmt"
	"strings"
	"time"
)

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

		fmt.Fprintf(&b, "%7d %7d  %-10s    %2d    %-12s %-10s %d\r\n",
			h.VNum, h.Atrium, builtOn, h.NumOfGuests,
			toTitle(toLower(ownerName)), lastPay, h.Key)

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
