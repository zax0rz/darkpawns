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

// NPCDoRaceSay is do_race_say for a switched mobile. C's GET_RACE reads the
// mobile prototype, while its PLR_NOSHOUT and PRF_NOREPEAT checks read the
// shared zeroed dummy_mob player-specials record.
func (w *World) NPCDoRaceSay(ch *MobInstance, arg string) {
	if ch == nil {
		return
	}
	arg = skipSpaces(arg)
	if ch.GetWis() == 0 || ch.GetInt() == 0 {
		ch.SendMessage("You are too stupid to communicate with language!\r\n")
		return
	}
	if arg == "" {
		ch.SendMessage("Yes, but WHAT do you want to say?\n\r")
		return
	}
	prototype := ch.Proto()
	if prototype == nil {
		return
	}
	var translate func(string) string
	var raceName string
	switch prototype.Race {
	case RaceHuman:
		translate, raceName = speakHuman, "Human"
	case RaceElf:
		translate, raceName = speakElven, "Elven"
	case RaceDwarf:
		translate, raceName = speakDwarven, "Dwarven"
	case RaceKender:
		translate, raceName = speakKender, "Kenderkin"
	case RaceMinotaur:
		translate, raceName = speakMinotaur, "Minotauran"
	case RaceRakshasa:
		translate, raceName = speakRakshasan, "Rakshasan"
	case RaceSsaur:
		translate, raceName = speakSsaur, "Ssauran"
	default:
		return
	}
	translated := translate(arg)
	verb := determineVerb(arg)
	verbMsg := fmt.Sprintf(" %s, ", verb)
	for _, player := range w.GetPlayersInRoom(ch.GetRoomVNum()) {
		if player.GetPosition() <= posSleeping {
			continue
		}
		if player.GetRace() == prototype.Race || player.GetLevel() >= lvlImmort {
			player.SendMessage(fmt.Sprintf("%s%s'(In %s) %s'\r\n", ch.GetName(), verbMsg, raceName, arg))
		} else {
			player.SendMessage(fmt.Sprintf("%s%s'%s'\r\n", ch.GetName(), verbMsg, translated))
		}
	}
	actorVerb := verb[:len(verb)-1]
	ch.SendMessage(fmt.Sprintf("You %s, '(In %s) %s'\r\n", actorVerb, raceName, arg))
}

// performTell — port of perform_tell().
