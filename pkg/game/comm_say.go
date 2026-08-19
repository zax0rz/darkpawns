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
	if ch.Flags&(1<<PrfNoRepeat) == 0 {
		ch.SendMessage(fmt.Sprintf("You%s'(In %s) %s'\r\n", verbMsg, raceName, arg))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// performTell — port of perform_tell().
