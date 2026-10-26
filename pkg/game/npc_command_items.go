package game

import "strings"

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

// npcWearDirect ports do_wear's all-items path for mobile callers, including
// its two_arguments parse and the existing NPC-specific perform_wear body.
func (w *World) npcWearDirect(me *MobInstance, argument string) CharacterCommandResult {
	itemArg, positionArg := twoArguments(argument)
	if itemArg != "all" || positionArg != "" {
		return CharacterCommandUnsupported
	}
	itemsWorn := 0
	for _, item := range append([]*ObjectInstance(nil), me.Inventory...) {
		if item == nil || item.Prototype == nil || !CanSeeObject(me, item) {
			continue
		}
		if where := findEqPos(item, ""); where >= 0 {
			itemsWorn++
			w.performMobileWear(me, item, where)
		}
	}
	if itemsWorn == 0 && w.MobileMessageSink != nil {
		w.MobileMessageSink(me, []byte("You don't seem to have anything wearable.\r\n"))
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
