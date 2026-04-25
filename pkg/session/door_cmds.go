package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

// Door subcommand constants, matching C port SCMD_* values.
const (
	doorSCMDOpen   = 0
	doorSCMDClose  = 1
	doorSCMDUnlock = 2
	doorSCMDLock   = 3
	doorSCMDPick   = 4
	doorSCMDBash   = 5
)

// doGenDoor handles door operations for all subcommands, mirroring the C
// do_gen_door() logic from act.movement.c.
func (s *Session) doGenDoor(subcmd int, args []string) {
	if len(args) == 0 {
		switch subcmd {
		case doorSCMDOpen:
			s.Send("Open what? (Try: open door north)")
		case doorSCMDClose:
			s.Send("Close what? (Try: close door north)")
		case doorSCMDUnlock:
			s.Send("Unlock what? (Try: unlock door north)")
		case doorSCMDLock:
			s.Send("Lock what? (Try: lock door north)")
		case doorSCMDPick:
			s.Send("Pick what? (Try: pick door north)")
		case doorSCMDBash:
			s.Send("Bash what? (Try: bash door north)")
		default:
			s.Send("Do what with which door?")
		}
		return
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.Send(fmt.Sprintf("%s what door?", cmdDoorName(subcmd)))
		return
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.Send("There are no doors here.")
		return
	}

	roomVNum := s.player.GetRoom()
	door, ok := dm.GetDoor(roomVNum, dir)
	if !ok || !door.CanSee() {
		s.Send(fmt.Sprintf("There is no door %s of here.", dir))
		return
	}

	switch subcmd {
	case doorSCMDOpen:
		s.doDoorOpen(door, roomVNum, dir)
	case doorSCMDClose:
		s.doDoorClose(door, roomVNum, dir)
	case doorSCMDUnlock:
		s.doDoorUnlock(door, roomVNum, dir)
	case doorSCMDLock:
		s.doDoorLock(door, roomVNum, dir)
	case doorSCMDPick:
		s.doDoorPick(door, roomVNum, dir)
	case doorSCMDBash:
		s.doDoorBash(door, roomVNum, dir)
	}
}

func cmdDoorName(subcmd int) string {
	switch subcmd {
	case doorSCMDOpen:
		return "open"
	case doorSCMDClose:
		return "close"
	case doorSCMDUnlock:
		return "unlock"
	case doorSCMDLock:
		return "lock"
	case doorSCMDPick:
		return "pick"
	case doorSCMDBash:
		return "bash"
	default:
		return "do"
	}
}

func (s *Session) doDoorOpen(door *systems.Door, roomVNum int, dir string) {
	if !door.Closed {
		s.Send("It's already open.")
		return
	}
	if door.Locked {
		s.Send("It's locked.")
		return
	}

	dm := getDoorManager(s)
	success, msg := dm.OpenDoor(roomVNum, dir)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s opens the door %s.", s.player.Name, dir))
		s.mirrorDoorState(door, false)
	} else {
		s.Send(msg)
	}
}

func (s *Session) doDoorClose(door *systems.Door, roomVNum int, dir string) {
	if door.Closed {
		s.Send("It's already closed.")
		return
	}

	dm := getDoorManager(s)
	success, msg := dm.CloseDoor(roomVNum, dir)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s closes the door %s.", s.player.Name, dir))
		s.mirrorDoorState(door, true)
	} else {
		s.Send(msg)
	}
}

func (s *Session) doDoorUnlock(door *systems.Door, roomVNum int, dir string) {
	if !door.Locked {
		s.Send("It's already unlocked.")
		return
	}
	if !door.Closed {
		s.Send("You must close it first.")
		return
	}

	keyVNum := s.findKeyForDoor(door)
	if keyVNum < 0 {
		s.Send("You don't have the right key.")
		return
	}

	dm := getDoorManager(s)
	success, msg := dm.UnlockDoor(roomVNum, dir, keyVNum)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s unlocks the door %s.", s.player.Name, dir))
	} else {
		s.Send(msg)
	}
}

func (s *Session) doDoorLock(door *systems.Door, roomVNum int, dir string) {
	if door.Locked {
		s.Send("It's already locked.")
		return
	}
	if !door.Closed {
		s.Send("You must close it first.")
		return
	}

	keyVNum := s.findKeyForDoor(door)
	if keyVNum < 0 {
		s.Send("You don't have the right key.")
		return
	}

	dm := getDoorManager(s)
	success, msg := dm.LockDoor(roomVNum, dir, keyVNum)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s locks the door %s.", s.player.Name, dir))
	} else {
		s.Send(msg)
	}
}

