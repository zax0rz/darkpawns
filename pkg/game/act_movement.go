// Package game: act_movement.go — movement commands, door handling, sleep/rest/stand/sit/wake, follow.
//
// Ported from src/act.movement.c (CircleMUD / Dark Pawns MUD).
package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Constants ported from structs.h / constants.c
// ---------------------------------------------------------------------------

// Room flag string constants — parser stores these as string names.
const (
	roomFlagDeath  = "death" // used in doSimpleMove
	roomFlagTunnel = "tunnel"
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

// movementLoss is the single shared per-sector movement-cost table, indexed by
// the SECT_* enum (from src/constants.c movement_loss[]).
//
// WARNING — do NOT "fix" indices 8/9 by copying C's inline comments verbatim.
// C's array comments are swapped relative to the enum: the comment at index 8
// reads "/* Flying */" but the enum is SECT_UNDERWATER=8, and index 9 reads
// "/* Underwater */" but the enum is SECT_FLYING=9. The array is indexed by
// SECT(room) at runtime, so the VALUES win and the comments are noise. Runtime
// behavior: UNDERWATER (sector 8) = 2, FLYING (sector 9) = 6.
// See docs/briefs/BRIEF-2026-07-11-glm-dp1029-movement-cost.md.
var movementLoss = []int{
	2, // INSIDE (0)
	2, // CITY (1)
	3, // FIELD (2)
	4, // FOREST (3)
	5, // HILLS (4)
	7, // MOUNTAIN (5)
	5, // WATER_SWIM (6)
	6, // WATER_NOSWIM (7)
	2, // UNDERWATER (sector 8) — C comment LIES ("Flying"); enum wins
	6, // FLYING (sector 9) — C comment LIES ("Underwater"); enum wins
	8, // DESERT (10)
	6, // FIRE (11)
	6, // EARTH (12)
	6, // WIND (13)
	6, // WATER (14)
	4, // SWAMP (15)
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
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return parser.Exit{}, false
	}
	ext, ok := room.Exits[dirs[dir]]
	return ext, ok
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
	if ch.Inventory != nil {
		for _, obj := range ch.Inventory.Items {
			if obj != nil && obj.VNum == key {
				return true
			}
		}
	}
	if ch.Equipment != nil {
		if obj, ok := ch.Equipment.GetItemInSlot(SlotHold); ok && obj != nil && obj.VNum == key {
			return true
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
	if ch.IsAffected(affCharm) && ch.GetFollowing() != "" {
		if leader, ok := w.GetPlayer(ch.GetFollowing()); ok && ch.GetRoom() == leader.GetRoom() {
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

	room := w.GetRoomInWorld(ch.GetRoom())
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

	// Validate sector types before indexing movementLoss (DP-924).
	// C uses SECT() macro with no bounds check (UB on bad values);
	// Go panics on out-of-range, so we guard explicitly.
	if room.Sector < 0 || room.Sector >= len(movementLoss) ||
		toRoom.Sector < 0 || toRoom.Sector >= len(movementLoss) {
		sendToChar(ch, "You can't go that way.\r\n")
		return false
	}

	// Movement points needed is avg of src and dest sector movement loss
	needMovement := (movementLoss[room.Sector] + movementLoss[toRoom.Sector]) >> 1

	// Deduct movement if mortal
	if ch.GetLevel() < lvlImmort && !ch.SpendMove(needMovement) {
		if needSpecialsCheck && ch.GetFollowing() != "" {
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

	wasIn := ch.GetRoom()

	// Leave message
	if !ch.IsAffected(affSneak) {
		w.roomMessage(wasIn, fmt.Sprintf("$n leaves %s.", dirs[dir]))
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

	// MobProg greet: C-style trigger for specific mobs
	w.MpGreet(ch, ext.ToRoom)
	// Lua greet triggers for all mobs in the new room with the script
	if ScriptEngine != nil {
		for _, mob := range w.GetMobsInRoom(ext.ToRoom) {
			if mob.HasScript("greet") {
				ctx := mob.CreateScriptContext(ch, nil, "")
				ctx.World = NewWorldScriptableAdapter(w)
				ctx.RoomVNum = ext.ToRoom
				if _, err := mob.RunScript("greet", ctx); err != nil {
					slog.Warn("greet script error", "mob_vnum", mob.GetVNum(), "error", err)
				}
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

	if ext.ExitInfo&parser.ExitClosed != 0 {
		if ext.Keywords != "" && !strings.Contains(ext.Keywords, "secret") {
			sendToChar(ch, fmt.Sprintf("The %s seems to be closed.\r\n", firstWord(ext.Keywords)))
		} else {
			sendToChar(ch, "Alas, you cannot go that way...\r\n")
		}
		return false
	}

	wasIn := ch.GetRoom()
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

// ---------------------------------------------------------------------------
// Door Commands
// ---------------------------------------------------------------------------

// findDoor locates a door by keyword or direction.
// Returns door index (0-5) or -1 on failure.
func findDoor(w *World, ch *Player, doorType, dir, cmdname string) int {
	room := w.GetRoomInWorld(ch.GetRoom())
	revealsSecrets := room != nil && room.HasFlag(20)
	if dir != "" {
		// A direction was specified
		door := searchBlock(dir, dirs, false)
		if door == -1 {
			sendToChar(ch, "That's not a direction.\r\n")
			return -1
		}
		ext, ok := getExit(w, ch, door)
		if !ok || strings.Contains(strings.ToLower(ext.Keywords), "secret") && !revealsSecrets {
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

	if room == nil {
		return -1
	}

	for door := 0; door < len(dirs); door++ {
		ext, ok := room.Exits[dirs[door]]
		if ok && ext.Keywords != "" && isName(doorType, ext.Keywords) &&
			(!strings.Contains(strings.ToLower(ext.Keywords), "secret") || revealsSecrets) {
			return door
		}
	}

	sendToChar(ch, fmt.Sprintf("There doesn't seem to be %s %s here.\r\n", an(doorType), doorType))
	return -1
}

// doDoorcmd executes a container or exit subcommand after the exact C
// precondition ladder has succeeded.
func doDoorcmd(w *World, ch *Player, obj *ObjectInstance, door int, scmd int) {
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil || obj == nil && (door < 0 || door >= len(dirs)) {
		return
	}

	var ext parser.Exit
	if obj == nil {
		var hasExt bool
		ext, hasExt = room.Exits[dirs[door]]
		if !hasExt {
			return
		}
	}

	otherRoomVNum := -1
	var backExt parser.Exit
	hasBack := false
	if obj == nil {
		otherRoomVNum = ext.ToRoom
	}
	if obj == nil && otherRoomVNum != -1 {
		otherRoom := w.GetRoomInWorld(otherRoomVNum)
		if otherRoom != nil {
			backDir := revDir[door]
			if backDir >= 0 && backDir < len(dirs) {
				backExt, hasBack = otherRoom.Exits[dirs[backDir]]
				if hasBack && backExt.ToRoom != ch.GetRoom() {
					hasBack = false
				}
			}
		}
	}

	switch scmd {
	case scmdOpen:
		if obj != nil {
			obj.SetValue(contFlags, obj.GetValue(contFlags)^contClosed)
		} else {
			ext.ExitInfo ^= parser.ExitClosed
			w.SetExitInfo(ch.GetRoom(), dirs[door], ext.ExitInfo)
			if hasBack {
				backExt.ExitInfo ^= parser.ExitClosed
				w.SetExitInfo(otherRoomVNum, dirs[revDir[door]], backExt.ExitInfo)
			}
		}
		sendToChar(ch, "Okay.\r\n")

	case scmdClose:
		if obj != nil {
			obj.SetValue(contFlags, obj.GetValue(contFlags)^contClosed)
		} else {
			ext.ExitInfo ^= parser.ExitClosed
			w.SetExitInfo(ch.GetRoom(), dirs[door], ext.ExitInfo)
			if hasBack {
				backExt.ExitInfo ^= parser.ExitClosed
				w.SetExitInfo(otherRoomVNum, dirs[revDir[door]], backExt.ExitInfo)
			}
		}
		sendToChar(ch, "Okay.\r\n")

	case scmdUnlock:
		if obj != nil {
			obj.SetValue(contFlags, obj.GetValue(contFlags)^contLocked)
		} else {
			ext.ExitInfo ^= parser.ExitLocked
			w.SetExitInfo(ch.GetRoom(), dirs[door], ext.ExitInfo)
			if hasBack {
				backExt.ExitInfo ^= parser.ExitLocked
				w.SetExitInfo(otherRoomVNum, dirs[revDir[door]], backExt.ExitInfo)
			}
		}
		sendToChar(ch, "*Click*\r\n")

	case scmdLock:
		if obj != nil {
			obj.SetValue(contFlags, obj.GetValue(contFlags)^contLocked)
		} else {
			ext.ExitInfo ^= parser.ExitLocked
			w.SetExitInfo(ch.GetRoom(), dirs[door], ext.ExitInfo)
			if hasBack {
				backExt.ExitInfo ^= parser.ExitLocked
				w.SetExitInfo(otherRoomVNum, dirs[revDir[door]], backExt.ExitInfo)
			}
		}
		sendToChar(ch, "*Click*\r\n")

	case scmdPick:
		if obj != nil {
			obj.SetValue(contFlags, obj.GetValue(contFlags)^contLocked)
		} else {
			ext.ExitInfo ^= parser.ExitLocked
			w.SetExitInfo(ch.GetRoom(), dirs[door], ext.ExitInfo)
			if hasBack {
				backExt.ExitInfo ^= parser.ExitLocked
				w.SetExitInfo(otherRoomVNum, dirs[revDir[door]], backExt.ExitInfo)
			}
		}
		sendToChar(ch, "The lock quickly yields to your skills.\r\n")
	}

	// Notify the room
	roomFormat := fmt.Sprintf("$n %ss ", cmdDoor[scmd])
	if scmd == scmdPick {
		roomFormat = "$n skillfully picks the lock on "
	}
	if obj != nil {
		roomFormat += "$p."
		if obj.Location.Kind == ObjInRoom {
			Act(w, false, ch, nil, obj, nil, roomFormat, "", ToRoom)
		}
	} else {
		roomFormat += "the "
		if ext.Keywords != "" {
			roomFormat += "$F."
		} else {
			roomFormat += "door."
		}
		Act(w, false, ch, nil, nil, nil, roomFormat, ext.Keywords, ToRoom)
	}

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
		for _, actor := range w.actChar(otherRoomVNum) {
			actor.SendMessage(msg)
		}
	}
}

var doorNumber = number

// okPick checks whether a pick attempt succeeds.
func okPick(w *World, ch *Player, keynum int, pickproof bool, scmd int) bool {
	percent := doorNumber(1, 101)
	if scmd != scmdPick {
		return true
	}

	var picks *ObjectInstance
	if ch.Equipment != nil {
		picks, _ = ch.Equipment.GetItemInSlot(SlotHold)
	}
	canBreak := 0
	switch {
	case keynum < 0:
		sendToChar(ch, "Odd - you can't seem to find a keyhole.\r\n")
	case (picks == nil || picks.VNum != 8027) && ch.GetLevel() < lvlImmort:
		sendToChar(ch, "You'll need to hold a set of lockpicks before you can pick a lock!\r\n")
	case pickproof:
		sendToChar(ch, "It resists your attempts to pick it.\r\n")
		canBreak = 2
	case percent > ch.GetSkill(SkillPickLock):
		sendToChar(ch, "You failed to pick the lock.\r\n")
		canBreak = 1
	default:
		return true
	}

	if picks != nil && canBreak != 0 && ch.GetLevel() < doorNumber(0, 30)+canBreak {
		Act(w, false, ch, nil, nil, nil, "$n curses as $e bends some of $s lockpicks.", "", ToRoom)
		sendToChar(ch, "You ruin your lockpicks in the process.\r\n")
		if err := ch.Equipment.Unequip(SlotHold, ch.Inventory); err != nil {
			slog.Error("unequip ruined lockpicks", "player", ch.GetName(), "error", err)
			return false
		}
		w.ExtractObject(picks, ch.GetRoom())
		broken, err := w.SpawnObject(8028, -1)
		if err != nil {
			slog.Error("spawn broken lockpicks", "player", ch.GetName(), "error", err)
			return false
		}
		if err := ch.Equipment.Equip(broken, ch.Inventory); err != nil {
			slog.Error("equip broken lockpicks", "player", ch.GetName(), "error", err)
			if moveErr := w.MoveObject(broken, LocInventoryPlayer(ch.GetName())); moveErr != nil {
				slog.Error("rollback broken lockpicks to inventory", "player", ch.GetName(), "error", moveErr)
			}
		}
	}
	return false
}

// doGenDoor generic door command handler (open/close/lock/unlock/pick).
func doGenDoor(w *World, ch *Player, argument string, scmd int) {
	fields := strings.Fields(argument)
	if len(fields) == 0 {
		sendToChar(ch, strings.ToUpper(cmdDoor[scmd][:1])+cmdDoor[scmd][1:]+" what?\r\n")
		return
	}
	doorType := fields[0]
	dir := ""
	if len(fields) > 1 {
		dir = fields[1]
	}

	var obj *ObjectInstance
	if ch.Inventory != nil {
		for _, item := range ch.Inventory.Items {
			if item != nil && canSeeObject(ch, item) && isName(doorType, item.GetKeywords()) {
				obj = item
				break
			}
		}
	}
	if obj == nil {
		for _, item := range w.GetItemsInRoom(ch.GetRoom()) {
			if item != nil && canSeeObject(ch, item) && isName(doorType, item.GetKeywords()) {
				obj = item
				break
			}
		}
	}

	door := -1
	if obj == nil {
		door = findDoor(w, ch, doorType, dir, cmdDoor[scmd])
		if door < 0 {
			return
		}
	}

	var keynum int
	var openable, open, unlocked, pickproof bool
	if obj != nil {
		keynum = obj.GetValue(contKey)
		flags := obj.GetValue(contFlags)
		openable = obj.GetTypeFlag() == ITEM_CONTAINER && flags&contCloseable != 0
		open = flags&contClosed == 0
		unlocked = flags&contLocked == 0
		pickproof = flags&contPickproofBit != 0
	} else {
		ext, ok := getExit(w, ch, door)
		if !ok {
			return
		}
		keynum = ext.Key
		openable = ext.ExitInfo&parser.ExitIsDoor != 0
		open = ext.ExitInfo&parser.ExitClosed == 0
		unlocked = ext.ExitInfo&parser.ExitLocked == 0
		pickproof = ext.ExitInfo&parser.ExitPickproof != 0
	}

	needed := flagsDoor[scmd]
	switch {
	case !openable:
		Act(nil, false, ch, nil, nil, nil, "You can't $F that!", cmdDoor[scmd], ToChar)
	case !open && needed&needOpen != 0:
		sendToChar(ch, "But it's already closed!\r\n")
	case open && needed&needClosed != 0:
		sendToChar(ch, "But it's currently open!\r\n")
	case unlocked && needed&needLocked != 0:
		sendToChar(ch, "Oh.. it wasn't locked, after all..\r\n")
	case !unlocked && needed&needUnlocked != 0:
		sendToChar(ch, "It seems to be locked.\r\n")
	case !hasKey(ch, keynum) && ch.GetLevel() < LVL_GOD && (scmd == scmdLock || scmd == scmdUnlock):
		sendToChar(ch, "You don't seem to have the proper key.\r\n")
	case okPick(w, ch, keynum, pickproof, scmd):
		doDoorcmd(w, ch, obj, door, scmd)
	}
}

func (w *World) DoOpen(ch *Player, argument string)   { doGenDoor(w, ch, argument, scmdOpen) }
func (w *World) DoClose(ch *Player, argument string)  { doGenDoor(w, ch, argument, scmdClose) }
func (w *World) DoUnlock(ch *Player, argument string) { doGenDoor(w, ch, argument, scmdUnlock) }
func (w *World) DoLock(ch *Player, argument string)   { doGenDoor(w, ch, argument, scmdLock) }
func (w *World) DoPick(ch *Player, argument string)   { doGenDoor(w, ch, argument, scmdPick) }

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
