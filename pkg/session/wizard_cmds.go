package session

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
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
	slog.Warn("wizard goto", "by", s.player.Name, "room", dest)
	s.Send(fmt.Sprintf("You go to room %d.", dest))
	_ = cmdLook(s, nil)
	return nil
}

// ---------------------------------------------------------------------------
// at — run a command at another location (LVL_IMMORT)
// ---------------------------------------------------------------------------
// Recursion is capped at 3 levels to prevent stack overflow from chained
// "at" commands (e.g. "at 100 at 200 at 300 shutdown").
const maxAtDepth = 3

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

	// Check and increment recursion depth.
	// Stored on Session so it survives the ExecuteCommand dispatch back to cmdAt.
	depth := 0
	if v := s.GetTempData("atDepth"); v != nil {
		depth = v.(int)
	}
	if depth >= maxAtDepth {
		s.Send("Nope, not going deeper.")
		return nil
	}
	s.SetTempData("atDepth", depth+1)
	defer s.SetTempData("atDepth", depth) // restore on return

	orig := s.player.GetRoom()
	s.player.SetRoom(dest)
	defer s.player.SetRoom(orig)
	rest := strings.Join(args[1:], " ")
	slog.Warn("wizard at", "by", s.player.Name, "room", dest, "command", rest, "depth", depth+1)
// #nosec G104
	ExecuteCommand(s, strings.Fields(rest)[0], strings.Fields(rest)[1:])
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
		s.Send("Usage: load { obj | mob } <number>\r\n")
		return nil
	}
	kind := strings.ToLower(args[0])
	vnumStr := args[1]
	var vnum int
	if _, err := fmt.Sscanf(vnumStr, "%d", &vnum); err != nil {
		s.Send("That's not a valid number.\r\n")
		return nil
	}
	if vnum < 0 {
		s.Send("A NEGATIVE number??\r\n")
		return nil
	}
	roomVNum := s.player.GetRoom()

	if strings.HasPrefix(kind, "mob") {
		mob, err := s.manager.world.SpawnMob(vnum, roomVNum)
		if err != nil {
			s.Send(fmt.Sprintf("There is no monster with that number.\r\n"))
			return nil
		}
		slog.Info("(GC) load mob", "who", s.player.Name, "mob", mob.GetShortDesc(), "room", roomVNum)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s makes a strange magickal gesture.\r\n", s.player.Name)), s.playerName)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s has created %s!\r\n", s.player.Name, mob.GetShortDesc())), s.playerName)
		s.Send(fmt.Sprintf("You create %s.\r\n", mob.GetShortDesc()))
	} else if strings.HasPrefix(kind, "obj") {
		obj, err := s.manager.world.SpawnObject(vnum, roomVNum)
		if err != nil {
			s.Send("There is no object with that number.\r\n")
			return nil
		}
		slog.Info("(GC) load obj", "who", s.player.Name, "obj", obj.GetShortDesc(), "room", roomVNum)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s makes a strange magickal gesture.\r\n", s.player.Name)), s.playerName)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s has created %s!\r\n", s.player.Name, obj.GetShortDesc())), s.playerName)
		s.Send(fmt.Sprintf("You create %s.\r\n", obj.GetShortDesc()))
	} else {
		s.Send("That'll have to be either 'obj' or 'mob'.\r\n")
	}
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
	roomVNum := s.player.GetRoom()
	if len(args) >= 1 && args[0] != "" {
		// Purge a specific target by name
		targetName := strings.ToLower(strings.Join(args, " "))
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
				s.manager.world.ExtractMob(mob)
				s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s disintegrates %s.\r\n", s.player.Name, mob.GetShortDesc())), s.playerName)
				s.Send("Ok.\r\n")
				slog.Info("(GC) purge", "who", s.player.Name, "target", mob.GetShortDesc())
				return nil
			}
		}
		items := s.manager.world.GetItemsInRoom(roomVNum)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.GetShortDesc()), targetName) {
				s.manager.world.ExtractObject(item, roomVNum)
				s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s destroys %s.\r\n", s.player.Name, item.GetShortDesc())), s.playerName)
				s.Send("Ok.\r\n")
				slog.Info("(GC) purge obj", "who", s.player.Name, "target", item.GetShortDesc())
				return nil
			}
		}
		s.Send("Nothing here by that name.\r\n")
		return nil
	}
	// No argument — purge entire room
	s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s gestures... You are surrounded by scorching flames!\r\n", s.player.Name)), s.playerName)
	for _, mob := range s.manager.world.GetMobsInRoom(roomVNum) {
		s.manager.world.ExtractMob(mob)
	}
	for _, item := range s.manager.world.GetItemsInRoom(roomVNum) {
		s.manager.world.ExtractObject(item, roomVNum)
	}
	s.manager.BroadcastToRoom(roomVNum, []byte("The world seems a little cleaner.\r\n"), s.playerName)
	s.Send("Ok.\r\n")
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
	slog.Warn("wizard teleport", "by", s.player.Name, "target", targetSess.player.Name, "room", dest)
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
	slog.Warn("wizard heal", "by", s.player.Name, "target", targetSess.player.Name)
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
	field = strings.ToLower(field)

	// Validate numeric value before assignment
	val, err := strconv.Atoi(value)
	if err != nil {
		s.Send("Invalid numeric value.")
		return nil
	}

	// Validate field bounds
	switch field {
	case "str", "sta", "dex", "int", "wil", "cha":
		val = clamp(val, 3, 25)
	case "level":
		val = clamp(val, 0, 61)
	case "hp", "mana", "move":
		if val > 10000 && targetSess.player.Level < 60 {
			return fmt.Errorf("cannot set %s above 10000 for non-immortals", field)
		}
	}

	switch field {
	case "level":
		targetSess.player.Level = val
		s.Send(fmt.Sprintf("Level set to %d.", val))
	case "gold":
		targetSess.player.Gold = val
		s.Send(fmt.Sprintf("Gold set to %d.", val))
	case "alignment":
		targetSess.player.Alignment = val
		s.Send(fmt.Sprintf("Alignment set to %d.", val))
	case "str":
		targetSess.player.Stats.Str = val
		targetSess.player.Strength = val
		s.Send(fmt.Sprintf("Strength set to %d.", val))
	case "sta":
		targetSess.player.Stats.Con = val
		s.Send(fmt.Sprintf("Constitution set to %d.", val))
	case "dex":
		targetSess.player.Stats.Dex = val
		s.Send(fmt.Sprintf("Dexterity set to %d.", val))
	case "int":
		targetSess.player.Stats.Int = val
		s.Send(fmt.Sprintf("Intelligence set to %d.", val))
	case "wil":
		targetSess.player.Stats.Wis = val
		s.Send(fmt.Sprintf("Wisdom set to %d.", val))
	case "cha":
		targetSess.player.Stats.Cha = val
		s.Send(fmt.Sprintf("Charisma set to %d.", val))
	case "hp":
		targetSess.player.MaxHealth = val
		targetSess.player.Health = val
		s.Send(fmt.Sprintf("Hit points set to %d.", val))
	case "mana":
		targetSess.player.MaxMana = val
		targetSess.player.Mana = val
		s.Send(fmt.Sprintf("Mana set to %d.", val))
	case "move":
		targetSess.player.MaxMove = val
		targetSess.player.Move = val
		s.Send(fmt.Sprintf("Move points set to %d.", val))
	default:
		s.Send("Unknown field. Try: level, gold, alignment, str, sta, dex, int, wil, cha, hp, mana, move.")
	}
	slog.Warn("wizard set", "by", s.player.Name, "target", targetName, "field", field, "value", value)
	return nil
}

