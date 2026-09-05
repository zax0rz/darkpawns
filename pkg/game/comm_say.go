package game

import "fmt"

func (w *World) doRaceSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if checkStupid(ch) {
		ch.SendMessage("You are too stupid to communicate with language!\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		ch.SendMessage("You cannot race-say!\r\n")
		return true
	}
	if arg == "" {
		ch.SendMessage("Yes, but WHAT do you want to say?\n\r")
		return true
	}

	var translate func(string) string
	var raceName string

	switch ch.Race {
	case RaceHuman:
		translate = speakHuman
		raceName = "Human"
	case RaceElf:
		translate = speakElven
		raceName = "Elven"
	case RaceDwarf:
		translate = speakDwarven
		raceName = "Dwarven"
	case RaceKender:
		translate = speakKender
		raceName = "Kenderkin"
	case RaceMinotaur:
		translate = speakMinotaur
		raceName = "Minotauran"
	case RaceRakshasa:
		translate = speakRakshasan
		raceName = "Rakshasan"
	case RaceSsaur:
		translate = speakSsaur
		raceName = "Ssauran"
	default:
		return true
	}

	translated := translate(arg)
	verb := determineVerb(arg)
	actorVerb := verb[:len(verb)-1]

	// Send to others in the room.
	verbMsg := fmt.Sprintf(" %s, ", verb)
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name {
			continue
		}
		if p.GetPosition() <= posSleeping {
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
		ch.SendMessage(fmt.Sprintf("You %s, '(In %s) %s'\r\n", actorVerb, raceName, arg))
	} else {
		ch.SendMessage("Ok.\n\r")
	}

	return true
}

// performTell — port of perform_tell().
