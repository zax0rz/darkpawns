package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// act_informative.go — ported from act.informative.c
// Player-level commands of an informative nature: look, exa, who, score, etc.
// ---------------------------------------------------------------------------

// Affect bit positions (from structs.h AFF_*)
const (
	affBlind      = 0  // AFF_BLIND
	affSenseLife  = 5  // AFF_SENSE_LIFE  Char can sense hidden life
	affInfravision = 10 // AFF_INFRAVISION Char can see in dark
)

// dirList is the canonical direction order.
var dirList = []string{"north", "east", "south", "west", "up", "down"}

// ---------------------------------------------------------------------------
// doLook — ACMD(do_look) — room, target, direction, or "read"
// ---------------------------------------------------------------------------

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

func (w *World) showObjToChar(obj *ObjectInstance, ch *Player, mode int) {
	var buf string

	switch {
	case mode == 0 && obj.Prototype.LongDesc != "":
		buf = obj.Prototype.LongDesc
	case mode == 5 || mode == 6:
		w.showObjExamine(obj, ch, mode == 6)
		return
	default:
		if obj.Prototype.ShortDesc != "" {
			buf = obj.Prototype.ShortDesc
		}
	}

	if mode != 3 {
		flags := w.getObjectExtraFlags(obj)
		if flags != "" {
			buf += " " + flags
		}
	}

	buf += "\r\n"
	ch.SendMessage(buf)
}

func (w *World) showObjExamine(obj *ObjectInstance, ch *Player, showExtras bool) {
	switch obj.Prototype.TypeFlag {
	case ITEM_NOTE:
		if obj.Prototype.ActionDesc != "" {
			ch.SendMessage("There is something written upon it:\r\n\r\n" + obj.Prototype.ActionDesc)
		} else {
			ch.SendMessage("It's blank.\r\n")
		}
		return
	case ITEM_DRINKCON:
		ch.SendMessage("A drink container.\r\n")
		return
	case ITEM_FOUNTAIN:
		ch.SendMessage("A fountain.\r\n")
		return
	case ITEM_CONTAINER:
		ch.SendMessage("A container.\r\n")
		return
	}
	if showExtras {
		flags := w.getObjectExtraFlags(obj)
		if flags != "" {
			ch.SendMessage("You see nothing special... " + flags + "\r\n")
			return
		}
	}
	ch.SendMessage("You see nothing special...\r\n")
}

func (w *World) getObjectExtraFlags(obj *ObjectInstance) string {
	var flags []string
	ef := obj.Prototype.ExtraFlags
	if len(ef) > 0 && ef[0]&1 != 0 {
		flags = append(flags, "(invisible)")
	}
	if len(ef) > 0 && ef[0]&4 != 0 {
		flags = append(flags, "(glowing)")
	}
	if len(ef) > 0 && ef[0]&8 != 0 {
		flags = append(flags, "(humming)")
	}
	if len(ef) > 1 && ef[1]&1 != 0 {
		flags = append(flags, "(blessed)")
	}
	return strings.Join(flags, " ")
}

// ---------------------------------------------------------------------------
// doScore — shows player stats (ACMD(do_score) in C)
// ---------------------------------------------------------------------------

func (w *World) doScore(ch *Player, me *MobInstance, cmd string, arg string) bool {
	classNames := map[int]string{
		0: "Magic-User", 1: "Cleric", 2: "Thief", 3: "Warrior",
	}
	raceNames := map[int]string{
		0: "Human", 1: "Elf", 2: "Dwarf", 3: "Halfling", 4: "Gnome", 5: "Kender",
	}
	cn := classNames[ch.Class]
	if cn == "" {
		cn = "Unknown"
	}
	rn := raceNames[ch.Race]
	if rn == "" {
		rn = "Unknown"
	}
	ch.SendMessage(fmt.Sprintf("You are %s.\r\n", ch.GetName()))
	ch.SendMessage(fmt.Sprintf("Level %d %s %s.\r\n", ch.Level, rn, cn))
	ch.SendMessage(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d\r\n",
		ch.Health, ch.MaxHealth,
		ch.Mana, ch.MaxMana,
		ch.Move, ch.MaxMove))
	ch.SendMessage(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d\r\n",
		ch.Stats.Str, ch.Stats.Int, ch.Stats.Wis,
		ch.Stats.Dex, ch.Stats.Con, ch.Stats.Cha))
	ch.SendMessage(fmt.Sprintf("AC: %d  Experience: %d  Gold: %d\r\n",
		ch.AC, ch.Exp, ch.Gold))
	return true
}