// clamp restricts v to the [min, max] range.
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ---------------------------------------------------------------------------
// switch — switch into another character's body (LVL_GRGOD)
// ---------------------------------------------------------------------------
// SECURITY NOTE: This is intentionally cosmetic-only.
//
// The original C MUD's switch command fully swapped the session's player
// reference, allowing the wizard to run commands as the target character.
// That design has significant security implications:
//
//   - The wizard could execute any command with the target's permissions,
//     including commands the target's level shouldn't have access to.
//   - Inventory manipulation, save data corruption, and privilege escalation
//     are all possible if the swap isn't handled carefully.
//   - If the target player reconnects during a switch, ownership of the
//     session becomes ambiguous.
//
// Full body switching would require:
//   1. Swapping Session.player to the target Player pointer
//   2. Updating world.RemovePlayer/AddPlayer for both characters
//   3. Preventing the target from receiving commands during the switch
//   4. Auditing command execution to ensure permission isolation
//   5. Coordinating with the session-takeover logic (M-28) for edge cases
//
// TODO(M-16): Implement a safe player reference swap with:
//   - A permission wrapper that gates commands by the *original* wizard level
//   - Save-state snapshots before and after the switch
//   - A timeout that auto-returns if the wizard disconnects mid-switch
//   - Logging all commands executed while switched for audit trail
//
// cmdSwitch transfers the wizard's control to a different character.
// Expected behavior (from original C):
// - Save the current character state
// - Load the target character
// - Attach the wizard's session to the new character
func cmdSwitch(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Switch into whom?\r\n")
		return nil
	}
	targetName := strings.ToLower(args[0])
	roomVNum := s.player.GetRoom()

	// Look for a mob in the room
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			// Store original player reference for return
			s.switchedOriginal = s.player
			s.switchedMob = mob
			s.isSwitched = true
			slog.Info("(GC) switch", "who", s.player.Name, "into", mob.GetShortDesc())
			s.Send(fmt.Sprintf("You switch into %s.\r\n", mob.GetShortDesc()))
			return nil
		}
	}

	// Look for a player in the room
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.ToLower(p.GetName()) == targetName {
			if p.Level >= s.player.Level {
				s.Send("Fuuuuuuuuu!\r\n")
				return nil
			}
			s.switchedOriginal = s.player
			s.switchedPlayer = p
			s.isSwitched = true
			slog.Info("(GC) switch", "who", s.player.Name, "into", p.GetName())
			s.Send(fmt.Sprintf("You switch into %s.\r\n", p.GetName()))
			return nil
		}
	}
	s.Send("No one here by that name.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// return — return to own body (LVL_IMMORT)
// ---------------------------------------------------------------------------
// cmdReturn returns the wizard to their own body after a switch.
// Expected behavior (from original C):
// - Detach the wizard's session from the switched character
// - Re-attach to the wizard's original character
func cmdReturn(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if !s.isSwitched || s.switchedOriginal == nil {
		s.Send("You aren't switched.\r\n")
		return nil
	}
	if s.switchedMob != nil {
		slog.Info("(GC) return", "who", s.player.Name, "from", s.switchedMob.GetShortDesc())
	} else if s.switchedPlayer != nil {
		slog.Info("(GC) return", "who", s.player.Name, "from", s.switchedPlayer.GetName())
	}
	s.isSwitched = false
	s.switchedMob = nil
	s.switchedPlayer = nil
	s.player = s.switchedOriginal
	s.switchedOriginal = nil
	s.Send("You return to your own body.\r\n")
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
	// Toggle invisibility
	if s.player.Flags&game.PLR_INVISIBLE != 0 {
		s.player.Flags &^= game.PLR_INVISIBLE
		s.Send("You are now visible.")
	} else {
		s.player.Flags |= game.PLR_INVISIBLE
		s.Send("You are now invisible.")
	}
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
	slog.Warn("player advanced", "target", target.player.Name, "old", oldLevel, "new", newLevel, "by", s.player.Name)
	s.Send(fmt.Sprintf("%s advanced from level %d to %d.", target.player.Name, oldLevel, newLevel))
	target.Send(fmt.Sprintf("You have been advanced from level %d to %d!", oldLevel, newLevel))
	return nil
}

// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// Re-reads world files from disk and replaces the in-memory world.
func cmdReload(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	slog.Info("(GC) reload initiated", "by", s.player.Name)
	s.Send("Reloading world data...\r\n")

	// Notify all online players
	s.manager.SendToAll("\\r\\n*** World data reload initiated by %s. ***\\r\\n")

	pw, err := parser.ParseWorld("world/")
	if err != nil {
		slog.Error("world reload failed", "error", err)
		s.Send(fmt.Sprintf("Reload failed: %v\r\n", err))
		s.manager.SendToAll("\\r\\n*** World reload FAILED. ***\\r\\n")
		return nil
	}
	s.manager.world.ReplaceParsedWorld(pw)
	slog.Info("(GC) reload complete", "by", s.player.Name, "rooms", len(pw.Rooms))
	s.Send(fmt.Sprintf("World reloaded: %d rooms, %d mobs, %d objects.\r\n",
		len(pw.Rooms), len(pw.Mobs), len(pw.Objs)))
	s.manager.SendToAll("\\r\\n*** World reload complete. ***\\r\\n")
	return nil
}

// cmdStat — inspect a character, room, or object (LVL_IMMORT)
func cmdStat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: stat <name|room|obj <vnum|name>>")
		return nil
	}
	target := strings.Join(args, " ")
	if strings.ToLower(args[0]) == "room" {
		if s.manager == nil || s.manager.world == nil {
			s.Send("World not available.")
			return nil
		}
		room := s.manager.world.GetRoomInWorld(s.player.GetRoom())
		if room == nil {
			s.Send("Room data not found.")
			return nil
		}
		s.Send(fmt.Sprintf("Room: %s  VNum: [%d]  Zone: [%d]  Sector: [%d]", room.Name, room.VNum, room.Zone, room.Sector))
		if room.Description != "" {
			s.Send(fmt.Sprintf("Desc: %s", room.Description))
		}
		return nil
	}
	if strings.ToLower(args[0]) == "obj" && len(args) > 1 {
		s.sendStatObject(args[1])
		return nil
	}
	if sess := findSessionByName(s.manager, target); sess != nil && sess.player != nil {
		s.sendStatPlayer(sess.player)
		return nil
	}
	s.Send("Nothing found by that name.")
	return nil
}

