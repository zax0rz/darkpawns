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
		s.Send("You must supply a room number or name.\r\n")
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

func cmdAt(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("You must supply a room number or a name.\r\n")
		return nil
	}
	dest, ok := findAtRoom(s, args[0])
	if !ok {
		return nil
	}
	if len(args) == 1 {
		s.Send("What do you want to do there?\r\n")
		return nil
	}

	orig := s.player.GetRoom()
	s.player.SetRoom(dest)
	command, commandArgs := args[1], args[2:]
	slog.Warn("wizard at", "by", s.player.Name, "room", dest, "command", strings.Join(args[1:], " "))
	if err := ExecuteCommand(s, command, commandArgs); err != nil {
		slog.Error("wizard at command failed", "command", strings.Join(args[1:], " "), "error", err)
	}
	// C only restores the actor when the nested command left them in the
	// temporary room (act.wizard.c:260-264). A nested movement command that
	// leaves that room intentionally keeps its resulting location.
	if s.player.GetRoom() == dest {
		s.player.SetRoom(orig)
	}
	return nil
}

// findAtRoom mirrors find_target_room (act.wizard.c:184-239). Numeric room
// vnums take the real_room path; otherwise C resolves a visible character and
// then a visible object, using the resolved entity's current room.
func findAtRoom(s *Session, raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		s.Send("You must supply a room number or a name.\r\n")
		return 0, false
	}

	if isLeadingDigit(raw) && !strings.ContainsRune(raw, '.') {
		vnum, ok := parseLeadingInt(raw)
		if !ok || s.manager.world.GetRoomInWorld(vnum) == nil {
			s.Send("No room exists with that number.\r\n")
			return 0, false
		}
		return vnum, true
	}

	if target, ok := s.manager.world.ResolveCharWorld(s.player, raw); ok {
		if target.Player != nil {
			return target.Player.GetRoom(), true
		}
		if target.Mob != nil {
			return target.Mob.GetRoom(), true
		}
	}
	if object, ok := s.manager.world.ResolveObjectWorld(s.player, raw); ok {
		if room := object.GetRoomVNum(); room > 0 {
			return room, true
		}
		s.Send("That object is not available.\r\n")
		return 0, false
	}

	s.Send("No such creature or object around.\r\n")
	return 0, false
}

func isLeadingDigit(value string) bool {
	return value[0] >= '0' && value[0] <= '9'
}

func parseLeadingInt(value string) (int, bool) {
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	vnum, err := strconv.Atoi(value[:end])
	return vnum, err == nil
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
		s.Send("Whom do you wish to teleport?\r\n")
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
