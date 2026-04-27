package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// do_quit
// ---------------------------------------------------------------------------

func (w *World) doQuit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		ch.SendMessage("No way, you're a monster!\r\n")
		return true
	}

	roomVNum := ch.GetRoomVNum()
	room := w.GetRoomInWorld(roomVNum)

	// Check valid quit rooms
	isValidRoom := false
	validRooms := []int{8004, 8008, 18201, 21202, 21258}
	for _, v := range validRooms {
		if roomVNum == v {
			isValidRoom = true
			break
		}
	}

	if !isValidRoom {
		// Check if player owns the room (has room key)
		if room == nil {
			ch.SendMessage("You can't quit here!\r\n")
			return true
		}
	}

	if ch.Position == combat.PosFighting {
		ch.SendMessage("No way!  You are fighting!\r\n")
		return true
	}

	// Kill duplicates
	w.RemovePlayer(ch.Name)

	// Save player
// #nosec G104
	SavePlayer(ch)

	// Extract — broadcast leave message
	msg := fmt.Sprintf("%s has left the game.\r\n", ch.Name)
	actToRoom(w, roomVNum, msg, ch.Name)

	ch.SendMessage("Good bye... Come again soon!\r\n")

	// Signal disconnect
	ch.Send <- []byte("CLOSE_CONNECTION")

	return true
}

// ---------------------------------------------------------------------------
// do_save
// ---------------------------------------------------------------------------

func (w *World) doSave(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if err := SavePlayer(ch); err != nil {
		ch.SendMessage("Could not save your data. Contact an admin!\r\n")
		return true
	}

	ch.SendMessage("Saving.\r\n")
	return true
}