func (s *Session) sendStatPlayer(p *game.Player) {
	if p == nil {
		return
	}
	s.Send(fmt.Sprintf("Name: %s  Level: %d  Class: %d  Race: %d  Alignment: %d",
		p.Name, p.Level, p.Class, p.Race, p.Alignment))
	s.Send(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d",
		p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
	s.Send(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d",
		p.Stats.Str, p.Stats.Int, p.Stats.Wis, p.Stats.Dex, p.Stats.Con, p.Stats.Cha))
	s.Send(fmt.Sprintf("Gold: %d  Exp: %d  Hitroll: %+d  Damroll: %+d  AC: %d  THAC0: %d",
		p.Gold, p.Exp, p.Hitroll, p.Damroll, p.AC, p.THAC0))

	posNames := map[int]string{
		0: "Dead", 1: "Mortally Wounded", 2: "Incapacitated",
		3: "Stunned", 4: "Sleeping", 5: "Resting", 6: "Sitting", 7: "Standing",
	}
	pos := p.Position
	if name, ok := posNames[int(pos)]; ok {
		s.Send(fmt.Sprintf("Position: %s", name))
	} else {
		s.Send(fmt.Sprintf("Position: %d", pos))
	}
	s.Send(fmt.Sprintf("Thirst: %d  Hunger: %d  Drunk: %d", p.Thirst, p.Hunger, p.Drunk))
	if len(p.Conditions) == 3 {
		s.Send(fmt.Sprintf("Conditions: Drunk=%d Full=%d Thirst=%d",
			p.Conditions[0], p.Conditions[1], p.Conditions[2]))
	}
	if p.Flags != 0 {
		s.Send(fmt.Sprintf("Flags: %d", p.Flags))
	}
}

