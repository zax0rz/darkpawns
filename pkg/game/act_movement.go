// Package game: act_movement.go — movement commands, door handling, sleep/rest/stand/sit/wake, follow.
//
// Ported from src/act.movement.c (CircleMUD / Dark Pawns MUD).

package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Constants ported from structs.h / constants.c
// ---------------------------------------------------------------------------

// Room flag string constants — parser stores these as string names.
const (
	roomFlagDeath   = "death"
	roomFlagIndoors = "indoors"
	roomFlagTunnel  = "tunnel"
)

// Sector type constants — from structs.h SECT_*.
const (
	SECT_INSIDE       = 0
	SECT_CITY         = 1
	SECT_FIELD        = 2
	SECT_FOREST       = 3
	SECT_HILLS        = 4
	SECT_MOUNTAIN     = 5
	SECT_WATER_SWIM   = 6
	SECT_WATER_NOSWIM = 7
	SECT_UNDERWATER   = 8
	SECT_FLYING       = 9
	SECT_DESERT       = 10
	SECT_FIRE         = 11
	SECT_EARTH        = 12
	SECT_WIND         = 13
	SECT_WATER        = 14
	SECT_SWAMP        = 15
)

// DoorState constants — from parser.Exit.DoorState in wld.go.
const (
	doorOpen   = 0
	doorClosed = 1
	doorLocked = 2
)

// Affect flag bit positions — from structs.h AFF_* constants.
const (
	affSneak     = 0 // AFF_SNEAK
	affHide      = 1 // AFF_HIDE
	affSleep     = 2 // AFF_SLEEP
	affCharm     = 3 // AFF_CHARM
	affFly       = 4 // AFF_FLY
	affWaterWalk = 5 // AFF_WATERWALK
	affGroup     = 6 // AFF_GROUP
)

// Direction name array (matching dirs[] from constants.c).
var dirs = []string{
	"north",
	"east",
	"south",
	"west",
	"up",
	"down",
}

// revDir reverses direction indices (2 ↔ 0, 3 ↔ 1, 5 ↔ 4).
var revDir = []int{
	2, // north → south
	3, // east → west
	0, // south → north
	1, // west → east
	5, // up → down
	4, // down → up
}

// movementLoss per sector type (from constants.c).
var movementLoss = []int{
	2, // INSIDE (0)
	2, // CITY (1)
	3, // FIELD (2)
	4, // FOREST (3)
	5, // HILLS (4)
	7, // MOUNTAIN (5)
	5, // WATER_SWIM (6)
	6, // WATER_NOSWIM (7)
	2, // FLYING — this is index 8 in C, but C has UNDERWATER=8 and FLYING=9
	6, // UNDERWATER (index 8 in C structs.h)
	8, // DESERT
	6, // FIRE
	6, // EARTH
	6, // WIND
	6, // WATER
	4, // SWAMP
}

// Door command indices.
const (
	scmdOpen   = 0
	scmdClose  = 1
	scmdUnlock = 2
	scmdLock   = 3
	scmdPick   = 4
)

// cmdDoor names (matching cmd_door[] in C).
var cmdDoor = []string{
	"open",
	"close",
	"unlock",
	"lock",
	"pick",
}

// Door subcommand requirement flags.
const (
	needOpen     = 1 << 0
	needClosed   = 1 << 1
	needUnlocked = 1 << 2
	needLocked   = 1 << 3
)

var flagsDoor = []int{
	needClosed | needUnlocked, // SCMD_OPEN
	needOpen,                  // SCMD_CLOSE
	needClosed | needLocked,   // SCMD_UNLOCK
	needClosed | needUnlocked, // SCMD_LOCK
	needClosed | needLocked,   // SCMD_PICK
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getExit returns the exit in the given direction for the player's current room.
// Direction is an index into dirs[] (0=north...5=down).
func getExit(w *World, ch *Player, dir int) (parser.Exit, bool) {
	if dir < 0 || dir >= len(dirs) {
		return parser.Exit{}, false
	}
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return parser.Exit{}, false
	}
	ext, ok := room.Exits[dirs[dir]]
	return ext, ok
}

