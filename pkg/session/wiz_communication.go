package session

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdGecho(s *Session, args []string) error {
	return cmdGechoText(s, strings.Join(args, " "))
}

func cmdGechoText(s *Session, msg string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if msg == "" {
		s.Send("That must be a mistake...\r\n")
		return nil
	}
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player != nil && sess.player != s.player {
			sess.Send(msg)
		}
	}
	s.manager.mu.RUnlock()
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.Send("Okay.\r\n")
	} else {
		s.Send(msg)
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
	return cmdSendText(s, args, "")
}

// cmdSendText preserves the un-tokenized argument remainder used by C's
// half_chop (src/interpreter.c:1372-1379). The target is the first token, and
// the message is the remaining text after skip_spaces, including internal and
// trailing whitespace.
func cmdSendText(s *Session, args []string, rawArgs string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?\r\n")
		return nil
	}
	targetName, msg, hasTarget := splitSendArgs(args, rawArgs)
	if !hasTarget {
		s.Send("Send what to who?\r\n")
		return nil
	}

	target, ok := s.manager.world.ResolveCharWorld(s.player, targetName)
	if !ok {
		s.Send("No-one by that name here.\r\n")
		return nil
	}

	// C's send_to_char is a no-op for NPCs. Player.SendMessage routes through
	// the manager's descriptor sink, so linkless players likewise receive no
	// bytes while the actor still gets the confirmation below.
	if target.Player != nil {
		target.Player.SendMessage(msg + "\r\n")
	}

	targetDisplayName := target.Combatant.GetName()
	slog.Warn("wizard send", "by", s.player.Name, "target", targetDisplayName, "message", msg)
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.Send("Sent.\r\n")
	} else {
		s.Send(fmt.Sprintf("You send '%s' to %s.\r\n", msg, targetDisplayName))
	}
	return nil
}

func splitSendArgs(args []string, rawArgs string) (target, msg string, ok bool) {
	if rawArgs != "" {
		remainder := strings.TrimLeft(rawArgs, cCommandWhitespace)
		if remainder == "" {
			return "", "", false
		}
		idx := strings.IndexAny(remainder, cCommandWhitespace)
		if idx < 0 {
			return remainder, "", true
		}
		return remainder[:idx], strings.TrimLeft(remainder[idx+1:], cCommandWhitespace), true
	}
	if len(args) == 0 {
		return "", "", false
	}
	return args[0], strings.Join(args[1:], " "), true
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
	return cmdWiznetText(s, strings.Join(args, " "))
}

// cmdWiznetText ports do_wiznet() (src/act.wizard.c:1912-2034). The C
// handler consumes the original argument remainder, so the transport path
// supplies rawArgs to preserve internal and trailing spaces.
func cmdWiznetText(s *Session, rawArgs string) error {
	if s.player.GetLevel() < LVL_IMMORT && !s.player.HasPLRFlag(game.PlrChosen) {
		s.Send("Huh?!?")
		return nil
	}

	argument := strings.TrimLeft(rawArgs, cCommandWhitespace)
	argument = collapseDoubledDollar(argument)
	if argument == "" {
		s.Send("Usage: wiznet <text> | #<level> <text> | *<emotetext> |\r\n " +
			"       wiznet @\r\n")
		return nil
	}

	emote := false
	noChosen := false
	level := LVL_IMMORT
	switch argument[0] {
	case '*':
		emote = true
		fallthrough
	case '#':
		// C uses one_argument to decide whether the prefix is numeric, then
		// half_chop to consume the actual first token. Those parsers differ
		// when fill words occur, so preserve both call sites here.
		prefix, _ := game.OneArgument(argument[1:])
		if cIsNumber(prefix) {
			first, remainder := wiznetHalfChop(argument[1:])
			level = cAtoi(first)
			if level < LVL_IMMORT {
				level = LVL_IMMORT
			}
			noChosen = true
			if level > s.player.GetLevel() {
				s.Send("You can't wizline above your own level.\r\n")
				return nil
			}
			argument = remainder
		} else if emote {
			argument = argument[1:]
		}
	case '@':
		return wiznetList(s)
	case '\\':
		argument = argument[1:]
	}

	if s.player.GetFlags()&(1<<uint(game.PrfNowiz)) != 0 {
		s.Send("You are offline!\r\n")
		return nil
	}

	argument = strings.TrimLeft(argument, cCommandWhitespace)
	if argument == "" {
		s.Send("Don't bother the gods like that!\r\n")
		return nil
	}

	levelPrefix := ""
	if noChosen {
		levelPrefix = fmt.Sprintf("<%d> ", level)
	}
	message := fmt.Sprintf("%s: %s%s%s\r\n", s.player.GetName(), levelPrefix, wiznetEmotePrefix(emote), argument)
	shadow := fmt.Sprintf("Someone: %s%s%s\r\n", levelPrefix, wiznetEmotePrefix(emote), argument)
	norepeat := s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0

	sessions := wiznetSessions(s.manager)
	for _, sess := range sessions {
		if !wiznetRecipientEligible(sess, level, noChosen) || (sess == s && norepeat) {
			continue
		}
		toSend := message
		if sess != s && s.player.GetLevel() != LVL_IMPL && !game.CanSee(sess.player, s.player) {
			toSend = shadow
		}
		sess.Send(toSend)
	}

	if norepeat {
		s.Send("Okay.\r\n")
	}
	return nil
}

