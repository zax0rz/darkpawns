package game

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ObservationResult is the game-owned boundary for look-family commands.
// Rules and visibility are resolved here; transports only render Messages and
// translate Room into their existing structured payload.
type ObservationResult struct {
	Messages []ObservationMessage
	Room     *RoomView
	Events   []SemanticEvent
}

// ObservationMessage is a deferred act() call. Literal marks world-authored
// descriptions whose '$' bytes must not be interpreted as act substitutions.
// Observation is single-viewer, so every message is routed ToChar.
type ObservationMessage struct {
	Format        string
	Literal       bool
	HideInvisible bool
	Actor         Actor
	Target        Actor
	Object        *ObjectInstance
	TargetObject  *ObjectInstance
	Argument      string
}

// SemanticEvent reserves the typed event portion of the result boundary for
// later consumers. Observation is read-only, so current operations emit none.
type SemanticEvent struct {
	Type string
}

// RoomView contains viewer-resolved room facts used by structured clients.
// The WebSocket adapter preserves the existing StateData/RoomState schema.
type RoomView struct {
	VNum        int
	Name        string
	Description string
	Exits       []string
	Doors       []DoorView
	Players     []string
	Mobs        []string
	Items       []string
}

// DoorView is the game-owned representation of a visible room door.
type DoorView struct {
	Direction string
	Closed    bool
	Locked    bool
}

func (r *ObservationResult) literal(ch *Player, text string) {
	r.Messages = append(r.Messages, ObservationMessage{
		Format:  normalizeObservationText(text),
		Literal: true,
		Actor:   ch,
	})
}

func (r *ObservationResult) act(ch *Player, target Actor, obj *ObjectInstance, format string) {
	r.Messages = append(r.Messages, ObservationMessage{
		Format: format,
		Actor:  ch,
		Target: target,
		Object: obj,
	})
}

// RenderObservationMessages is the shared text adapter. Both telnet and
// WebSocket sessions receive these viewer-resolved messages through the
// canonical act() delivery path.
func (w *World) RenderObservationMessages(result ObservationResult) {
	for _, message := range result.Messages {
		format := message.Format
		if message.Literal {
			format = strings.ReplaceAll(format, "$", "$$")
		}
		Act(
			w,
			message.HideInvisible,
			message.Actor,
			message.Target,
			message.Object,
			message.TargetObject,
			format,
			message.Argument,
			ToChar,
		)
	}
}

// DoLook routes look/read syntax to the canonical observation operations.
func (w *World) DoLook(ch *Player, cmd, arg string) ObservationResult {
	var result ObservationResult
	if ch == nil {
		return result
	}
	if ch.GetPosition() < posSleeping {
		result.literal(ch, "You can't see anything but stars!")
		return result
	}
	if ch.IsAffected(affBlind) {
		result.literal(ch, "You can't see a damned thing, you're blind!")
		return result
	}

	first, rest := splitArg(strings.ToLower(arg))
	if strings.EqualFold(cmd, "read") {
		if first == "" {
			result.literal(ch, "Read what?")
			return result
		}
		return w.DoLookTarget(ch, first)
	}

	switch {
	case first == "":
		return w.DoLookRoom(ch, true)
	case isAbbrev(first, "in"):
		return w.DoLookIn(ch, rest)
	case directionIndex(first) >= 0:
		return w.DoLookDirection(ch, directionIndex(first))
	case isAbbrev(first, "at"):
		return w.DoLookTarget(ch, rest)
	default:
		return w.DoLookTarget(ch, first)
	}
}

// DoLookRoom observes the viewer's current room.
func (w *World) DoLookRoom(ch *Player, ignoreBrief bool) ObservationResult {
	if ch == nil {
		return ObservationResult{}
	}
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		var result ObservationResult
		result.literal(ch, "You are in a void.")
		return result
	}
	return w.observeRoom(ch, room, ignoreBrief, true)
}

// DoLookRoomAt observes a specific room without moving the viewer. It is used
// by entry-state rendering and preserves the same visibility rules as a normal
// room look.
func (w *World) DoLookRoomAt(ch *Player, roomVNum int, ignoreBrief bool) ObservationResult {
	if ch == nil {
		return ObservationResult{}
	}
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		var result ObservationResult
		result.literal(ch, "You are in a void.")
		return result
	}
	return w.observeRoom(ch, room, ignoreBrief, true)
}

