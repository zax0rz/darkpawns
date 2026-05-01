package game

import (
	"fmt"
	"strings"
)

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
