package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

const (
	actorItemMagicUse    = 0
	actorItemMagicQuaff  = 1
	actorItemMagicRecite = 2
)

// CharacterItemMagic ports do_use and mag_objectmagic for switched mobile
// bodies. Command names select the ordinary interpreter subcmd; direct C
// callers retain the exact subcmd supplied at the call site.
func CharacterItemMagic(ctx CharacterCommandContext, commandName string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	subcmd := actorItemMagicSubcmd(ctx, commandName)
	itemArg, targetArgs := halfChop(ctx.RawArgs)
	if itemArg == "" {
		name := commandName
		if ctx.DirectCall {
			switch subcmd {
			case actorItemMagicQuaff:
				name = "quaff"
			case actorItemMagicRecite:
				name = "recite"
			default:
				name = "use"
			}
		}
		mobileSend(ctx, mob, fmt.Sprintf("What do you want to %s?\r\n", name))
		return CharacterCommandHandled
	}

	item := mobileHeldItem(mob, itemArg)
	if item == nil {
		switch subcmd {
		case actorItemMagicQuaff, actorItemMagicRecite:
			item = mobileCarriedVisibleItem(ctx.World, mob, itemArg)
		case actorItemMagicUse:
			if cIsName(itemArg, "tattoo") {
				// use_tattoo returns without effects for IS_NPC(ch) (tattoo.c:31-81).
				return CharacterCommandHandled
			}
			item = mobileEquippedVisibleUseItem(mob, itemArg)
		}
	}
	if item == nil {
		mobileSend(ctx, mob, fmt.Sprintf("You don't seem to have %s %s.\r\n", an(itemArg), itemArg))
		return CharacterCommandHandled
	}

	switch subcmd {
	case actorItemMagicQuaff:
		if item.GetTypeFlag() != ITEM_POTION {
			mobileSend(ctx, mob, "You can only quaff potions.")
			return CharacterCommandHandled
		}
	case actorItemMagicRecite:
		if item.GetTypeFlag() != ITEM_SCROLL {
			mobileSend(ctx, mob, "You can only recite scrolls.")
			return CharacterCommandHandled
		}
	case actorItemMagicUse:
		if item.GetTypeFlag() != ITEM_WAND && item.GetTypeFlag() != ITEM_STAFF {
			mobileSend(ctx, mob, "You can't seem to figure out how to use it.\r\nTry holding it.(?)\r\n")
			return CharacterCommandHandled
		}
	default:
		return CharacterCommandHandled
	}

	mobileMagicItem(ctx, mob, item, subcmd, targetArgs)
	return CharacterCommandHandled
}

func actorItemMagicSubcmd(ctx CharacterCommandContext, commandName string) int {
	if ctx.DirectCall {
		return ctx.Subcmd
	}
	switch commandName {
	case "quaff":
		return actorItemMagicQuaff
	case "recite":
		return actorItemMagicRecite
	default:
		return actorItemMagicUse
	}
}

func mobileHeldItem(mob *MobInstance, name string) *ObjectInstance {
	item := mob.Equipped(17) // C WEAR_HOLD; do_use checks it with isname before any other lookup.
	if item != nil && cIsName(name, item.GetKeywords()) {
		return item
	}
	return nil
}

func mobileCarriedVisibleItem(w *World, mob *MobInstance, name string) *ObjectInstance {
	item, _ := w.ResolveObjectInInventory(mob, name)
	return item
}

func mobileEquippedVisibleUseItem(mob *MobInstance, name string) *ObjectInstance {
	var match *ObjectInstance
	for position := 0; position < NumWears; position++ {
		item := mob.Equipped(position)
		if item != nil && cIsName(name, item.GetKeywords()) && CanSeeObject(mob, item) {
			match = item // C do_use keeps the last visible match in WEAR_* order.
		}
	}
	return match
}

