package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdGoto(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	dest, ok := findGotoRoom(s, strings.Join(args, " "))
	if !ok {
		return nil
	}

	oldRoom := s.player.GetRoom()
	poofOut := s.player.PoofOut
	if poofOut == "" {
		poofOut = "$n disappears in a puff of smoke."
	}
	game.Act(s.manager.world, true, s.player, nil, nil, nil, poofOut, "", game.ToRoom)

	if err := s.manager.world.PlayerTransfer(s.player, dest); err != nil {
		slog.Error("wizard goto transfer failed", "by", s.player.Name, "from", oldRoom, "to", dest, "error", err)
		return err
	}

	poofIn := s.player.PoofIn
	if poofIn == "" {
		poofIn = "$n appears with an ear-splitting bang."
	}
	game.Act(s.manager.world, true, s.player, nil, nil, nil, poofIn, "", game.ToRoom)
	if err := cmdMovementLook(s); err != nil {
		slog.Error("wizard goto room look failed", "by", s.player.Name, "room", dest, "error", err)
		return err
	}
	return nil
}

// findGotoRoom mirrors find_target_room (src/act.wizard.c:184-239). The
// handler receives C's one_argument result, so fill words are discarded before
// numeric-room, visible-character, and visible-object resolution.
func findGotoRoom(s *Session, raw string) (int, bool) {
	roomName, _ := game.OneArgument(raw)
	if roomName == "" {
		s.Send("You must supply a room number or name.\r\n")
		return 0, false
	}

	if isLeadingDigit(roomName) && !strings.ContainsRune(roomName, '.') {
		vnum, ok := parseLeadingInt(roomName)
		if !ok || s.manager.world.GetRoomInWorld(vnum) == nil {
			s.Send("No room exists with that number.\r\n")
			return 0, false
		}
		return gotoRoomAllowed(s, vnum)
	}

	if target, ok := s.manager.world.ResolveCharWorld(s.player, roomName); ok {
		if target.Player != nil {
			return gotoRoomAllowed(s, target.Player.GetRoom())
		}
		if target.Mob != nil {
			return gotoRoomAllowed(s, target.Mob.GetRoom())
		}
	}
	if object, ok := s.manager.world.ResolveObjectWorld(s.player, roomName); ok {
		if room := object.GetRoomVNum(); room > 0 {
			return gotoRoomAllowed(s, room)
		}
		s.Send("That object is not available.\r\n")
		return 0, false
	}

	s.Send("No such creature or object around.\r\n")
	return 0, false
}

func gotoRoomAllowed(s *Session, roomVNum int) (int, bool) {
	if s.player.GetLevel() < LVL_GRGOD {
		room := s.manager.world.GetRoomInWorld(roomVNum)
		if game.HasRoomFlag(room, "GODROOM") {
			s.Send("You are not godly enough to use that room!\r\n")
			return 0, false
		}
		if game.HasRoomFlag(room, "PRIVATE") {
			occupants := len(s.manager.world.GetPlayersInRoom(roomVNum)) + len(s.manager.world.GetMobsInRoom(roomVNum))
			if occupants > 1 {
				s.Send("There's a private conversation going on in that room.\r\n")
				return 0, false
			}
		}
	}
	return roomVNum, true
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

// cmdDig mirrors do_dig (src/new_cmds2.c:818-881). It creates a bare exit in
// both directions and leaves the runtime world mutation in game.World.
func cmdDig(s *Session, args []string) error {
	directionArg, roomArg := parseDigArguments(args)
	if directionArg == "" || roomArg == "" {
		s.Send("Format: dig <dir> <room number>\r\n")
		return nil
	}

	targetVNum, ok := parseDigRoomNumber(roomArg)
	if !ok {
		targetVNum = 0
	}
	targetRoom := s.manager.world.GetRoomInWorld(targetVNum)
	if targetVNum == 0 || targetRoom == nil {
		s.Send(fmt.Sprintf("There is no room with the number %d.\r\n", targetVNum))
		return nil
	}
	currentRoom := s.manager.world.GetRoomInWorld(s.player.GetRoomVNum())
	if currentRoom == nil {
		return nil
	}

	// C lets LVL_SET_BUILD (LVL_GOD+1) edit across zones. Lower builders must
	// have their saved OLC zone set to the current zone and may only target the
	// current zone. Session.olcZone mirrors GET_OLC_ZONE's saved field.
	if s.player.GetLevel() < LVL_GOD+1 {
		if currentRoom.Zone != s.olcZone {
			s.Send("You don't have permission to edit this zone.\r\n")
			return nil
		}
		if targetRoom.Zone != currentRoom.Zone {
			s.Send("You don't have permission to edit that zone.\r\n")
			return nil
		}
	}

	direction, reverse := digDirections(directionArg)
	if direction == "" {
		s.Send("Valid dirs are n,s,e,w,u and d.\r\n")
		direction, reverse = "north", "south"
	}
	s.manager.world.CreateRoomExit(currentRoom.VNum, direction, targetVNum)
	s.manager.world.CreateRoomExit(targetRoom.VNum, reverse, currentRoom.VNum)
	s.Send(fmt.Sprintf("You make an exit %s to room %d.\r\n", directionArg, targetVNum))
	return nil
}

func parseDigArguments(args []string) (string, string) {
	next := func() string {
		for len(args) > 0 {
			value := strings.ToLower(args[0])
			args = args[1:]
			switch value {
			case "in", "from", "with", "the", "on", "at", "to":
				continue
			default:
				return value
			}
		}
		return ""
	}
	return next(), next()
}

func parseDigRoomNumber(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	index := 0
	sign := 1
	if value[0] == '+' || value[0] == '-' {
		if value[0] == '-' {
			sign = -1
		}
		index++
	}
	start := index
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == start {
		return 0, false
	}
	number, err := strconv.Atoi(value[start:index])
	if err != nil {
		return 0, false
	}
	return sign * number, true
}

func digDirections(value string) (string, string) {
	if value == "" {
		return "", ""
	}
	switch strings.ToLower(value[:1]) {
	case "n":
		return "north", "south"
	case "e":
		return "east", "west"
	case "s":
		return "south", "north"
	case "w":
		return "west", "east"
	case "u":
		return "up", "down"
	case "d":
		return "down", "up"
	default:
		return "", ""
	}
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
