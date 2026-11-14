package game

import (
	"fmt"
	"log/slog"
	"strings"
)

// CharacterGet ports do_get for a mobile body. Player bodies retain DoGet's
// established path; switched NPCs use their own carrying list and room, and
// their successful pickup reaches the C room onget hook with the NPC CharRef.
func CharacterGet(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	getForMobile(ctx, mob)
	return CharacterCommandHandled
}

func getForMobile(ctx CharacterCommandContext, mob *MobInstance) {
	w := ctx.World
	if mobCarriedItemCount(mob) >= mobMaxCarryCount(mob) {
		mobileSend(ctx, mob, "Your arms are already full!\r\n")
		return
	}
	arg1, arg2 := twoArguments(ctx.RawArgs)
	if arg1 == "" {
		mobileSend(ctx, mob, "Get what?\r\n")
		return
	}
	if arg2 == "" {
		getFromRoomMobile(ctx, mob, arg1)
		return
	}

	contDotmode := findAllDots(arg2)
	if contDotmode != findIndiv {
		keyword := strings.TrimPrefix(arg2, "all.")
		if contDotmode == findAlldot && keyword == "" {
			mobileSend(ctx, mob, "Get from all of what?\r\n")
			return
		}
		found := false
		containers := mob.InventorySnapshot()
		containers = append(containers, w.GetItemsInRoom(mob.GetRoomVNum())...)
		for _, cont := range containers {
			if !canSeeObject(mob, cont) || (contDotmode == findAlldot && !isnameWithAbbrevs(keyword, cont.GetKeywords())) {
				continue
			}
			if cont.GetTypeFlag() != ITEM_CONTAINER {
				if contDotmode == findAlldot {
					found = true
					mobileActToChar(ctx, mob, cont, nil, "$p is not a container.")
				}
				continue
			}
			found = true
			mode := findObjRoom
			if cont.Location.Kind == ObjInInventory && cont.Location.OwnerKind == OwnerMob && cont.Location.MobID == mob.GetID() {
				mode = findObjInv
			}
			getFromContainerMobile(ctx, mob, cont, arg1, mode)
		}
		if !found {
			if contDotmode == findAll {
				mobileSend(ctx, mob, "You can't seem to find any containers.\r\n")
			} else {
				mobileSend(ctx, mob, fmt.Sprintf("You can't seem to find any %ss here.\r\n", keyword))
			}
		}
		return
	}

	var cont *ObjectInstance
	mode := findObjRoom
	for _, obj := range mob.InventorySnapshot() {
		if canSeeObject(mob, obj) && isnameWithAbbrevs(arg2, obj.GetKeywords()) {
			cont = obj
			mode = findObjInv
			break
		}
	}
	if cont == nil {
		for _, obj := range w.GetItemsInRoom(mob.GetRoomVNum()) {
			if canSeeObject(mob, obj) && isnameWithAbbrevs(arg2, obj.GetKeywords()) {
				cont = obj
				break
			}
		}
	}
	if cont == nil {
		mobileSend(ctx, mob, fmt.Sprintf("You don't have %s %s.\r\n", an(arg2), arg2))
		return
	}
	if cont.GetTypeFlag() != ITEM_CONTAINER {
		mobileActToChar(ctx, mob, cont, nil, "$p is not a container.")
		return
	}
	getFromContainerMobile(ctx, mob, cont, arg1, mode)
}

func getFromRoomMobile(ctx CharacterCommandContext, mob *MobInstance, arg string) {
	if mob.IsAffected(affMount) {
		mobileSend(ctx, mob, "You can't reach it from your mount.\r\n")
		return
	}
	dotmode := findAllDots(arg)
	keyword := strings.TrimPrefix(arg, "all.")
	if dotmode == findAlldot && keyword == "" {
		mobileSend(ctx, mob, "Get all of what?\r\n")
		return
	}
	items := ctx.World.GetItemsInRoom(mob.GetRoomVNum())
	if dotmode == findIndiv {
		for _, obj := range items {
			if canSeeObject(mob, obj) && isnameWithAbbrevs(arg, obj.GetKeywords()) {
				performGetFromRoomMobile(ctx, mob, obj)
				return
			}
		}
		mobileSend(ctx, mob, fmt.Sprintf("You don't see %s %s here.\r\n", an(arg), arg))
		return
	}
	found := false
	for _, obj := range items {
		if canSeeObject(mob, obj) && (dotmode == findAll || isnameWithAbbrevs(keyword, obj.GetKeywords())) {
			found = true
			performGetFromRoomMobile(ctx, mob, obj)
		}
	}
	if found {
		return
	}
	if dotmode == findAll {
		mobileSend(ctx, mob, "There doesn't seem to be anything here.\r\n")
	} else {
		mobileSend(ctx, mob, fmt.Sprintf("You don't see any %ss here.\r\n", keyword))
	}
}

