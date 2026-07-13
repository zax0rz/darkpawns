package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Communication commands (ported from act.comm.c)
// ---------------------------------------------------------------------------

// cmdRaceSay speaks in the player's racial tongue with syllable substitution.
// Delegates to game.World.doRaceSay (via ExecRaceSay) so that same-race
// players and immortals hear the original text while others hear the
// phonetically-translated version.
// Source: act.comm.c do_race_say() — wired to ExecRaceSay bridge in comm_say.go
func cmdRaceSay(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What do you want to say?")
		return nil
	}

	msg := sanitizeMessage(strings.Join(args, " "))
	s.manager.world.ExecRaceSay(s.player, msg)
	return nil
}

// cmdQcomm handles question communication (question asked to all questing players).
// Source: act.comm.c do_qcomm() — requires PRF_QUEST flag to participate.
func cmdQcomm(s *Session, args []string) error {
	// Quest flag check — act.comm.c do_qcomm() PRF_FLAGGED(ch, PRF_QUEST)
	if s.player.GetFlags()&(1<<uint(game.PrfQuest)) == 0 {
		s.Send("You aren't even part of the quest!")
		return nil
	}

	if len(args) == 0 {
		s.Send("What is your question?")
		return nil
	}

	msg := sanitizeMessage(strings.Join(args, " "))
	formatted := fmt.Sprintf("%s asks '%s'", s.player.Name, msg)

	s.Send(fmt.Sprintf("You ask '%s'", msg))

	// Broadcast to all online players
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || sess == s {
			continue
		}
		if sess.player.GetFlags()&(1<<uint(game.PrfQuest)) == 0 {
			continue
		}
		sess.Send(formatted)
	}
	s.manager.mu.RUnlock()

	return nil
}

// ---------------------------------------------------------------------------
// Whisper
// ---------------------------------------------------------------------------

// cmdWhisper whispers a private message to a visible character in the same room.
func cmdWhisper(s *Session, args []string) error {
	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 {
		game.SendToChar(s.player, "Sorry, you cannot do that.")
		return nil
	}

	if len(args) < 2 {
		game.SendToChar(s.player, "Whom do you want to whisper to.. and what??")
		return nil
	}

	// NOTE: no ROOM_SOUNDPROOF gate here — act.comm.c do_spec_comm() (whisper
	// AND ask) only checks PLR_NOSHOUT. SOUNDPROOF ("Shouts, gossip blocked")
	// gates ranged comm (do_tell/gossip/shout), not same-room whisper/ask.

	targetName := args[0]
	message := sanitizeMessage(strings.Join(args[1:], " "))

	// Word filter + spam check (opt-in moderation layer; passthrough when
	// no modChecker is configured — not part of the C game).
	filtered, block := filterCommMessage(s, message)
	if block {
		game.SendToChar(s.player, "Your message was blocked.")
		return nil
	}
	message = filtered

	target, found := s.manager.world.ResolveCharInRoom(s.player, targetName)
	if !found {
		game.SendToChar(s.player, "No-one by that name here.")
		return nil
	}

	var victim game.Actor
	switch {
	case target.Player != nil:
		victim = target.Player
	case target.Mob != nil:
		victim = target.Mob
	default:
		game.SendToChar(s.player, "No-one by that name here.")
		return nil
	}

	if victim == s.player {
		game.SendToChar(s.player, "You can't get your mouth close enough to your ear...")
		return nil
	}

	game.Act(nil, false, s.player, victim, nil, nil,
		"$n whispers to you, '$T'", message, game.ToVict)
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		game.SendToChar(s.player, "Okay.")
	} else {
		game.Act(nil, false, s.player, victim, nil, nil,
			"You whisper to $N, '$T'", message, game.ToChar)
	}
	game.Act(s.manager.world, false, s.player, victim, nil, nil,
		"$n whispers something to $N.", "", game.ToNotVict)

	return nil
}

// ---------------------------------------------------------------------------
// Ask
// ---------------------------------------------------------------------------

// cmdAsk asks a question to a player in the same room.
func cmdAsk(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Ask whom what?")
		return nil
	}

	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 {
		s.Send("You cannot communicate at all!")
		return nil
	}

	targetName := args[0]
	message := sanitizeMessage(strings.Join(args[1:], " "))
	roomVNum := s.player.GetRoomVNum()

	var targetSess *Session
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		if strings.EqualFold(sess.player.Name, targetName) && sess.player.GetRoomVNum() == roomVNum {
			targetSess = sess
			break
		}
	}
	s.manager.mu.RUnlock()

	if targetSess == nil {
		s.Send("No one by that name is here.")
		return nil
	}

	targetSess.Send(fmt.Sprintf("\x1B[1;36m%s asks, '%s'\033[0m\r\n", s.player.Name, message))
	s.Send(fmt.Sprintf("You ask %s, '%s'", targetSess.player.Name, message))

	roomText := fmt.Sprintf("%s asks %s something.\r\n", s.player.Name, targetSess.player.Name)
	broadcastToRoomText(s, roomVNum, roomText)

	return nil
}
