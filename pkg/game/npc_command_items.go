package game

import "strings"

// CharacterPut ports do_put for a mobile body. C resolves the body from
// descriptor->character, while the descriptor remains the immortal's; the
// object's inventory and room lookup therefore belong to the mobile.
func CharacterPut(ctx CharacterCommandContext) CharacterCommandResult {
	me, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	return ctx.World.npcPutDirect(me, ctx.RawArgs)
}

// CharacterWear ports do_wear for a mobile body. The player equipment path is
// intentionally left alone; only the shared dispatcher selects this callback
// when the resolved body is a mobile.
func CharacterWear(ctx CharacterCommandContext) CharacterCommandResult {
	me, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	return ctx.World.npcWearDirect(me, ctx.RawArgs)
}

// CharacterWield ports do_wield for a mobile body.
func CharacterWield(ctx CharacterCommandContext) CharacterCommandResult {
	me, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	return ctx.World.npcWearAtDirect(me, ctx.RawArgs, eqWearWield, "Wield")
}

// CharacterGrab ports do_grab for a mobile body. The hold and grab entries
// have separate command gates but share this C handler.
func CharacterGrab(ctx CharacterCommandContext) CharacterCommandResult {
	me, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	return ctx.World.npcGrabDirect(me, ctx.RawArgs)
}

// CharacterRemove ports do_remove for a mobile body.
func CharacterRemove(ctx CharacterCommandContext) CharacterCommandResult {
	me, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	return ctx.World.npcRemoveDirect(me, ctx.RawArgs)
}

func (w *World) npcPutDirect(me *MobInstance, argument string) CharacterCommandResult {
	itemArg, containerArg := twoArguments(argument)
	itemMode := findAllDots(itemArg)
	containerMode := findAllDots(containerArg)
	private := func(message string) { me.SendMessage(message) }

	if itemArg == "" {
		private("Put what in what?\r\n")
		return CharacterCommandHandled
	}
	if containerMode != findIndiv {
		private("You can only put things into one container at a time.\r\n")
		return CharacterCommandHandled
	}
	if containerArg == "" {
		what := "it"
		if itemMode != findIndiv {
			what = "them"
		}
		private("What do you want to put " + what + " in?\r\n")
		return CharacterCommandHandled
	}

	var container *ObjectInstance
	for _, object := range me.InventorySnapshot() {
		if object != nil && canSeeObject(me, object) && isnameWithAbbrevs(containerArg, object.GetKeywords()) {
			container = object
			break
		}
	}
	if container == nil {
		for _, object := range w.GetItemsInRoom(me.GetRoomVNum()) {
			if object != nil && canSeeObject(me, object) && isnameWithAbbrevs(containerArg, object.GetKeywords()) {
				container = object
				break
			}
		}
	}
	if container == nil {
		private("You don't see " + an(containerArg) + " " + containerArg + " here.\r\n")
		return CharacterCommandHandled
	}
	if container.GetTypeFlag() != ITEM_CONTAINER {
		Act(w, false, me, nil, container, nil, "$p is not a container.", "", ToChar)
		return CharacterCommandHandled
	}
	if contIsClosed(container) {
		private("You'd better open it first!\r\n")
		return CharacterCommandHandled
	}
	if itemMode == findIndiv {
		var object *ObjectInstance
		for _, carried := range me.InventorySnapshot() {
			if carried != nil && canSeeObject(me, carried) && isnameWithAbbrevs(itemArg, carried.GetKeywords()) {
				object = carried
				break
			}
		}
		if object == nil {
			private("You aren't carrying " + an(itemArg) + " " + itemArg + ".\r\n")
			return CharacterCommandHandled
		}
		if object == container {
			private("You attempt to fold it into itself, but fail.\r\n")
			return CharacterCommandHandled
		}
		w.npcPerformPut(me, object, container)
		return CharacterCommandHandled
	}

	keyword := strings.TrimPrefix(itemArg, "all.")
	found := false
	for _, object := range me.InventorySnapshot() {
		if object == nil || object == container || !canSeeObject(me, object) ||
			(itemMode != findAll && !isnameWithAbbrevs(keyword, object.GetKeywords())) {
			continue
		}
		found = true
		w.npcPerformPut(me, object, container)
	}
	if !found {
		if itemMode == findAll {
			private("You don't seem to have anything to put in it.\r\n")
		} else {
			private("You don't seem to have any " + keyword + "s.\r\n")
		}
	}
	return CharacterCommandHandled
}

