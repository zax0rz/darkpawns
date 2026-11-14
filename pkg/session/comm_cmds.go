package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// Communication command handlers.
// Source: src/act.comm.c — tell, reply, shout, gossip, emote, say, write, page, afk, ignore

// ---------------------------------------------------------------------------
// Tell / Reply
// ---------------------------------------------------------------------------

// filterCommMessage checks a message against word filters and records it for spam.
// Returns the (potentially filtered) message and whether it should be blocked.
func filterCommMessage(s *Session, message string) (string, bool) {
	if s.manager.modChecker != nil && s.player != nil {
		s.manager.modChecker.RecordMessage(s.player.Name)
		return s.manager.modChecker.CheckMessage(s.player.Name, message)
	}
	return message, false
}

func filteredCommMessage(s *Session, message string) (string, bool) {
	return filterCommMessage(s, sanitizeMessage(message))
}

// cmdTell sends a private message to another player.
// Source: act.comm.c do_tell() lines 901-931, perform_tell()
func cmdTell(s *Session, args []string) error {
	argument := strings.Join(args, " ")
	if len(args) >= 2 {
		filtered, block := filteredCommMessage(s, strings.Join(args[1:], " "))
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		argument = args[0] + " " + filtered
	}
	s.manager.world.DoTell(s.player, argument)
	return nil
}

// cmdReply replies to the last person who told you.
// Source: act.comm.c do_reply() lines 934-975
func cmdReply(s *Session, args []string) error {
	if len(args) > 0 {
		message, block := filteredCommMessage(s, strings.Join(args, " "))
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		s.manager.world.DoReply(s.player, message)
		return nil
	}
	s.manager.world.DoReply(s.player, "")
	return nil
}

// ---------------------------------------------------------------------------
// Shout / Gossip
// ---------------------------------------------------------------------------

// cmdChannel sends a message through one of the generic communication
// channels. The world layer owns each channel's C-specific gates and fanout;
// this command seam only preserves the shared sanitize/filter path.
func cmdChannel(s *Session, args []string, channel string) error {
	message := sanitizeMessage(strings.Join(args, " "))
	if len(args) > 0 {
		filtered, block := filterCommMessage(s, message)
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		message = filtered
	}
	s.manager.world.DoChannel(s.player, message, channel)
	return nil
}

// cmdShout broadcasts a message to all players in the same zone.
// Source: act.comm.c do_gen_comm() SCMD_SHOUT lines 1286-1289
// Original: zone-scoped; receivers must be POS_RESTING or higher.
func cmdShout(s *Session, args []string) error { return cmdChannel(s, args, "shout") }

// cmdGossip broadcasts a message to everyone online.
// Source: act.comm.c do_gen_comm() SCMD_GOSSIP lines 1286+
func cmdGossip(s *Session, args []string) error { return cmdChannel(s, args, "gossip") }

// ---------------------------------------------------------------------------
// Emote / Say
// ---------------------------------------------------------------------------

// cmdEmote broadcasts a roleplay action to the room.
// Source: src/act.wizard.c do_echo(), SCMD_EMOTE branch.
func cmdEmote(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Yes.. but what?")
		return nil
	}
	// C do_echo's SCMD_EMOTE branch (act.wizard.c:135-141): a muted (PLR_NOSHOUT)
	// or INT-0 actor cannot express themselves. Fires after the empty-argument
	// gate, before any bytes reach the room.
	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 || s.player.GetInt() == 0 {
		s.Send("You try to express yourself but cannot!")
		return nil
	}
	action := sanitizeMessage(strings.Join(args, " "))

	// Word filter + spam check
	filtered, block := filterCommMessage(s, action)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	action = filtered

	// src/act.wizard.c:144-151: room act first (including arena broadcast),
	// then the actor's repeat preference. Shared Act renders for each recipient.
	format := "$n " + action
	game.Act(s.manager.world, false, s.player, nil, nil, nil, format, "", game.ToRoom)
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.Send("Okay.\r\n")
	} else {
		game.Act(s.manager.world, false, s.player, nil, nil, nil, format, "", game.ToChar)
	}
	return nil
}

// cmdSay sends a message to the room with punctuation-based variants.
// Source: act.comm.c do_say() lines 824-870
func cmdSay(s *Session, args []string) error {
	text := sanitizeMessage(strings.Join(args, " "))
	if len(args) > 0 {
		filtered, block := filterCommMessage(s, text)
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		text = filtered
	}
	s.manager.world.DoSay(s.player, text)
	return nil
}

// ---------------------------------------------------------------------------
// Think
// ---------------------------------------------------------------------------

