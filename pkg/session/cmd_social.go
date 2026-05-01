package session

import (
	"strings"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdSocial(s *Session, social *game.Social, args []string) error {
	msgs := social.Messages
	if len(msgs) == 0 {
		return nil
	}

	// Extract target name from args
	targetName := strings.TrimSpace(strings.Join(args, " "))

	// Check if this is a 3-message (self-only) social
	// Convention: 3 messages means: [0]=char_no_arg, [1]=others_no_arg, [2]="#" (terminator)
	threeMsg := len(msgs) == 3 && msgs[2] == "#"

	// Helper to replace $n/$N/$m/$M/$s/$S/$e in a message
	// $n = character name, $N = target name
	// $m = target obj pronoun (him/her/it), $M = target obj pronoun
	// $s = target poss pronoun (his/her/its), $S = target poss pronoun
	// $e = target subj pronoun (he/she/it)
	actSubst := func(msg string, charName, targetName string, targetSex int) string {
		msg = strings.ReplaceAll(msg, "$n", charName)
		msg = strings.ReplaceAll(msg, "$N", targetName)
		switch targetSex {
		case 0:
			msg = strings.ReplaceAll(msg, "$m", "him")
			msg = strings.ReplaceAll(msg, "$M", "him")
			msg = strings.ReplaceAll(msg, "$s", "his")
			msg = strings.ReplaceAll(msg, "$S", "his")
			msg = strings.ReplaceAll(msg, "$e", "he")
		case 1:
			msg = strings.ReplaceAll(msg, "$m", "her")
			msg = strings.ReplaceAll(msg, "$M", "her")
			msg = strings.ReplaceAll(msg, "$s", "her")
			msg = strings.ReplaceAll(msg, "$S", "her")
			msg = strings.ReplaceAll(msg, "$e", "she")
		default:
			msg = strings.ReplaceAll(msg, "$m", "it")
			msg = strings.ReplaceAll(msg, "$M", "it")
			msg = strings.ReplaceAll(msg, "$s", "its")
			msg = strings.ReplaceAll(msg, "$S", "its")
			msg = strings.ReplaceAll(msg, "$e", "it")
		}
		return msg
	}

	// Helper to send a message to char; skip "#" sentinel
	sendToChar := func(msg string) {
		if msg == "#" || msg == "" {
			return
		}
		s.sendText(msg)
	}

	// Helper to send message to everyone in room except sender
	sendToRoom := func(msg string) {
		if msg == "#" || msg == "" {
			return
		}
		s.manager.BroadcastToRoom(s.player.GetRoom(), []byte(msg), s.player.Name)
	}

	// Helper to send to a specific player (victim)
	sendToVictim := func(msg string, victimName string) {
		if msg == "#" || msg == "" {
			return
		}
		victim, ok := s.manager.world.GetPlayer(victimName)
		if ok {
			victim.SendMessage(msg)
		}
	}

	if targetName == "" || threeMsg {
		// No argument or 3-message self-only social
		if len(msgs) >= 1 {
			sendToChar(actSubst(msgs[0], s.player.Name, "", 0))
		}
		if len(msgs) >= 2 {
			sendToRoom(actSubst(msgs[1], s.player.Name, "", 0))
		}
		return nil
	}

	// Try to find target in the room
	// Check players first
	victimName := ""
	victimSex := 2
	players := s.manager.world.GetPlayersInRoom(s.player.GetRoom())
	for _, p := range players {
		if strings.EqualFold(p.Name, targetName) && p.Name != s.player.Name {
			victimName = p.Name
			victimSex = p.Sex
			break
		}
	}

	// Also check mobs in room
	if victimName == "" {
		mobs := s.manager.world.GetMobsInRoom(s.player.GetRoom())
		for _, m := range mobs {
			if strings.EqualFold(m.GetShortDesc(), targetName) || strings.EqualFold(m.GetName(), targetName) {
				victimName = m.GetShortDesc()
				victimSex = m.GetSex()
				break
			}
		}
	}

	// Player name with self-target
	if victimName == "" && strings.EqualFold(targetName, s.player.Name) {
		victimName = s.player.Name
	}

	if victimName == "" {
		// Not found message
		// Messages[5] = not_found (6th entry, 0-indexed=5)
		if len(msgs) >= 6 && msgs[5] != "#" {
			sendToChar(actSubst(msgs[5], s.player.Name, targetName, 0))
		} else if len(msgs) >= 5 {
			// vict_found used as fallback when targeting someone not there
			_ = msgs
			s.sendText("They aren't here.")
		}
		return nil
	}

	if strings.EqualFold(victimName, s.player.Name) {
		// Targetting self
		// Messages[6] = char_auto (7th entry, 0-indexed=6)
		// Messages[7] = others_auto (8th entry)
		if len(msgs) >= 7 && msgs[6] != "#" {
			sendToChar(actSubst(msgs[6], s.player.Name, s.player.Name, 0))
		}
		if len(msgs) >= 8 && msgs[7] != "#" {
			sendToRoom(actSubst(msgs[7], s.player.Name, s.player.Name, 0))
		}
		return nil
	}

	// Normal target found
	// Messages[2] = char_found
	// Messages[3] = others_found
	// Messages[4] = vict_found
	if len(msgs) >= 3 && msgs[2] != "#" {
		sendToChar(actSubst(msgs[2], s.player.Name, victimName, victimSex))
	}
	if len(msgs) >= 4 && msgs[3] != "#" {
		sendToRoom(actSubst(msgs[3], s.player.Name, victimName, victimSex))
	}
	if len(msgs) >= 5 && msgs[4] != "#" {
		sendToVictim(actSubst(msgs[4], s.player.Name, victimName, victimSex), victimName)
	}

	return nil
}

// cmdLook shows the current room.