// ---------------------------------------------------------------------------
// doWho — shows connected players (ACMD(do_who) in C)
// ---------------------------------------------------------------------------

func (w *World) doWho(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Players currently connected:\r\n")
	for _, p := range w.players {
		if p.Flags&PLR_INVISIBLE != 0 && ch.Level < LVL_IMMORT {
			continue
		}
		if p.Level >= 31 {
			ch.SendMessage(fmt.Sprintf("[%5s] %-12s %s\r\n", w.getWhoTitle(p), p.GetName(), "God"))
		} else {
			ch.SendMessage(fmt.Sprintf("[%5s] %-12s %s\r\n", w.getWhoTitle(p), p.GetName(), "Adventurer"))
		}
	}
	ch.SendMessage("\r\n")
	return true
}

func (w *World) getWhoTitle(ch *Player) string {
	return fmt.Sprintf("%2d", ch.Level)
}

// ---------------------------------------------------------------------------
// doInventory — shows carried items (ACMD(do_inventory) in C)
// ---------------------------------------------------------------------------

func (w *World) doInventory(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("You are carrying:\r\n")
	for i, item := range ch.Inventory.Items {
		if item == nil {
			continue
		}
		ch.SendMessage(fmt.Sprintf("[%2d] %s\r\n", i+1, item.Prototype.ShortDesc))
	}
	return true
}

// ---------------------------------------------------------------------------
// doEquipment — shows worn items (ACMD(do_equipment) in C)
// ---------------------------------------------------------------------------

func (w *World) doEquipment(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("You are using:\r\n")
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if !ok {
			continue
		}
		ch.SendMessage(fmt.Sprintf("%-15s : %s\r\n", slot.String(), item.Prototype.ShortDesc))
	}
	return true
}

// ---------------------------------------------------------------------------
// doWhere — shows mobs in world (ACMD(do_where) in C)
// ---------------------------------------------------------------------------