func (w *World) npcPerformPut(me *MobInstance, object, container *ObjectInstance) {
	if container.GetTotalWeight()+object.GetWeight() > container.GetValue(contCapacity) {
		Act(w, false, me, nil, object, container, "$p won't fit in $P.", "", ToChar)
		return
	}
	if err := w.MoveObjectToContainer(object, container); err != nil {
		Act(w, false, me, nil, object, container, "You can't put that in there.", "", ToChar)
		return
	}
	Act(w, false, me, nil, object, container, "You put $p in $P.", "", ToChar)
	Act(w, true, me, nil, object, container, "$n puts $p in $P.", "", ToRoom)
}

func (w *World) npcWearAtDirect(me *MobInstance, argument string, where int, verb string) CharacterCommandResult {
	item, _ := oneArgument(argument)
	if item == "" {
		me.SendMessage(verb + " what?\r\n")
		return CharacterCommandHandled
	}
	var object *ObjectInstance
	for _, carried := range me.InventorySnapshot() {
		if carried != nil && canSeeObject(me, carried) && isnameWithAbbrevs(item, carried.GetKeywords()) {
			object = carried
			break
		}
	}
	if object == nil {
		me.SendMessage("You don't seem to have " + an(item) + " " + item + ".\r\n")
		return CharacterCommandHandled
	}
	w.performMobileWear(me, object, where)
	return CharacterCommandHandled
}

func (w *World) npcGrabDirect(me *MobInstance, argument string) CharacterCommandResult {
	item, _ := oneArgument(argument)
	if item == "" {
		me.SendMessage("Hold what?\r\n")
		return CharacterCommandHandled
	}
	var object *ObjectInstance
	for _, carried := range me.InventorySnapshot() {
		if carried != nil && canSeeObject(me, carried) && isnameWithAbbrevs(item, carried.GetKeywords()) {
			object = carried
			break
		}
	}
	if object == nil {
		me.SendMessage("You don't seem to have " + an(item) + " " + item + ".\r\n")
		return CharacterCommandHandled
	}
	itemType := object.GetTypeFlag()
	if !canWearObject(object, eqWearHold) && itemType != ITEM_WAND && itemType != ITEM_STAFF && itemType != ITEM_SCROLL && itemType != ITEM_POTION {
		me.SendMessage("You can't hold that.\r\n")
		return CharacterCommandHandled
	}
	w.performMobileWear(me, object, eqWearHold)
	return CharacterCommandHandled
}

func (w *World) npcRemoveDirect(me *MobInstance, argument string) CharacterCommandResult {
	item, _ := oneArgument(argument)
	if item == "" {
		me.SendMessage("Remove what?\r\n")
		return CharacterCommandHandled
	}
	dotmode := findAllDots(item)
	if item == "all." {
		dotmode = findAlldot
	}
	found := false
	for position := range NumWears {
		object := me.Equipped(position)
		if object == nil {
			continue
		}
		matches := false
		switch dotmode {
		case findAll:
			matches = true
		case findAlldot:
			keyword := strings.TrimPrefix(item, "all.")
			matches = keyword != "" && canSeeObject(me, object) && isnameWithAbbrevs(keyword, object.GetKeywords())
		case findIndiv:
			matches = canSeeObject(me, object) && isnameWithAbbrevs(item, object.GetKeywords())
		}
		if !matches {
			continue
		}
		found = true
		if object.HasExtraFlag(0, extraFlagNoDrop) {
			w.mobileEquipmentAct(me, object, "You can't remove $p, it must be CURSED!")
			continue
		}
		if me.UnequipItem(position) == nil {
			continue
		}
		Act(w, false, me, nil, object, nil, "You stop using $p.", "", ToChar)
		Act(w, true, me, nil, object, nil, "$n stops using $p.", "", ToRoom)
	}
	if !found {
		if dotmode == findAll {
			me.SendMessage("You're not using anything.\r\n")
		} else if dotmode == findAlldot && strings.TrimPrefix(item, "all.") == "" {
			me.SendMessage("Remove all of what?\r\n")
		} else if dotmode == findAlldot {
			me.SendMessage("You don't seem to be using any " + strings.TrimPrefix(item, "all.") + "s.\r\n")
		} else {
			me.SendMessage("You don't seem to be using " + an(item) + " " + item + ".\r\n")
		}
	}
	return CharacterCommandHandled
}

