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

	for i := range control {
		h := &control[i]
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
			builtOn = time.Unix(h.BuiltOn, 0).Local().Format("Mon Jan _2")
		}

		lastPay := "None"
		if h.LastPayment != 0 {
			lastPay = time.Unix(h.LastPayment, 0).Local().Format("Mon Jan _2")
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

	// C's atoi() returns zero for a non-numeric token. Keep that behavior at
	// this command boundary; inventing an "invalid vnum" branch changes the
	// player-facing path (house.c:398-404).
	virtHouse := cAtoi(args[0])
	message := ""
	var save bool

	w.mu.Lock()
	if len(w.HouseControl) >= MaxHouses {
		message = "Max houses already defined.\r\n"
	} else {
		// First arg: house vnum
		realHouse := w.rooms[virtHouse]
		if realHouse == nil {
			message = "No such room exists.\r\n"
		} else if findHouse(w.HouseControl, virtHouse) >= 0 {
			message = "House already exists.\r\n"
		} else {
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
				message = fmt.Sprintf("'%s' is not a valid direction.\r\n", dirName)
			} else {
				destVNum := toRoom(realHouse, exitNum)
				if destVNum < 0 {
					message = fmt.Sprintf("There is no exit %s from room %d.\r\n", dirName, virtHouse)
				} else {
					destRoom := w.rooms[destVNum]
					if destRoom == nil {
						message = "Destination room does not exist.\r\n"
					} else if toRoom(destRoom, revDir[exitNum]) != virtHouse {
						message = "A house's exit must be a two-way door.\r\n"
					} else if getPlayerIDByName == nil {
						message = "Player lookup not available.\r\n"
					} else {
						// The live lookup reads World.players and therefore takes the
						// same world read lock used by Player.SendMessage. Release
						// the writer lock around it; otherwise a valid build can
						// deadlock before it reaches the player-facing response.
						lookup := getPlayerIDByName
						w.mu.Unlock()
						owner := lookup(args[2])
						if owner < 0 {
							sendToChar(ch, fmt.Sprintf("Unknown player '%s'.\r\n", toLower(args[2])))
							return
						}
						w.mu.Lock()

						if owner < 0 {
							message = fmt.Sprintf("Unknown player '%s'.\r\n", toLower(args[2]))
						} else {
							now := time.Now().Unix()
							w.HouseControl = append(w.HouseControl, HouseControl{
								VNum:    virtHouse,
								Atrium:  destVNum,
								ExitNum: exitNum,
								BuiltOn: now,
								Owner:   owner,
								Key:     -1, // NOTHING
							})

							setRoomFlag(realHouse, RoomFlagHouse)
							setRoomFlag(realHouse, RoomFlagPriv)
							setRoomFlag(destRoom, RoomFlagAtrium)
							message = "House built.  Mazel tov!\r\n"
							save = true
						}
					}
				}
			}
		}
	}
	w.mu.Unlock()

	if message != "" {
		sendToChar(ch, message)
	}
	if save {
		w.saveHouseControl()
	}
}

// HcontrolDestroyHouse deletes a house.
// In C: hcontrol_destroy_house()
func (w *World) HcontrolDestroyHouse(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, HcontrolFormat)
		return
	}

	vnum := cAtoi(args[0])

	w.mu.Lock()
	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		w.mu.Unlock()
		sendToChar(ch, "Unknown house.\r\n")
		return
	}

	h := w.HouseControl[i]

	// Clear atrium flag
	realAtrium := w.rooms[h.Atrium]
	if realAtrium != nil {
		removeRoomFlag(realAtrium, RoomFlagAtrium)
	}

	// Clear house flags
	realHouse := w.rooms[h.VNum]
	if realHouse != nil {
		removeRoomFlag(realHouse, RoomFlagHouse)
		removeRoomFlag(realHouse, RoomFlagCrash)
	}

	// Delete house file
	houseDeleteFile(h.VNum)

	// Remove from slice
	w.HouseControl = append(w.HouseControl[:i], w.HouseControl[i+1:]...)

	// Re-set atrium flags on remaining houses that may share this atrium
	for j := range w.HouseControl {
		ra := w.rooms[w.HouseControl[j].Atrium]
		if ra != nil {
			setRoomFlag(ra, RoomFlagAtrium)
		}
	}
	w.mu.Unlock()

	sendToChar(ch, "House deleted.\r\n")
	w.saveHouseControl()
}

// HcontrolPayHouse records a payment for a house.
// In C: hcontrol_pay_house()
func (w *World) HcontrolPayHouse(ch *Player, arg string) {
	args := strings.Fields(arg)
	if len(args) < 1 || args[0] == "" {
		sendToChar(ch, HcontrolFormat)
		return
	}

	vnum := cAtoi(args[0])

	w.mu.Lock()
	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		w.mu.Unlock()
		sendToChar(ch, "Unknown house.\r\n")
		return
	}

	chName := ch.GetName()
	MudLog(fmt.Sprintf("Payment for house %d collected by %s.", vnum, chName), 0, LVL_IMMORT, true)

	w.HouseControl[i].LastPayment = time.Now().Unix()
	w.mu.Unlock()

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

	vnum := cAtoi(args[0])
	keyVNum := cAtoi(args[1])

	w.mu.Lock()
	i := findHouse(w.HouseControl, vnum)
	if i < 0 {
		w.mu.Unlock()
		sendToChar(ch, "That house doesn't exist!\r\n")
		return
	}

	// Validate key object exists
	if _, ok := w.objs[keyVNum]; !ok {
		w.mu.Unlock()
		sendToChar(ch, "That object doesn't exist!\r\n")
		return
	}

	w.HouseControl[i].Key = keyVNum
	w.mu.Unlock()

	w.saveHouseControl()
	sendToChar(ch, "House key set.\r\n")
}

// Hcontrol is the dispatcher for the hcontrol command.
// In C: ACMD(do_hcontrol)
func (w *World) Hcontrol(ch *Player, argument string) {
	// Defense-in-depth: this command mutates persistent house state, so
	// enforce the GRGOD gate here even if another caller reaches the dispatcher.
	if ch.GetLevel() < LVL_GRGOD {
		sendToChar(ch, "Huh?!?\r\n")
		return
	}

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

// cAtoi mirrors C's atoi for the house-control command. It returns zero when
// no leading integer is present and otherwise stops at the first non-digit,
// which is the branch-driving behavior used by house.c.
func cAtoi(value string) int {
	var number int
	if _, err := fmt.Sscanf(value, "%d", &number); err != nil {
		return 0
	}
	return number
}

// ---------------------------------------------------------------------------
// do_house — player-facing house command for guest management
// ---------------------------------------------------------------------------

// DoHouse handles the "house" command for guest management and ownership transfer.
// In C: ACMD(do_house)
