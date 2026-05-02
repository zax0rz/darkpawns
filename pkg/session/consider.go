package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdConsider compares the player against a target.
// Source: act.informative.c ACMD(do_consider)
func cmdConsider(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Consider killing who?")
		return nil
	}

	targetName := strings.TrimSpace(strings.Join(args, " "))
	roomVNum := s.player.GetRoom()

	// Look for target in room — players first, then mobs
	var targetNameFound string
	var targetSex int
	var targetLevel int
	var targetHP int

	// Check players
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	var targetPlayer *game.Player
	for _, p := range players {
		if strings.EqualFold(p.Name, targetName) && p.Name != s.player.Name {
			targetPlayer = p
			targetNameFound = p.Name
			targetSex = p.Sex
			targetLevel = p.Level
			targetHP = p.Health
			break
		}
	}

	// Check mobs if no player found
	var targetMob *game.MobInstance
	if targetPlayer == nil {
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, m := range mobs {
			mobName := m.GetName()
			if mobName == "" {
				mobName = m.GetShortDesc()
			}
			if strings.EqualFold(mobName, targetName) || strings.EqualFold(m.GetShortDesc(), targetName) {
				targetMob = m
				targetNameFound = mobName
				targetSex = m.GetSex()
				targetLevel = m.GetLevel()
				targetHP = m.GetHP()
				break
			}
		}
	}

	if targetNameFound == "" {
		s.Send("They aren't here.")
		return nil
	}

	// Calculate damdiff: compare damage rolls
	playerDamroll := s.player.Level * 2
	// Add str_app equivalent: roughly +1 per 3 strength above 15
	if s.player.Stats.Str > 15 {
		playerDamroll += (s.player.Stats.Str - 15) / 3
	}
	playerDamroll += s.player.Damroll // bonus from gear/affects

	targetDamroll := targetLevel * 2
	if targetMob != nil {
		targetDamroll += targetMob.GetDamroll()
	} else if targetPlayer != nil {
		if targetPlayer.Stats.Str > 15 {
			targetDamroll += (targetPlayer.Stats.Str - 15) / 3
		}
		targetDamroll += targetPlayer.Damroll
	}

	damdiff := targetDamroll - playerDamroll

	// First line based on damdiff (EXACT strings from original C including trailing space and comma)
	var firstLine string
	switch {
	case damdiff > 20:
		firstLine = "$N looks like $E could eat you for lunch, "
	case damdiff > 10:
		firstLine = "$N looks like $E could tear you up in a fight, "
	case damdiff > 5:
		firstLine = "$N looks like $E could hurt you in a fight, "
	case damdiff > -3:
		firstLine = "$N looks like a fair fight, "
	case damdiff > -5:
		firstLine = "$N looks like an easy kill, "
	case damdiff > -10:
		firstLine = "$N looks like a very easy kill, "
	default:
		firstLine = "$N might not even be worth the effort to kill, "
	}

	// Calculate hitdiff: compare hit points
	playerMaxHP := s.player.MaxHealth
	hitdiff := targetHP - playerMaxHP

	// Second line based on hitdiff
	var secondLine string
	switch {
	case hitdiff > 30:
		secondLine = "and you would need a lot of help to beat $M."
	case hitdiff > 10:
		secondLine = "and you would need some help to beat $M."
	case hitdiff > -10:
		secondLine = "and you wouldn't need any help at all to beat $M."
	case hitdiff > -30:
		secondLine = "and you think you could beat $M without too much help."
	case hitdiff > -60:
		secondLine = "and you think it would be a fair fight for any group of your size."
	default:
		secondLine = ""
	}

	// Substitute $N, $E, $S, $M with appropriate pronouns
	msg := resolvePronouns(firstLine + secondLine, targetNameFound, targetSex)

	s.Send(msg)

	// Broadcast consider action to the room
	broadcastConsider(s, targetNameFound, roomVNum)

	return nil
}

// resolvePronouns replaces $N, $E, $S, $M tokens in a consider message.
func resolvePronouns(msg string, targetName string, sex int) string {
	var subject, object, possessive string

	switch sex {
	case 0: // neutral
		subject = "it"
		object = "it"
		possessive = "its"
	case 1: // male
		subject = "he"
		object = "him"
		possessive = "his"
	case 2: // female
		subject = "she"
		object = "her"
		possessive = "her"
	default:
		subject = "it"
		object = "it"
		possessive = "its"
	}

	msg = strings.ReplaceAll(msg, "$N", targetName)
	msg = strings.ReplaceAll(msg, "$E", subject)
	msg = strings.ReplaceAll(msg, "$S", possessive)
	msg = strings.ReplaceAll(msg, "$M", object)

	return msg
}

// broadcastConsider sends the consider action message to everyone in the room except the considerer.
func broadcastConsider(s *Session, targetName string, roomVNum int) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "consider",
			From: s.player.Name,
			Text: fmt.Sprintf("%s considers %s.", s.player.Name, targetName),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}
