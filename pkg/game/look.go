package game

import (
	"fmt"
	"strings"
)
import "github.com/zax0rz/darkpawns/pkg/parser"

func (w *World) doLook(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetPosition() < posSleeping {
		ch.SendMessage("You can't see anything but stars!\r\n")
		return true
	}
	if ch.IsAffected(affBlind) {
		ch.SendMessage("You can't see a damned thing, you're blind!\r\n")
		return true
	}

	first, rest := splitArg(arg)

	if cmd == "read" {
		if first == "" {
			ch.SendMessage("Read what?\r\n")
		} else {
			w.lookAtTarget(ch, first)
		}
		return true
	}

	if first == "" {
		w.lookAtRoom(ch, true)
	} else if first == "in" {
		w.lookInObj(ch, rest)
	} else if idx := indexOf(dirList, first); idx >= 0 {
		w.lookInDirection(ch, idx)
	} else if first == "at" {
		w.lookAtTarget(ch, rest)
	} else {
		w.lookAtTarget(ch, first)
	}
	return true
}

// ---------------------------------------------------------------------------
// lookAtRoom — renders full room description
// ---------------------------------------------------------------------------

func (w *World) lookAtRoom(ch *Player, ignoreBrief bool) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		ch.SendMessage("You are in a void.\r\n")
		return
	}

	// Dark or blind check
	isDark := w.isRoomDark(room.VNum)
	isBlind := ch.IsAffected(affBlind)

	if isBlind || (isDark && !chCanSeeInDark(ch)) {
		if isBlind {
			ch.SendMessage("Darkness\r\n\r\n")
			ch.SendMessage("You see nothing but infinite darkness...\r\n")
		} else {
			ch.SendMessage("Darkness\r\n\r\n")
			ch.SendMessage("It is too dark here to see much of anything...\r\n")
		}
		// List dark-detectable mobs & players
		for _, m := range w.GetMobsInRoom(room.VNum) {
			if m.VNum == ch.ID {
				continue
			}
			ch.SendMessage("You hear someone or something moving around nearby.\r\n")
		}
		for _, p := range w.GetPlayersInRoom(room.VNum) {
			if p.GetName() == ch.GetName() {
				continue
			}
			if p.Level >= 31 {
				continue
			}
			if p.IsAffected(affSneak) {
				continue
			}
			if p.IsAffected(affHide) {
				continue
			}
			ch.SendMessage("You hear someone or something moving around nearby.\r\n")
		}
		return
	}

	// Room name line
	roomLine := room.Name
	if ch.RoomFlags {
		roomLine = fmt.Sprintf("[%5d] %s", room.VNum, room.Name)
	}
	ch.SendMessage(roomLine + "\r\n\r\n")

	// Room description
	if !ch.AutoExit || ignoreBrief || w.roomIsDeath(room) {
		if room.Description != "" {
			ch.SendMessage(room.Description + "\r\n")
		}
	}

	// Autoexits
	if ch.AutoExit {
		w.doAutoExits(ch)
	}

	// List objects in room
	w.listObjToChar(room, ch)

	// List characters in room
	w.listCharToChar(room, ch)
}

// ---------------------------------------------------------------------------
// listObjToChar — lists visible objects in room (port of list_obj_to_char)
// ---------------------------------------------------------------------------

func (w *World) listObjToChar(room *parser.Room, ch *Player) {
	items := w.roomItems[room.VNum]
	if len(items) == 0 {
		return
	}
	// Group by short desc
	type group struct {
		shortDesc string
		count     int
	}
	groups := make(map[string]*group)
	var order []string
	for _, item := range items {
		if !chCanSeeObj(ch, item) {
			continue
		}
		sd := item.Prototype.ShortDesc
		if sd == "" {
			sd = item.Prototype.LongDesc
		}
		if g, ok := groups[sd]; ok {
			g.count++
		} else {
			groups[sd] = &group{shortDesc: sd, count: 1}
			order = append(order, sd)
		}
	}
	for _, k := range order {
		g := groups[k]
		if g.count > 1 {
			ch.SendMessage(fmt.Sprintf("%s [%d here]\r\n", g.shortDesc, g.count))
		} else {
			ch.SendMessage(g.shortDesc + "\r\n")
		}
	}
}