func (s *Session) doDoorPick(door *systems.Door, roomVNum int, dir string) {
	if !door.Locked {
		s.Send("It's not locked.")
		return
	}
	if !door.Closed {
		s.Send("You must close it first.")
		return
	}
	if door.Pickproof {
		s.Send("This lock is too complex to pick.")
		return
	}

	if !s.findItemByVNum(8027) {
		s.Send("You don't have any lockpicks.")
		return
	}

	skillLevel := s.player.GetSkill(game.SkillPickLock)
	if skillLevel == 0 {
		s.Send("You have no idea how to pick locks.")
		return
	}

	dm := getDoorManager(s)
	success, msg := dm.PickDoor(roomVNum, dir, skillLevel)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s picks the lock on the door %s.", s.player.Name, dir))
	} else {
		s.Send(msg)
	}
}

func (s *Session) doDoorBash(door *systems.Door, roomVNum int, dir string) {
	if !door.Closed {
		s.Send("It's already open.")
		return
	}
	if !door.Bashable {
		s.Send("This door cannot be bashed.")
		return
	}
	if door.Hp <= 0 {
		s.Send("The door has already been destroyed.")
		return
	}

	// Rough strength calc — stats system coming later
	str := 50 + s.player.Strength/2
	dm := getDoorManager(s)
	success, msg := dm.BashDoor(roomVNum, dir, str)
	if success {
		s.Send(msg)
		doorBroadcast(s, fmt.Sprintf("%s bashes down the %s door!", s.player.Name, dir))
	} else {
		s.Send(msg)
	}
}

func (s *Session) findKeyForDoor(door *systems.Door) int {
	if door.KeyVNum >= 0 {
		if playerHasKey(s, door.KeyVNum) {
			return door.KeyVNum
		}
		return -1
	}
	// No specific key required — look for any ITEM_KEY (type 6)
	for _, item := range s.player.Inventory.Items {
		if item.GetTypeFlag() == 6 {
			return item.VNum
		}
	}
	return -1
}

func (s *Session) findItemByVNum(vnum int) bool {
	for _, item := range s.player.Inventory.Items {
		if item.VNum == vnum {
			return true
		}
	}
	return false
}

func (s *Session) mirrorDoorState(door *systems.Door, closed bool) {
	if door.ToRoom <= 0 {
		return
	}

	dm := getDoorManager(s)
	odir := getOppositeDirection(door.Direction)
	mirrorDoor, ok := dm.GetDoor(door.ToRoom, odir)
	if !ok {
		return
	}

	mirrorDoor.Closed = closed
	if !closed && mirrorDoor.Locked {
		mirrorDoor.Locked = false
	}

	var msgText string
	if closed {
		msgText = fmt.Sprintf("The door %s closes.", odir)
	} else {
		msgText = fmt.Sprintf("The door %s opens.", odir)
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "door",
			Text: msgText,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(door.ToRoom, msg, "")
}

func getOppositeDirection(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	case "up":
		return "down"
	case "down":
		return "up"
	default:
		return ""
	}
}

// cmdBashDoor handles the 'bash' command for doors (not combat bash).
func cmdBashDoor(s *Session, args []string) error {
	s.doGenDoor(doorSCMDBash, args)
	return nil
}

// cmdOpen handles the 'open' command.
func cmdOpen(s *Session, args []string) error {
	s.doGenDoor(doorSCMDOpen, args)
	return nil
}

// cmdClose handles the 'close' command.
func cmdClose(s *Session, args []string) error {
	s.doGenDoor(doorSCMDClose, args)
	return nil
}

// cmdLock handles the 'lock' command.
func cmdLock(s *Session, args []string) error {
	s.doGenDoor(doorSCMDLock, args)
	return nil
}

// cmdUnlock handles the 'unlock' command.
func cmdUnlock(s *Session, args []string) error {
	s.doGenDoor(doorSCMDUnlock, args)
	return nil
}

// cmdPick handles the 'pick' command.
func cmdPick(s *Session, args []string) error {
	s.doGenDoor(doorSCMDPick, args)
	return nil
}

// cmdKnock handles the 'knock' command.
func cmdKnock(s *Session, args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = resolveDirection(strings.ToLower(args[0]))
	}
	if dir == "" {
		s.Send("Knock on what?  Try north, south, east, west, up, or down.")
		return nil
	}

	roomVNum := s.player.GetRoom()
	room, ok := s.manager.world.GetRoom(roomVNum)
	if !ok {
		return nil
	}

	exit, exists := room.Exits[dir]
	if !exists {
		s.Send("There is nothing to knock on in that direction.")
		return nil
	}

	var doorDesc string
	if exit.Keywords != "" {
		doorDesc = exit.Keywords
	} else {
		doorDesc = dir
	}

	s.Send(fmt.Sprintf("You knock on the %s.", doorDesc))
	doorBroadcast(s, fmt.Sprintf("%s knocks on the %s.", s.player.Name, doorDesc))

	if exit.ToRoom > 0 {
		s.manager.BroadcastToRoom(exit.ToRoom,
			[]byte(fmt.Sprintf("Someone knocks on the door from the other side.")), "")
	}

	return nil
}
