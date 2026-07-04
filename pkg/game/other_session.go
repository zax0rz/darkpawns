package game

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
