package game

import (
	"fmt"
	"strings"
)

func (w *World) DoHouse(ch *Player, argument string) {
	arg1, rest := oneArgument(argument)
	arg2, _ := oneArgument(rest)

	// Check we're in a house room
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil || !roomHasFlagLocal(room, RoomFlagHouse) {
		ch.SendMessage("You must be in your house to set guests.\r\n")
		return
	}

	i := findHouse(w.HouseControl, room.VNum)
	if i < 0 {
		ch.SendMessage("Um.. this house seems to be screwed up.\r\n")
		return
	}

	w.mu.RLock()
	h := w.HouseControl[i]
	w.mu.RUnlock()

	// Only owner (or immortals) can set guests
	if int64(ch.GetID()) != h.Owner && ch.GetLevel() < LVL_IMMORT {
		ch.SendMessage("Only the primary owner can set guests.\r\n")
		return
	}

	if arg1 == "" {
		ch.SendMessage("house guest <name>\r\nhouse transfer <name>\r\n")
		return
	}

	switch {
	case isAbbrev(arg1, "guest"):
		w.doHouseGuest(ch, i, arg2)
	case isAbbrev(arg1, "transfer"):
		w.doHouseTransfer(ch, i, arg2)
	default:
		// C has no fallback after the guest/transfer branches: an unknown
		// subcommand is intentionally silent (house.c:688-718).
	}
}

// doHouseGuest handles "house guest" subcommand: list, add, or remove guests.
func (w *World) doHouseGuest(ch *Player, houseIdx int, arg string) {
	if arg == "" {
		w.listHouseGuests(ch, houseIdx)
		return
	}

	// C performs the player lookup before it enters the guest-list mutation;
	// the callback consults World and must not run while World.mu is held.
	if getPlayerIDByName == nil {
		ch.SendMessage("Player lookup not available.\r\n")
		return
	}
	id := getPlayerIDByName(arg)
	if id < 0 {
		ch.SendMessage("No such player.\r\n")
		return
	}

	w.mu.Lock()
	h := &w.HouseControl[houseIdx]
	for j := 0; j < h.NumOfGuests; j++ {
		if h.Guests[j] == id {
			for k := j; k < h.NumOfGuests-1; k++ {
				h.Guests[k] = h.Guests[k+1]
			}
			h.NumOfGuests--
			w.mu.Unlock()
			w.saveHouseControl()
			ch.SendMessage("Guest deleted.\r\n")
			return
		}
	}
	if h.NumOfGuests >= MaxGuests {
		w.mu.Unlock()
		ch.SendMessage("You've already reached the maximum number of guests in your house!\r\n")
		return
	}
	h.Guests[h.NumOfGuests] = id
	h.NumOfGuests++
	w.mu.Unlock()
	w.saveHouseControl()
	ch.SendMessage("Guest added.\r\n")
}

// listHouseGuests snapshots the control record before consulting the player
// lookup and sending output. Both operations can reacquire World.mu.
func (w *World) listHouseGuests(ch *Player, houseIdx int) {
	w.mu.RLock()
	h := w.HouseControl[houseIdx]
	w.mu.RUnlock()

	cleaned := h
	cleaned.NumOfGuests = 0
	for j := 0; j < h.NumOfGuests; j++ {
		gName := ""
		if getPlayerNameByID != nil {
			gName = getPlayerNameByID(h.Guests[j])
		}
		if gName != "" {
			cleaned.Guests[cleaned.NumOfGuests] = h.Guests[j]
			cleaned.NumOfGuests++
		}
	}
	if cleaned.NumOfGuests != h.NumOfGuests {
		w.mu.Lock()
		w.HouseControl[houseIdx].NumOfGuests = cleaned.NumOfGuests
		w.HouseControl[houseIdx].Guests = cleaned.Guests
		w.mu.Unlock()
		w.saveHouseControl()
		h = cleaned
	}

	ch.SendMessage("Guests of your house:\r\n")
	if h.NumOfGuests == 0 {
		ch.SendMessage("  None.\r\n")
		return
	}
	for j := 0; j < h.NumOfGuests; j++ {
		gName := ""
		if getPlayerNameByID != nil {
			gName = getPlayerNameByID(h.Guests[j])
		}
		if gName != "" {
			ch.SendMessage(toTitle(toLower(gName)) + "\r\n")
		}
	}
}

// doHouseTransfer handles "house transfer" subcommand: change ownership.
func (w *World) doHouseTransfer(ch *Player, houseIdx int, arg string) {
	if arg == "" {
		ch.SendMessage("Transfer your house to whom?\r\n")
		return
	}

	if getPlayerIDByName == nil {
		ch.SendMessage("Player lookup not available.\r\n")
		return
	}

	id := getPlayerIDByName(arg)
	if id < 0 {
		ch.SendMessage("No such player.\r\n")
		return
	}

	w.mu.Lock()
	w.HouseControl[houseIdx].Owner = id
	w.mu.Unlock()

	chName := ch.GetName()
	ch.SendMessage("House transfered.\r\n")
	MudLog(fmt.Sprintf("%s transfered %s house to %s.", chName, hshr(ch), toTitle(toLower(arg))),
		0, LVL_IMMORT, true)
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// isAbbrev checks if arg is a case-insensitive abbreviation of name.
// In C: is_abbrev() — prefix match of length >= 1.
func isAbbrev(arg, name string) bool {
	if len(arg) == 0 || len(name) == 0 {
		return false
	}
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(arg))
}