func wiznetEmotePrefix(emote bool) string {
	if emote {
		return "<--- "
	}
	return ""
}

func wiznetHalfChop(input string) (first, remainder string) {
	input = strings.TrimLeft(input, cCommandWhitespace)
	if input == "" {
		return "", ""
	}
	end := strings.IndexAny(input, cCommandWhitespace)
	if end < 0 {
		return strings.ToLower(input), ""
	}
	return strings.ToLower(input[:end]), strings.TrimLeft(input[end:], cCommandWhitespace)
}

func wiznetSessions(m *Manager) []*Session {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, sess := range m.sessions {
		sessions = append(sessions, sess)
	}
	m.mu.RUnlock()
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].connectedAt.After(sessions[j].connectedAt)
	})
	return sessions
}

func wiznetRecipientEligible(sess *Session, level int, noChosen bool) bool {
	if sess == nil || !sess.authenticated || sess.player == nil || !sess.hasTransport() {
		return false
	}
	if sess.player.GetLevel() < level && (noChosen || !sess.player.HasPLRFlag(game.PlrChosen)) {
		return false
	}
	flags := sess.player.GetFlags()
	if flags&(1<<uint(game.PrfNowiz)) != 0 {
		return false
	}
	return !sess.player.HasPLRFlag(game.PlrMailing) || !sess.player.HasPLRFlag(game.PlrWriting)
}

func wiznetList(s *Session) error {
	var online, offline strings.Builder
	anyOnline := false
	anyOffline := false
	for _, sess := range wiznetSessions(s.manager) {
		if sess == nil || !sess.authenticated || sess.player == nil || sess.player.GetLevel() < LVL_IMMORT {
			continue
		}
		if sess.player.GetFlags()&(1<<uint(game.PrfNowiz)) != 0 {
			continue
		}
		if s.player.GetLevel() != LVL_IMPL && !game.CanSee(s.player, sess.player) {
			continue
		}
		if sess.hasTransport() {
			if !anyOnline {
				online.WriteString("Gods online:\r\n")
				anyOnline = true
			}
			fmt.Fprintf(&online, "  %s", sess.player.GetName())
			if sess.player.HasPLRFlag(game.PlrWriting) {
				online.WriteString(" (Writing)\r\n")
			} else if sess.player.HasPLRFlag(game.PlrMailing) {
				online.WriteString(" (Writing mail)\r\n")
			} else {
				online.WriteString("\r\n")
			}
		} else {
			if !anyOffline {
				offline.WriteString("Gods offline:\r\n")
				anyOffline = true
			}
			fmt.Fprintf(&offline, "  %s\r\n", sess.player.GetName())
		}
	}
	if anyOnline {
		s.Send(online.String())
	}
	if anyOffline {
		s.Send(offline.String())
	}
	return nil
}

// cmdZreset — reset a zone by VNum (LVL_GOD)
// Original: act.wizard.c do_zreset() — reset_zone() is async via spawner