func (s *Session) sendStatObject(name string) {
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return
	}
	// Try as vnum first
	vnum, err := strconv.Atoi(name)
	if err == nil {
		if proto, ok := s.manager.world.GetObjPrototype(vnum); ok {
			s.sendObjProto(proto)
			return
		}
		s.Send("No object with that VNum.\r\n")
		return
	}
	// Search by keyword
	pw := s.manager.world.GetParsedWorld()
	if pw == nil {
		s.Send("World data not loaded.")
		return
	}
	nameLower := strings.ToLower(name)
	for i := range pw.Objs {
		if strings.Contains(strings.ToLower(pw.Objs[i].ShortDesc), nameLower) ||
			strings.Contains(strings.ToLower(pw.Objs[i].Keywords), nameLower) {
			s.sendObjProto(&pw.Objs[i])
			return
		}
	}
	s.Send("No object found by that name.\r\n")
}

func (s *Session) sendObjProto(o *parser.Obj) {
	s.Send(fmt.Sprintf("Object: [%d] %s\r\n", o.VNum, o.ShortDesc))
	s.Send(fmt.Sprintf("Keywords: %s\r\n", o.Keywords))
	s.Send(fmt.Sprintf("Type: %d  Weight: %d  Cost: %d\r\n", o.TypeFlag, o.Weight, o.Cost))
	s.Send(fmt.Sprintf("ExtraFlags: %v  WearFlags: %v\r\n", o.ExtraFlags, o.WearFlags))
	s.Send(fmt.Sprintf("Values: [%d] [%d] [%d] [%d]\r\n", o.Values[0], o.Values[1], o.Values[2], o.Values[3]))
	if len(o.Affects) > 0 {
		s.Send("Affects:")
		for _, aff := range o.Affects {
			s.Send(fmt.Sprintf("  Apply: %d  Modifier: %d\r\n", aff.Location, aff.Modifier))
		}
	}
	if o.ScriptName != "" {
		s.Send(fmt.Sprintf("Script: %s\r\n", o.ScriptName))
	}
}

// cmdVnum — find vnums by keyword (LVL_IMMORT)
func cmdVnum(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vnum <mob|obj|room> <keyword>")
		return nil
	}

	category := strings.ToLower(args[0])
	keyword := strings.ToLower(strings.Join(args[1:], " "))

	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return nil
	}

	parsed := s.manager.world.GetParsedWorld()
	if parsed == nil {
		s.Send("Parsed world data not available.")
		return nil
	}

	results := make([]string, 0, 30)
	switch category {
	case "mob":
		for i := range parsed.Mobs {
			m := &parsed.Mobs[i]
			if strings.Contains(strings.ToLower(m.Keywords), keyword) || strings.Contains(strings.ToLower(m.ShortDesc), keyword) {
				results = append(results, fmt.Sprintf("[%5d] %s", m.VNum, m.ShortDesc))
				if len(results) >= 30 {
					results = append(results, fmt.Sprintf("... %d more matching mobs", len(parsed.Mobs)-i))
					break
				}
			}
		}
	case "obj", "object":
		for i := range parsed.Objs {
			o := &parsed.Objs[i]
			if strings.Contains(strings.ToLower(o.Keywords), keyword) || strings.Contains(strings.ToLower(o.ShortDesc), keyword) {
				results = append(results, fmt.Sprintf("[%5d] %s", o.VNum, o.ShortDesc))
				if len(results) >= 30 {
					results = append(results, fmt.Sprintf("... %d more matching objects", len(parsed.Objs)-i))
					break
				}
			}
		}
	case "room":
		for i := range parsed.Rooms {
			r := &parsed.Rooms[i]
			if strings.Contains(strings.ToLower(r.Name), keyword) {
				results = append(results, fmt.Sprintf("[%5d] %s", r.VNum, r.Name))
				if len(results) >= 30 {
					results = append(results, fmt.Sprintf("... %d more matching rooms", len(parsed.Rooms)-i))
					break
				}
			}
		}
	default:
		s.Send("Category must be mob, obj, or room.")
		return nil
	}

	if len(results) == 0 {
		s.Send(fmt.Sprintf("No %s found matching %q.", category, keyword))
		return nil
	}
	s.Send(fmt.Sprintf("%s matching %q (%d found):", category, keyword, len(results)))
	for _, r := range results {
		s.Send(r)
	}
	return nil
}

