package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
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
// Source: act.comm.c do_emote() — "$n laughs." style
func cmdEmote(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Yes.. but what?")
		return nil
	}
	// C do_echo's SCMD_EMOTE branch (act.wizard.c:135-141): a muted (PLR_NOSHOUT)
	// or INT-0 actor cannot express themselves. Fires after the empty-argument
	// gate, before any bytes reach the room.
	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 || s.player.Stats.Int == 0 {
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

	// Sender sees their own name, exactly like the room does — C do_echo
	// sends the same "$n <text>" act() line TO_CHAR (oracle-proven:
	// command-surface-punctuation scenario; was an invented "You emit:").
	s.Send(fmt.Sprintf("%s %s", s.player.Name, action))

	// Room sees: "$n $message"
	text := fmt.Sprintf("%s %s", s.player.Name, action)
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "emote",
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
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

	if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 || s.player.Stats.Int == 0 {
		s.Send("You try to think aloud, but cannot!")
		return nil
	}

	message := strings.Join(args, " ")
	broadcastToRoom(s, fmt.Sprintf("%s thinks . o O ( %s )", s.player.Name, message))

	if s.player.Flags&(1<<game.PrfNoRepeat) == 0 {
		s.Send(fmt.Sprintf("You think . o O ( %s )", message))
	} else {
		s.Send("Ok.")
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
	if len(args) < 2 {
		s.Send("Write?  With what?  ON what?  What are you trying to do?!?")
		return nil
	}

	// Last arg is the item name, everything before is the message
	itemName := args[len(args)-1]
	message := strings.Join(args[:len(args)-1], " ")

	// Find the item in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.Send(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Check if item is writable (ITEM_NOTE type = 16)
	if item.GetTypeFlag() != game.ITEM_NOTE {
		s.Send("You can't write on that!")
		return nil
	}

	// Check if player has a pen in inventory
	hasPen := false
	if s.player.Inventory != nil {
		for _, invItem := range s.player.Inventory.FindItems("") {
			if invItem != nil && invItem.GetTypeFlag() == game.ITEM_PEN {
				hasPen = true
				break
			}
		}
	}
	if !hasPen {
		s.Send("You need a pen to write on something!")
		return nil
	}

	// Store message in extra descriptions
	if item.CustomData == nil {
		item.CustomData = make(map[string]interface{})
	}
	var extraDescs []parser.ExtraDesc
	if raw, ok := item.CustomData["extra_descs"]; ok {
		if currentDescs, ok := raw.([]parser.ExtraDesc); ok {
			extraDescs = currentDescs
		}
	}
	// Append new description for looking at the note
	extraDescs = append(extraDescs, parser.ExtraDesc{
		Keywords:    item.GetKeywords(),
		Description: message + "\r\n",
	})
	item.CustomData["extra_descs"] = extraDescs

	s.Send(fmt.Sprintf("You write '%s' on %s.", message, item.GetShortDesc()))
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
		defer s.manager.mu.RUnlock()
		for _, target := range s.manager.sessions {
			if target.player != nil && target.authenticated && target.player.GetPosition() > combat.PosSleeping {
				target.Send(cActText)
			}
		}
		return nil
	}

	target := s
	if !strings.EqualFold(targetName, "self") && !strings.EqualFold(targetName, "me") {
		target = findSessionByName(s.manager, targetName)
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

// cmdHoller sends a message on the global holler channel.
// Source: act.comm.c do_gen_comm() SCMD_HOLLER
func cmdHoller(s *Session, args []string) error { return cmdChannel(s, args, "holler") }

// cmdGratz sends a message on the gratz channel.
// Source: act.comm.c do_gen_comm() SCMD_GRATZ
func cmdGratz(s *Session, args []string) error { return cmdChannel(s, args, "grats") }

// cmdNewbieChannel sends a message on the newbie channel.
// Source: act.comm.c do_gen_comm() SCMD_NEWBIE
// Named cmdNewbieChannel to avoid conflict with cmdNewbie (wizard command).
func cmdNewbieChannel(s *Session, args []string) error { return cmdChannel(s, args, "newbie") }

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

// ---------------------------------------------------------------------------
// Ignore
// ---------------------------------------------------------------------------

// cmdIgnore toggles ignore status for a player.
func cmdIgnore(s *Session, args []string) error {
	if len(args) == 0 {
		// List ignored players
		ignored := s.player.GetIgnoredPlayers()
		if len(ignored) == 0 {
			s.Send("You are not ignoring anyone.")
			return nil
		}
		s.Send("You are ignoring:\n" + strings.Join(ignored, "\n"))
		return nil
	}

	targetName := args[0]

	// Can't ignore self
	if strings.EqualFold(targetName, s.player.Name) {
		s.Send("You can't ignore yourself.")
		return nil
	}

	// Toggle ignore
	if s.player.IsIgnoring(targetName) {
		s.player.RemoveIgnore(targetName)
		s.Send(fmt.Sprintf("%s is no longer ignored.", targetName))
	} else {
		s.player.AddIgnore(targetName)
		s.Send(fmt.Sprintf("%s is now ignored.", targetName))
	}
	return nil
}