func (w *World) observeRoom(ch *Player, room *parser.Room, ignoreBrief, includeView bool) ObservationResult {
	var result ObservationResult
	if room == nil {
		result.literal(ch, "You are in a void.")
		return result
	}

	isBlind := ch.IsAffected(affBlind)
	isDark := w.isRoomDark(room.VNum) && !chCanSeeInDark(ch)
	if isBlind || isDark {
		cyan, normal := observationColors(ch, "\x1b[36m"), observationColors(ch, "\x1b[0m")
		if isBlind {
			result.literal(ch, cyan+"Darkness\r\n"+normal+"\r\nYou see nothing but infinite darkness...")
			return result
		}
		result.literal(ch, cyan+"Darkness\r\n"+normal+"\r\nIt is too dark here to see much of anything...")
		w.appendDarkOccupants(&result, ch, room)
		return result
	}

	showDescription := ch.GetFlags()&(1<<uint(PrfBrief)) == 0 || ignoreBrief || w.roomIsDeath(room)
	view := w.buildRoomView(ch, room, showDescription)
	if includeView {
		result.Room = &view
	}

	cyan, normal := observationColors(ch, "\x1b[36m"), observationColors(ch, "\x1b[0m")
	roomName := room.Name
	if ch.GetRoomFlags() {
		roomName = formatRoomFlags(room)
	}
	result.literal(ch, cyan+roomName+normal)
	if showDescription && room.Description != "" {
		result.literal(ch, room.Description)
	}
	if ch.GetAutoExit() {
		autoExits := w.autoExitsText(ch, room)
		// The C command path's ignore_brief room render carries one leading
		// spacer before autoexits; directional/movement room renders do not.
		if ignoreBrief && !ch.GetRoomFlags() {
			autoExits = " " + autoExits
		}
		result.literal(ch, autoExits)
	}

	green, yellow := observationColors(ch, "\x1b[32m"), observationColors(ch, "\x1b[33m")
	for _, line := range w.roomObjectLines(ch, room) {
		result.literal(ch, green+line+normal)
	}
	for _, line := range w.roomCharacterLines(ch, room, &view) {
		result.literal(ch, yellow+line+normal)
	}
	return result
}

func (w *World) appendDarkOccupants(result *ObservationResult, ch *Player, room *parser.Room) {
	for _, mob := range sortedMobs(w.GetMobsInRoom(room.VNum)) {
		if mob == nil {
			continue
		}
		if mob.IsAffected(affSneak) || mob.IsAffected(affHide) {
			continue
		}
		if mob.IsAffected(affInfravision) {
			result.literal(ch, "You see a pair of glowing red eyes.")
		} else {
			result.literal(ch, "You hear someone or something moving around nearby.")
		}
	}
	for _, player := range sortedPlayers(w.GetPlayersInRoom(room.VNum)) {
		if player == nil || player == ch || player.GetLevel() >= LVL_IMMORT {
			continue
		}
		if player.IsAffected(affSneak) || player.IsAffected(affHide) {
			continue
		}
		if player.IsAffected(affInfravision) {
			result.literal(ch, "You see a pair of glowing red eyes.")
		} else {
			result.literal(ch, "You hear someone or something moving around nearby.")
		}
	}
}

func (w *World) buildRoomView(ch *Player, room *parser.Room, showDescription bool) RoomView {
	view := RoomView{
		VNum:  room.VNum,
		Name:  room.Name,
		Exits: make([]string, 0, len(room.Exits)),
	}
	if showDescription {
		view.Description = normalizeObservationText(room.Description)
	}
	for _, direction := range dirList {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		view.Exits = append(view.Exits, direction)
		if exit.Keywords != "" || exit.DoorState != doorOpen {
			view.Doors = append(view.Doors, DoorView{
				Direction: direction,
				Closed:    exit.DoorState != doorOpen,
				Locked:    exit.DoorState == doorLocked,
			})
		}
	}
	for _, item := range w.GetItemsInRoom(room.VNum) {
		if item != nil && chCanSeeObj(ch, item) {
			view.Items = append(view.Items, normalizeObservationText(item.GetLongDesc()))
		}
	}
	return view
}