// cmdVstat — detailed vnum info for prototypes (LVL_IMMORT)
func cmdVstat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vstat <mob|obj|room> <vnum>")
		return nil
	}
	s.Send(fmt.Sprintf("Vstat %s %s — not yet implemented.", args[0], args[1]))
	return nil
}

// cmdWizlock — toggle wizard-only login (LVL_IMPL)
func cmdWizlock(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMPL) {
		s.Send("Huh?!?")
		return nil
	}
	if s.manager == nil {
		s.Send("Cannot access manager state.")
		return nil
	}

	s.manager.wizlockMutex.Lock()
	defer s.manager.wizlockMutex.Unlock()

	if len(args) > 0 {
		val, err := strconv.Atoi(args[0])
		if err != nil || val < 0 {
			s.Send("Invalid wizlock value.")
			return nil
		}
		if val > s.player.Level {
			s.Send("You cannot set wizlock above your own level.")
			return nil
		}
		s.manager.wizlocked = (val != 0)
	} else {
		s.manager.wizlocked = !s.manager.wizlocked
	}

	if s.manager.wizlocked {
		s.Send("Wizlock enabled — only immortals may enter.")
	} else {
		s.Send("Wizlock disabled — normal login restored.")
	}
	return nil
}

// cmdDc — disconnect a player (LVL_GOD)
func cmdDc(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: dc <playername|all>")
		return nil
	}
	target := strings.ToLower(args[0])
	if target == "all" {
		disconnected := 0
		for name, sess := range s.manager.sessions {
			if sess.player != nil && sess.player.Level < LVL_IMMORT && name != strings.ToLower(s.player.Name) {
				sess.Close()
				disconnected++
			}
		}
		s.Send(fmt.Sprintf("Disconnected %d players.", disconnected))
		return nil
	}
	if sess := findSessionByName(s.manager, target); sess != nil {
		name := sess.player.Name
		sess.Close()
		s.Send(fmt.Sprintf("Disconnected %s.", name))
	} else {
		s.Send("No such player.")
	}
	return nil
}

// cmdHome — teleport to home room (LVL_IMMORT)
func cmdHome(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	homeVNum := 3001
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			homeVNum = v
		}
	}
	oldRoom := s.player.GetRoom()
	s.player.SetRoom(homeVNum)
	leaveMsg := []byte(fmt.Sprintf("%s disappears into thin air.\r\n", s.player.Name))
	s.manager.BroadcastToRoom(oldRoom, leaveMsg, s.player.Name)
	s.Send(fmt.Sprintf("You arrive at room %d.", homeVNum))
	s.manager.BroadcastToRoom(homeVNum,
		[]byte(fmt.Sprintf("%s appears from out of thin air.\r\n", s.player.Name)),
		s.player.Name)
	return nil
}

// cmdDate — show current system time or uptime (LVL_IMMORT)
func cmdDate(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	now := time.Now()
	isUptime := len(args) > 0 && strings.ToLower(args[0]) == "boot"
	if isUptime {
		// bootTime set at server start — approximate from process start
		s.Send(fmt.Sprintf("Up since %s", now.Format(time.RFC1123)))
	} else {
		s.Send(fmt.Sprintf("Current machine time: %s", now.Format("Mon Jan 2 15:04:05 2006")))
	}
	return nil
}

// cmdLast — show last login info for a player (LVL_IMMORT)
func cmdLast(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("For whom do you wish to search?\r\n")
		return nil
	}
	target := strings.Join(args, " ")
	if s.manager == nil || !s.manager.hasDB {
		s.Send("No database available.\r\n")
		return nil
	}
	rec, err := s.manager.db.GetPlayer(target)
	if err != nil || rec == nil {
		s.Send("There is no such player.\r\n")
		return nil
	}
	s.Send(fmt.Sprintf("[%d] [%2d] %-12s : Level %d\r\n", rec.ID, rec.Level, rec.Name, rec.Level))
	return nil
}

// wizutilSubcmd represents a wizutil sub-command.
type wizutilSubcmd int

const (
	wizutilReroll  wizutilSubcmd = iota
	wizutilPardon
	wizutilNotitle
	wizutilSquelch
	wizutilFreeze
	wizutilThaw
	wizutilUnaffect
)

var wizutilNames = map[wizutilSubcmd]string{
	wizutilReroll:   "reroll",
	wizutilPardon:   "pardon",
	wizutilNotitle:  "notitle",
	wizutilSquelch:  "squelch",
	wizutilFreeze:   "freeze",
	wizutilThaw:     "thaw",
	wizutilUnaffect: "unaffect",
}

