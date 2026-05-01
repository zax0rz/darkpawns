package game

import "fmt"

func (w *World) doRaceSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if checkStupid(ch) {
		sendToChar(ch, "You are too stupid to communicate with language!\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot race-say!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to say?\r\n")
		return true
	}

	var translate func(string) string
	var raceName string

	switch ch.Race {
	case raceDwarf, raceDeepDwarf:
		translate = speakDwarven
		raceName = "Dwarven"
	case raceElf, raceSurfaceElf:
		translate = speakElven
		raceName = "Elven"
	case raceGnoll:
		translate = speakGnoll
		raceName = "Gnoll"
	case raceDraconian:
		translate = speakDraconian
		raceName = "Draconian"
	case raceGiantish:
		translate = speakGiantish
		raceName = "Giantish"
	case raceUndead:
		translate = speakDeadspeak
		raceName = "Deadspeak"
	case raceDrow, raceRakshasa:
		translate = speakRakshasan
		raceName = "Rakshasan"
	default:
		return true
	}

	translated := translate(arg)
	verb := determineVerb(arg)

	// Send to others in the room.
	verbMsg := fmt.Sprintf(" %s, ", verb)
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name {
			continue
		}

		// Same race / immortals hear the original with race tag.
		// Other races hear the translated version.
		if p.Race == ch.Race || p.GetLevel() >= lvlImmort || p.IsNPC() {
			p.SendMessage(fmt.Sprintf("%s%s'(In %s) %s'\r\n", ch.Name, verbMsg, raceName, arg))
		} else {
			p.SendMessage(fmt.Sprintf("%s%s'%s'\r\n", ch.Name, verbMsg, translated))
		}
	}

	// Self-message.
	if ch.Flags&prfNoRepeat == 0 {
		ch.SendMessage(fmt.Sprintf("You%s'(In %s) %s'\r\n", verbMsg, raceName, arg))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// doSay — port of do_say().
func (w *World) doSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if checkStupid(ch) {
		sendToChar(ch, "You are too stupid to communicate with language!\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot speak!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to say?\r\n")
		return true
	}

	// Drunk substitution.
	speech := arg
	if ch.Conditions[condDrunk] > 10 {
		speech = speakDrunk(arg)
	}

	verb := determineVerb(arg)

	msg := fmt.Sprintf("$n %s, '%s'", verb, speech)
	msg = deleteAnsiControls(msg)
	w.roomMessage(ch.RoomVNum, msg)

	if ch.Flags&prfNoRepeat == 0 {
		selfMsg := fmt.Sprintf("You %s, '%s'\r\n", verb, arg)
		selfMsg = deleteAnsiControls(selfMsg)
		sendToChar(ch, selfMsg)
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// doGSay — port of do_gsay().
func (w *World) doGSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if !ch.InGroup {
		sendToChar(ch, "But you are not a member of any group!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to group-say?\r\n")
		return true
	}

	msg := fmt.Sprintf("$n tells the group, '%s'", arg)
	msg = deleteAnsiControls(msg)

	// Find group leader.
	var leader *Player
	if ch.Following != "" {
		if l, ok := w.GetPlayer(ch.Following); ok {
			leader = l
		}
	}
	if leader == nil {
		leader = ch
	}

	// Broadcast to group members in the room.
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name || !p.InGroup {
			continue
		}
		if p.Following == leader.Name || p.Name == leader.Name || ch.Following == p.Name {
			p.SendMessage(fmt.Sprintf("\x1B[1;37m%s\033[0m\r\n", msg))
		}
	}

	if ch.Flags&prfNoRepeat == 0 {
		selfMsg := fmt.Sprintf("You tell the group, '%s'\r\n", arg)
		selfMsg = deleteAnsiControls(selfMsg)
		ch.SendMessage(fmt.Sprintf("\x1B[1;37m%s\033[0m\r\n", selfMsg))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// performTell — port of perform_tell().
