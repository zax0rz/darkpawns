package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"

	"github.com/zax0rz/darkpawns/pkg/common"
)

func cmdGecho(s *Session, args []string) error {
	return cmdGechoText(s, strings.Join(args, " "))
}

func cmdGechoText(s *Session, msg string) error {
	if msg == "" {
		s.Send("That must be a mistake...\r\n")
		return nil
	}
	for _, sess := range s.manager.playingSessionsInDescriptorOrder() {
		if sess.player != s.player {
			sess.Send(msg)
		}
	}
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
	if len(args) == 0 {
		s.Send("Yes.. but what?")
		return nil
	}
	msg := strings.Join(args, " ")
	if len(msg) > 500 {
		s.Send("Maximum echo length is 500 characters.")
		return nil
	}
	// C do_echo (src/act.wizard.c:147-151): the room copy is
	// act(buf, FALSE, ch, 0, 0, TO_ROOM) — sleepers get nothing — then
	// NOREPEAT gets OK (config.c:92) and everyone else gets the act TO_CHAR
	// copy. C doubles every $ in user input before queuing
	// (src/comm.c:1968-1977), so a typed "$n" reaches act as "$$n" and prints
	// literally; Go has no input doubling, so the argument is escaped here.
	msg = strings.ReplaceAll(msg, "$", "$$")
	game.Act(s.manager.world, false, s.player, nil, nil, nil, msg, "", game.ToRoom)
	slog.Warn("wizard echo", "by", s.player.Name, "room", s.player.RoomVNum, "message", msg)
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.Send("Okay.\r\n")
	} else {
		game.Act(s.manager.world, false, s.player, nil, nil, nil, msg, "", game.ToChar)
	}
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
	return cmdForceText(s, strings.Join(args, " "))
}

// cmdForceText retains half_chop's command remainder for the producer and
// nested producer paths (src/act.wizard.c:1863-1880), including internal/trailing spaces.
func cmdForceText(s *Session, argument string) error {
	targetName, forceCmd := wiznetHalfChop(argument)
	if targetName == "" || forceCmd == "" {
		s.Send("Whom do you wish to force do what?\r\n")
		return nil
	}

	// C treats "room" and "all" specially only for LVL_GRGOD and above.
	// Below that level they fall through to ordinary get_char_vis lookup.
	if s.player.GetLevel() >= LVL_GRGOD && strings.EqualFold(targetName, "room") {
		s.Send("Okay.\r\n")
		// src/act.wizard.c:1884-1885: before visiting any room target.
		game.MudLog(fmt.Sprintf("(GC) %s forced room %d to %s", s.player.Name, s.player.GetRoom(), forceCmd), game.MudlogNormal, max(s.player.GetLevel()+1, s.player.GetInvisLevel()), true)
		forceRoom(s, forceCmd)
		return nil
	}
	if s.player.GetLevel() >= LVL_GRGOD && strings.EqualFold(targetName, "all") {
		forceAll(s, forceCmd)
		return nil
	}

	// C get_char_vis checks the room first and then the global character list.
	// ResolveCharWorld retains that target resolution for both players and
	// mobiles; only a player needs a direct session lookup.
	resolved, found := s.manager.world.ResolveCharWorld(s.player, targetName)
	body := resolved.Combatant
	if !found || body == nil {
		s.Send(noPersonHere)
		return nil
	}
	var targetSession *Session
	if resolved.Player != nil {
		targetSession = findForceSession(s, targetName)
		if targetSession == nil {
			s.Send(noPersonHere)
			return nil
		}
	} else if resolved.Mob != nil {
		targetSession = s.manager.combatSession(body)
	} else {
		s.Send(noPersonHere)
		return nil
	}

	if body.GetLevel() >= s.player.GetLevel() {
		s.Send("No, no, no!\r\n")
		return nil
	}

	s.Send("Okay.\r\n")
	// src/act.wizard.c:1875-1880: notify, log, then interpret.
	if s.player.GetLevel() < LVL_IMPL {
		if resolved.Player != nil {
			game.Act(s.manager.world, true, s.player, resolved.Player, nil, nil, fmt.Sprintf("$n has forced you to '%s'.", forceCmd), "", game.ToVict)
		} else if targetActor, ok := body.(game.Actor); ok {
			forceMobNotice(s, targetActor, targetSession, forceCmd)
		}
	}
	game.MudLog(fmt.Sprintf("(GC) %s forced %s to %s", s.player.Name, body.GetName(), forceCmd), game.MudlogNormal, max(s.player.GetLevel()+1, s.player.GetInvisLevel()), true)
	if resolved.Player != nil {
		forceSessionCommand(s, targetSession, forceCmd, false)
	} else {
		forceMobCommand(s, body, targetSession, forceCmd)
	}
	return nil
}