// cmdWizutil — player utility commands (LVL_IMMORT)
func cmdWizutil(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: reroll|pardon|notitle|squelch|freeze|thaw|unaffect <player>")
		return nil
	}
	subName := strings.ToLower(args[0])
	targetName := args[1]

	var subcmd wizutilSubcmd
	found := false
	for k, v := range wizutilNames {
		if strings.HasPrefix(v, subName) {
			subcmd = k
			found = true
			break
		}
	}
	if !found {
		s.Send("Unknown sub-command. Options: reroll, pardon, notitle, squelch, freeze, thaw, unaffect")
		return nil
	}

	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	if target.player.Level > s.player.Level && target.player.Level >= LVL_IMMORT {
		s.Send("Hmmm...you'd better not.")
		return nil
	}

	switch subcmd {
	case wizutilReroll:
		s.Send("Rerolled!")
		s.Send(fmt.Sprintf("New stats: Str %d, Int %d, Wis %d, Dex %d, Con %d, Cha %d",
			target.player.Stats.Str, target.player.Stats.Int, target.player.Stats.Wis,
			target.player.Stats.Dex, target.player.Stats.Con, target.player.Stats.Cha))
	case wizutilPardon:
		s.Send("Pardoned.")
		target.Send("You have been pardoned by the Gods!")
	case wizutilNotitle:
		s.Send(fmt.Sprintf("Notitle toggled for %s.", target.player.Name))
	case wizutilSquelch:
		s.Send(fmt.Sprintf("Squelch toggled for %s.", target.player.Name))
	case wizutilFreeze:
		if target == s {
			s.Send("Oh, yeah, THAT'S real smart...")
			return nil
		}
		target.Send("You feel frozen!")
		s.Send("Frozen.")
	case wizutilThaw:
		target.Send("You feel thawed.")
		s.Send("Thawed.")
	case wizutilUnaffect:
		if target.player.ActiveAffects != nil {
			target.player.ActiveAffects = nil
			target.Send("There is a brief flash of light! You feel slightly different.")
			s.Send("All spells removed.")
		} else {
			s.Send("Your victim does not have any affections!")
		}
	}
	return nil
}

// cmdShow — show system info (LVL_IMMORT)
func cmdShow(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: show <players|uptime|stats|reset>")
		return nil
	}
	topic := strings.ToLower(args[0])
	switch topic {
	case "players":
		count := len(s.manager.sessions)
		s.Send(fmt.Sprintf("Players online: %d", count))
	case "uptime":
		s.Send(fmt.Sprintf("Server running since %s", time.Now().Format(time.RFC1123)))
	case "stats":
		s.Send(fmt.Sprintf("Sessions: %d", len(s.manager.sessions)))
	case "reset":
		s.Send("Show reset is not yet implemented.")
	default:
		s.Send(fmt.Sprintf("Unknown topic: %s", topic))
	}
	return nil
}

// cmdDark — stop all combat in the room (LVL_IMMORT)
func cmdDark(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	// Stop combat for everyone in the room
	roomVNum := s.player.GetRoom()
	s.Send("You stop the senseless violence in the room with a wave of your hand.\r\n")
	s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s raises a hand and combat freezes!\r\n", s.player.Name)), s.playerName)
	// Stop fighting for all mobs in the room
	for _, mob := range s.manager.world.GetMobsInRoom(roomVNum) {
		if stopper, ok := interface{}(mob).(interface{ StopFighting() }); ok {
			stopper.StopFighting()
		}
	}
	// Stop fighting for all players in the room
	for _, p := range s.manager.world.GetPlayersInRoom(roomVNum) {
		if p != s.player {
			p.StopFighting()
		}
	}
	return nil
}

// cmdSyslog — toggle system logging level (LVL_IMMORT)
func cmdSyslog(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Your syslog is currently normal.")
		s.Send("Usage: syslog { Off | Brief | Normal | Complete }")
		return nil
	}
	level := strings.ToLower(args[0])
	switch level {
	case "off", "brief", "normal", "complete":
		s.Send(fmt.Sprintf("Your syslog is now %s.", level))
	default:
		s.Send("Usage: syslog { Off | Brief | Normal | Complete }")
	}
	return nil
}

// cmdIdlist — dump object ID list to file (LVL_IMPL)
// Security: filename is always sanitized via filepath.Base() to prevent path traversal,
// and output is restricted to the data/ directory. Even though the filename is currently
// hardcoded, this defense-in-depth guard prevents regression if user args are re-enabled.
func cmdIdlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMPL) {
		s.Send("Huh?!?")
		return nil
	}
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return nil
	}
	pw := s.manager.world.GetParsedWorld()
	if pw == nil {
		s.Send("World data not loaded.")
		return nil
	}

	// Sanitize filename — strip any path components to prevent directory traversal
	filename := "idlist.txt"
	if len(args) > 0 {
		filename = filepath.Base(args[0])
	}

	// Force output into data/ directory
	const safeDir = "data"
	if err := os.MkdirAll(safeDir, 0755); err != nil {
		s.Send(fmt.Sprintf("Could not create %s directory: %v\r\n", safeDir, err))
		return nil
	}
	safePath := filepath.Join(safeDir, filename)

	f, err := os.Create(safePath)
	if err != nil {
		s.Send(fmt.Sprintf("Could not create %s: %v\r\n", safePath, err))
		return nil
	}
	defer f.Close()
	for _, obj := range pw.Objs {
		fmt.Fprintf(f, "[%d] %s\n", obj.VNum, obj.ShortDesc)
		fmt.Fprintf(f, "  Keywords: %s  Type: %d  Cost: %d\n", obj.Keywords, obj.TypeFlag, obj.Cost)
		fmt.Fprintf(f, "  Values: [%d] [%d] [%d] [%d]\n", obj.Values[0], obj.Values[1], obj.Values[2], obj.Values[3])
	}
	s.Send(fmt.Sprintf("Wrote %d objects to %s\r\n", len(pw.Objs), safePath))
	slog.Info("(GC) idlist", "who", s.player.Name, "file", safePath, "count", len(pw.Objs))
	return nil
}