// ---------------------------------------------------------------------------
// listCharToChar — lists visible mobs/players in room (port of list_char_to_char)
// ---------------------------------------------------------------------------

func (w *World) listCharToChar(room *parser.Room, ch *Player) {
	// Mobs
	for _, m := range w.GetMobsInRoom(room.VNum) {
		if m.VNum == ch.ID {
			continue
		}
		if !chCanSee(ch, m) {
			continue
		}
		w.listOneChar(ch, m)
	}

	// Players
	for _, p := range w.GetPlayersInRoom(room.VNum) {
		if p.GetName() == ch.GetName() {
			continue
		}
		if p.Level >= 31 {
			continue
		}
		if p.IsAffected(affHide) {
			if ch.IsAffected(affSenseLife) {
				ch.SendMessage("You sense a hidden presence in the room.\r\n")
			}
			continue
		}
		w.listOneChar(ch, p)
	}
}

// listOneChar — prints "<name> is standing here."-style line
func (w *World) listOneChar(ch *Player, target interface{}) {
	var name string
	switch v := target.(type) {
	case *Player:
		name = v.GetName()
	case *MobInstance:
		name = v.GetShortDesc()
		if name == "" {
			name = v.Prototype.ShortDesc
		}
	}
	ch.SendMessage(name + " is here.\r\n")
}

// ---------------------------------------------------------------------------
// lookAtTarget — port of look_at_target
// ---------------------------------------------------------------------------

