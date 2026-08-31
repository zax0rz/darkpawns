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
		s.Send("That must be a mistake...\r\n")
		return nil
	}
	msg := strings.Join(args, " ")
	if len(msg) > 500 {
		s.Send("Maximum gecho length is 500 characters.")
		return nil
	}
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player != nil {
			sess.Send(msg)
		}
	}
	s.manager.mu.RUnlock()
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
	// C act(buf, ..., TO_ROOM) excludes the echoer; the TO_CHAR copy below
	// delivers their own view (or OK under norepeat).
	s.manager.BroadcastToRoom(s.player.RoomVNum, []byte(msg), s.player.Name)
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
// force — force command on another character (LVL_GOD)
//
// Source: src/act.wizard.c:1856-1906. The C handler deliberately has no
// command denylist or cooldown; its command surface is part of the game (R2).
// ---------------------------------------------------------------------------
func cmdForce(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 || strings.TrimSpace(strings.Join(args[1:], " ")) == "" {
		s.Send("Whom do you wish to force do what?\r\n")
		return nil
	}

	forceCmd := strings.Join(args[1:], " ")
	targetName := args[0]

	// C treats "room" and "all" specially only for LVL_GRGOD and above.
	// Below that level they fall through to ordinary get_char_vis lookup.
	if s.player.GetLevel() >= LVL_GRGOD && strings.EqualFold(targetName, "room") {
		s.Send("Okay.\r\n")
		for _, target := range forceTargets(s, true) {
			forceSessionCommand(s, target, forceCmd, true)
		}
		return nil
	}
	if s.player.GetLevel() >= LVL_GRGOD && strings.EqualFold(targetName, "all") {
		s.Send("Okay.\r\n")
		for _, target := range forceTargets(s, false) {
			forceSessionCommand(s, target, forceCmd, true)
		}
		return nil
	}

	// C get_char_vis checks the room first and then the global character list.
	// ResolveCharWorld carries those visibility, self/me, abbreviation, and
	// ordinal rules; the session lookup then selects the connected PC that C's
	// descriptor-backed force path can command.
	target := findForceSession(s, targetName)
	if target == nil || target.player == nil {
		s.Send(noPersonHere)
		return nil
	}

	if target.player.Level >= s.player.Level {
		s.Send("No, no, no!\r\n")
		return nil
	}

	s.Send("Okay.\r\n")
	forceSessionCommand(s, target, forceCmd, s.player.GetLevel() < LVL_IMPL)
	return nil
}

func findForceSession(s *Session, name string) *Session {
	if s == nil || s.player == nil || s.manager == nil || s.manager.world == nil {
		return nil
	}
	target, ok := s.manager.world.ResolveCharWorld(s.player, name)
	if !ok || target.Player == nil {
		return nil
	}

	s.manager.mu.RLock()
	defer s.manager.mu.RUnlock()
	for _, sess := range s.manager.sessions {
		if sess.player == target.Player {
			return sess
		}
	}
	return nil
}

// forceTargets returns the connected player sessions that C's room/all loops
// would visit. C skips any victim whose level is not below the caster's.
func forceTargets(s *Session, sameRoom bool) []*Session {
	s.manager.mu.RLock()
	defer s.manager.mu.RUnlock()

	targets := make([]*Session, 0, len(s.manager.sessions))
	for _, target := range s.manager.sessions {
		if target.player == nil || target.player.Level >= s.player.Level {
			continue
		}
		if sameRoom && target.player.GetRoom() != s.player.GetRoom() {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

// forceSessionCommand mirrors command_interpreter(vict, to_force). The C
// helper is called directly, so the victim's aliases are not expanded.
func forceSessionCommand(caster, target *Session, command string, notifyVictim bool) {
	if notifyVictim {
		target.Send(fmt.Sprintf("%s has forced you to '%s'.\r\n", caster.player.Name, command))
	}
	cmd, args := splitCommandInput(command)
	if cmd == "" {
		return
	}
	if err := executeCommand(target, cmd, args, false); err != nil {
		// command_interpreter is void in C; errors are diagnostic only and must
		// not create a second player-facing error response.
		slog.Error("forced command failed", "target", target.player.Name, "command", command, "error", err)
	}
	slog.Info("forced", "target", target.player.Name, "command", command, "by", caster.player.Name)
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