func mobileMagicItem(ctx CharacterCommandContext, mob *MobInstance, item *ObjectInstance, subcmd int, targetArgs string) {
	spellArgument, _ := OneArgument(targetArgs)
	switch item.GetTypeFlag() {
	case ITEM_WAND, ITEM_STAFF:
		if mobileMagicEquipdHold(mob, targetArgs) {
			if item.GetTypeFlag() == ITEM_STAFF {
				mobileStaffTap(ctx, mob, item, targetArgs)
			} else {
				mobileWandPoint(ctx, mob, item, spellArgument)
			}
		} else if item.GetTypeFlag() == ITEM_WAND {
			mobileWandUse(ctx, mob, item, spellArgument)
		} else {
			mobileStaffUse(ctx, mob, item, spellArgument)
		}
	case ITEM_SCROLL:
		if item.GetVNum() == 8052 && mob.GetFightingBody() != nil {
			mobileActToChar(ctx, mob, nil, nil, "You can't concentrate enough!.")
			return
		}
		if mob.GetPosition() == combat.PosSitting {
			mobileSend(ctx, mob, "You can't do this sitting!!\r\n")
			return
		}
		target, targetObj, found := mobileResolveMagicTarget(ctx.World, mob, spellArgument)
		if spellArgument != "" && !found {
			mobileActToChar(ctx, mob, item, nil, "There is nothing to here to affect with $p.")
			return
		}
		if spellArgument == "" {
			target = mob
		}
		mobileActToChar(ctx, mob, item, nil, "You recite $p which dissolves.")
		mobileActionDescription(ctx, mob, item, targetObj, "$n recites $p.", true)
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
		for i := 1; i < 4; i++ {
			if !spells.CallMagicWithArgument(mob, target, targetObj, item.GetValue(i), item.GetValue(0), spells.CastScroll, ctx.World, spellArgument) {
				break
			}
		}
		ctx.World.ExtractObject(item, mob.GetRoomVNum())
	case ITEM_POTION:
		if mob.GetPosition() == combat.PosSitting {
			mobileSend(ctx, mob, "You can't do this sitting!\r\n")
			return
		}
		mobileActToChar(ctx, mob, item, nil, "You quaff $p.")
		mobileActionDescription(ctx, mob, item, nil, "$n quaffs $p.", true)
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
		for i := 1; i < 4; i++ {
			if !spells.CallMagicWithArgument(mob, mob, nil, item.GetValue(i), item.GetValue(0), spells.CastPotion, ctx.World, spellArgument) {
				break
			}
		}
		ctx.World.ExtractObject(item, mob.GetRoomVNum())
	default:
		return
	}
}

func mobileMagicEquipdHold(mob *MobInstance, targetArgs string) bool {
	if targetArgs == "" {
		return false
	}
	last := -1
	for position := 0; position < NumWears; position++ {
		item := mob.Equipped(position)
		if item != nil && cIsName(targetArgs, item.GetKeywords()) {
			last = position
		}
	}
	return last == 16 // mag_objectmagic compares wear_pos+1 with WEAR_HOLD (17).
}

func mobileResolveMagicTarget(w *World, mob *MobInstance, targetName string) (Actor, *ObjectInstance, bool) {
	if targetName == "" {
		return nil, nil, false
	}
	if resolved, ok := w.ResolveCharInRoom(mob, targetName); ok {
		if actor := asActor(resolved.Combatant); actor != nil {
			return actor, nil, true
		}
	}
	if item, _ := w.GenericFindObject(mob, targetName); item != nil {
		return nil, item, true
	}
	return nil, nil, false
}

func mobileMagicObjectTargetAllowed(spellNum int) bool {
	info := spells.GetSpellInfo(spellNum)
	return info != nil && (info.HasTarget(spells.TarObjInv) || info.HasTarget(spells.TarObjRoom) ||
		info.HasTarget(spells.TarObjEquip) || info.HasTarget(spells.TarObjWorld))
}