// npcGetDirect ports the command's parsed all-from-container path for mobile
// callers. The direct C do_get sites in fight.c pass "all corpse".
func (w *World) npcGetDirect(me *MobInstance, argument string) CharacterCommandResult {
	itemArg, containerArg := twoArguments(argument)
	if containerArg == "" {
		return CharacterCommandUnsupported
	}
	itemFilter := ""
	switch {
	case strings.EqualFold(itemArg, "all"):
	case len(itemArg) > 4 && strings.EqualFold(itemArg[:4], "all."):
		itemFilter = itemArg[4:]
	default:
		return CharacterCommandUnsupported
	}
	if mobCarriedItemCount(me) >= mobMaxCarryCount(me) {
		return CharacterCommandHandled
	}

	var container *ObjectInstance
	inInventory := false
	for _, candidate := range me.Inventory {
		if candidate != nil && canSeeObject(me, candidate) && isnameWithAbbrevs(containerArg, candidate.GetKeywords()) {
			container = candidate
			inInventory = true
			break
		}
	}
	if container == nil {
		for _, candidate := range w.GetItemsInRoom(me.GetRoomVNum()) {
			if candidate != nil && canSeeObject(me, candidate) && isnameWithAbbrevs(containerArg, candidate.GetKeywords()) {
				container = candidate
				break
			}
		}
	}
	if container == nil || container.GetTypeFlag() != ITEM_CONTAINER || contIsClosed(container) {
		return CharacterCommandHandled
	}

	items := append([]*ObjectInstance(nil), container.Contains...)
	found := false
	for _, item := range items {
		if item == nil || !canSeeObject(me, item) ||
			(itemFilter != "" && !isnameWithAbbrevs(itemFilter, item.GetKeywords())) {
			continue
		}
		found = true
		if mobCarriedItemCount(me) >= mobMaxCarryCount(me) ||
			(!inInventory && (!item.IsTakeable() || mobCarriedWeight(me)+item.GetWeight() > mobMaxCarryWeight(me))) {
			continue
		}
		if err := w.MoveObjectToMobInventoryFront(item, me); err != nil {
			continue
		}
		Act(w, false, me, nil, item, container, "You get $p from $P.", "", ToChar)
		Act(w, true, me, nil, item, container, "$n gets $p from $P.", "", ToRoom)
		if item.GetTypeFlag() == ITEM_MONEY && item.GetValue(0) > 0 {
			me.SetGold(me.GetGold() + item.GetValue(0))
			w.ExtractObject(item, me.GetRoomVNum())
		}
	}
	if !found {
		Act(w, false, me, nil, container, nil, "$p seems to be empty.", "", ToChar)
	}
	return CharacterCommandHandled
}

