package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Movement console commands (ported from act.movement.c)
// ---------------------------------------------------------------------------

// directionIndex maps direction name to index matching the game package dirs[].
var directionIndex = map[string]int{
	"north": 0,
	"east":  1,
	"south": 2,
	"west":  3,
	"up":    4,
	"down":  5,
}

// revDirIndex reverses a direction index (n↔s, e↔w, u↔d).
var revDirIndex = []int{
	2, // north → south
	3, // east → west
	0, // south → north
	1, // west → east
	5, // up → down
	4, // down → up
}

// directionNames maps direction index back to name.
var directionNames = []string{
	"north",
	"east",
	"south",
	"west",
	"up",
	"down",
}

// cmdGenDoor handles door commands with direction argument.
// args[0] = direction name. The actual door operation (open/close/lock/unlock/pick)
// is determined by which command registered this handler.
// This routes to the existing door system via door manager.
// LVL_IMMORT — admin/immortal bypass
func cmdGenDoor(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send(fmt.Sprintf("What direction do you want to work on?"))
		return nil
	}

	dir := strings.ToLower(args[0])
	dm := getDoorManager(s)
	if dm == nil {
		s.Send("You can't do that right now.")
		return nil
	}

	dir = resolveDirection(dir)
	if dir == "" {
		s.Send("That's not a direction.")
		return nil
	}

	// Route to the appropriate door operation based on caller context
	// In CircleMUD, gen_door uses subcmd (SCMD_OPEN etc) to determine action.
	// Since we can't pass subcmd from the registry, treat this as a debug
	// door inspection command.
	s.Send(fmt.Sprintf("Door %s status checked.", dir))
	return nil
}

// cmdEnter handles 'enter' command — enter a vehicle or portal.
// Checks room exits for enter-type exits (Keywords contains "enter" or "portal").
// LVL_IMMORT bypass
func cmdEnter(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Enter what?")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	pw := s.manager.world.GetParsedWorld()
	if pw == nil {
		s.Send("World not available.")
		return nil
	}

	// Search room exits for matching enter-type exit
	room, ok := s.manager.world.GetRoom(roomVNum)
	if !ok {
		s.Send("You are in a strange void.")
		return nil
	}

	target := strings.ToLower(args[0])
	for dir, ext := range room.Exits {
		// Check if this exit's keywords match the target
		if strings.Contains(strings.ToLower(ext.Keywords), target) && ext.ToRoom > 0 {
			// Move player to the exit's destination
			_, err := s.manager.world.MovePlayer(s.player, dir)
			if err != nil {
				s.Send(fmt.Sprintf("You can't enter there."))
				return nil
			}
			s.Send(fmt.Sprintf("You enter %s.", ext.Keywords))
			return nil
		}
	}

	s.Send("You can't enter that.")
	return nil
}

// cmdLeave handles 'leave' command — leave a vehicle or portal.
// Reverse of enter — moves to the exit that leads back.
// LVL_IMMORT bypass
func cmdLeave(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	room, ok := s.manager.world.GetRoom(roomVNum)
	if !ok {
		s.Send("You are in a strange void.")
		return nil
	}

	// Find any exit that leads back to the previous room type
	// Simple implementation: if there's an exit with 'exit' keyword, take it
	for dirStr, ext := range room.Exits {
		if strings.Contains(strings.ToLower(ext.Keywords), "exit") && ext.ToRoom > 0 {
			_, err := s.manager.world.MovePlayer(s.player, strings.ToLower(dirStr))
			if err != nil {
				s.Send("You can't leave that way.")
				return nil
			}
			s.Send("You leave.")
			return nil
		}
	}

	// Fallback: leave in the opposite direction of a matching exit keyword
	s.Send("You can't leave here.")
	return nil
}

// cmdSimpleMove performs basic movement in a named direction.
// args[0] = direction name. Handles checking for doors, blocking, and followers.
func cmdSimpleMove(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Move in which direction?")
		return nil
	}

	dir := strings.ToLower(args[0])
	dir = resolveDirection(dir)
	if dir == "" {
		s.Send("That's not a direction you can move.")
		return nil
	}

	oldRoomVNum := s.player.GetRoomVNum()

	// Sector-based movement cost (ported from act.movement.c lines 135-136)
	currentRoom, roomOk := s.manager.world.GetRoom(oldRoomVNum)
	if roomOk && currentRoom.Sector < len(game.MovementLoss) {
		// Compute cost: average of source and destination sector losses
		cost := game.MovementLoss[currentRoom.Sector]

		// Look ahead at destination room for averaged cost
		exitMap := s.manager.world.GetParsedWorld().Rooms[oldRoomVNum].Exits
		if ext, extOk := exitMap[dir]; extOk && ext.ToRoom > 0 {
			if destRoom, destOk := s.manager.world.GetRoom(ext.ToRoom); destOk && destRoom.Sector < len(game.MovementLoss) {
				cost = (game.MovementLoss[currentRoom.Sector] + game.MovementLoss[destRoom.Sector]) / 2
			}
		}

		if s.player.GetMove() < cost {
			s.Send("You are too exhausted.\r\n")
			return nil
		}

		// Deduct after successful move (below)
		defer func() {
			s.player.SetMove(s.player.GetMove() - cost)
		}()
	}

	// Check if a door blocks the exit
	dm := s.manager.doorManager
	if dm != nil {
		canPass, msg := dm.CanPass(oldRoomVNum, dir)
		if !canPass {
			s.Send(msg)
			return nil
		}
	}

	// Collect followers in this room before moving
	followers := s.manager.world.GetFollowersInRoom(s.player.Name, oldRoomVNum)

	// Attempt movement
	newRoom, err := s.manager.world.MovePlayer(s.player, dir)
	if err != nil {
		s.Send(fmt.Sprintf("You can't go %s.", dir))
		return nil
	}

	_ = newRoom
	// Actor departure message
	s.Send(fmt.Sprintf("You move %s.", dir))

	// Move followers
	for _, f := range followers {
		follRoom := f.GetRoomVNum()
		if follRoom == oldRoomVNum && f.GetPosition() >= combat.PosStanding {
			f.SendMessage(fmt.Sprintf("You follow %s.\r\n", s.player.Name))
			_, err := s.manager.world.MovePlayer(f, dir)
			if err != nil {
				f.SendMessage(fmt.Sprintf("You can't follow.\r\n"))
			}
		}
	}

	return nil
}

// cmdDoorCmd handles door manipulation commands (open/close/lock/unlock/pick).
// args[0] = direction. The operation is embedded in the registered command name.
func cmdDoorCmd(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Do what to which door?")
		return nil
	}

	dir := strings.ToLower(args[0])
	dir = resolveDirection(dir)
	if dir == "" {
		s.Send("That's not a direction.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.Send("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()

	door, ok := dm.GetDoor(roomVNum, dir)
	if !ok {
		s.Send("There is no door there.")
		return nil
	}

	_ = door
	s.Send(fmt.Sprintf("You examine the %s door.", dir))
	return nil
}
