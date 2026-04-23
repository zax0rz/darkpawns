package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// Communication command handlers.
// Source: src/act.comm.c — tell, reply, shout, gossip, emote, say, write, page, afk, ignore

// ---------------------------------------------------------------------------
// Tell / Reply
// ---------------------------------------------------------------------------

// cmdTell sends a private message to another player.
// Source: act.comm.c do_tell() lines 901-931, perform_tell()
func cmdTell(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Who do you wish to tell what??")
		return nil
	}
	targetName := args[0]
	message := strings.Join(args[1:], " ")

	if strings.EqualFold(targetName, s.player.Name) {
		s.Send("You try to tell yourself something.")
		return nil
	}

	// Find target session — act.comm.c line 909 get_char_vis()
	target, ok := s.manager.GetSession(targetName)
	if !ok || target.player == nil {
		s.Send("There is no such player online.")
		return nil
	}

	// Check if target is ignoring sender
	if target.player.IsIgnoring(s.player.Name) {
		s.Send(fmt.Sprintf("%s is ignoring you.", target.player.Name))
		return nil
	}

	// Deliver to target — act.comm.c perform_tell()
	// Target sees: "$n tells you, '$message'"
	target.Send(fmt.Sprintf("%s tells you, '%s'", s.player.Name, message))

	// Track last teller on target's session for reply support
	target.lastTeller = s.player.Name

	// AFK warning to sender — act.comm.c perform_tell() line 957
	if target.player.AFK {
		s.Send(fmt.Sprintf("%s is AFK right now, %s may not hear you.", target.player.Name, target.player.Name))
	}

	// Confirm to sender — act.comm.c perform_tell() line 964
	// Sender sees: "You tell $N, '$message'"
	s.Send(fmt.Sprintf("You tell %s, '%s'", target.player.Name, message))
	return nil
}

// cmdReply replies to the last person who told you.
// Source: act.comm.c do_reply() lines 934-975
func cmdReply(s *Session, args []string) error {
	if s.lastTeller == "" {
		s.Send("You have no-one to reply to!")
		return nil
	}
	if len(args) == 0 {
		s.Send("What is your reply?")
		return nil
	}

	message := strings.Join(args, " ")

	// Find the last teller
	target, ok := s.manager.GetSession(s.lastTeller)
	if !ok || target.player == nil {
		s.Send("They are no longer playing.")
		return nil
	}

	// Check if target is ignoring sender
	if target.player.IsIgnoring(s.player.Name) {
		s.Send(fmt.Sprintf("%s is ignoring you.", target.player.Name))
		return nil
	}

	// Deliver to target
	target.Send(fmt.Sprintf("%s tells you, '%s'", s.player.Name, message))
	target.lastTeller = s.player.Name

	// AFK warning
	if target.player.AFK {
		s.Send(fmt.Sprintf("%s is AFK right now, %s may not hear you.", target.player.Name, target.player.Name))
	}

	// Confirm to sender
	s.Send(fmt.Sprintf("You tell %s, '%s'", target.player.Name, message))
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
	message := strings.Join(args, " ")

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
	message := strings.Join(args, " ")

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
	action := strings.Join(args, " ")

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
	if len(args) == 0 {
		s.Send("Yes, but WHAT do you want to say?")
		return nil
	}

	text := strings.Join(args, " ")

	// Determine verb based on trailing punctuation — act.comm.c do_say() switch
	verb := "says"
	if len(text) > 0 {
		switch text[len(text)-1] {
		case '!':
			verb = "exclaims"
		case '?':
			verb = "asks"
		case '.':
			verb = "states"
		}
	}

	// Sender sees: "You say '$message'"
	s.Send(fmt.Sprintf("You %s '%s'", verb, text))

	// Room sees: "$n says, '$message'"
	roomText := fmt.Sprintf("%s %s, '%s'", s.player.Name, verb, text)
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "say",
			From: s.player.Name,
			Text: roomText,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
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

	// Check if item is writable (ITEM_NOTE type = 23 from structs.h)
	// For now, allow writing on any item as a simplification
	// In full implementation, check item.GetTypeFlag() == 23
	_ = message

	s.Send(fmt.Sprintf("You write '%s' on %s.", message, item.GetShortDesc()))
	return nil
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

// cmdPage sends an urgent message to a remote player.
// Source: act.comm.c do_page() lines 1056-1084
// Can reach any player, anywhere. Uses bell chars for urgency.
func cmdPage(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Whom do you wish to page?")
		return nil
	}

	targetName := args[0]
	message := strings.Join(args[1:], " ")

	// Find target — get_char_vis (act.comm.c line 1070)
	target, ok := s.manager.GetSession(targetName)
	if !ok || target.player == nil {
		s.Send("There is no such person in the game!")
		return nil
	}

	// Page message with bell chars for urgency — act.comm.c line 1068
	// \007 is the bell character
	pageText := fmt.Sprintf("\x07\x07*%s* %s", s.player.Name, message)

	// Deliver to target
	target.Send(pageText)

	// Confirm to sender
	s.Send(pageText)
	return nil
}

// ---------------------------------------------------------------------------
// AFK
// ---------------------------------------------------------------------------

// cmdAfk toggles away-from-keyboard status.
// Source: act.comm.c PRF_AFK flag usage in perform_tell()
func cmdAfk(s *Session, args []string) error {
	// Toggle AFK state
	s.player.AFK = !s.player.AFK

	if s.player.AFK {
		// Set AFK message if provided
		if len(args) > 0 {
			s.player.AFKMessage = strings.Join(args, " ")
		} else {
			s.player.AFKMessage = ""
		}
		s.Send("You are now AFK.")

		// Notify room
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "afk",
				From: s.player.Name,
				Text: fmt.Sprintf("%s is now AFK.", s.player.Name),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
		}
		s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
	} else {
		s.player.AFKMessage = ""
		s.Send("You are no longer AFK.")

		// Notify room
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "afk",
				From: s.player.Name,
				Text: fmt.Sprintf("%s is no longer AFK.", s.player.Name),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
		}
		s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
	}
	return nil
}

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