func mobileWandPoint(ctx CharacterCommandContext, mob *MobInstance, item *ObjectInstance, targetName string) {
	target, targetObj, found := mobileResolveMagicTarget(ctx.World, mob, targetName)
	if !found || targetObj != nil && !mobileMagicObjectTargetAllowed(item.GetValue(3)) {
		mobileActToChar(ctx, mob, item, nil, "At what should $p be pointed?")
		return
	}
	if target != nil {
		if target == mob {
			mobileActToChar(ctx, mob, item, nil, "You point $p at yourself.")
			Act(ctx.World, false, mob, nil, item, nil, "$n points $p at $mself.", "", ToRoom)
		} else {
			mobileActToCharWithTarget(ctx, mob, target, item, nil, "You point $p at $N.")
			mobileActionDescriptionWithTarget(ctx, mob, target, item, nil, "$n points $p at $N.", true)
		}
	} else {
		mobileActToChar(ctx, mob, item, targetObj, "You point $p at $P.")
		mobileActionDescription(ctx, mob, item, targetObj, "$n points $p at $P.", true)
	}
	if !ctx.World.takeMagicItemCharge(item) {
		mobileActToChar(ctx, mob, item, nil, "It seems powerless.")
		Act(ctx.World, false, mob, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	var objectTarget any
	if targetObj != nil {
		objectTarget = targetObj
	}
	spells.CallMagicWithArgument(mob, target, objectTarget, item.GetValue(3), mobileMagicItemLevel(item), spells.CastWand, ctx.World, targetName)
}

func mobileWandUse(ctx CharacterCommandContext, mob *MobInstance, item *ObjectInstance, targetName string) {
	target, targetObj, found := mobileResolveMagicTarget(ctx.World, mob, targetName)
	if !found || targetObj != nil && !mobileMagicObjectTargetAllowed(item.GetValue(3)) {
		mobileActToChar(ctx, mob, item, nil, "You can't use $p like that.")
		return
	}
	if target != nil {
		if target == mob {
			mobileActToChar(ctx, mob, item, nil, "Your $p bathes you in a blinding glow!")
			mobileActionDescriptionWithTarget(ctx, mob, mob, item, nil, "$n's $p bathes $m in a blinding glow!", false)
		} else {
			mobileActToCharWithTarget(ctx, mob, target, item, nil, "Your $p flares up with a blinding glow that surges toward $N!")
			mobileActionDescriptionWithTarget(ctx, mob, target, item, nil, "$n's $p flares up with a blinding glow that surges toward $N!", true)
		}
	} else {
		mobileActToChar(ctx, mob, item, targetObj, "Your $p flares up with a blinding glow that surges toward $P!")
		mobileActionDescription(ctx, mob, item, targetObj, "$n's $p flares up with a blinding glow that surges toward $P!", true)
	}
	if !ctx.World.takeMagicItemCharge(item) {
		mobileActToChar(ctx, mob, item, nil, "It seems powerless.")
		Act(ctx.World, false, mob, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	var objectTarget any
	if targetObj != nil {
		objectTarget = targetObj
	}
	spells.CallMagicWithArgument(mob, target, objectTarget, item.GetValue(3), mobileMagicItemLevel(item), spells.CastWand, ctx.World, targetName)
}

func mobileStaffTap(ctx CharacterCommandContext, mob *MobInstance, item *ObjectInstance, targetArgs string) {
	spellArgument, _ := OneArgument(targetArgs)
	mobileActToChar(ctx, mob, item, nil, "You tap $p three times on the ground.")
	mobileActionDescription(ctx, mob, item, nil, "$n taps $p three times on the ground.", true)
	if !ctx.World.takeMagicItemCharge(item) {
		mobileActToChar(ctx, mob, item, nil, "It seems powerless.")
		Act(ctx.World, false, mob, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	for _, actor := range ctx.World.actChar(mob.GetRoomVNum()) {
		if actor != mob {
			spells.CallMagicWithArgument(mob, actor, nil, item.GetValue(3), mobileMagicItemLevel(item), spells.CastStaff, ctx.World, spellArgument)
		}
	}
}

func mobileStaffUse(ctx CharacterCommandContext, mob *MobInstance, item *ObjectInstance, targetName string) {
	spellArgument, _ := OneArgument(targetName)
	var targetObj *ObjectInstance
	if targetName != "" {
		if _, ok := ctx.World.ResolveCharInRoom(mob, targetName); !ok {
			targetObj, _ = ctx.World.GenericFindObject(mob, targetName)
		}
	}
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		mobileActionDescription(ctx, mob, item, targetObj, item.Prototype.ActionDesc, false)
	} else {
		Act(ctx.World, true, mob, nil, item, nil, "$n's $p sparks blindingly, bathing you in its glow.", "", ToRoom)
	}
	mobileActToChar(ctx, mob, item, nil, "Your $p radiates an ethereal glow that lights the room.")
	if !ctx.World.takeMagicItemCharge(item) {
		mobileActToChar(ctx, mob, item, nil, "It seems powerless.")
		Act(ctx.World, false, mob, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	spellNum := item.GetValue(3)
	info := spells.GetSpellInfo(spellNum)
	if info != nil && (info.HasRoutine(spells.RoutineMasses) || info.HasRoutine(spells.RoutineAreas)) {
		spells.CallMagicWithArgument(mob, nil, nil, spellNum, mobileMagicItemLevel(item), spells.CastStaff, ctx.World, spellArgument)
		return
	}
	for _, actor := range ctx.World.actChar(mob.GetRoomVNum()) {
		if actor != mob {
			spells.CallMagicWithArgument(mob, actor, nil, spellNum, mobileMagicItemLevel(item), spells.CastStaff, ctx.World, spellArgument)
		}
	}
}

func mobileMagicItemLevel(item *ObjectInstance) int {
	level := item.GetValue(0)
	if level == 0 {
		return defaultMagicItemLevel
	}
	return level
}

func mobileActionDescription(ctx CharacterCommandContext, mob *MobInstance, item, targetObj *ObjectInstance, fallback string, hideFallback bool) {
	format, hide := fallback, hideFallback
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		format, hide = item.Prototype.ActionDesc, false
	}
	Act(ctx.World, hide, mob, nil, item, targetObj, format, "", ToRoom)
}

func mobileActionDescriptionWithTarget(ctx CharacterCommandContext, mob *MobInstance, target Actor, item, targetObj *ObjectInstance, fallback string, hideFallback bool) {
	format, hide := fallback, hideFallback
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		format, hide = item.Prototype.ActionDesc, false
	}
	Act(ctx.World, hide, mob, target, item, targetObj, format, "", ToRoom)
}

func mobileActToCharWithTarget(ctx CharacterCommandContext, mob *MobInstance, target Actor, item, targetObj *ObjectInstance, format string) {
	ActWithPrivateSink(ctx.World, false, mob, target, item, targetObj, format, "", ToChar, mob, ctx.PrivateOutput)
}