// getExitByDirStr returns an exit by direction string name.
func getExitByDirStr(w *World, ch *Player, dirStr string) (parser.Exit, string, bool) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return parser.Exit{}, "", false
	}
	ext, ok := room.Exits[dirStr]
	return ext, dirStr, ok
}

// hasBoat checks if a player can traverse water sectors.
func hasBoat(w *World, ch *Player) bool {
	if ch.GetLevel() >= lvlImmort {
		return true
	}
	if ch.IsAffected(affWaterWalk) {
		return true
	}
	if ch.IsAffected(affFly) {
		return true
	}
	// Check inventory for ITEM_BOAT
	if ch.Inventory != nil {
		for _, obj := range ch.Inventory.Items {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
	}
	// Check equipment for ITEM_BOAT
	if ch.Equipment != nil {
		for _, obj := range ch.Equipment.Slots {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
	}
	return false
}

// hasKey checks if a player has a key object by vnum.
func hasKey(ch *Player, key int) bool {
	if key <= 0 {
		return false
	}
	if ch.Inventory != nil {
		for _, obj := range ch.Inventory.Items {
			if obj.VNum == key {
				return true
			}
		}
	}
	if ch.Equipment != nil {
		for _, obj := range ch.Equipment.Slots {
			if obj != nil && obj.VNum == key {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Core Movement
// ---------------------------------------------------------------------------

// doSimpleMove moves a character assuming no master and no followers.
// Returns true on success.
func doSimpleMove(w *World, ch *Player, dir int, needSpecialsCheck bool) bool {
	// Charmed check
	if ch.IsAffected(affCharm) && ch.Following != "" {
		if leader, ok := w.GetPlayer(ch.Following); ok && ch.RoomVNum == leader.RoomVNum {
			sendToChar(ch, "The thought of leaving your master makes you weep.\r\n")
			return false
		}
	}

	ext, ok := getExit(w, ch, dir)
	if !ok {
		sendToChar(ch, "Alas, you cannot go that way...\r\n")
		return false
	}

	toRoom := w.GetRoomInWorld(ext.ToRoom)
	if toRoom == nil {
		sendToChar(ch, "Alas, you cannot go that way...\r\n")
		return false
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return false
	}

	// Water sector boat check
	if room.Sector == SECT_WATER_NOSWIM || toRoom.Sector == SECT_WATER_NOSWIM {
		if !hasBoat(w, ch) {
			sendToChar(ch, "You need a boat to go there.\r\n")
			return false
		}
	}

	// Movement points needed is avg of src and dest sector movement loss
	needMovement := (movementLoss[room.Sector] + movementLoss[toRoom.Sector]) >> 1

	if ch.GetMove() < needMovement {
		if needSpecialsCheck && ch.Following != "" {
			sendToChar(ch, "You are too exhausted to follow.\r\n")
		} else {
			sendToChar(ch, "You are too exhausted.\r\n")
		}
		return false
	}

	// Room tunnel check
	if roomHasFlagStatic(toRoom, roomFlagTunnel) {
		players := w.GetPlayersInRoom(toRoom.VNum)
		if len(players) >= 1 {
			sendToChar(ch, "There isn't enough room there!\r\n")
			return false
		}
	}

	wasIn := ch.RoomVNum

	// Leave message
	if !ch.IsAffected(affSneak) {
		w.roomMessage(wasIn, fmt.Sprintf("$n leaves %s.", dirs[dir]))
	}

	// Deduct movement
	if ch.GetLevel() < lvlImmort {
		ch.SetMove(ch.GetMove() - needMovement)
	}

	// Move character
	ch.SetRoom(ext.ToRoom)

	// Arrival message
	if !ch.IsAffected(affSneak) {
		var direct string
		switch dir {
		case 0:
			direct = "south"
		case 1:
			direct = "west"
		case 2:
			direct = "north"
		case 3:
			direct = "east"
		}

		switch dir {
		case 0, 1, 2, 3:
			var msg string
			if ch.IsAffected(affFly) {
				msg = fmt.Sprintf("$n flies in from the %s.", direct)
			} else if toRoom.Sector == SECT_UNDERWATER {
				msg = fmt.Sprintf("$n swims in from the %s.", direct)
			} else {
				msg = fmt.Sprintf("$n arrives from the %s.", direct)
			}
			w.roomMessage(toRoom.VNum, msg)
		case 4:
			if ch.IsAffected(affFly) {
				w.roomMessage(toRoom.VNum, "$n flies in from below.")
			} else if toRoom.Sector == SECT_UNDERWATER {
				w.roomMessage(toRoom.VNum, "$n swims in from below.")
			} else {
				w.roomMessage(toRoom.VNum, "$n climbs in from below.")
			}
		case 5:
			if ch.IsAffected(affFly) {
				w.roomMessage(toRoom.VNum, "$n flies in from above.")
			} else if toRoom.Sector == SECT_UNDERWATER {
				w.roomMessage(toRoom.VNum, "$n swims in from above.")
			} else {
				w.roomMessage(toRoom.VNum, "$n climbs in from above.")
			}
		}
	}

	// Death trap check
	if roomHasFlagStatic(toRoom, roomFlagDeath) && ch.GetLevel() < lvlImmort {
		ch.TakeDamage(ch.GetHP() + 1)
		sendToChar(ch, "You have entered a death trap!\r\n")
		return false
	}

	return true
}

// roomHasFlagStatic checks a room's Flags slice for a string flag.
func roomHasFlagStatic(room *parser.Room, flag string) bool {
	for _, f := range room.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// performMove moves a character and all followers.
// Returns true on success.
func performMove(w *World, ch *Player, dir int, needSpecialsCheck bool) bool {
	if ch == nil || dir < 0 || dir >= len(dirs) {
		return false
	}

	ext, ok := getExit(w, ch, dir)
	if !ok || ext.ToRoom == -1 {
		sendToChar(ch, "Alas, you cannot go that way...\r\n")
		return false
	}

	if ext.DoorState == doorClosed || ext.DoorState == doorLocked {
		if ext.Keywords != "" && !strings.Contains(ext.Keywords, "secret") {
			sendToChar(ch, fmt.Sprintf("The %s seems to be closed.\r\n", firstWord(ext.Keywords)))
		} else {
			sendToChar(ch, "Alas, you cannot go that way...\r\n")
		}
		return false
	}

	wasIn := ch.RoomVNum
	if !doSimpleMove(w, ch, dir, needSpecialsCheck) {
		return false
	}

	// Followers
	followers := w.GetFollowers(ch.Name)
	for _, f := range followers {
		if f.GetRoom() == wasIn && f.GetPosition() >= combat.PosStanding {
			sendToChar(f, fmt.Sprintf("You follow %s.\r\n", ch.Name))
			f.SetAffect(affHide, false)
			performMove(w, f, dir, true)
		}
	}
	return true
}

// doMove maps ACMD command index to direction and executes move.
func doMove(w *World, ch *Player, cmd int) {
	performMove(w, ch, cmd-1, false)
}

// ---------------------------------------------------------------------------
// Door Commands
// ---------------------------------------------------------------------------

// findDoor locates a door by keyword or direction.
// Returns door index (0-5) or -1 on failure.
func findDoor(w *World, ch *Player, doorType, dir, cmdname string) int {
	if dir != "" {
		// A direction was specified
		door := searchBlock(dir, dirs, false)
		if door == -1 {
			sendToChar(ch, "That's not a direction.\r\n")
			return -1
		}
		ext, ok := getExit(w, ch, door)
		if !ok {
			sendToChar(ch, "I really don't see how you can do anything there.\r\n")
			return -1
		}
		if ext.Keywords != "" {
			if isName(doorType, ext.Keywords) {
				return door
			}
			sendToChar(ch, fmt.Sprintf("I see no %s there.\r\n", doorType))
			return -1
		}
		// No keywords on exit — it's just a direction, return the door index
		return door
	}

	// Try to locate by keyword
	if doorType == "" {
		sendToChar(ch, fmt.Sprintf("What is it you want to %s?\r\n", cmdname))
		return -1
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return -1
	}

	for door := 0; door < len(dirs); door++ {
		ext, ok := room.Exits[dirs[door]]
		if ok && ext.Keywords != "" && isName(doorType, ext.Keywords) {
			return door
		}
	}

	sendToChar(ch, fmt.Sprintf("There doesn't seem to be %s %s here.\r\n", an(doorType), doorType))
	return -1
}

// doDoorcmd executes a door subcommand (open/close/lock/unlock/pick).
func doDoorcmd(w *World, ch *Player, _ *ObjectInstance, door int, scmd int) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil || door < 0 || door >= len(dirs) {
		return
	}

	ext, hasExt := room.Exits[dirs[door]]
	if !hasExt {
		return
	}

	otherRoomVNum := ext.ToRoom
	var backExt parser.Exit
	hasBack := false
	if otherRoomVNum != -1 {
		otherRoom := w.GetRoomInWorld(otherRoomVNum)
		if otherRoom != nil {
			backDir := revDir[door]
			if backDir >= 0 && backDir < len(dirs) {
				backExt, hasBack = otherRoom.Exits[dirs[backDir]]
				if hasBack && backExt.ToRoom != ch.RoomVNum {
					hasBack = false
				}
			}
		}
	}

	doorName := "door"
	if ext.Keywords != "" {
		doorName = firstWord(ext.Keywords)
	}

	switch scmd {
	case scmdOpen:
		if ext.DoorState == doorClosed {
			ext.DoorState = doorOpen
			if hasBack {
				backExt.DoorState = doorOpen
			}
		}
		sendToChar(ch, "OK.\r\n")
		// Update the exit in room map
		room.Exits[dirs[door]] = ext
		if hasBack {
			otherRoom := w.GetRoomInWorld(otherRoomVNum)
			otherRoom.Exits[dirs[revDir[door]]] = backExt
		}

	case scmdClose:
		if ext.DoorState == doorOpen {
			ext.DoorState = doorClosed
			if hasBack {
				backExt.DoorState = doorClosed
			}
		}
		sendToChar(ch, "OK.\r\n")
		room.Exits[dirs[door]] = ext
		if hasBack {
			otherRoom := w.GetRoomInWorld(otherRoomVNum)
			otherRoom.Exits[dirs[revDir[door]]] = backExt
		}

	case scmdUnlock:
		if ext.DoorState == doorLocked {
			ext.DoorState = doorClosed
			if hasBack {
				backExt.DoorState = doorClosed
			}
		}
		sendToChar(ch, "*Click*\r\n")
		room.Exits[dirs[door]] = ext
		if hasBack {
			otherRoom := w.GetRoomInWorld(otherRoomVNum)
			otherRoom.Exits[dirs[revDir[door]]] = backExt
		}

	case scmdLock:
		if ext.DoorState == doorClosed {
			ext.DoorState = doorLocked
			if hasBack {
				backExt.DoorState = doorLocked
			}
		}
		sendToChar(ch, "*Click*\r\n")
		room.Exits[dirs[door]] = ext
		if hasBack {
			otherRoom := w.GetRoomInWorld(otherRoomVNum)
			otherRoom.Exits[dirs[revDir[door]]] = backExt
		}

	case scmdPick:
		if ext.DoorState == doorLocked {
			ext.DoorState = doorClosed
			if hasBack {
				backExt.DoorState = doorClosed
			}
		}
		sendToChar(ch, "The lock quickly yields to your skills.\r\n")
		room.Exits[dirs[door]] = ext
		if hasBack {
			otherRoom := w.GetRoomInWorld(otherRoomVNum)
			otherRoom.Exits[dirs[revDir[door]]] = backExt
		}
	}

	// Notify the room
	w.roomMessage(ch.RoomVNum, fmt.Sprintf("$n %ss the %s.", cmdDoor[scmd], doorName))

	// Notify the other room for open/close
	if (scmd == scmdOpen || scmd == scmdClose) && hasBack {
		backName := "door"
		if backExt.Keywords != "" {
			backName = firstWord(backExt.Keywords)
		}
		suffix := "ed"
		if scmd == scmdClose {
			suffix = "d"
		}
		msg := fmt.Sprintf("The %s %s %s%s from the other side.\r\n",
			backName, verbIs(backName), cmdDoor[scmd], suffix)
		players := w.GetPlayersInRoom(otherRoomVNum)
		for _, p := range players {
			p.SendMessage(msg)
		}
	}
}

// okPick checks whether a pick attempt succeeds.
func okPick(_ *World, ch *Player, keynum int, _ bool, _ int) bool {
	if keynum > 0 {
		sendToChar(ch, "The lock seems to be magical.\r\n")
		return false
	}

	percent := rand.Intn(101) + 1
	chance := 40 + (ch.GetLevel() * 5)
	if chance > 95 {
		chance = 95
	}
	if percent > chance {
		sendToChar(ch, "You failed to pick the lock.\r\n")
		return false
	}
	return true
}

// doGenDoor generic door command handler (open/close/lock/unlock/pick).
func doGenDoor(w *World, ch *Player, argument string, scmd int) {
	arg := strings.TrimSpace(argument)
	parts := strings.SplitN(arg, " ", 2)
	doorType := parts[0]
	dir := ""
	if len(parts) > 1 {
		dir = parts[1]
	}

	if doorType == "" {
		sendToChar(ch, fmt.Sprintf("What is it you want to %s?\r\n", cmdDoor[scmd]))
		return
	}

	door := findDoor(w, ch, doorType, dir, cmdDoor[scmd])
	if door == -1 {
		return
	}

	ext, ok := getExit(w, ch, door)
	if !ok {
		return
	}

	// Has keyword = proper door, else just direction exit
	if ext.Keywords != "" {
		// For lock/unlock, check key
		if scmd == scmdLock || scmd == scmdUnlock {
			if ext.Key > 0 && !hasKey(ch, ext.Key) {
				sendToChar(ch, "You don't seem to have the proper key.\r\n")
				return
			}
		}

		// Check exit state requirements
		needed := flagsDoor[scmd]
		if needed&needClosed != 0 && ext.DoorState != doorClosed && ext.DoorState != doorLocked {
			sendToChar(ch, "It's not closed.\r\n")
			return
		}
		if needed&needOpen != 0 && ext.DoorState != doorOpen {
			sendToChar(ch, "It's not open.\r\n")
			return
		}
		if needed&needUnlocked != 0 && ext.DoorState != doorClosed {
			sendToChar(ch, "It's not closed.\r\n")
			return
		}
		if needed&needLocked != 0 && ext.DoorState != doorLocked {
			sendToChar(ch, "It's not locked.\r\n")
			return
		}

		if scmd == scmdPick && ext.Key > 0 {
			// Magic lock — can't pick
			sendToChar(ch, "The lock seems to be magical.\r\n")
			return
		}

		if scmd == scmdPick {
			if !okPick(w, ch, ext.Key, false, scmd) {
				return
			}
		}
	}

	doDoorcmd(w, ch, nil, door, scmd)
}

// ---------------------------------------------------------------------------
// do_enter / do_leave
// ---------------------------------------------------------------------------

// doEnter handles the 'enter' command.
func doEnter(w *World, ch *Player, argument string) {
	arg := strings.TrimSpace(argument)
	if arg == "" {
		sendToChar(ch, "Enter what?\r\n")
		return
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return
	}

	for door := 0; door < len(dirs); door++ {
		ext, ok := room.Exits[dirs[door]]
		if !ok {
			continue
		}
		if ext.Keywords != "" && isName(arg, ext.Keywords) {
			if ext.DoorState == doorClosed || ext.DoorState == doorLocked {
				sendToChar(ch, "It seems to be closed.\r\n")
				return
			}
			doSimpleMove(w, ch, door, false)
			return
		}
	}

	sendToChar(ch, fmt.Sprintf("You don't see a %s here.\r\n", arg))
}

// doLeave handles the 'leave' command.
func doLeave(w *World, ch *Player) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return
	}

	for door := 0; door < len(dirs); door++ {
		ext, ok := room.Exits[dirs[door]]
		if !ok {
			continue
		}
		if ext.Keywords != "" && isName("leave", ext.Keywords) {
			if ext.DoorState == doorClosed || ext.DoorState == doorLocked {
				sendToChar(ch, "It seems to be closed.\r\n")
				return
			}
			doSimpleMove(w, ch, door, false)
			return
		}
	}

	sendToChar(ch, "You don't see a way out.\r\n")
}

// ---------------------------------------------------------------------------
// Position Commands: stand, sit, rest, sleep, wake
// ---------------------------------------------------------------------------

// doStand handles the 'stand' command.
func doStand(w *World, ch *Player) {
	switch ch.GetPosition() {
	case combat.PosSleeping:
		sendToChar(ch, "You wake and stand up.\r\n")
		w.roomMessage(ch.RoomVNum, "$n wakes and stands up.")
	case combat.PosSitting:
		sendToChar(ch, "You stand up.\r\n")
		w.roomMessage(ch.RoomVNum, "$n stands up.")
	case combat.PosResting:
		sendToChar(ch, "You stop resting and stand up.\r\n")
		w.roomMessage(ch.RoomVNum, "$n stops resting.")
	case combat.PosFighting:
		sendToChar(ch, "You are already fighting!\r\n")
		return
	case combat.PosStanding:
		sendToChar(ch, "You are already standing.\r\n")
		return
	}
	ch.SetPosition(combat.PosStanding)
	ch.SetAffect(affSleep, false)
}

// doSit handles the 'sit' command.
func doSit(w *World, ch *Player) {
	switch ch.GetPosition() {
	case combat.PosSleeping:
		sendToChar(ch, "You wake and sit up.\r\n")
		w.roomMessage(ch.RoomVNum, "$n wakes and sits up.")
	case combat.PosResting:
		sendToChar(ch, "You stop resting and sit up.\r\n")
		w.roomMessage(ch.RoomVNum, "$n sits up.")
	case combat.PosStanding:
		sendToChar(ch, "You sit down.\r\n")
		w.roomMessage(ch.RoomVNum, "$n sits down.")
	case combat.PosSitting:
		sendToChar(ch, "You are already sitting.\r\n")
		return
	case combat.PosFighting:
		sendToChar(ch, "Sit down? You are fighting!\r\n")
		return
	}
	ch.SetPosition(combat.PosSitting)
}

// doRest handles the 'rest' command.
func doRest(w *World, ch *Player) {
	switch ch.GetPosition() {
	case combat.PosSleeping:
		sendToChar(ch, "You wake and start resting.\r\n")
		w.roomMessage(ch.RoomVNum, "$n wakes and starts resting.")
	case combat.PosSitting:
		sendToChar(ch, "You rest.\r\n")
		w.roomMessage(ch.RoomVNum, "$n rests.")
	case combat.PosStanding:
		sendToChar(ch, "You sit down and rest.\r\n")
		w.roomMessage(ch.RoomVNum, "$n sits down and rests.")
	case combat.PosResting:
		sendToChar(ch, "You are already resting.\r\n")
		return
	case combat.PosFighting:
		sendToChar(ch, "Rest? You are fighting!\r\n")
		return
	}
	ch.SetPosition(combat.PosResting)
}

// doSleep handles the 'sleep' command.
func doSleep(w *World, ch *Player) {
	switch ch.GetPosition() {
	case combat.PosSleeping:
		sendToChar(ch, "You are already sleeping.\r\n")
	case combat.PosResting:
		sendToChar(ch, "You lie down and go to sleep.\r\n")
		w.roomMessage(ch.RoomVNum, "$n lies down and goes to sleep.")
		ch.SetPosition(combat.PosSleeping)
	case combat.PosSitting:
		sendToChar(ch, "You lie down and go to sleep.\r\n")
		w.roomMessage(ch.RoomVNum, "$n lies down and goes to sleep.")
		ch.SetPosition(combat.PosSleeping)
	case combat.PosStanding:
		sendToChar(ch, "You lie down and go to sleep.\r\n")
		w.roomMessage(ch.RoomVNum, "$n lies down and goes to sleep.")
		ch.SetPosition(combat.PosSleeping)
	case combat.PosFighting:
		sendToChar(ch, "Sleep? You are fighting!\r\n")
	}
}

// doWake handles the 'wake' command.
func doWake(w *World, ch *Player, argument string) {
	arg := strings.TrimSpace(argument)
	if arg == "" {
		if ch.GetPosition() == combat.PosSleeping {
			sendToChar(ch, "You wake.\r\n")
			w.roomMessage(ch.RoomVNum, "$n wakes.")
			ch.SetPosition(combat.PosStanding)
		} else {
			sendToChar(ch, "You are already awake.\r\n")
		}
		return
	}

	if ch.IsFighting() {
		sendToChar(ch, "You can't do that while fighting.\r\n")
		return
	}

	players := w.GetPlayersInRoom(ch.RoomVNum)
	for _, p := range players {
		if isName(arg, p.Name) && p.GetPosition() == combat.PosSleeping {
			sendToChar(p, fmt.Sprintf("You are awakened by %s.\r\n", ch.Name))
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("$n awakens %s.", p.Name))
			p.SetPosition(combat.PosStanding)
			return
		}
	}

	sendToChar(ch, "You don't see anyone sleeping by that name.\r\n")
}

// ---------------------------------------------------------------------------
// Follow Command
// ---------------------------------------------------------------------------

// doFollow handles the 'follow' command.
func doFollow(w *World, ch *Player, argument string) {
	arg := strings.TrimSpace(argument)
	if arg == "" {
		if ch.Following != "" {
			sendToChar(ch, fmt.Sprintf("You are currently following %s.\r\n", ch.Following))
		} else {
			sendToChar(ch, "Follow who?\r\n")
		}
		return
	}

	if strings.EqualFold(arg, ch.Name) {
		if ch.Following != "" {
			prevLeader := ch.Following
			ch.Following = ""
			ch.InGroup = false
			sendToChar(ch, "You stop following.\r\n")
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("$n stops following %s.", prevLeader))
		} else {
			sendToChar(ch, "You are not following anyone.\r\n")
		}
		return
	}

	if ch.IsAffected(affCharm) {
		sendToChar(ch, "You can't change what you follow while charmed!\r\n")
		return
	}

	target, ok := w.GetPlayer(arg)
	if !ok || target == nil {
		sendToChar(ch, fmt.Sprintf("You don't see '%s' here.\r\n", arg))
		return
	}

	if target == ch {
		sendToChar(ch, "You can't follow yourself (try 'cancel' to stop following).\r\n")
		return
	}

	ch.Following = target.Name
	sendToChar(ch, fmt.Sprintf("You now follow %s.\r\n", target.Name))
	sendToChar(target, fmt.Sprintf("%s now follows you.\r\n", ch.Name))
}

// ---------------------------------------------------------------------------
// Keyword matching helpers
// ---------------------------------------------------------------------------

// searchBlock returns the index of the first element with the given prefix.
// Returns -1 if not found.
func searchBlock(name string, list []string, exact bool) int {
	lower := strings.ToLower(name)
	for i, s := range list {
		if exact {
			if s == lower {
				return i
			}
		} else {
			if strings.HasPrefix(s, lower) {
				return i
			}
		}
	}
	return -1
}

// isName checks if name matches any keyword in the keyword string.
// Keywords are space-separated. Matching is partial (prefix).
// Replacement for C's isname().
func isName(name string, keywords string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, kw := range strings.Fields(keywords) {
		if strings.HasPrefix(strings.ToLower(kw), name) {
			return true
		}
	}
	return false
}

// firstWord returns the first word of a keyword string.
func firstWord(keywords string) string {
	parts := strings.Fields(keywords)
	if len(parts) > 0 {
		return parts[0]
	}
	return keywords
}

// an is defined in act_item.go

// verbIs returns "are" if the word ends with 's', "is" otherwise.
// Rough approximation of "this door is" vs "these doors are".
func verbIs(s string) string {
	if s != "" && s[len(s)-1] == 's' {
		return "are"
	}
	return "is"
}
