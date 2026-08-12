package game

import "fmt"

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

	// C do_save (act.other.c): "Saving %s.\r\n" with GET_NAME(ch).
	ch.SendMessage(fmt.Sprintf("Saving %s.\r\n", ch.Name))
	return true
}
