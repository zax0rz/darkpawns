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
	argument := strings.Join(args, " ")
	if len(args) >= 2 {
		message := sanitizeMessage(strings.Join(args[1:], " "))
		filtered, block := filterCommMessage(s, message)
		if block {
			game.SendToChar(s.player, "Your message was blocked.")
			return nil
		}
		argument = args[0] + " " + filtered
	}
	s.manager.world.DoSpecComm(s.player, argument, false)
	return nil
}

// ---------------------------------------------------------------------------
// Ask
// ---------------------------------------------------------------------------

// cmdAsk asks a question to a player in the same room.
func cmdAsk(s *Session, args []string) error {
	argument := strings.Join(args, " ")
	if len(args) >= 2 {
		message := sanitizeMessage(strings.Join(args[1:], " "))
		filtered, block := filterCommMessage(s, message)
		if block {
			game.SendToChar(s.player, "Your message was blocked.")
			return nil
		}
		argument = args[0] + " " + filtered
	}
	s.manager.world.DoSpecComm(s.player, argument, true)
	return nil
}