// cmdThink shows a flavor "thinking" message, optionally with a private
// thought broadcast to the room as "$n thinks . o O ( msg )".
// Source: act.comm.c ACMD(do_think) lines 1356-1386
func cmdThink(s *Session, args []string) error {
	if len(args) == 0 {
		broadcastToRoom(s, fmt.Sprintf("%s thinks about life, the universe, and everything.", s.player.Name))
		s.Send("You think about life, the universe, and everything.")
		return nil
	}

	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 || s.player.GetInt() == 0 {
		s.Send("You try to think aloud, but cannot!")
		return nil
	}

	message := strings.Join(args, " ")
	broadcastToRoom(s, fmt.Sprintf("%s thinks . o O ( %s )", s.player.Name, message))

	if s.player.Flags&(1<<game.PrfNoRepeat) == 0 {
		s.Send(fmt.Sprintf("You think . o O ( %s )", message))
	} else {
		s.Send("Ok.\n\r")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Write
// ---------------------------------------------------------------------------

// cmdWrite writes a message on a writable item (paper/scroll).
// Source: act.comm.c do_write() lines 978-1054
// Simplified: requires "pen" and "paper" in inventory.
func cmdWrite(s *Session, args []string) error {
	// Faithful port of do_write (src/act.comm.c:1024-1094); the game layer owns
	// C's two_arguments parsing and every refusal byte (DP-1516 item 7).
	s.manager.world.DoWrite(s.player, strings.Join(args, " "))
	return nil
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

// cmdPage sends an urgent message to one or more remote players.
// Source: act.comm.c do_page() lines 1056-1084
func cmdPage(s *Session, args []string) error {
	return cmdPageText(s, strings.Join(args, " "))
}

// cmdPageText keeps the raw remainder passed by C's half_chop. Unlike the
// usual tokenized command arguments, this preserves internal and trailing
// whitespace in the page body (act.comm.c:1112-1114).
func cmdPageText(s *Session, argument string) error {
	targetName, message := splitPageArguments(argument)
	if targetName == "" {
		s.Send("Whom do you wish to page?")
		return nil
	}

	// Page message with bell chars for urgency — act.comm.c line 1068
	// \007 is the bell character
	pageText := fmt.Sprintf("\x07\x07*%s* %s", s.player.Name, message)
	// C stores its CRLF in buf and act() appends another CRLF. Keep both
	// endings; the extra blank line is observable when two act() calls target
	// the same descriptor (notably page self).
	cActText := pageText + "\r\n\r\n"

	if strings.EqualFold(targetName, "all") {
		if s.player.GetLevel() <= LVL_GOD {
			s.Send("You will never be godly enough to do that!")
			return nil
		}
		s.manager.mu.RLock()
		sessions := make([]*Session, 0, len(s.manager.sessions))
		for _, target := range s.manager.sessions {
			sessions = append(sessions, target)
		}
		s.manager.mu.RUnlock()
		for _, target := range sessions {
			if target.isConPlaying() && target.player.GetPosition() > combat.PosSleeping {
				target.Send(cActText)
			}
		}
		return nil
	}

	target := s
	if !strings.EqualFold(targetName, "self") && !strings.EqualFold(targetName, "me") {
		// C do_page resolves with get_char_vis (src/act.comm.c:1082-1083) and
		// then requires a live descriptor: room first, then the players-only
		// world pass, both CAN_SEE-gated. The session fallback is fixture-only
		// and stays visibility-gated so an invisible target still gets C's
		// refusal.
		target = nil
		if resolved, ok := s.manager.world.ResolveCharWorld(s.player, targetName); ok && resolved.Player != nil {
			target = findSessionForPlayer(s.manager, resolved.Player)
		}
		if target == nil {
			if sess := findSessionByName(s.manager, targetName); sess != nil && sess.player != nil && game.CanSee(s.player, sess.player) {
				target = sess
			}
		}
	}
	if target == nil || target.player == nil || !target.authenticated {
		s.Send("There is no such person in the game!")
		return nil
	}
	if target.player.GetPosition() > combat.PosSleeping {
		target.Send(cActText)
	}
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.Send("Okay.")
	} else {
		s.Send(cActText)
	}

	return nil
}

// splitPageArguments mirrors C any_one_arg + skip_spaces: the target is the
// first case-folded token, while the message keeps all remaining spacing.
func splitPageArguments(argument string) (target, message string) {
	argument = strings.TrimLeft(argument, cCommandWhitespace)
	if argument == "" {
		return "", ""
	}
	if idx := strings.IndexAny(argument, cCommandWhitespace); idx >= 0 {
		return strings.ToLower(argument[:idx]), strings.TrimLeft(argument[idx+1:], cCommandWhitespace)
	}
	return strings.ToLower(argument), ""
}

// ---------------------------------------------------------------------------
// Auction / Gratz / Newbie Channel / Clan Tell
// ---------------------------------------------------------------------------

// cmdAuction sends a message on the auction channel.
// Source: act.comm.c do_gen_comm() SCMD_AUCTION
func cmdAuction(s *Session, args []string) error { return cmdChannel(s, args, "auction") }

// cmdCTell sends a message on the clan tell channel.
// Source: act.comm.c do_ctell()
func cmdCTell(s *Session, args []string) error {
	message := sanitizeMessage(strings.Join(args, " "))
	if len(args) > 0 {
		filtered, block := filterCommMessage(s, message)
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		message = filtered
	}
	// C do_ctell handles the immortal clan-number syntax and all sender/channel
	// gates inside the command path (act.comm.c:1451-1565). Keep those gates in
	// the game layer so direct ExecCTell callers and player dispatch agree.
	s.manager.world.ExecCTell(s.player, message)
	return nil
}

// ---------------------------------------------------------------------------
// AFK
// ---------------------------------------------------------------------------