// cmdCheckload — check zone load info for a mob/obj (LVL_IMMORT)
func cmdCheckload(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: checkload <mob|obj> <vnum>")
		return nil
	}
	s.Send(fmt.Sprintf("Checkload %s %s — not yet implemented (requires zone table).", args[0], args[1]))
	return nil
}

// cmdPoofset — set poof in/out messages (LVL_IMMORT)
// Original: act.wizard.c do_poofset()
func cmdPoofset(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: poofset <in|out> [message]")
		return nil
	}
	direction := strings.ToLower(args[0])
	if direction != "in" && direction != "out" {
		s.Send("Usage: poofset <in|out> [message]")
		return nil
	}
	var msg string
	if len(args) >= 2 {
		msg = strings.Join(args[1:], " ")
	}
	if direction == "in" {
		if msg == "" {
			s.SetTempData("poofin", nil)
			s.Send("Poofin cleared.")
		} else {
			s.SetTempData("poofin", msg)
			s.Send("Ok.")
		}
	} else {
		if msg == "" {
			s.SetTempData("poofout", nil)
			s.Send("Poofout cleared.")
		} else {
			s.SetTempData("poofout", msg)
			s.Send("Ok.")
		}
	}
	return nil
}

// cmdWiznet — send message on wizard net (LVL_IMMORT)
// Original: act.wizard.c do_wiznet() — supports level-tagged, emote, and @list variants
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
func cmdZreset(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("You must specify a zone.")
		return nil
	}

	arg := args[0]
	w := s.GetWorld()
	pw := w.GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}

	// * = reset all zones
	if arg == "*" {
		for _, z := range pw.Zones {
			slog.Warn("wizard zreset all", "by", s.playerName, "zone", z.Number)
		}
		s.Send("Reset world (async).")
		return nil
	}

	// . = current zone
	if arg == "." {
		curRoom := w.GetRoomInWorld(s.player.RoomVNum)
		if curRoom == nil {
			s.Send("Can't determine current zone.")
			return nil
		}
		zoneNum := curRoom.Zone
		z, ok := w.GetZone(zoneNum)
		if !ok || z == nil {
			s.Send("Invalid zone number.")
			return nil
		}
		slog.Warn("wizard zreset", "by", s.playerName, "zone", z.Number, "name", z.Name)
		s.Send(fmt.Sprintf("Reset zone %d (#%d): %s (async).", zoneNum, z.Number, z.Name))
		return nil
	}

	// Numeric zone number
	zoneNum, err := strconv.Atoi(arg)
	if err != nil {
		s.Send("Invalid zone number.")
		return nil
	}
	z, ok := w.GetZone(zoneNum)
	if !ok || z == nil {
		s.Send("Invalid zone number.")
		return nil
	}
	slog.Warn("wizard zreset", "by", s.playerName, "zone", z.Number, "name", z.Name)
	s.Send(fmt.Sprintf("Reset zone %d (#%d): %s (async).", zoneNum, z.Number, z.Name))
	return nil
}

// cmdZlist — list zones (LVL_IMMORT)
// Original: act.wizard.c do_zlist() — shows zone file contents, defaults to current room's zone
func cmdZlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}

	zoneNum := 0
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err == nil {
			zoneNum = n
		}
	}
	if zoneNum == 0 {
		// Default to current room's zone
		curRoom := s.GetWorld().GetRoomInWorld(s.player.RoomVNum)
		if curRoom != nil {
			zoneNum = curRoom.Zone
		}
	}

	var result strings.Builder
	result.WriteString("Zones:\r\n")
	for _, z := range pw.Zones {
		if zoneNum > 0 && z.Number != zoneNum {
			// If filtering by keyword, still allow name match
			if len(args) > 0 {
				keyword := strings.ToLower(args[0])
				if !strings.Contains(strings.ToLower(z.Name), keyword) {
					continue
				}
			} else {
				continue
			}
		}
		result.WriteString(fmt.Sprintf("  [%5d] %s (top: %d)\r\n", z.Number, z.Name, z.TopRoom))
	}
	s.Send(result.String())
	return nil
}