func (w *World) lookAtTarget(ch *Player, arg string) {
	if arg == "" {
		ch.SendMessage("Look at what?\r\n")
		return
	}

	// Search: mobs then players in room, then objects
	foundPlayer, foundMob := w.findCharInRoom(ch, ch.RoomVNum, arg)
	foundObj := w.findObjNear(ch, arg)
	var found bool

	if foundPlayer != nil {
		w.lookAtChar(ch, foundPlayer)
		return
	}
	if foundMob != nil {
		w.lookAtChar(ch, foundMob)
		return
	}

	// Extra descs on room objects
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room != nil {
		for _, item := range w.roomItems[room.VNum] {
			for _, ed := range item.Prototype.ExtraDescs {
				if strings.Contains(strings.ToLower(ed.Keywords), strings.ToLower(arg)) {
					ch.SendMessage(ed.Description + "\r\n")
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// Extra descs on inventory items
	if !found {
		for _, item := range ch.Inventory.Items {
			if item == nil {
				continue
			}
			for _, ed := range item.Prototype.ExtraDescs {
				if strings.Contains(strings.ToLower(ed.Keywords), strings.ToLower(arg)) {
					ch.SendMessage(ed.Description + "\r\n")
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// Extra descs on equipped items
	if !found {
		for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
			item, ok := ch.Equipment.GetItemInSlot(slot)
			if !ok {
				continue
			}
			for _, ed := range item.Prototype.ExtraDescs {
				if strings.Contains(strings.ToLower(ed.Keywords), strings.ToLower(arg)) {
					ch.SendMessage(ed.Description + "\r\n")
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	if foundObj != nil {
		if !found {
			w.showObjToChar(foundObj, ch, 5)
		} else {
			w.showObjToChar(foundObj, ch, 6)
		}
	} else if !found {
		ch.SendMessage("You do not see that here.\r\n")
	}
}

// ---------------------------------------------------------------------------
// lookAtChar — describes a character to the looker
// ---------------------------------------------------------------------------

func (w *World) lookAtChar(ch *Player, target interface{}) {
	var buf string
	switch v := target.(type) {
	case *Player:
		buf = fmt.Sprintf("%s is in excellent condition.\r\n", v.GetName())
	case *MobInstance:
		shortDesc := v.GetShortDesc()
		if shortDesc == "" {
			shortDesc = v.Prototype.ShortDesc
		}
		buf = fmt.Sprintf("%s is in excellent condition.\r\n", shortDesc)
	}
	ch.SendMessage(buf)
}

// ---------------------------------------------------------------------------
// lookInDirection — port of look_in_direction
// ---------------------------------------------------------------------------

func (w *World) lookInDirection(ch *Player, dir int) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		ch.SendMessage("You are in a void.\r\n")
		return
	}

	exit, ok := w.getExitForDirection(room, dir)
	if !ok || exit.ToRoom <= 0 {
		ch.SendMessage("No exit that way.\r\n")
		return
	}

	if exit.DoorState > 0 {
		ch.SendMessage("The door is closed.\r\n")
		return
	}

	dest := w.GetRoomInWorld(exit.ToRoom)
	if dest == nil {
		ch.SendMessage("Nothing special there...\r\n")
		return
	}

	if w.isRoomDark(dest.VNum) && !chCanSeeInDark(ch) {
		ch.SendMessage("It's too dark to see.\r\n")
		return
	}

	ch.SendMessage(dest.Name + "\r\n")
	w.listObjToChar(dest, ch)
	w.listCharToChar(dest, ch)
}

// ---------------------------------------------------------------------------
// lookInObj — port of look_in_obj
// ---------------------------------------------------------------------------

func (w *World) lookInObj(ch *Player, arg string) {
	obj := w.findObjNear(ch, arg)
	if obj == nil {
		ch.SendMessage("You do not see that here.\r\n")
		return
	}
	if obj.Prototype.TypeFlag != ITEM_DRINKCON &&
		obj.Prototype.TypeFlag != ITEM_FOUNTAIN &&
		obj.Prototype.TypeFlag != ITEM_CONTAINER {
		ch.SendMessage("That is not a container.\r\n")
		return
	}
	ch.SendMessage("When you look inside, you see:\r\n")
	// List container contents
	contents := obj.GetContents()
	if len(contents) == 0 {
		ch.SendMessage("  It's empty.\r\n")
	} else {
		for _, item := range contents {
			ch.SendMessage(fmt.Sprintf("  %s\r\n", item.GetShortDesc()))
		}
	}
}

// ---------------------------------------------------------------------------
// doAutoExits — shows abbreviated exit list (for autoexit pref)
// ---------------------------------------------------------------------------

func (w *World) doAutoExits(ch *Player) {
	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return
	}

	var exits []string
	for _, dir := range dirList {
		exit, ok := room.Exits[dir]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		if exit.DoorState > 0 {
			if ch.Level >= 31 {
				exits = append(exits, fmt.Sprintf("(%s)", dir))
			}
			continue
		}
		exits = append(exits, dir)
	}

	if len(exits) == 0 {
		ch.SendMessage("[Exits: None! ]\r\n")
	} else {
		ch.SendMessage(fmt.Sprintf("[Exits: %s ]\r\n", strings.Join(exits, " ")))
	}
}

// ---------------------------------------------------------------------------
// doExits — verbose exit listing (ACMD(do_exits))
// ---------------------------------------------------------------------------

func (w *World) doExits(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsAffected(affBlind) {
		ch.SendMessage("You can't see a damned thing, you're blind!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		ch.SendMessage("You're in a void.\r\n")
		return true
	}

	ch.SendMessage("Obvious exits:\r\n")
	anyExit := false
	for _, dir := range dirList {
		exit, ok := room.Exits[dir]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		anyExit = true
		if exit.DoorState > 0 {
			continue
		}
		dest := w.GetRoomInWorld(exit.ToRoom)
		if dest == nil {
			ch.SendMessage(fmt.Sprintf("%-5s - somewhere\r\n", dir))
			continue
		}
		if ch.Level >= 31 {
			ch.SendMessage(fmt.Sprintf("%-5s - [%5d] %s\r\n", dir, dest.VNum, dest.Name))
		} else if w.isRoomDark(dest.VNum) && !chCanSeeInDark(ch) {
			ch.SendMessage(fmt.Sprintf("%-5s - Too dark to tell\r\n", dir))
		} else {
			ch.SendMessage(fmt.Sprintf("%-5s - %s\r\n", dir, dest.Name))
		}
	}
	if !anyExit {
		ch.SendMessage(" None.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// getExitForDirection resolves a direction index to a room exit.
// ---------------------------------------------------------------------------

func (w *World) getExitForDirection(room *parser.Room, dir int) (parser.Exit, bool) {
	if dir >= 0 && dir < len(dirList) {
		exit, ok := room.Exits[dirList[dir]]
		return exit, ok
	}
	return parser.Exit{}, false
}

// ---------------------------------------------------------------------------
// showObjToChar — displays object info to character (mode 0-6)
// ---------------------------------------------------------------------------

