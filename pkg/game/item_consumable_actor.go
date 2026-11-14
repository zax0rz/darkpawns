package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// CharacterDrink ports drink/sip for a switched mobile. C reads conditions
// through the shared dummy_mob player-specials record; MobInstance's
// GetCondition/SetCondition mirror that shared state.
func CharacterDrink(ctx CharacterCommandContext, commandName string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	subcmd := scmdDrink
	if commandName == "sip" {
		subcmd = scmdSip
	}
	drinkForMobile(ctx, mob, subcmd)
	return CharacterCommandHandled
}

func drinkForMobile(ctx CharacterCommandContext, mob *MobInstance, subcmd int) {
	arg, _ := oneArgument(ctx.RawArgs)
	if arg == "" {
		mobileSend(ctx, mob, "Drink from what?\r\n")
		return
	}
	var item *ObjectInstance
	for _, candidate := range mob.InventorySnapshot() {
		if canSeeObject(mob, candidate) && isnameWithAbbrevs(arg, candidate.GetKeywords()) {
			item = candidate
			break
		}
	}
	onGround := false
	if item == nil {
		for _, candidate := range ctx.World.GetItemsInRoom(mob.GetRoomVNum()) {
			if canSeeObject(mob, candidate) && isnameWithAbbrevs(arg, candidate.GetKeywords()) {
				item = candidate
				onGround = true
				break
			}
		}
	}
	if item == nil {
		mobileActToChar(ctx, mob, nil, nil, "You can't find it!")
		return
	}
	objType := item.GetTypeFlag()
	if objType != ITEM_DRINKCON && objType != ITEM_FOUNTAIN {
		mobileSend(ctx, mob, "You can't drink from that!\r\n")
		return
	}
	if onGround && objType == ITEM_DRINKCON {
		mobileSend(ctx, mob, "You have to be holding that to drink from it.\r\n")
		return
	}
	if mob.GetCondition(CondDrunk) > 10 && mob.GetCondition(CondThirst) > 0 {
		mobileSend(ctx, mob, "You can't seem to get close enough to your mouth.\r\n")
		Act(ctx.World, true, mob, nil, nil, nil, "$n tries to drink but misses $s mouth!", "", ToRoom)
		return
	}
	if mob.GetCondition(CondFull) > 20 && mob.GetCondition(CondThirst) > 0 {
		mobileSend(ctx, mob, "Your stomach can't contain anymore!\r\n")
		return
	}
	if mob.GetCondition(CondThirst) > 40 {
		mobileSend(ctx, mob, "If you drink any more, you'll explode!\r\n")
		return
	}
	level, claimed := ctx.World.claimDrinkDose(item)
	if !claimed {
		mobileSend(ctx, mob, "It's empty.\r\n")
		return
	}
	liqIndex := item.GetValue(2)
	liqName := DrinkName(liqIndex)
	amount := 0
	if subcmd == scmdDrink {
		mobileSend(ctx, mob, fmt.Sprintf("You drink the %s.\r\n", liqName))
		drunkAff, _, _ := GetDrinkAffects(liqIndex)
		if drunkAff > 0 {
			amount = (25 - mob.GetCondition(CondThirst)) / drunkAff
		} else {
			amount = consumableNumber(3, 8)
		}
	} else {
		Act(ctx.World, true, mob, nil, item, nil, "$n sips from $p.", "", ToRoom)
		mobileSend(ctx, mob, fmt.Sprintf("It tastes like %s.\r\n", liqName))
	}
	amount = min(amount, level)
	weight := min(amount, item.GetWeight())
	item.SetWeight(item.GetWeight() - weight)
	drunkAff, fullAff, thirstAff := GetDrinkAffects(liqIndex)
	gainConditionMobile(ctx, mob, CondDrunk, drunkAff*amount/4)
	// PLR_VAMPIRE is a player-specials flag. The NPCs' shared dummy_mob flag
	// set is zero in this path, so C always takes the ordinary sustenance arm.
	gainConditionMobile(ctx, mob, CondFull, fullAff*amount/4)
	gainConditionMobile(ctx, mob, CondThirst, thirstAff*amount/4)
	if mob.GetCondition(CondDrunk) > 10 {
		mobileSend(ctx, mob, "You feel drunk.\r\n")
	}
	if mob.GetCondition(CondThirst) > 20 {
		mobileSend(ctx, mob, "You don't feel thirsty any more.\r\n")
	}
	if mob.GetCondition(CondFull) > 20 {
		mobileSend(ctx, mob, "You are full.\r\n")
	}
	if item.GetValue(3) != 0 {
		mobileSend(ctx, mob, "Oops, it tasted rather strange!\r\n")
		Act(ctx.World, true, mob, nil, nil, nil, "$n chokes and utters some strange sounds.", "", ToRoom)
		applyPoisonMobile(mob, amount*3, "poisoned drink")
	}
	if ctx.World.depleteDrink(item, amount-1) && item.VNum == 20 {
		ctx.World.ExtractObject(item, mob.GetRoomVNum())
	}
}

// CharacterEat ports eat/taste for a mobile body. The C dummy_mob condition
// values start at -1; gain_condition treats that sentinel as immutable, so
// food consumption is visible while fullness remains unchanged for all NPCs.
func CharacterEat(ctx CharacterCommandContext, commandName string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	subcmd := scmdEat
	if commandName == "taste" {
		subcmd = scmdTaste
	}
	eatForMobile(ctx, mob, subcmd)
	return CharacterCommandHandled
}

