package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

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
	if err := ExecuteCommand(s, strings.Fields(rest)[0], strings.Fields(rest)[1:]); err != nil {
		slog.Error("wizard at command failed", "command", rest, "error", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// load — load a mob or object (LVL_IMMORT)
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