// forceMobNotice is do_force's act(buf1, TRUE, ch, NULL, vict, TO_VICT) for a
// mobile, reaching a switched descriptor when one is attached
// (src/act.wizard.c:1875,1890).
func forceMobNotice(s *Session, body game.Actor, targetSession *Session, forceCmd string) {
	var privateSink func(string)
	if targetSession != nil {
		privateSink = func(text string) { targetSession.Send(text) }
	}
	game.ActWithPrivateSink(s.manager.world, true, s.player, body, nil, nil, fmt.Sprintf("$n has forced you to '%s'.", forceCmd), "", game.ToVict, body, privateSink)
}

// forceMobCommand is do_force's command_interpreter(vict, to_force) for a
// mobile (src/act.wizard.c:1880,1891).
func forceMobCommand(s *Session, body combat.Combatant, targetSession *Session, forceCmd string) {
	if result := s.manager.executeCharacterCommand(body, targetSession, forceCmd); result == game.CharacterCommandUnsupported {
		slog.Warn("forced character command unsupported", "target", body.GetName(), "command", forceCmd)
	}
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
// would visit in C's exact order: world[room].people (most recently arrived
// first) for "room", and descriptor_list (newest connection first, descending
// connectionNumber) for "all". C skips any victim whose level is not below the
// caster's.
func forceTargets(s *Session, sameRoom bool) []*Session {
	if sameRoom {
		players := s.manager.world.GetPlayersInRoom(s.player.GetRoom())
		s.manager.mu.RLock()
		byPlayer := make(map[*game.Player]*Session, len(s.manager.sessions))
		for _, target := range s.manager.sessions {
			if target.player != nil {
				byPlayer[target.player] = target
			}
		}
		s.manager.mu.RUnlock()

		targets := make([]*Session, 0, len(players))
		for _, p := range players {
			if p == nil || p.Level >= s.player.Level {
				continue
			}
			if target, ok := byPlayer[p]; ok {
				targets = append(targets, target)
			}
		}
		return targets
	}

	sessions := s.manager.playingSessionsInDescriptorOrder()
	targets := make([]*Session, 0, len(sessions))
	for _, target := range sessions {
		if target.player == nil || target.player.Level >= s.player.Level {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

// forceRoom runs do_force's "room" loop over every lower-level body in the
// room (players and mobiles) in C's world[ch->in_room].people order.
func forceRoom(s *Session, forceCmd string) {
	chars := s.manager.world.GetAllCharsInRoom(s.player.GetRoom())
	s.manager.mu.RLock()
	byPlayer := make(map[*game.Player]*Session, len(s.manager.sessions))
	for _, target := range s.manager.sessions {
		if target.player != nil {
			byPlayer[target.player] = target
		}
	}
	s.manager.mu.RUnlock()

	for _, c := range chars {
		switch body := c.(type) {
		case *game.Player:
			if body == nil || body.GetLevel() >= s.player.GetLevel() {
				continue
			}
			if target, ok := byPlayer[body]; ok {
				forceSessionCommand(s, target, forceCmd, true)
			}
		case *game.MobInstance:
			if body == nil || body.GetLevel() >= s.player.GetLevel() {
				continue
			}
			targetSession := s.manager.combatSession(body)
			forceMobNotice(s, body, targetSession, forceCmd)
			forceMobCommand(s, body, targetSession, forceCmd)
		}
	}
}

// forceAll is do_force's "all" branch (src/act.wizard.c:1894-1906): "Okay.",
// the mudlog, then the notice and command_interpreter() for each lower-level
// playing descriptor. do_shutdown reaches it as do_force(ch, "all save")
// (src/act.wizard.c:1113).
func forceAll(s *Session, forceCmd string) {
	s.Send("Okay.\r\n")
	// src/act.wizard.c:1896-1897: before visiting any descriptor.
	game.MudLog(fmt.Sprintf("(GC) %s forced all to %s", s.player.Name, forceCmd), game.MudlogNormal, max(s.player.GetLevel()+1, s.player.GetInvisLevel()), true)
	for _, target := range forceTargets(s, false) {
		forceSessionCommand(s, target, forceCmd, true)
	}
}

// forceSessionCommand mirrors command_interpreter(vict, to_force). The C
// helper is called directly, so the victim's aliases are not expanded.
func forceSessionCommand(caster, target *Session, command string, notifyVictim bool) {
	if notifyVictim {
		game.Act(caster.manager.world, true, caster.player, target.player, nil, nil, fmt.Sprintf("$n has forced you to '%s'.", command), "", game.ToVict)
	}
	cmd, args := splitCommandInput(command)
	if cmd == "" {
		return
	}
	_, rawArgs := wiznetHalfChop(command)
	if err := executeCommandRaw(target, cmd, args, false, rawArgs); err != nil {
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
		// C sends the gate reply with no terminator (src/act.wizard.c:1924):
		// the prompt frame supplies the only line break, so these bytes ride
		// the literal path — the text event's line-ending append would print
		// a blank line C never sends (PF-063 cluster 4).
		s.sendLiteralText("Huh?!?")
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
		if game.CIsNumber(prefix) {
			first, remainder := wiznetHalfChop(argument[1:])
			level = common.AtoiC(first)
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
	return m.sessionsInDescriptorOrder()
}

func wiznetRecipientEligible(sess *Session, level int, noChosen bool) bool {
	// !d->connected (act.wizard.c:2010): a god at the menu hears nothing.
	if sess == nil || !sess.IsPlaying() || sess.player == nil || !sess.hasTransport() {
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

// wiznetList is do_wiznet's '@' case (act.wizard.c:1948-1980). Both loops
// walk playing descriptors (!d->connected), so a linkless god or one at the
// menu is in neither list. The first lists level-qualified gods without
// NOWIZ, letting an Implementor see past CAN_SEE; the second lists the gods
// who set NOWIZ, with no Implementor bypass.
func wiznetList(s *Session) error {
	var out strings.Builder
	playing := func(sess *Session) bool {
		return sess != nil && sess.player != nil && sess.hasTransport() && sess.IsPlaying() &&
			sess.player.GetLevel() >= LVL_IMMORT
	}
	nowiz := func(sess *Session) bool {
		return sess.player.GetFlags()&(1<<uint(game.PrfNowiz)) != 0
	}
	sessions := wiznetSessions(s.manager)
	any := false
	for _, sess := range sessions {
		if !playing(sess) || nowiz(sess) ||
			(!game.CanSee(s.player, sess.player) && s.player.GetLevel() != LVL_IMPL) {
			continue
		}
		if !any {
			out.WriteString("Gods online:\r\n")
			any = true
		}
		fmt.Fprintf(&out, "  %s", sess.player.GetName())
		if sess.player.HasPLRFlag(game.PlrWriting) {
			out.WriteString(" (Writing)\r\n")
		} else if sess.player.HasPLRFlag(game.PlrMailing) {
			out.WriteString(" (Writing mail)\r\n")
		} else {
			out.WriteString("\r\n")
		}
	}
	any = false
	for _, sess := range sessions {
		if !playing(sess) || !nowiz(sess) || !game.CanSee(s.player, sess.player) {
			continue
		}
		if !any {
			out.WriteString("Gods offline:\r\n")
			any = true
		}
		fmt.Fprintf(&out, "  %s\r\n", sess.player.GetName())
	}
	if out.Len() > 0 {
		s.Send(out.String())
	}
	return nil
}

// cmdZreset — reset a zone by VNum (LVL_GOD)
// Original: act.wizard.c do_zreset() — reset_zone() is async via spawner