func eatForMobile(ctx CharacterCommandContext, mob *MobInstance, subcmd int) {
	arg, _ := oneArgument(ctx.RawArgs)
	if arg == "" {
		mobileSend(ctx, mob, "Eat what?\r\n")
		return
	}
	var item *ObjectInstance
	for _, candidate := range mob.InventorySnapshot() {
		if canSeeObject(mob, candidate) && isnameWithAbbrevs(arg, candidate.GetKeywords()) {
			item = candidate
			break
		}
	}
	found := item != nil
	if !found && mob.IsAffected(affWerewolf) {
		for _, candidate := range ctx.World.GetItemsInRoom(mob.GetRoomVNum()) {
			if canSeeObject(mob, candidate) && isnameWithAbbrevs(arg, candidate.GetKeywords()) {
				item = candidate
				break
			}
		}
	}
	if item == nil {
		mobileSend(ctx, mob, fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg), arg))
		return
	}
	objType := item.GetTypeFlag()
	if !found {
		if objType == ITEM_CONTAINER && item.GetValue(3) != 0 {
			eatWerewolfCorpseMobile(ctx, mob, item)
		} else {
			mobileSend(ctx, mob, "Eat what?!?\r\n")
		}
		return
	}
	if subcmd == scmdTaste && (objType == ITEM_DRINKCON || objType == ITEM_FOUNTAIN) {
		drinkForMobile(ctx, mob, scmdSip)
		return
	}
	if objType != ITEM_FOOD && mob.GetLevel() < LVL_GOD {
		mobileSend(ctx, mob, "You can't eat THAT!\r\n")
		return
	}
	if mob.GetCondition(CondFull) > 40 {
		mobileActToChar(ctx, mob, nil, nil, "You are too full to eat more!")
		return
	}
	if subcmd == scmdEat {
		mobileActToChar(ctx, mob, item, nil, "You eat $p.")
		Act(ctx.World, true, mob, nil, item, nil, "$n eats $p.", "", ToRoom)
	} else {
		mobileActToChar(ctx, mob, item, nil, "You nibble a little bit of $p.")
		Act(ctx.World, true, mob, nil, item, nil, "$n tastes a little bit of $p.", "", ToRoom)
	}
	amount := 0
	if subcmd == scmdEat {
		amount = item.GetValue(0)
	}
	gainConditionMobile(ctx, mob, CondFull, amount)
	if mob.GetCondition(CondFull) > 20 {
		mobileActToChar(ctx, mob, nil, nil, "You are full.")
	}
	if item.GetValue(3) != 0 && mob.GetLevel() < LVL_IMMORT {
		mobileSend(ctx, mob, "Oops, that tasted rather strange!\r\n")
		Act(ctx.World, true, mob, nil, nil, nil, "$n coughs and utters some strange sounds.", "", ToRoom)
		applyPoisonMobile(mob, amount*2, item.GetShortDesc())
	}
	if subcmd == scmdEat {
		ctx.World.ExtractObject(item, mob.GetRoomVNum())
		return
	}
	newVal := item.GetValue(0) - 1
	item.SetValue(0, newVal)
	if newVal == 0 {
		mobileSend(ctx, mob, "There's nothing left now.\r\n")
		ctx.World.ExtractObject(item, mob.GetRoomVNum())
	}
}

func eatWerewolfCorpseMobile(ctx CharacterCommandContext, mob *MobInstance, corpse *ObjectInstance) {
	mobileActToChar(ctx, mob, corpse, nil, "You savagely rip into $p, feeding your insatiable appetite.")
	Act(ctx.World, true, mob, nil, corpse, nil, "$n savagely rips into $p, crunching through flesh and bone alike.", "", ToRoom)
	addMobDummyConditionIfInRange(CondFull, 0, 40, mob.GetLevel()/2)
	for _, item := range append([]*ObjectInstance(nil), corpse.Contains...) {
		if err := ctx.World.MoveObjectToRoomFront(item, mob.GetRoomVNum()); err != nil {
			slog.Error("mobile werewolf corpse spill failed", "object", item.ID, "error", err)
			return
		}
	}
	ctx.World.ExtractObject(corpse, mob.GetRoomVNum())
	flesh, err := ctx.World.SpawnObject(19, -1)
	if err != nil {
		slog.Error("mobile werewolf flesh spawn failed", "error", err)
		return
	}
	flesh.SetValue(0, 0)
	flesh.SetValue(3, 1)
	flesh.SetTimer(MaxNPCCorpseTime)
	if err := ctx.World.MoveObjectToRoomFront(flesh, mob.GetRoomVNum()); err != nil {
		slog.Error("mobile werewolf flesh placement failed", "error", err)
		ctx.World.ExtractObject(flesh, -1)
	}
}

func gainConditionMobile(ctx CharacterCommandContext, mob *MobInstance, condition, amount int) {
	current, intoxicated, changed := gainMobDummyCondition(condition, amount)
	if !changed {
		return
	}
	if current > 1 {
		return
	}
	switch condition {
	case CondFull:
		if current == 1 {
			mobileSend(ctx, mob, "Your stomach growls with hunger.\r\n")
		} else {
			mobileSend(ctx, mob, "You are hungry.\r\n")
		}
	case CondThirst:
		if current == 1 {
			mobileSend(ctx, mob, "You feel a bit parched.\r\n")
		} else {
			mobileSend(ctx, mob, "You are thirsty.\r\n")
		}
	case CondDrunk:
		if intoxicated && current == 1 {
			mobileSend(ctx, mob, "Your head starts to clear.\r\n")
		} else if intoxicated && current == 0 {
			mobileSend(ctx, mob, "You are now sober.\r\n")
		}
	}
}

func applyPoisonMobile(mob *MobInstance, duration int, source string) {
	mob.JoinAffect(engine.NewAffectDirect(spells.SpellPoison, engine.ApplyNone, duration, 0, engine.AFFPoison, source), false, false)
}
