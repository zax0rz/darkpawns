package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

func cmdGecho(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Echo what?")
		return nil
	}
	msg := strings.Join(args, " ")
	if len(msg) > 500 {
		s.Send("Maximum gecho length is 500 characters.")
		return nil
	}
	for _, sess := range s.manager.sessions {
		if sess.player != nil {
			sess.Send(msg)
		}
	}
	slog.Warn("wizard gecho", "message", msg, "by", s.player.Name)
	return nil
}

// ---------------------------------------------------------------------------
// echo — echo message to room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdEcho(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes.. but what?")
		return nil
	}
	msg := strings.Join(args, " ")
	if len(msg) > 500 {
		s.Send("Maximum echo length is 500 characters.")
		return nil
	}
	broadcastToRoomText(s, s.player.RoomVNum, msg)
	slog.Warn("wizard echo", "by", s.player.Name, "room", s.player.RoomVNum, "message", msg)
	s.Send(msg)
	return nil
}

// ---------------------------------------------------------------------------
// send — send message to another character (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdSend(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Send what to who?")
		return nil
	}

	targetName := args[0]
	msg := strings.Join(args[1:], " ")

	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("No one by that name online.")
		return nil
	}

	target.Send(msg)
	slog.Warn("wizard send", "by", s.player.Name, "target", target.player.Name, "message", msg)
	s.Send(fmt.Sprintf("You send '%s' to %s.", msg, target.player.Name))
	return nil
}

// ---------------------------------------------------------------------------
// force — force command on another character (LVL_GRGOD)
// ---------------------------------------------------------------------------
//
// *** INTENTIONAL STUB — command is logged but NOT executed. ***
//
// This is a safety-first stub. Before activating, the following MUST be implemented:
//
//  1. Privilege level: The forced command must execute at the TARGET's privilege
//     level, not the wizard's. A level-50 wizard force on a level-1 player must not
//     let that player run immortal commands.
//
//  TODO: Add a forced-player privilege mask or temporary level override that
//  scopes only the single forced command execution.
//
//  2. No transitive force chains: If player A forces player B, player B must NOT
//     be able to force player C. This prevents force amplification attacks.
//
//  TODO: Add a session flag (e.g. s.isForced) that is checked before any force
//  command can execute, and cleared after the forced command completes.
//
//  3. Dangerous command blocklist: Commands like "force", "shutdown", "purge",
//     "set", and "advance" should not be forceable even with correct privilege.
//
//  TODO: Add a denylist of commands that cannot be forced, similar to the original
//  C codebase's do_force() restrictions.
//
//  4. Audit trail: Every forced command must be logged with full context (wizard,
//     target, command text, timestamp). The current slog.Info call covers this.
//
//  5. Rate limiting: A wizard should not be able to spam force commands on a target.
//
//  TODO: Add a per-target cooldown or global force-rate limit.
//
func cmdForce(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Whom do you wish to force do what?")
		return nil
	}

	forceCmd := args[1]
	targetName := strings.ToLower(args[0])

	if targetName == "all" {
		s.Send("OK.")
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
		for _, sess := range s.manager.sessions {
			if sess.player == nil {
				continue
			}
			slog.Info("force all", "target", sess.player.Name, "command", forceCmd, "by", s.player.Name)
		}
		return nil
	}

	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("No one by that name here.")
		return nil
	}

	if target.player.Level >= s.player.Level {
		s.Send("You cannot force that player.")
		return nil
	}

	slog.Info("forced", "target", target.player.Name, "command", forceCmd, "by", s.player.Name)
	s.Send(fmt.Sprintf("Forced %s to '%s'.", target.player.Name, forceCmd))
	return nil
}

// ---------------------------------------------------------------------------
// shutdown — shut down the server (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdWiznet(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: wiznet <text> | #<level> <text> | *<emote> | @")
		return nil
	}
	fullArg := strings.Join(args, " ")

	// wiznet @ — list gods online
	if fullArg == "@" {
		var online, offline strings.Builder
		online.WriteString("Gods online:\r\n")
		offline.WriteString("Gods offline:\r\n")
		anyOnline := false
		anyOffline := false
		s.manager.mu.RLock()
		for _, sess := range s.manager.sessions {
			if sess.player == nil || sess.player.Level < LVL_IMMORT {
				continue
			}
			// Simple distinction: all immortals in session are "online"
			online.WriteString(fmt.Sprintf("  %s\r\n", sess.player.Name))
			anyOnline = true
		}
		s.manager.mu.RUnlock()
		if anyOnline {
			s.Send(online.String())
		}
		if anyOffline {
			s.Send(offline.String())
		}
		return nil
	}

	// Check for level prefix: #<level> <text>
	level := LVL_IMMORT
	text := fullArg
	if len(args[0]) > 0 && args[0][0] == '#' {
		lvlStr := args[0][1:]
		lvl, err := strconv.Atoi(lvlStr)
		if err == nil && lvl >= LVL_IMMORT {
			level = lvl
			if level > s.player.Level {
				s.Send("You can't wizline above your own level.")
				return nil
			}
			text = strings.Join(args[1:], " ")
		}
	}

	// Check for emote prefix: *<text>
	isEmote := false
	if len(args[0]) > 0 && args[0][0] == '*' {
		isEmote = true
		text = strings.Join(args, " ")[1:]
	}

	if len(text) == 0 {
		s.Send("Don't bother the gods like that!")
		return nil
	}

	fromName := s.playerName
	msg := fmt.Sprintf("%s: %s%s\r\n", fromName, map[bool]string{true: "<--- ", false: ""}[isEmote], text)
	shadowMsg := fmt.Sprintf("Someone: %s%s\r\n", map[bool]string{true: "<--- ", false: ""}[isEmote], text)

	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || sess.player.Level < level {
			continue
		}
		if sess.player.Level >= level {
			toSend := msg
			if sess.player.Level < s.player.Level {
				toSend = shadowMsg
			}
			sess.Send(toSend)
		}
	}
	s.manager.mu.RUnlock()
	return nil
}

// cmdZreset — reset a zone by VNum (LVL_GOD)
// Original: act.wizard.c do_zreset() — reset_zone() is async via spawner