func (w *World) doWhere(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Players in your area:\r\n")
	for _, p := range w.players {
		if p.RoomVNum == ch.RoomVNum {
			ch.SendMessage(fmt.Sprintf("%-20s : here\r\n", p.GetName()))
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// doLevels — shows class level titles (ACMD(do_levels) in C)
// ---------------------------------------------------------------------------

func (w *World) doLevels(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Level progression:\r\n")
	for lvl := 1; lvl <= ch.Level && lvl <= 50; lvl++ {
		ch.SendMessage(fmt.Sprintf("Level %2d: %d exp\r\n", lvl, 1000*lvl*lvl))
	}
	return true
}

// ---------------------------------------------------------------------------
// doColor / doToggle — configuration commands
// ---------------------------------------------------------------------------

func (w *World) doColor(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Color is not yet implemented.\r\n")
	return true
}

func (w *World) doToggle(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Toggles are not yet implemented.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doAbils / doSkills — shows abilities and learned skills
// ---------------------------------------------------------------------------

func (w *World) doAbils(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Abilities:\r\n")
	abilNames := []string{"Str", "Int", "Wis", "Dex", "Con", "Cha"}
	abilVals := []int{ch.Stats.Str, ch.Stats.Int, ch.Stats.Wis,
		ch.Stats.Dex, ch.Stats.Con, ch.Stats.Cha}
	for i := range abilNames {
		ch.SendMessage(fmt.Sprintf("%s: %d\r\n", abilNames[i], abilVals[i]))
	}
	return true
}

func (w *World) doSkills(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Skills:\r\n")
	for _, skill := range ch.SkillManager.GetLearnedSkills() {
		ch.SendMessage(fmt.Sprintf("%-20s %3d%%\r\n", skill.Name, skill.Level))
	}
	return true
}

// ---------------------------------------------------------------------------
// doUsers — shows all descriptors (ACMD(do_users) in C)
// ---------------------------------------------------------------------------

func (w *World) doUsers(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Connected users:\r\n")
	for _, p := range w.players {
		ch.SendMessage(fmt.Sprintf("%-20s %6d hp  Room %d\r\n",
			p.GetName(), p.Health, p.RoomVNum))
	}
	return true
}

// ---------------------------------------------------------------------------
// doExamine — examine objects or mobs
// ---------------------------------------------------------------------------

func (w *World) doExamine(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Examine what?\r\n")
		return true
	}
	w.lookAtTarget(ch, arg)

	// If it's a container, show contents
	obj := w.findObjNear(ch, arg)
	if obj != nil && (obj.Prototype.TypeFlag == ITEM_DRINKCON ||
		obj.Prototype.TypeFlag == ITEM_FOUNTAIN ||
		obj.Prototype.TypeFlag == ITEM_CONTAINER) {
		ch.SendMessage("When you look inside, you see:\r\n")
		w.lookInObj(ch, arg)
	}
	return true
}

// ---------------------------------------------------------------------------
// doCoins — shows money on person
// ---------------------------------------------------------------------------

func (w *World) doCoins(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage(fmt.Sprintf("You have %d gold coins.\r\n", ch.Gold))
	return true
}

// ---------------------------------------------------------------------------
// doDescription — sets character description
// ---------------------------------------------------------------------------

func (w *World) doDescription(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.Description = arg
	ch.SendMessage("Description set.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doCommands — shows available commands
// ---------------------------------------------------------------------------

func (w *World) doCommands(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Commands available:\r\n")
	ch.SendMessage("look, read, examine, score, who, inventory, equipment\r\n")
	ch.SendMessage("time, weather, exits, consider, where\r\n")
	ch.SendMessage("north, east, south, west, up, down\r\n")
	ch.SendMessage("get, drop, give, put, wear, wield, remove\r\n")
	ch.SendMessage("open, close, lock, unlock\r\n")
	ch.SendMessage("kill, backstab, flee, assist, hit, murder\r\n")
	ch.SendMessage("sneak, hide, steal, practice, train, level\r\n")
	ch.SendMessage("skills, abilities, help, save, quit\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doDiagnose — check a mob's condition
// ---------------------------------------------------------------------------

func (w *World) doDiagnose(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Diagnose not yet fully implemented.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doConsider — compare target to player (ACMD(do_consider) in C)
// ---------------------------------------------------------------------------

func (w *World) doConsider(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Consider who?\r\n")
		return true
	}
	ch.SendMessage("You consider your options...\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doHelp — shows help topics (ACMD(do_help) in C)
// ---------------------------------------------------------------------------

func (w *World) doHelp(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Usage: help <topic>\r\n")
		return true
	}
	ch.SendMessage(fmt.Sprintf("No help on '%s' available.\r\n", arg))
	return true
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func splitArg(arg string) (string, string) {
	arg = strings.TrimSpace(arg)
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func indexOf(list []string, item string) int {
	for i, s := range list {
		if s == item {
			return i
		}
	}
	return -1
}

func chCanSee(ch *Player, target interface{}) bool {
	if ch.IsAffected(affBlind) {
		return false
	}
	return true
}

// mobCanSee checks whether a mob can see. Uses the mob's AffectFlags for blindness.
func mobCanSee(m *MobInstance) bool {
	if m.Prototype != nil {
		for _, aff := range m.Prototype.AffectFlags {
			if strings.EqualFold(aff, "BLIND") {
				return false
			}
		}
	}
	return true
}

func chCanSeeObj(ch *Player, obj *ObjectInstance) bool {
	ef := obj.Prototype.ExtraFlags
	if len(ef) > 0 && ef[0]&1 != 0 {
		// ITEM_INVISIBLE - immortals can see invisible items
		if ch.Level >= 31 {
			return true
		}
		return false
	}
	return chCanSee(ch, nil)
}

func chCanSeeInDark(ch *Player) bool {
	return ch.IsAffected(affInfravision) || ch.Level >= 31
}

func (w *World) isRoomDark(vnum int) bool {
	room := w.GetRoomInWorld(vnum)
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if f == "dark" {
			return true
		}
	}
	return false
}

func (w *World) roomIsDeath(room *parser.Room) bool {
	for _, f := range room.Flags {
		if f == "death" {
			return true
		}
	}
	return false
}

// findCharInRoom finds a character by name in the same room.
// Returns the player and mob — exactly one will be non-nil.
func (w *World) findCharInRoom(ch *Player, roomVNum int, name string) (*Player, *MobInstance) {
	argLower := strings.ToLower(name)
	// Check players first
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(p.GetName()), argLower) {
			return p, nil
		}
	}
	// Check mobs
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(m.Prototype.ShortDesc), argLower) {
			return nil, m
		}
	}
	return nil, nil
}

// findObjNear finds an object near the player (inventory, equipment, room).
func (w *World) findObjNear(ch *Player, name string) *ObjectInstance {
	argLower := strings.ToLower(name)
	// Check inventory
	for _, item := range ch.Inventory.Items {
		if item != nil && strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	// Check equipment
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if ok && strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	// Check room items
	for _, item := range w.roomItems[ch.RoomVNum] {
		if strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Under-ported helpers (from act.informative.c / act.wizard.c / spec_procs2.c)
// ---------------------------------------------------------------------------

// FindTargetRoom resolves a target room string to a VNum (from act.wizard.c:184).
func (w *World) FindTargetRoom(ch *Player, raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return -1
	}
	vnum := 0
	if _, err := fmt.Sscanf(raw, "%d", &vnum); err == nil && vnum > 0 {
		if _, ok := w.rooms[vnum]; ok {
			return vnum
		}
		return -1
	}
	lower := strings.ToLower(raw)
	for vnum, room := range w.rooms {
		if room == nil {
			continue
		}
		if strings.Contains(strings.ToLower(room.Name), lower) {
			return vnum
		}
	}
	return -1
}

// PrintObjectLocation formats where an object is (in room, carried, worn, inside another).
func (w *World) PrintObjectLocation(num int, obj *ObjectInstance, ch *Player, recur bool) string {
	var b strings.Builder
	if num > 0 {
		b.WriteString(fmt.Sprintf("O%3d. %-25s - ", num, obj.Prototype.ShortDesc))
	} else {
		b.WriteString(fmt.Sprintf("%33s", " - "))
	}
	switch {
	case obj.RoomVNum > 0:
		if room, ok := w.rooms[obj.RoomVNum]; ok && room != nil {
			b.WriteString(fmt.Sprintf("[%5d] %s\r\n", obj.RoomVNum, room.Name))
		} else {
			b.WriteString(fmt.Sprintf("[%5d] (unknown room)\r\n", obj.RoomVNum))
		}
	case obj.Location.Kind == ObjInInventory || obj.Location.Kind == ObjEquipped:
		name := "someone"
		if obj.Location.OwnerKind == OwnerPlayer {
			if p, ok := w.players[obj.Location.PlayerName]; ok {
				name = p.GetName()
			}
		} else if obj.Location.OwnerKind == OwnerMob {
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				name = m.GetName()
			}
		}
		if obj.Location.Kind == ObjEquipped {
			b.WriteString(fmt.Sprintf("worn by %s\r\n", name))
		} else {
			b.WriteString(fmt.Sprintf("carried by %s\r\n", name))
		}
	case obj.Location.Kind == ObjInContainer:
		if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok {
			b.WriteString(fmt.Sprintf("inside %s\r\n", container.Prototype.ShortDesc))
		} else {
			b.WriteString("in an unknown container\r\n")
		}
	default:
		b.WriteString("in an unknown location\r\n")
	}
	return b.String()
}

// KenderSteal attempts to pilfer a random item from a mob (from spec_procs2.c:594).
func (w *World) KenderSteal(ch *Player, mob *MobInstance) {
	if mob == nil || len(mob.Inventory) == 0 {
		return
	}
	for _, obj := range mob.Inventory {
		if obj == nil || obj.Prototype == nil || !chCanSeeObj(ch, obj) {
			continue
		}
		if int(w.randPct()%601) >= ch.GetLevel() {
			continue
		}
		percent := int(w.randPct()%100) + 1
		if mob.GetPosition() < posSleeping {
			percent = -1
		}
		if ch.GetLevel() >= lvlImmort {
			percent = 101
		}
		if ch.GetLevel() <= 10 || mob.GetLevel() <= 10 {
			return
		}
		if percent < 0 {
			mob.RemoveFromInventory(obj)
// #nosec G104
			ch.Inventory.AddItem(obj)
			ch.SendMessage("You stealthily filch an item.\r\n")
			return
		}
	}
}

// FindClassBitvector maps a class letter to a bitvector bit for skill/spell filtering.
func FindClassBitvector(arg byte) int64 {
	switch arg {
	case 'm':
		return 1 << 0 // mage
	case 'c':
		return 1 << 1 // cleric
	case 't':
		return 1 << 2 // thief
	case 'w':
		return 1 << 3 // warrior
	case 'a':
		return 1 << 4 // magus
	case 'v':
		return 1 << 5 // avatar
	case 's':
		return 1 << 6 // assassin
	case 'p':
		return 1 << 7 // paladin
	case 'n':
		return 1 << 8 // ninja
	case 'i':
		return 1 << 9 // psionic
	default:
		return 0
	}
}

// randPct returns a simple pseudo-random uint64 for game RNG needs.
func (w *World) randPct() uint64 {
	return uint64(time.Now().UnixNano()) * 6364136223846793005 % (1 << 32)
}