func (w *World) roomObjectLines(ch *Player, room *parser.Room) []string {
	type group struct {
		line  string
		count int
		item  *ObjectInstance
	}
	groups := make(map[string]*group)
	order := make([]string, 0)
	for _, item := range w.GetItemsInRoom(room.VNum) {
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		line := normalizeObservationText(item.GetLongDesc())
		if line == "" {
			line = normalizeObservationText(item.GetShortDesc())
		}
		key := fmt.Sprintf("%d:%v:%d:%s", item.GetVNum(), item.GetExtraFlags(), item.GetWeight(), line)
		if existing := groups[key]; existing != nil {
			existing.count++
			continue
		}
		groups[key] = &group{line: line, count: 1, item: item}
		order = append(order, key)
	}
	// C's oc_add_front builds this display list in reverse discovery order.
	lines := make([]string, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		entry := groups[order[i]]
		line := entry.line + objectVisibleFlags(ch, entry.item)
		if entry.count > 1 {
			line += fmt.Sprintf("[%d]", entry.count)
		}
		lines = append(lines, line)
	}
	return lines
}

func (w *World) roomCharacterLines(ch *Player, room *parser.Room, view *RoomView) []string {
	var lines []string
	for _, mob := range sortedMobs(w.GetMobsInRoom(room.VNum)) {
		if mob == nil {
			continue
		}
		if !canSee(ch, mob) || mob.IsAffected(affHide) {
			if mob.IsAffected(affHide) && ch.IsAffected(affSenseLife) {
				lines = append(lines, "You sense a hidden presence in the room.")
			}
			continue
		}
		line := mobPresenceLine(mob, ch)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		view.Mobs = append(view.Mobs, line)
	}
	for _, player := range sortedPlayers(w.GetPlayersInRoom(room.VNum)) {
		if player == nil || player == ch {
			continue
		}
		if !canSee(ch, player) || player.IsAffected(affHide) {
			if player.IsAffected(affHide) && ch.IsAffected(affSenseLife) {
				lines = append(lines, "You sense a hidden presence in the room.")
			}
			continue
		}
		line := playerPresenceLine(player, ch)
		lines = append(lines, line)
		view.Players = append(view.Players, player.GetName())
	}
	return lines
}

// DoLookTarget resolves room features and nearby objects in the authoritative
// observation order, then characters. Room extra descriptions deliberately
// win over objects with the same keyword.
func (w *World) DoLookTarget(ch *Player, arg string) ObservationResult {
	var result ObservationResult
	arg = strings.TrimSpace(arg)
	if arg == "" {
		result.literal(ch, "Look at what?")
		return result
	}
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		result.literal(ch, "You do not see that here.")
		return result
	}

	foundObj := w.findObservationObject(ch, arg)
	if desc, ok := findExtraDescription(arg, room.ExtraDescs); ok {
		result.literal(ch, desc)
		return result
	}
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if !ok || !chCanSeeObj(ch, item) {
			continue
		}
		if desc, found := findExtraDescription(arg, item.GetExtraDescs()); found {
			result.literal(ch, desc)
			appendObjectLook(&result, ch, item, 6)
			return result
		}
	}
	for _, item := range ch.Inventory.Items {
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		if desc, found := findExtraDescription(arg, item.GetExtraDescs()); found {
			result.literal(ch, desc)
			appendObjectLook(&result, ch, item, 6)
			return result
		}
	}
	for _, item := range w.GetItemsInRoom(room.VNum) {
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		if desc, found := findExtraDescription(arg, item.GetExtraDescs()); found {
			result.literal(ch, desc)
			appendObjectLook(&result, ch, item, 6)
			return result
		}
	}
	if target, ok := w.ResolveCharInRoom(ch, arg); ok {
		w.appendCharacterLook(&result, ch, target)
		return result
	}
	if foundObj != nil {
		appendObjectLook(&result, ch, foundObj, 5)
		return result
	}
	result.literal(ch, "You do not see that here.")
	return result
}

