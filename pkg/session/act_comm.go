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
	if blocked := qcommGuard(s); blocked {
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

// cmdQsay — "qsay <message>" quest-say (act.comm.c do_qcomm/SCMD_QSAY, level 0).
// Broadcasts "<name> quest-says, '<msg>'" to PRF_QUEST participants. C colors it &W...&n.
func cmdQsay(s *Session, args []string) error {
	if blocked := qcommGuard(s); blocked {
		return nil
	}
	if len(args) == 0 {
		s.Send(qcommEmptyMsg("qsay"))
		return nil
	}
	msg := sanitizeMessage(strings.Join(args, " "))
	self := fmt.Sprintf("You quest-say, '%s'", msg)
	other := fmt.Sprintf("%s quest-says, '%s'", s.player.Name, msg)
	broadcastQuest(s, self, other)
	return nil
}

// cmdQecho — "qecho <text>" immortal quest-echo (act.comm.c do_qcomm/SCMD_QECHO, LVL_IMMORT).
// Echoes the raw text verbatim to PRF_QUEST participants (no prefix, no color).
func cmdQecho(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if blocked := qcommGuard(s); blocked {
		return nil
	}
	if len(args) == 0 {
		s.Send(qcommEmptyMsg("qecho"))
		return nil
	}
	msg := sanitizeMessage(strings.Join(args, " "))
	broadcastQuest(s, msg, msg)
	return nil
}

// qcommGuard enforces the two do_qcomm pre-checks (act.comm.c:1306-1314) shared by every qcomm
// variant: PRF_QUEST required, and PLR_NOSHOUT (the "mute" flag this same pipeline sets) blocks
// the channel. Returns true if the caller should abort (message already sent).
func qcommGuard(s *Session) bool {
	if s.player.GetFlags()&(1<<uint(game.PrfQuest)) == 0 {
		s.Send("You aren't even part of the quest!")
		return true
	}
	if s.player.GetFlags()&(1<<uint(game.PlrNoshout)) != 0 {
		s.Send("You cannot quest-say!")
		return true
	}
	return false
}

// qcommEmptyMsg returns C's do_qcomm empty-argument chide for the given command name
// (act.comm.c:1320): "<Cmd>?  Yes, fine, <cmd> we must, but WHAT??" (capitalized first letter).
func qcommEmptyMsg(cmd string) string {
	c := []byte(cmd)
	if len(c) > 0 && c[0] >= 'a' && c[0] <= 'z' {
		c[0] -= 32
	}
	return fmt.Sprintf("%s?  Yes, fine, %s we must, but WHAT??", string(c), cmd)
}

// broadcastQuest sends a self/other message to every other PRF_QUEST participant online.
func broadcastQuest(s *Session, selfMsg, otherMsg string) {
	s.Send(selfMsg)
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || sess == s {
			continue
		}
		if sess.player.GetFlags()&(1<<uint(game.PrfQuest)) == 0 {
			continue
		}
		sess.Send(otherMsg)
	}
	s.manager.mu.RUnlock()
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
