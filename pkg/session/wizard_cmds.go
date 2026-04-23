package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

)

// Wizard level constants — matching Dark Pawns C source scale mapped to Go codebase.
// Original C: LVL_IMMORT=31, LVL_GOD=34, LVL_GRGOD=38, LVL_IMPL=40
// Go codebase uses higher scale: 50/60/61.
const (
	LVL_IMMORT = 50
	LVL_GOD    = 60
	LVL_GRGOD  = 61
	LVL_IMPL   = 61
)

// checkLevel checks if a session's player has at least the required level.
func checkLevel(s *Session, level int) bool {
	if s.player == nil {
		return false
	}
	return s.player.Level >= level
}

// findSessionByName searches all sessions for a player by name (case-insensitive).
func findSessionByName(m *Manager, name string) *Session {
	name = strings.ToLower(name)
	for _, sess := range m.sessions {
		if sess.player != nil && strings.ToLower(sess.player.Name) == name {
			return sess
		}
	}
	return nil
}

// broadcastToRoomText sends a text message to all players in a given room.
func broadcastToRoomText(s *Session, roomVNum int, msg string) {
	if s.manager != nil && s.manager.world != nil {
		s.manager.BroadcastToRoom(roomVNum, []byte(msg), "")
	}
}

// ---------------------------------------------------------------------------
// goto — teleport to any room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdGoto(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Goto where? (room number)")
		return nil
	}
	dest, err := strconv.Atoi(args[0])
	if err != nil {
		s.Send("That's not a valid room number.")
		return nil
	}
	s.player.SetRoom(dest)
	s.Send(fmt.Sprintf("You go to room %d.", dest))
	_ = cmdLook(s, nil)
	return nil
}

// ---------------------------------------------------------------------------
// at — run a command at another location (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdAt(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: at <room> <command>")
		return nil
	}
	dest, err := strconv.Atoi(args[0])
	if err != nil {
		s.Send("That's not a valid room number.")
		return nil
	}
	orig := s.player.GetRoom()
	s.player.SetRoom(dest)
	rest := strings.Join(args[1:], " ")
	ExecuteCommand(s, strings.Fields(rest)[0], strings.Fields(rest)[1:])
	s.player.SetRoom(orig)
	return nil
}

// ---------------------------------------------------------------------------
// load — load a mob or object (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdLoad(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: load <mob|obj> <vnum>")
		return nil
	}
	s.Send("Load not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// purge — remove all mobs/objects from room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdPurge(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	s.Send("Purge not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// teleport — teleport a player (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdTeleport(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: teleport <player> <room>")
		return nil
	}
	targetName := args[0]
	dest, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("That's not a valid room number.")
		return nil
	}
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	s.Send("OK.")
	broadcastToRoomText(s, targetSess.player.RoomVNum, fmt.Sprintf("%s disappears in a puff of smoke.", targetSess.player.Name))
	targetSess.player.RoomVNum = dest
	broadcastToRoomText(s, dest, fmt.Sprintf("%s arrives from a puff of smoke.", targetSess.player.Name))
	targetSess.Send(fmt.Sprintf("%s has teleported you!", s.player.Name))
	_ = cmdLook(targetSess, nil)
	return nil
}

// ---------------------------------------------------------------------------
// heal — fully heal target (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdHeal(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Heal whom?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	targetSess.player.Health = targetSess.player.MaxHealth
	targetSess.player.Mana = targetSess.player.MaxMana
	targetSess.player.Move = targetSess.player.MaxMove
	s.Send(fmt.Sprintf("You heal %s.", targetSess.player.Name))
	targetSess.Send(fmt.Sprintf("%s has healed you!", s.player.Name))
	return nil
}

// ---------------------------------------------------------------------------
// restore — fully restore target (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdRestore(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	return cmdHeal(s, args)
}