func (w *World) appendCharacterLook(result *ObservationResult, ch *Player, target CharTarget) {
	if target.Player != nil {
		player := target.Player
		if description := player.GetDescription(); description != "" {
			result.literal(ch, description)
		} else {
			result.act(ch, player, nil, "You see nothing special about $M.")
		}
		race := strings.ToLower(RaceNames[player.GetRace()])
		result.literal(ch, fmt.Sprintf("%s is %s.", player.GetName(), race))
		result.act(ch, player, nil, "$N "+diagCondition(player.GetHP(), player.GetMaxHP()))
		appendPlayerEquipment(result, ch, player)
		return
	}
	if target.Mob != nil {
		mob := target.Mob
		if mob.Prototype != nil && mob.Prototype.DetailedDesc != "" {
			result.literal(ch, mob.Prototype.DetailedDesc)
		} else {
			result.act(ch, mob, nil, "You see nothing special about $M.")
		}
		result.act(ch, mob, nil, "$N "+diagCondition(mob.GetHP(), mob.GetMaxHP()))
		appendMobEquipment(result, ch, mob)
	}
}

// DoLookDirection renders C's direction preface, exit state, and—when open—the
// complete destination room without moving the character.
func (w *World) DoLookDirection(ch *Player, direction int) ObservationResult {
	var result ObservationResult
	if ch.IsFighting() {
		result.literal(ch, "You're a little busy right now!")
		return result
	}
	if direction < 0 || direction >= len(dirList) {
		return result
	}
	result.literal(ch, fmt.Sprintf("You look %swards.", dirList[direction]))
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return result
	}
	exit, ok := w.getExitForDirection(room, direction)
	if !ok {
		return result
	}
	if exit.Description != "" {
		result.literal(ch, exit.Description)
		return result
	}
	secret := strings.Contains(strings.ToLower(exit.Keywords), "secret")
	revealSecret := roomHasFlagBit(room.Flags, 20)
	if exit.DoorState != doorOpen && exit.Keywords != "" && (revealSecret || !secret) {
		result.literal(ch, fmt.Sprintf("The %s is closed.", fname(exit.Keywords)))
	} else if exit.DoorState == doorOpen && exit.Keywords != "" && (revealSecret || !secret) {
		result.literal(ch, fmt.Sprintf("The %s is open.", fname(exit.Keywords)))
	}
	if secret && !revealSecret {
		result.literal(ch, "You see nothing special.")
	}
	if exit.ToRoom <= 0 || exit.DoorState != doorOpen {
		return result
	}
	destination := w.GetRoomInWorld(exit.ToRoom)
	if destination == nil {
		result.literal(ch, "You see nothing special.")
		return result
	}
	destinationResult := w.observeRoom(ch, destination, false, false)
	result.Messages = append(result.Messages, destinationResult.Messages...)
	return result
}

// DoLookIn renders container and drink-container contents using instance
// values and searches inventory, room, and equipment.
func (w *World) DoLookIn(ch *Player, arg string) ObservationResult {
	var result ObservationResult
	arg = strings.TrimSpace(arg)
	if arg == "" {
		result.literal(ch, "Look in what?")
		return result
	}
	object, location := w.findObservationObjectWithLocation(ch, arg)
	if object == nil {
		article := "a"
		if startsWithVowel(arg) {
			article = "an"
		}
		result.literal(ch, fmt.Sprintf("There doesn't seem to be %s %s here.", article, arg))
		return result
	}
	if object.GetTypeFlag() != ITEM_DRINKCON && object.GetTypeFlag() != ITEM_FOUNTAIN && object.GetTypeFlag() != ITEM_CONTAINER {
		result.literal(ch, "There's nothing inside that!")
		return result
	}
	if object.GetTypeFlag() == ITEM_CONTAINER {
		if object.GetValue(contFlags)&contClosed != 0 {
			result.literal(ch, "It is closed.")
			return result
		}
		where := map[observationObjectLocation]string{
			observationInventory: "carried",
			observationRoom:      "here",
			observationEquipment: "used",
		}[location]
		result.literal(ch, fmt.Sprintf("%s (%s): ", fname(object.GetKeywords()), where))
		contents := visibleObjectShortLines(ch, object.GetContents())
		if len(contents) == 0 {
			result.literal(ch, "Nothing.")
		} else {
			for _, line := range contents {
				result.literal(ch, line)
			}
		}
		return result
	}

	capacity, amount, liquid := object.GetValue(0), object.GetValue(1), object.GetValue(2)
	if amount <= 0 {
		result.literal(ch, "It is empty.")
		return result
	}
	if capacity <= 0 || amount > capacity {
		result.literal(ch, "Its contents seem somewhat murky.")
		return result
	}
	fullness := []string{"less than half ", "about half ", "more than half ", ""}
	index := (amount * 3) / capacity
	if index < 0 || index >= len(fullness) {
		index = 0
	}
	color := "UNDEFINED"
	if liquid >= 0 && liquid < len(Liquids) {
		color = Liquids[liquid].Color
	}
	result.literal(ch, fmt.Sprintf("It's %sfull of a %s liquid.", fullness[index], color))
	return result
}

