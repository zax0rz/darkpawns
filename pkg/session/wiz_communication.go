package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
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
//
// Safety measures implemented:
//   1. ForcedPrivilegeLevel wired into getEffectiveLevel (checkLevel path)
//   2. IsForced flag prevents transitive force chains
//   3. Command denylist blocks dangerous commands
//   4. 3-second cooldown between force commands on the same target
// ---------------------------------------------------------------------------
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

	// --- Safety 3: Command denylist ---
	denyList := []string{"force", "shutdown", "purge", "set", "advance", "switch", "wiznet"}
	cmdLower := strings.ToLower(forceCmd)
	for _, denied := range denyList {
		if cmdLower == denied {
			s.Send(fmt.Sprintf("You cannot force '%s'.", forceCmd))
			slog.Warn("force denied: blocked command", "command", forceCmd, "by", s.player.Name)
			return nil
		}
	}

	if targetName == "all" {
		// Force-all still respects denylist (checked above) but skips per-target checks
		s.Send("OK.")
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
		for _, sess := range s.manager.sessions {
			if sess.player == nil {
				continue
			}
			// Safety 2: skip already-forced targets (no transitive chains)
			if sess.IsForced {
				continue
			}
			sess.IsForced = true
			sess.ForcedPrivilegeLevel = sess.player.Level
			sess.LastForceTime = time.Now()
			slog.Info("force all", "target", sess.player.Name, "command", forceCmd, "by", s.player.Name)
			// Execute the forced command
			forceArgs := strings.Fields(forceCmd)
			if len(forceArgs) > 0 {
				_ = ExecuteCommand(sess, forceArgs[0], forceArgs[1:])
			}
			sess.IsForced = false
			sess.ForcedPrivilegeLevel = 0
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

	// --- Safety 2: No transitive force chains ---
	if target.IsForced {
		s.Send("That player is already executing a forced command.")
		slog.Warn("force denied: transitive chain blocked", "target", target.player.Name, "by", s.player.Name)
		return nil
	}

	// --- Safety 4: Rate limiting (3-second cooldown per target) ---
	if !target.LastForceTime.IsZero() && time.Since(target.LastForceTime) < 3*time.Second {
		s.Send(fmt.Sprintf("You must wait before forcing %s again.", target.player.Name))
		return nil
	}

	// --- Safety 1: Set privilege level to target's level ---
	target.ForcedPrivilegeLevel = target.player.Level
	target.IsForced = true
	target.LastForceTime = time.Now()

	// Execute the forced command on the target
	forceArgs := strings.Fields(forceCmd)
	var execErr error
	if len(forceArgs) > 0 {
		execErr = ExecuteCommand(target, forceArgs[0], forceArgs[1:])
	}

	// Clear force state
	target.IsForced = false
	target.ForcedPrivilegeLevel = 0

	slog.Info("forced", "target", target.player.Name, "command", forceCmd, "by", s.player.Name)
	s.Send(fmt.Sprintf("Forced %s to '%s'.", target.player.Name, forceCmd))
	return execErr
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
			fmt.Fprintf(&online, "  %s\r\n", sess.player.Name)
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