func getFromContainerMobile(ctx CharacterCommandContext, mob *MobInstance, cont *ObjectInstance, arg string, mode int) {
	if contIsClosed(cont) {
		mobileActToChar(ctx, mob, cont, nil, "$p is closed.")
		return
	}
	dotmode := findAllDots(arg)
	if dotmode == findIndiv {
		for _, obj := range cont.Contains {
			if canSeeObject(mob, obj) && isnameWithAbbrevs(arg, obj.GetKeywords()) {
				performGetFromContainerMobile(ctx, mob, obj, cont, mode)
				return
			}
		}
		mobileActToChar(ctx, mob, cont, nil, fmt.Sprintf("There doesn't seem to be %s %s in $p.", an(arg), arg))
		return
	}
	keyword := strings.TrimPrefix(arg, "all.")
	if dotmode == findAlldot && keyword == "" {
		mobileSend(ctx, mob, "Get all of what?\r\n")
		return
	}
	items := append([]*ObjectInstance(nil), cont.Contains...)
	found := false
	for _, obj := range items {
		if canSeeObject(mob, obj) && (dotmode == findAll || isnameWithAbbrevs(keyword, obj.GetKeywords())) {
			found = true
			performGetFromContainerMobile(ctx, mob, obj, cont, mode)
		}
	}
	if found {
		return
	}
	if dotmode == findAll {
		mobileActToChar(ctx, mob, cont, nil, "$p seems to be empty.")
	} else {
		mobileActToChar(ctx, mob, cont, nil, fmt.Sprintf("You can't seem to find any %ss in $p.", keyword))
	}
}

func canTakeObjMobile(ctx CharacterCommandContext, mob *MobInstance, obj *ObjectInstance) bool {
	switch {
	case mobCarriedItemCount(mob) >= mobMaxCarryCount(mob):
		mobileActToChar(ctx, mob, obj, nil, "$p: you can't carry that many items.")
		return false
	case mobCarriedWeight(mob)+obj.GetTotalWeight() > mobMaxCarryWeight(mob):
		mobileActToChar(ctx, mob, obj, nil, "$p: you can't carry that much weight.")
		return false
	}
	for _, wearFlag := range obj.GetWearFlags() {
		if wearFlag&1 != 0 {
			return true
		}
	}
	mobileActToChar(ctx, mob, obj, nil, "$p: you can't take that!")
	return false
}

func performGetFromRoomMobile(ctx CharacterCommandContext, mob *MobInstance, obj *ObjectInstance) {
	if obj.Location.Kind == ObjNowhere || !canTakeObjMobile(ctx, mob, obj) {
		return
	}
	if err := ctx.World.MoveObjectToMobInventoryFront(obj, mob); err != nil {
		slog.Error("mobile get from room failed", "mob_vnum", mob.GetVNum(), "obj_vnum", obj.VNum, "error", err)
		mobileSend(ctx, mob, "You can't carry that much.\n")
		return
	}
	mobileActToChar(ctx, mob, obj, nil, "You get $p.")
	Act(ctx.World, true, mob, nil, obj, nil, "$n gets $p.", "", ToRoom)
	getCheckMoneyMobile(ctx, mob, obj)
	ctx.World.RunRoomObjTriggerScriptForActor(mob, obj, rsOnGet, "onget")
}

func performGetFromContainerMobile(ctx CharacterCommandContext, mob *MobInstance, obj, cont *ObjectInstance, mode int) {
	if mode != findObjInv && !canTakeObjMobile(ctx, mob, obj) {
		return
	}
	if mobCarriedItemCount(mob) >= mobMaxCarryCount(mob) {
		mobileActToChar(ctx, mob, obj, nil, "$p: you can't hold any more items.")
		return
	}
	if err := ctx.World.MoveObjectToMobInventoryFront(obj, mob); err != nil {
		slog.Error("mobile get from container failed", "mob_vnum", mob.GetVNum(), "obj_vnum", obj.VNum, "error", err)
		if rbErr := ctx.World.MoveObjectToContainer(obj, cont); rbErr != nil {
			slog.Error("rollback after mobile get failed", "mob_vnum", mob.GetVNum(), "obj_vnum", obj.VNum, "error", rbErr)
		}
		return
	}
	mobileActToChar(ctx, mob, obj, cont, "You get $p from $P.")
	Act(ctx.World, true, mob, nil, obj, cont, "$n gets $p from $P.", "", ToRoom)
	getCheckMoneyMobile(ctx, mob, obj)
	ctx.World.RunRoomObjTriggerScriptForActor(mob, obj, rsOnGet, "onget")
}