// DoExits lists only open obvious exits, with immortal vnums exactly where C
// exposes them.
func (w *World) DoExits(ch *Player) ObservationResult {
	var result ObservationResult
	if ch.IsAffected(affBlind) {
		result.literal(ch, "You can't see a damned thing, you're blind!")
		return result
	}
	room := w.GetRoomInWorld(ch.GetRoom())
	result.literal(ch, "Obvious exits:")
	if room == nil {
		result.literal(ch, " None.")
		return result
	}
	count := 0
	for _, direction := range dirList {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom <= 0 || exit.DoorState != doorOpen {
			continue
		}
		destination := w.GetRoomInWorld(exit.ToRoom)
		if destination == nil {
			continue
		}
		label := strings.ToUpper(direction[:1]) + direction[1:]
		if ch.GetLevel() >= LVL_IMMORT {
			result.literal(ch, fmt.Sprintf("%-5s - [%5d] %s", label, destination.VNum, destination.Name))
		} else if w.isRoomDark(destination.VNum) && !chCanSeeInDark(ch) {
			result.literal(ch, fmt.Sprintf("%-5s - Too dark to tell", label))
		} else {
			result.literal(ch, fmt.Sprintf("%-5s - %s", label, destination.Name))
		}
		count++
	}
	if count == 0 {
		result.literal(ch, " None.")
	}
	return result
}

// DoExamine is C's look-at-target followed by the container contents probe.
func (w *World) DoExamine(ch *Player, arg string) ObservationResult {
	first, _ := splitArg(arg)
	if first == "" {
		var result ObservationResult
		result.literal(ch, "Examine what?")
		return result
	}
	result := w.DoLookTarget(ch, first)
	object := w.findObservationObject(ch, first)
	if object == nil {
		return result
	}
	if object.GetTypeFlag() == ITEM_DRINKCON || object.GetTypeFlag() == ITEM_FOUNTAIN || object.GetTypeFlag() == ITEM_CONTAINER {
		result.literal(ch, "When you look inside, you see:")
		inside := w.DoLookIn(ch, first)
		result.Messages = append(result.Messages, inside.Messages...)
	}
	return result
}

// DoDiagnose ports do_diagnose/diag_char_to_char. With no explicit target C
// diagnoses the current opponent, otherwise it asks who.
func (w *World) DoDiagnose(ch *Player, arg string) ObservationResult {
	var result ObservationResult
	first, _ := splitArg(arg)
	if first == "" {
		first = ch.GetFighting()
		if first == "" {
			result.literal(ch, "Diagnose who?")
			return result
		}
	}
	target, ok := w.ResolveCharInRoom(ch, first)
	if !ok {
		result.literal(ch, "No-one by that name here.")
		return result
	}
	if target.Player != nil {
		result.act(ch, target.Player, nil, "$N "+diagCondition(target.Player.GetHP(), target.Player.GetMaxHP()))
	} else if target.Mob != nil {
		result.act(ch, target.Mob, nil, "$N "+diagCondition(target.Mob.GetHP(), target.Mob.GetMaxHP()))
	}
	return result
}