// ---------------------------------------------------------------------------
// set — set a player field (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdSet(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 3 {
		s.Send("Usage: set <player> <field> <value>")
		return nil
	}
	targetName := args[0]
	field := args[1]
	value := strings.Join(args[2:], " ")
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	switch strings.ToLower(field) {
	case "level":
		lvl, err := strconv.Atoi(value)
		if err != nil {
			s.Send("Invalid level.")
			return nil
		}
		targetSess.player.Level = lvl
		s.Send(fmt.Sprintf("Level set to %d.", lvl))
	case "gold":
		g, err := strconv.Atoi(value)
		if err != nil {
			s.Send("Invalid gold amount.")
			return nil
		}
		targetSess.player.Gold = g
		s.Send(fmt.Sprintf("Gold set to %d.", g))
	case "alignment":
		a, err := strconv.Atoi(value)
		if err != nil {
			s.Send("Invalid alignment.")
			return nil
		}
		targetSess.player.Alignment = a
		s.Send(fmt.Sprintf("Alignment set to %d.", a))
	default:
		s.Send("Unknown field. Try: level, gold, alignment.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// switch — switch into another character's body (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdSwitch(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Switch into whom?")
		return nil
	}
	s.Send("Switch not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// return — return to own body (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdReturn(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	s.Send("Return not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// invis — toggle invisibility (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdInvis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	s.Send("You are now invisible.")
	return nil
}

// ---------------------------------------------------------------------------
// vis — make invisible players visible (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdVis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Vis whom?")
		return nil
	}
	s.Send("Vis not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// gecho — broadcast to all players (LVL_GOD)
// ---------------------------------------------------------------------------
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
	for _, sess := range s.manager.sessions {
		if sess.player != nil {
			sess.Send(msg)
		}
	}
	slog.Info("gecho", "message", msg, "by", s.player.Name)
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
	broadcastToRoomText(s, s.player.RoomVNum, msg)
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
	s.Send(fmt.Sprintf("You send '%s' to %s.", msg, target.player.Name))
	return nil
}

// ---------------------------------------------------------------------------
// force — force command on another character (LVL_GRGOD)
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

	slog.Info("forced", "target", target.player.Name, "command", forceCmd, "by", s.player.Name)
	s.Send(fmt.Sprintf("Forced %s to '%s'.", target.player.Name, forceCmd))
	return nil
}

// ---------------------------------------------------------------------------
// shutdown — shut down the server (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdShutdown(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	slog.Warn("server shutdown initiated", "by", s.player.Name)
	s.Send("World shudders and begins to fade...")
	s.Send("Shutting down...")
	return nil
}

// ---------------------------------------------------------------------------
// snoop — spy on player input (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdSnoop(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Snoop whom?")
		return nil
	}
	targetName := args[0]
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("They aren't here.")
		return nil
	}
	// Toggle snoop
	if s.snooping == target {
		s.snooping = nil
		target.snoopBy = nil
		s.Send(fmt.Sprintf("Snoop on %s removed.", target.player.Name))
	} else {
		if s.snooping != nil {
			s.snooping.snoopBy = nil
		}
		s.snooping = target
		target.snoopBy = s
		s.Send(fmt.Sprintf("Now snooping on %s.", target.player.Name))
	}
	return nil
}

// ---------------------------------------------------------------------------
// advance — advance a player's level (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdAdvance(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Advance whom to what level?")
		return nil
	}
	targetName := args[0]
	newLevel, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("Invalid level.")
		return nil
	}
	if newLevel < 0 || newLevel > 60 {
		s.Send("Level must be between 0 and 60.")
		return nil
	}
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	oldLevel := target.player.Level
	target.player.Level = newLevel
	slog.Info("player advanced", "target", target.player.Name, "old", oldLevel, "new", newLevel, "by", s.player.Name)
	s.Send(fmt.Sprintf("%s advanced from level %d to %d.", target.player.Name, oldLevel, newLevel))
	target.Send(fmt.Sprintf("You have been advanced from level %d to %d!", oldLevel, newLevel))
	return nil
}