func getCheckMoneyMobile(ctx CharacterCommandContext, mob *MobInstance, obj *ObjectInstance) {
	amount := obj.GetValue(0)
	if obj.GetTypeFlag() != ITEM_MONEY || amount <= 0 {
		return
	}
	ctx.World.mu.Lock()
	if obj.Location.Kind == ObjNowhere {
		ctx.World.mu.Unlock()
		return
	}
	ctx.World.extractObjectLocked(obj)
	ctx.World.mu.Unlock()
	if amount > 1 {
		mobileSend(ctx, mob, fmt.Sprintf("There were %d coins.\r\n", amount))
	}
	mob.SetGold(mob.GetGold() + amount)
}

// CharacterDrop ports the ordinary do_drop lane for a mobile. Junk and donate
// remain separate commands with their own experience and destination behavior.
func CharacterDrop(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	dropForMobile(ctx, mob)
	return CharacterCommandHandled
}

func dropForMobile(ctx CharacterCommandContext, mob *MobInstance) {
	arg, rest := oneArgument(ctx.RawArgs)
	if arg == "" {
		mobileSend(ctx, mob, "What do you want to drop?\r\n")
		return
	}
	if CIsNumber(arg) {
		amount := atoi(arg)
		coinName, _ := oneArgument(rest)
		if strings.EqualFold(coinName, "coin") || strings.EqualFold(coinName, "coins") {
			performDropGoldMobile(ctx, mob, amount)
		} else {
			mobileSend(ctx, mob, "Sorry, you can't do that to more than one item at a time.\r\n")
		}
		return
	}
	dotmode := findAllDots(arg)
	items := mob.InventorySnapshot()
	if dotmode == findAll {
		if len(items) == 0 {
			mobileSend(ctx, mob, "You don't seem to be carrying anything.\r\n")
			return
		}
		for _, obj := range items {
			performDropMobile(ctx, mob, obj)
		}
		return
	}
	if dotmode == findAlldot {
		keyword := arg[4:]
		if keyword == "" {
			mobileSend(ctx, mob, "What do you want to drop all of?\r\n")
			return
		}
		found := false
		for _, obj := range items {
			if isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				performDropMobile(ctx, mob, obj)
				found = true
			}
		}
		if !found {
			mobileSend(ctx, mob, fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return
	}
	for _, obj := range items {
		if isnameWithAbbrevs(arg, obj.GetKeywords()) {
			performDropMobile(ctx, mob, obj)
			return
		}
	}
	mobileSend(ctx, mob, fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg), arg))
}

func performDropMobile(ctx CharacterCommandContext, mob *MobInstance, obj *ObjectInstance) {
	if obj.HasExtraFlag(0, extraFlagNoDrop) && mob.GetLevel() < lvlImmort {
		mobileActToChar(ctx, mob, obj, nil, "You can't drop $p, it must be CURSED!")
		return
	}
	mobileActToChar(ctx, mob, obj, nil, "You drop $p.")
	Act(ctx.World, true, mob, nil, obj, nil, "$n drops $p.", "", ToRoom)
	if err := ctx.World.MoveObjectToRoomFront(obj, mob.GetRoomVNum()); err != nil {
		slog.Error("mobile drop failed", "mob_vnum", mob.GetVNum(), "obj_vnum", obj.VNum, "error", err)
		return
	}
	ctx.World.RunRoomObjTriggerScriptForActor(mob, obj, rsOnDrop, "ondrop")
}

func performDropGoldMobile(ctx CharacterCommandContext, mob *MobInstance, amount int) {
	if amount <= 0 {
		mobileSend(ctx, mob, "Heh heh heh.. we are jolly funny today, eh?\r\n")
		return
	}
	if mob.GetGold() < amount {
		mobileSend(ctx, mob, "You don't have that many coins!\r\n")
		return
	}
	money := ctx.World.createMoneyObject(amount)
	if err := ctx.World.MoveObjectToRoomFront(money, mob.GetRoomVNum()); err != nil {
		slog.Error("mobile drop gold failed", "mob_vnum", mob.GetVNum(), "amount", amount, "error", err)
		mobileSend(ctx, mob, "You can't drop that right now.\r\n")
		return
	}
	mob.SetGold(mob.GetGold() - amount)
	mob.SetWaitState(1)
	mobileSend(ctx, mob, "You drop some gold.\r\n")
	Act(ctx.World, false, mob, nil, nil, nil, fmt.Sprintf("$n drops %s.", createMoneyDesc(amount)), "", ToRoom)
}

func mobileSend(ctx CharacterCommandContext, mob *MobInstance, message string) {
	if ctx.PrivateOutput != nil {
		ctx.PrivateOutput(message)
		return
	}
	mob.SendMessage(message)
}

func mobileActToChar(ctx CharacterCommandContext, mob *MobInstance, obj, victObj *ObjectInstance, format string) {
	ActWithPrivateSink(ctx.World, false, mob, nil, obj, victObj, format, "", ToChar, mob, ctx.PrivateOutput)
}
