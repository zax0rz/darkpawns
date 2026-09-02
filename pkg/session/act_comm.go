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
	msg := sanitizeMessage(strings.Join(args, " "))
	s.manager.world.ExecRaceSay(s.player, msg)
	return nil
}

// cmdRaceSayText preserves the raw argument remainder delivered by the
// telnet transport. C's do_race_say keeps internal spacing and only strips
// leading spaces before dispatch, so the transport-aware path must not rebuild
// the message from tokenized words (act.comm.c:641-676).
func cmdRaceSayText(s *Session, msg string) error {
	s.manager.world.ExecRaceSay(s.player, sanitizeMessage(msg))
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
	return cmdQsayText(s, strings.Join(args, " "))
}

func cmdQsayText(s *Session, msg string) error {
	if blocked := qcommGuard(s); blocked {
		return nil
	}
	msg = strings.TrimLeft(msg, cCommandWhitespace)
	if msg == "" {
		s.Send(qcommEmptyMsg("qsay"))
		return nil
	}
	// C delete_ansi_controls runs on the argument before the SCMD_QSAY
	// templates are built. The &W/&n wrappers are part of those templates and
	// must remain in the player-facing act bytes (act.comm.c:1325-1341).
	msg = game.DeleteANSIControls(sanitizeMessage(msg))
	self := fmt.Sprintf("&WYou quest-say, '%s'&n", msg)
	other := fmt.Sprintf("&W%s quest-says, '%s'&n", s.player.Name, msg)
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		self = "Okay."
	}
	broadcastQuest(s, self, other)
	return nil
}

// cmdQecho — "qecho <text>" immortal quest-echo (act.comm.c do_qcomm/SCMD_QECHO, LVL_IMMORT).
// Echoes the raw text to PRF_QUEST participants (no prefix, no color).
func cmdQecho(s *Session, args []string) error {
	return cmdQechoText(s, strings.Join(args, " "))
}

func cmdQechoText(s *Session, msg string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if blocked := qcommGuard(s); blocked {
		return nil
	}
	if msg == "" {
		s.Send(qcommEmptyMsg("qecho"))
		return nil
	}
	msg = game.CapitalizeSentence(game.DeleteANSIControls(sanitizeMessage(msg)))
	self := msg
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		self = "Okay."
	}
	broadcastQuest(s, self, msg)
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
		flags := sess.player.GetFlags()
		if flags&(1<<uint(game.PrfQuest)) == 0 || flags&(1<<uint(game.PlrWriting)) != 0 {
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