// Legacy game callers (mobprogs/specprocs/death display) now consume the same
// canonical results instead of maintaining a second observation implementation.
func (w *World) doLook(ch *Player, me *MobInstance, cmd, arg string) bool {
	_ = me
	result := w.DoLook(ch, cmd, arg)
	w.RenderObservationMessages(result)
	return true
}

func (w *World) lookAtRoom(ch *Player, ignoreBrief bool) {
	w.RenderObservationMessages(w.DoLookRoom(ch, ignoreBrief))
}

func (w *World) listObjToChar(room *parser.Room, ch *Player) {
	var result ObservationResult
	for _, line := range w.roomObjectLines(ch, room) {
		result.literal(ch, line)
	}
	w.RenderObservationMessages(result)
}

func (w *World) getExitForDirection(room *parser.Room, direction int) (parser.Exit, bool) {
	if direction < 0 || direction >= len(dirList) {
		return parser.Exit{}, false
	}
	exit, ok := room.Exits[dirList[direction]]
	return exit, ok
}

func (w *World) autoExitsText(ch *Player, room *parser.Room) string {
	var exits []string
	for _, direction := range dirList {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		if exit.DoorState != doorOpen {
			if ch.GetLevel() >= LVL_IMMORT {
				exits = append(exits, "("+direction+")")
			}
			continue
		}
		exits = append(exits, direction)
	}
	if len(exits) == 0 {
		exits = append(exits, "None!")
	}
	cyan, normal := observationColors(ch, "\x1b[36m"), observationColors(ch, "\x1b[0m")
	return fmt.Sprintf("%s[ Exits: %s ]%s", cyan, strings.Join(exits, " "), normal)
}

func diagCondition(hp, maxHP int) string {
	percent := -1
	if maxHP > 0 {
		percent = (100 * hp) / maxHP
	}
	switch {
	case percent >= 100:
		return "is in excellent condition."
	case percent >= 90:
		return "has a few scratches."
	case percent >= 75:
		return "has some small wounds and bruises."
	case percent >= 50:
		return "has quite a few wounds."
	case percent >= 30:
		return "has some big nasty wounds and scratches."
	case percent >= 15:
		return "looks pretty hurt."
	case percent >= 0:
		return "is in awful condition."
	default:
		return "is bleeding awfully from big wounds."
	}
}

func appendObjectLook(result *ObservationResult, ch *Player, object *ObjectInstance, mode int) {
	if object == nil {
		return
	}
	if mode == 6 {
		flags := objectVisibleFlags(ch, object)
		if flags == "" {
			// C show_obj_to_char(mode 6) emits an otherwise empty line.
			result.literal(ch, " ")
		} else {
			result.literal(ch, flags)
		}
		return
	}
	if object.GetTypeFlag() == ITEM_NOTE {
		if object.Prototype != nil && object.Prototype.ActionDesc != "" {
			result.literal(ch, "There is something written upon it:\r\n\r\n"+object.Prototype.ActionDesc)
		} else {
			result.literal(ch, "It's blank.")
		}
		return
	}
	text := "You see nothing special.."
	if object.GetTypeFlag() == ITEM_DRINKCON {
		text = "It looks like a drink container."
	}
	result.literal(ch, text+objectVisibleFlags(ch, object))
}

func objectVisibleFlags(ch *Player, object *ObjectInstance) string {
	if object == nil {
		return ""
	}
	var flags []string
	if object.HasExtraFlag(0, 5) {
		flags = append(flags, "(invisible)")
	}
	if object.HasExtraFlag(0, 8) && ch.IsAffected(affDetectAlign) {
		flags = append(flags, "(blue glow)")
	}
	if object.HasExtraFlag(0, 6) && ch.IsAffected(affDetectMagic) {
		flags = append(flags, "(yellow glow)")
	}
	if object.HasExtraFlag(0, 0) {
		flags = append(flags, "(glowing)")
	}
	if object.HasExtraFlag(0, 1) {
		flags = append(flags, "(humming)")
	}
	if len(flags) == 0 {
		return ""
	}
	return " " + strings.Join(flags, " ")
}

func appendPlayerEquipment(result *ObservationResult, ch, target *Player) {
	if target.Equipment == nil {
		return
	}
	items := target.Equipment.GetEquippedItems()
	if len(items) == 0 {
		return
	}
	result.act(ch, target, nil, "\r\n$N is using:")
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item := items[slot]
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		result.literal(ch, fmt.Sprintf("%-20s%s%s", equipmentWhere(slot), item.GetShortDesc(), objectVisibleFlags(ch, item)))
	}
}

