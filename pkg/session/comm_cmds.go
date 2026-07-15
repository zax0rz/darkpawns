package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

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

// roomIsSoundproof returns true if the player's current room has the
// ROOM_SOUNDPROOF flag (bit 5 — from structs.h ROOM_SOUNDPROOF).
func roomIsSoundproof(s *Session) bool {
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return false
	}
	return room.HasFlag(5) // ROOM_SOUNDPROOF = bit 5
}

// cmdTell sends a private message to another player.
// Source: act.comm.c do_tell() lines 901-931, perform_tell()
func cmdTell(s *Session, args []string) error {
	argument := strings.Join(args, " ")
	if len(args) >= 2 {
		message := sanitizeMessage(strings.Join(args[1:], " "))
		filtered, block := filterCommMessage(s, message)
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
	message := sanitizeMessage(strings.Join(args, " "))
	if len(args) > 0 {
		filtered, block := filterCommMessage(s, message)
		if block {
			s.sendText("Your message was blocked.")
			return nil
		}
		message = filtered
	}
	s.manager.world.DoReply(s.player, message)
	return nil
}

// ---------------------------------------------------------------------------
// Shout / Gossip
// ---------------------------------------------------------------------------

// cmdShout broadcasts a message to all players in the same zone.
// Source: act.comm.c do_gen_comm() SCMD_SHOUT lines 1286-1289
// Original: zone-scoped; receivers must be POS_RESTING or higher.
func cmdShout(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Yes, shout, fine, shout we must, but WHAT???")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))

	// Word filter + spam check
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered

	// ROOM_SOUNDPROOF check — act.comm.c do_gen_comm() line 1289
	if roomIsSoundproof(s) {
		s.Send("The walls seem to absorb your words.\r\n")
		return nil
	}

	// Get the shouter's zone
	senderRoom, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return nil
	}
	senderZone := senderRoom.Zone

	s.Send(fmt.Sprintf("You shout, '%s'", message))

	text := fmt.Sprintf("%s shouts, '%s'", s.player.Name, message)

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "shout",
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}

	s.manager.mu.RLock()
	for name, sess := range s.manager.sessions {
		if name == s.player.Name || sess.player == nil {
			continue
		}
		// Restrict to same zone — act.comm.c line 1287
		targetRoom, ok := s.manager.world.GetRoom(sess.player.GetRoom())
		if !ok || targetRoom.Zone != senderZone {
			continue
		}
		// Skip players who are deafened / writing / in soundproof rooms
		// (simplified: just deliver to all in zone)
		select {
		case sess.send <- msg:
		default:
			slog.Warn("shout send channel full — dropping message", "target", name)
		}
	}
	s.manager.mu.RUnlock()
	return nil
}

// cmdGossip broadcasts a message to everyone online.
// Source: act.comm.c do_gen_comm() SCMD_GOSSIP lines 1286+
func cmdGossip(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Yes, gossip, fine, gossip we must, but WHAT???")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))

	// Word filter + spam check
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered

	s.Send(fmt.Sprintf("You gossip, '%s'", message))

	text := fmt.Sprintf("%s gossips, '%s'", s.player.Name, message)

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "gossip",
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}

	s.manager.mu.RLock()
	for name, sess := range s.manager.sessions {
		if name == s.player.Name || sess.player == nil {
			continue
		}
		select {
		case sess.send <- msg:
		default:
			slog.Warn("gossip send channel full — dropping message", "target", name)
		}
	}
	s.manager.mu.RUnlock()
	return nil
}

// ---------------------------------------------------------------------------
// Emote / Say
// ---------------------------------------------------------------------------

// cmdEmote broadcasts a roleplay action to the room.
// Source: act.comm.c do_emote() — "$n laughs." style
func cmdEmote(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Emote what?")
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

	// Sender sees: "You emit: $message"
	s.Send(fmt.Sprintf("You emit: %s", action))

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
		s.Send("Write what on what?")
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
// Extended: supports multiple targets as "page target1 target2 ... msg"
// Can reach any player, anywhere. Uses bell chars for urgency.
func cmdPage(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Whom do you wish to page?")
		return nil
	}

	// Multi-target: all args except the last are treated as target names.
	// The last arg is the message (single word).
	// Source extension: do_page() originally used half_chop for single target;
	// this Go version iterates through multiple target names.
	targetNames := args[:len(args)-1]
	message := args[len(args)-1]

	// Page message with bell chars for urgency — act.comm.c line 1068
	// \007 is the bell character
	pageText := fmt.Sprintf("\x07\x07*%s* %s", s.player.Name, message)

	var matched []string

	for _, targetName := range targetNames {
		// Find target — get_char_vis (act.comm.c line 1070)
		target, ok := s.manager.GetSession(targetName)
		if !ok || target.player == nil {
			s.Send("No one by that name is playing.\r\n")
			continue
		}

		// Deliver to target
		target.Send(pageText)
		matched = append(matched, target.player.Name)
	}

	if len(matched) > 0 {
		// Confirm to sender listing who was paged
		s.Send(fmt.Sprintf("You page %s with '%s'\r\n",
			strings.Join(matched, ", "), message))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Auction / Gratz / Newbie Channel / Clan Tell
// ---------------------------------------------------------------------------

// cmdAuction sends a message on the auction channel.
// Source: act.comm.c do_gen_comm() SCMD_AUCTION
func cmdAuction(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Auction what?")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered
	s.manager.world.ExecGenComm(s.player, "auction", message)
	return nil
}

// cmdGratz sends a message on the gratz channel.
// Source: act.comm.c do_gen_comm() SCMD_GRATZ
func cmdGratz(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Gratz whom?")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered
	s.manager.world.ExecGenComm(s.player, "gratz", message)
	return nil
}

// cmdNewbieChannel sends a message on the newbie channel.
// Source: act.comm.c do_gen_comm() SCMD_NEWBIE
// Named cmdNewbieChannel to avoid conflict with cmdNewbie (wizard command).
func cmdNewbieChannel(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Newbie what?")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered
	s.manager.world.ExecGenComm(s.player, "newbie", message)
	return nil
}

// cmdCTell sends a message on the clan tell channel.
// Source: act.comm.c do_ctell()
func cmdCTell(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What do you want to tell your clan?")
		return nil
	}
	message := sanitizeMessage(strings.Join(args, " "))
	filtered, block := filterCommMessage(s, message)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	message = filtered
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
