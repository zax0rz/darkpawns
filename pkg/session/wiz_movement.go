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
	raw := strings.Join(args, " ")
	targetName, remainder := game.OneArgument(raw)
	if targetName == "" {
		s.Send("Whom do you wish to teleport?\r\n")
		return nil
	}
	target, ok := s.manager.world.ResolveCharWorld(s.player, targetName)
	if !ok || target.Combatant == nil {
		s.Send("No-one by that name here.\r\n")
		return nil
	}
	if target.Combatant == s.player {
		s.Send("Use 'goto' to teleport yourself.\r\n")
		return nil
	}
	if target.Combatant.GetLevel() >= s.player.GetLevel() {
		s.Send("Maybe you shouldn't do that.\r\n")
		return nil
	}
	destination, _ := game.OneArgument(remainder)
	if destination == "" {
		s.Send("Where do you wish to send this person?\r\n")
		return nil
	}
	dest, ok := findGotoRoom(s, destination)
	if !ok {
		return nil
	}

	s.Send("Okay.\r\n")
	game.Act(s.manager.world, false, target.Combatant, nil, nil, nil,
		"$n disappears in a puff of smoke.", "", game.ToRoom)
	if target.Combatant.IsNPC() {
		target.Combatant.(*game.MobInstance).SetRoom(dest)
	} else {
		target.Combatant.(*game.Player).SetRoom(dest)
	}
	slog.Warn("wizard teleport", "by", s.player.Name, "target", target.Combatant.GetName(), "room", dest)
	game.Act(s.manager.world, false, target.Combatant, nil, nil, nil,
		"$n arrives from a puff of smoke.", "", game.ToRoom)
	game.Act(s.manager.world, false, s.player, target.Combatant, nil, nil,
		"$n has teleported you!", "", game.ToVict)
	if !target.Combatant.IsNPC() {
		if targetSession := findSessionForPlayer(s.manager, target.Combatant.(*game.Player)); targetSession != nil {
			if err := cmdLook(targetSession, nil); err != nil {
				slog.Error("wizard teleport look failed", "target", target.Combatant.GetName(), "error", err)
			}
		}
	}
	return nil
}

// cmdTransfer mirrors do_trans (src/act.wizard.c:309-364). Unlike teleport,
// transfer has only a one_argument target and always lands the target in the
// actor's current room. The C "all" branch is available only to GRGOD and
// iterates connected characters below the actor's level.
func cmdTransfer(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT+1) {
		s.Send("Huh?!?")
		return nil
	}

	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName == "" {
		s.Send("Whom do you wish to transfer?\r\n")
		return nil
	}

	if strings.EqualFold(targetName, "all") && s.player.GetLevel() >= LVL_GRGOD {
		s.manager.mu.RLock()
		targets := make([]*Session, 0, len(s.manager.sessions))
		for _, target := range s.manager.sessions {
			if target == nil || target.player == nil || target == s || !target.authenticated {
				continue
			}
			if target.player.GetLevel() >= s.player.GetLevel() {
				continue
			}
			targets = append(targets, target)
		}
		s.manager.mu.RUnlock()

		for _, target := range targets {
			transferCharacter(s, target.player)
		}
		s.Send("Okay.\r\n")
		return nil
	}

	target, ok := s.manager.world.ResolveCharWorld(s.player, targetName)
	if !ok || target.Combatant == nil {
		s.Send("No-one by that name here.\r\n")
		return nil
	}
	if target.Combatant == s.player {
		s.Send("That doesn't make much sense, does it?\r\n")
		return nil
	}
	if !target.Combatant.IsNPC() && s.player.GetLevel() < target.Combatant.GetLevel() {
		s.Send("Go transfer someone your own size.\r\n")
		return nil
	}

	transferCharacter(s, target.Combatant)
	return nil
}

func transferCharacter(s *Session, target game.Actor) {
	if s == nil || s.manager == nil || s.manager.world == nil || target == nil {
		return
	}

	game.Act(s.manager.world, false, target, nil, nil, nil,
		"$n disappears in a blaze of hellfire!", "", game.ToRoom)
	var err error
	if target.IsNPC() {
		err = s.manager.world.MobTransfer(target.(*game.MobInstance), s.player.GetRoom())
	} else {
		err = s.manager.world.PlayerTransfer(target.(*game.Player), s.player.GetRoom())
	}
	if err != nil {
		slog.Error("wizard transfer failed", "by", s.player.Name, "target", target.GetName(), "error", err)
		return
	}
	game.Act(s.manager.world, false, target, nil, nil, nil,
		"$n arrives from a puff of smoke.", "", game.ToRoom)
	game.Act(s.manager.world, false, s.player, target, nil, nil,
		"$n has transferred you!", "", game.ToVict)
	if !target.IsNPC() {
		if targetSession := findSessionForPlayer(s.manager, target.(*game.Player)); targetSession != nil {
			if err := cmdLook(targetSession, nil); err != nil {
				slog.Error("wizard transfer look failed", "target", target.GetName(), "error", err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// heal — fully heal target (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdHome(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	raw := strings.Join(args, " ")
	first, _ := game.OneArgument(raw)
	if first != "" && isLeadingDigit(first) {
		// C sets PLR_LOADROOM before find_target_room, including when the
		// destination is rejected. Keep that mutation on the same path.
		s.player.SetPlrFlag(game.PlrLoadroom, true)
		destination, ok := findGotoRoom(s, raw)
		if !ok {
			s.Send("That room does not exist.\n")
			return nil
		}
		s.player.SetLoadRoom(destination)
		s.Send(fmt.Sprintf("Home room set to %d.\n", destination))
		return nil
	}
	if first != "" {
		s.Send("Home or Home <room-number>\n")
		return nil
	}

	location := s.player.GetLoadRoom()
	if location <= 0 || s.manager.world.GetRoomInWorld(location) == nil {
		s.Send("That room does not exist.\n")
		s.player.SetLoadRoom(1)
		location = 1
		s.Send("Error in your home room. Now set to Limbo.\n")
	}

	// The C source builds this line with overlapping sprintf(buf, "%s ...",
	// buf); the oracle's actual libc result is the suffix only.
	s.Send(" pulled into a different reality.\r\n")
	// The telnet transport canonicalizes a message ending in LF to one CRLF;
	// using LF here preserves C's single blank line without triggering the
	// transport's implicit terminator for a message ending in CR.
	s.Send("\n")
	poofOut := s.player.PoofOut
	if poofOut == "" {
		poofOut = "$n disappears in a blaze of hellfire!"
	} else {
		// C's stored poof text is passed through act() as data and the
		// original command path preserves literal $-codes in this field.
		// Escape them before the shared act formatter processes the message.
		poofOut = strings.ReplaceAll(poofOut, "$", "$$")
	}
	game.Act(s.manager.world, true, s.player, nil, nil, nil, poofOut, "", game.ToRoom)
	if err := s.manager.world.PlayerTransfer(s.player, location); err != nil {
		slog.Error("wizard home transfer failed", "by", s.player.Name, "to", location, "error", err)
		return err
	}
	game.Act(s.manager.world, true, s.player, nil, nil, nil, poofOut, "", game.ToRoom)
	if err := cmdLook(s, nil); err != nil {
		slog.Error("wizard home room look failed", "by", s.player.Name, "room", location, "error", err)
		return err
	}
	return nil
}

// cmdDate — show current system time or uptime (LVL_IMMORT)