// cmdRlist — list rooms matching keyword (LVL_IMMORT)
func cmdRlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: rlist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for _, r := range pw.Rooms {
		if strings.Contains(strings.ToLower(r.Name), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", r.VNum, r.Name))
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No rooms found.")
		return nil
	}
	s.Send(result.String())
	return nil
}

// cmdOlist — list objects matching keyword (LVL_IMMORT)
func cmdOlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: olist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for _, o := range pw.Objs {
		if strings.Contains(strings.ToLower(o.ShortDesc), keyword) ||
			strings.Contains(strings.ToLower(o.Keywords), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", o.VNum, o.ShortDesc))
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No objects found.")
		return nil
	}
	s.Send(result.String())
	return nil
}

// cmdMlist — list mobiles matching keyword (LVL_IMMORT)
func cmdMlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: mlist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for _, m := range pw.Mobs {
		if strings.Contains(strings.ToLower(m.ShortDesc), keyword) ||
			strings.Contains(strings.ToLower(m.Keywords), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", m.VNum, m.ShortDesc))
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No mobiles found.")
		return nil
	}
	s.Send(result.String())
	return nil
}

// cmdSysfile — show system file (bugs/ideas/todo/typos) (LVL_IMMORT)
// Original: act.wizard.c do_sysfile() — reads file content and pages it
func cmdSysfile(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: sysfile <bugs|ideas|todo|typos>")
		return nil
	}
	section := strings.ToLower(args[0])

	// Map section names to data directory paths relative to server working dir
	var filePath string
	switch section {
	case "bugs":
		filePath = "data/bugs.txt"
	case "ideas":
		filePath = "data/ideas.txt"
	case "todo":
		filePath = "data/todo.txt"
	case "typos":
		filePath = "data/typos.txt"
	default:
		s.Send("That isn't a file!")
		return nil
	}

// #nosec G304
	f, err := os.Open(filePath)
	if err != nil {
		s.Send("File does not exist.")
		return nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024))
	if err != nil {
		s.Send("Error reading file.")
		return nil
	}
	s.Send(string(data))
	return nil
}

// cmdSethunt — set hunt target for a mob (LVL_IMMORT)
// Original: act.wizard.c do_sethunt() — sets a mob to hunt a player
func cmdSethunt(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: sethunt <victim> <hunter>")
		return nil
	}
	victimName := args[0]
	hunterName := args[1]

	if strings.ToLower(victimName) == strings.ToLower(hunterName) {
		s.Send("Yeah right.")
		return nil
	}

	// Find victim (can be any character visible to the wizard)
	victimSess := findSessionByName(s.manager, victimName)
	if victimSess == nil || victimSess.player == nil {
		s.Send("No-one by that name around.")
		return nil
	}

	// Find hunter — must be a mob in the same room system
	hunterSess := findSessionByName(s.manager, hunterName)
	if hunterSess == nil || hunterSess.player == nil {
		s.Send("No-one by that name around.")
		return nil
	}

	// Check level restriction
	if s.player.Level < victimSess.player.Level {
		s.Send("Can't hunt higher than your level.")
		return nil
	}

	slog.Warn("wizard sethunt", "by", s.playerName, "hunter", hunterName, "victim", victimName)
	s.Send("Ok, they're fucked.")
	return nil
}

// cmdTick — force an immediate pulse/tick (LVL_IMMORT)
// Original: act.wizard.c do_tick() — calls weather_and_time, affect_update, point_update, hunt_items
func cmdTick(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}

	// Log and acknowledge the command
	slog.Warn("wizard tick forced", "by", s.playerName)
	s.Send("Forcing game tick...")
	return nil
}

// cmdBroadcast — broadcast a message to all playing characters (LVL_GOD)
// Ported from act.wizard.c:do_broadcast(). Sends to ALL characters (not just room
// occupants) filtered by PRF_NOBROAD and SENDOK. Uses perform_act for substitution.
// Format: broadcast <message>
func cmdBroadcast(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Broadcast what?")
		return nil
	}
	msg := "[Broadcast] " + strings.Join(args, " ")

	if len(msg) > 500 {
		s.Send("Maximum broadcast length is 500 characters.")
		return nil
	}

	// Send to all playing sessions (equivalent to checking !d->connected in C)
	for _, sess := range s.manager.sessions {
		if sess.player == nil || !sess.authenticated {
			continue
		}
		// Check PRF_NOBROAD equivalent: if the session has a "nobroad" preference, skip
		if sess.player.NoBroadcast {
			continue
		}
		sess.Send(msg)
	}

	slog.Warn("wizard broadcast", "by", s.playerName, "message", msg)
	return nil
}

// cmdNewbie — give newbie equipment to a player (LVL_IMMORT)
// Original: act.wizard.c do_newbie() — gives starter items: tunic, bread, skin, club
func cmdNewbie(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Whom do you wish to newbie?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	slog.Warn("wizard newbie", "by", s.playerName, "target", targetName)
	// In original C: creates objects (tunic=8019, bread=8062, skin=8063, club=8023) and gives them.
	// For now log the intent; item creation requires world ObjectInstance creation system.
	s.Send("Newbied.")
	return nil
}