func appendMobEquipment(result *ObservationResult, ch *Player, target *MobInstance) {
	if len(target.Equipment) == 0 {
		return
	}
	result.act(ch, target, nil, "\r\n$N is using:")
	positions := make([]int, 0, len(target.Equipment))
	for position := range target.Equipment {
		positions = append(positions, position)
	}
	sort.Ints(positions)
	for _, position := range positions {
		item := target.Equipment[position]
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		where := "<used>             "
		if position >= 0 && position < len(WhereNames) {
			where = WhereNames[position]
		}
		result.literal(ch, where+item.GetShortDesc()+objectVisibleFlags(ch, item))
	}
}

func equipmentWhere(slot EquipmentSlot) string {
	switch slot {
	case SlotLight:
		return "<used as light>"
	case SlotFinger, SlotFingerR, SlotFingerL:
		return "<worn on finger>"
	case SlotNeck, SlotNeck1, SlotNeck2:
		return "<worn around neck>"
	case SlotBody:
		return "<worn on body>"
	case SlotHead:
		return "<worn on head>"
	case SlotLegs:
		return "<worn on legs>"
	case SlotFeet:
		return "<worn on feet>"
	case SlotHands:
		return "<worn on hands>"
	case SlotArms:
		return "<worn on arms>"
	case SlotShield:
		return "<worn as shield>"
	case SlotAbout:
		return "<worn about body>"
	case SlotWaist:
		return "<worn about waist>"
	case SlotWrist, SlotWristR, SlotWristL:
		return "<worn around wrist>"
	case SlotWield:
		return "<wielded>"
	case SlotHold:
		return "<held>"
	case SlotEar:
		return "<worn on ear>"
	default:
		return "<used>"
	}
}

type observationObjectLocation int

const (
	observationNowhere observationObjectLocation = iota
	observationInventory
	observationRoom
	observationEquipment
)

func (w *World) findObservationObject(ch *Player, name string) *ObjectInstance {
	object, _ := w.findObservationObjectWithLocation(ch, name)
	return object
}

func (w *World) findObservationObjectWithLocation(ch *Player, name string) (*ObjectInstance, observationObjectLocation) {
	for _, item := range ch.Inventory.Items {
		if observationObjectMatches(ch, item, name) {
			return item, observationInventory
		}
	}
	for _, item := range w.GetItemsInRoom(ch.GetRoom()) {
		if observationObjectMatches(ch, item, name) {
			return item, observationRoom
		}
	}
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if ok && observationObjectMatches(ch, item, name) {
			return item, observationEquipment
		}
	}
	return nil, observationNowhere
}

func observationObjectMatches(ch *Player, object *ObjectInstance, name string) bool {
	return object != nil && chCanSeeObj(ch, object) && isnameWithAbbrevs(name, object.GetKeywords())
}

func findExtraDescription(name string, descriptions []parser.ExtraDesc) (string, bool) {
	for _, description := range descriptions {
		if isnameWithAbbrevs(name, description.Keywords) {
			return description.Description, true
		}
	}
	return "", false
}

func visibleObjectShortLines(ch *Player, objects []*ObjectInstance) []string {
	var lines []string
	for _, object := range objects {
		if object != nil && chCanSeeObj(ch, object) {
			lines = append(lines, object.GetShortDesc()+objectVisibleFlags(ch, object))
		}
	}
	return lines
}

func playerPresenceLine(player, viewer *Player) string {
	name := player.GetName()
	title := strings.TrimSpace(player.GetTitle())
	if title == "" {
		title = "the " + ClassNames[player.GetClass()]
	}
	if title != "" {
		name += " " + title
	}
	if player.IsAffected(affInvisible) {
		name += " (invisible)"
	}
	if player.IsAffected(affHide) {
		name += " (hidden)"
	}
	if player.GetPosition() == posFighting {
		target := player.GetFighting()
		if target == "" {
			return name + " is here struggling with thin air."
		}
		if strings.EqualFold(target, viewer.GetName()) {
			target = "YOU!"
		} else {
			target += "!"
		}
		return name + " is here, fighting " + target
	}
	return name + positionPresence(player.GetPosition())
}

