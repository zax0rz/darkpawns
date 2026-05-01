package game

import "fmt"

func (w *World) performTell(ch *Player, vict *Player, arg string) {
	msg := fmt.Sprintf("$n tells you, '%s'", arg)
	msg = deleteAnsiControls(msg)
	vict.SendMessage(fmt.Sprintf("\033[0;31m%s\033[0m\r\n", msg))

	// AFK notice.
	if vict.Flags&prfAfk != 0 {
		ch.SendMessage(fmt.Sprintf("%s is AFK right now, %s may not hear you.\r\n",
			vict.Name, hisHer(vict.Sex)))
	}

	// Echo to sender.
	if ch.Flags&prfNoRepeat == 0 {
		echo := fmt.Sprintf("You tell $N, '%s'", arg)
		echo = deleteAnsiControls(echo)
		ch.SendMessage(fmt.Sprintf("\033[0;31m%s\033[0m\r\n", echo))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	// Track for reply.
	w.setLastTeller(vict.ID, ch.ID)
}

// doTell — port of do_tell().
func (w *World) doTell(ch *Player, me *MobInstance, cmd string, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Who do you want to tell what??\r\n")
		return true
	}

	vict := w.getCharVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is playing.\r\n")
		return true
	}
	if vict.Name == ch.Name {
		sendToChar(ch, "You try to tell yourself something.\r\n")
		return true
	}
	if ch.Flags&prfNoTell != 0 && ch.GetLevel() < lvlImmort {
		sendToChar(ch, "You can't tell other people while you have notell on.\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot tell anyone anything!\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}
	if !vict.IsNPC() && vict.Flags&plrWriting != 0 {
		ch.SendMessage(fmt.Sprintf("%s's writing a message right now; try again later.\r\n", vict.Name))
		return true
	}

	victNotellOrSP := (vict.Flags&prfNoTell != 0 || w.roomHasFlag(vict.RoomVNum, "soundproof"))
	if victNotellOrSP && ch.GetLevel() < lvlImmort {
		ch.SendMessage(fmt.Sprintf("%s can't hear you.\r\n", vict.Name))
		return true
	}

	w.performTell(ch, vict, msg)
	return true
}

// doReply — port of do_reply().
func (w *World) doReply(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	lastID := w.getLastTeller(ch.ID)
	if lastID == noBody {
		sendToChar(ch, "You have no one to reply to!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "What is your reply?\r\n")
		return true
	}

	// Find the last teller by ID.
	var tch *Player
	for _, p := range w.allPlayers() {
		if p.ID == lastID {
			tch = p
			break
		}
	}
	if tch == nil {
		sendToChar(ch, "They are no longer playing.\r\n")
		return true
	}

	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot tell anyone anything!\r\n")
		return true
	}
	if !tch.IsNPC() && tch.Flags&plrWriting != 0 {
		sendToChar(ch, "They are writing now, try later.\r\n")
		return true
	}

	tchNotellOrSP := (tch.Flags&prfNoTell != 0 || w.roomHasFlag(tch.RoomVNum, "soundproof"))
	if tchNotellOrSP {
		sendToChar(ch, "They can't hear you.\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}

	w.performTell(ch, tch, arg)
	return true
}

// doSpecComm — port of do_spec_comm() (shout, whisper, ask).