// npcWearDirect ports do_wear for mobile callers, including the two_arguments
// parse and the NPC-specific perform_wear body.
func (w *World) npcWearDirect(me *MobInstance, argument string) CharacterCommandResult {
	itemArg, positionArg := twoArguments(argument)
	dotmode := findAllDots(itemArg)
	if itemArg == "" {
		me.SendMessage("Wear what?\r\n")
		return CharacterCommandHandled
	}
	if dotmode == findAlldot && strings.TrimPrefix(itemArg, "all.") == "" {
		me.SendMessage("Wear all of what?\r\n")
		return CharacterCommandHandled
	}
	if positionArg != "" && dotmode != findIndiv {
		me.SendMessage("You can't specify the same body location for more than one item!\r\n")
		return CharacterCommandHandled
	}
	itemsWorn := 0
	keyword := strings.TrimPrefix(itemArg, "all.")
	found := false
	for _, item := range me.InventorySnapshot() {
		if item == nil || !canSeeObject(me, item) || (dotmode == findAlldot && !isnameWithAbbrevs(keyword, item.GetKeywords())) || (dotmode == findIndiv && !isnameWithAbbrevs(itemArg, item.GetKeywords())) {
			continue
		}
		found = true
		where := findEqPos(item, positionArg)
		if dotmode == findAll {
			where = findEqPos(item, "")
		}
		if where >= 0 {
			itemsWorn++
			w.performMobileWear(me, item, where)
		} else if positionArg != "" {
			me.SendMessage("'" + positionArg + "'?  What part of your body is THAT?\r\n")
			return CharacterCommandHandled
		} else if positionArg == "" {
			Act(w, false, me, nil, item, nil, "You can't wear $p.", "", ToChar)
		}
	}
	if !found {
		if dotmode == findAlldot {
			me.SendMessage("You don't seem to have any " + keyword + "s.\r\n")
		} else if dotmode == findIndiv {
			me.SendMessage("You don't seem to have " + an(itemArg) + " " + itemArg + ".\r\n")
		} else if itemsWorn == 0 {
			me.SendMessage("You don't seem to have anything wearable.\r\n")
		}
	} else if itemsWorn == 0 && dotmode == findAll {
		me.SendMessage("You don't seem to have anything wearable.\r\n")
	}
	return CharacterCommandHandled
}

func (w *World) npcRescueDirect(me *MobInstance, argument string) CharacterCommandResult {
	targetName, _ := oneArgument(argument)
	if targetName == "" {
		return CharacterCommandHandled
	}
	target, found := w.resolveCharInRoomAt(me, me.GetRoomVNum(), targetName)
	if !found || target.Combatant == nil {
		return CharacterCommandHandled
	}
	mobDoRescueResolved(w, me, target.Combatant)
	return CharacterCommandHandled
}

// npcBeheadDirect ports do_behead's direct mobile call. Its player-only output
// is absent for a mobile, while object lookup and performBehead side effects
// remain shared with the player command.
func (w *World) npcBeheadDirect(me *MobInstance, argument string) CharacterCommandResult {
	targetName, _ := oneArgument(argument)
	if targetName == "" {
		return CharacterCommandHandled
	}
	items := w.GetItemsInRoom(me.GetRoomVNum())
	var target *ObjectInstance
	for _, obj := range items {
		if obj != nil && isnameWithAbbrevs(targetName, obj.GetKeywords()) && canSeeObject(me, obj) {
			target = obj
			break
		}
	}
	if target == nil || target.GetTypeFlag() != ITEM_CONTAINER || target.GetValue(3) == 0 || strings.Contains(target.GetKeywords(), "headless") {
		return CharacterCommandHandled
	}
	wielded := me.Equipped(mobWearWield)
	slashWeapon := wielded != nil && wielded.GetValue(3) == 3
	result := performBehead(w, me, target, wielded != nil, slashWeapon,
		func(headObj *ObjectInstance) bool {
			return headObj.IsTakeable() && mobCarriedWeight(me)+headObj.GetWeight() <= mobMaxCarryWeight(me) && mobCarriedItemCount(me)+1 <= mobMaxCarryCount(me)
		},
		func(headObj *ObjectInstance) error { return w.MoveObjectToMobInventoryFront(headObj, me) },
	)
	if result.Success {
		if slashWeapon {
			Act(w, true, me, nil, target, nil, "$n beheads $p!", "", ToRoom)
		} else {
			Act(w, true, me, nil, target, nil, "$n rips the head off $p!", "", ToRoom)
		}
	}
	return CharacterCommandHandled
}