func mobPresenceLine(mob *MobInstance, viewer *Player) string {
	if mob.Prototype != nil && mob.Prototype.LongDesc != "" && mob.GetPosition() == mob.Prototype.DefaultPos {
		return normalizeObservationText(mob.Prototype.LongDesc)
	}
	name := mob.GetShortDesc()
	if mob.IsAffected(affInvisible) {
		name = "*" + name
	}
	if mob.GetPosition() == posFighting {
		if mob.FightingTarget == "" {
			return name + " is here struggling with thin air."
		}
		target := mob.FightingTarget
		if strings.EqualFold(target, viewer.GetName()) {
			target = "YOU!"
		} else {
			target += "!"
		}
		return name + " is here, fighting " + target
	}
	return name + positionPresence(mob.GetPosition())
}

func positionPresence(position int) string {
	positions := []string{
		" is lying here, dead.",
		" is lying here, mortally wounded.",
		" is lying here, incapacitated.",
		" is lying here, stunned.",
		" is sleeping here.",
		" is resting here.",
		" is sitting here.",
		"!FIGHTING!",
		" is standing here.",
	}
	if position < 0 || position >= len(positions) {
		return " is standing here."
	}
	return positions[position]
}

func formatRoomFlags(room *parser.Room) string {
	roomFlagNames := []string{
		"DARK", "DEATH", "!MOB", "INDOORS", "PEACEFUL", "SOUNDPROOF", "!TRACK", "!MAGIC",
		"TUNNEL", "PRIVATE", "GODROOM", "HOUSE", "HCRSH", "ATRIUM", "OLC", "*", "NEUTRAL",
		"BFR", "REGENROOM", "NO_WHO_ROOM", "**", "FLOW_NORTH", "FLOW_SOUTH", "FLOW_EAST",
		"FLOW_WEST", "FLOW_UP", "FLOW_DOWN", "ARENA",
	}
	var flags []string
	for bit, name := range roomFlagNames {
		if roomHasFlagBit(room.Flags, bit) {
			flags = append(flags, name)
		}
	}
	flagText := "NOBITS"
	if len(flags) > 0 {
		flagText = strings.Join(flags, " ")
	}
	sector := "Unknown"
	sectorNames := []string{
		"Inside", "City", "Field", "Forest", "Hills", "Mountain",
		"Water Swim", "Water Noswim", "Underwater", "Flying", "Desert",
		"Fire", "Earth", "Wind", "Water",
	}
	if room.Sector >= 0 && room.Sector < len(sectorNames) {
		sector = sectorNames[room.Sector]
	}
	return fmt.Sprintf("[%5d] %s [ %s ] [ %s ]", room.VNum, room.Name, flagText, sector)
}

func observationColors(ch *Player, code string) string {
	flags := ch.GetFlags()
	level := 0
	if flags&(1<<uint(PrfColor1)) != 0 {
		level++
	}
	if flags&(1<<uint(PrfColor2)) != 0 {
		level += 2
	}
	if level < 2 {
		return ""
	}
	return code
}

func directionIndex(name string) int {
	for index, direction := range dirList {
		if isAbbrev(name, direction) {
			return index
		}
	}
	return -1
}

func normalizeObservationText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimRight(text, "\n")
	return strings.ReplaceAll(text, "\n", "\r\n")
}

func startsWithVowel(text string) bool {
	if text == "" {
		return false
	}
	return strings.ContainsRune("aeiouyAEIOUY", rune(text[0]))
}

func sortedPlayers(players []*Player) []*Player {
	result := append([]*Player(nil), players...)
	sort.Slice(result, func(i, j int) bool { return result[i].GetName() < result[j].GetName() })
	return result
}

func sortedMobs(mobs []*MobInstance) []*MobInstance {
	result := append([]*MobInstance(nil), mobs...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].GetVNum() != result[j].GetVNum() {
			return result[i].GetVNum() < result[j].GetVNum()
		}
		return result[i].ID < result[j].ID
	})
	return result
}
