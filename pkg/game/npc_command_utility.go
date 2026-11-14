package game

import (
	"fmt"
	"log/slog"
	"strings"
)

// CharacterAppraise ports the NPC body of do_appraise. C's NPC skill lookup
// reads dummy_mob's zero-valued player_specials skill array, so mobile bodies
// always take the failed-estimate arm (src/db.c:104,1281; src/utils.h:343).
func CharacterAppraise(ctx CharacterCommandContext) CharacterCommandResult {
	if ctx.World == nil || ctx.Actor == nil {
		return CharacterCommandUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		ctx.World.ExecAppraise(player, nil, ctx.RawArgs)
		return CharacterCommandHandled
	}
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	ctx.World.characterMobAppraise(mob, ctx.RawArgs)
	return CharacterCommandHandled
}

func (w *World) characterMobAppraise(mob *MobInstance, argument string) {
	percent := randRange(1, 101) // C draws before parsing or checking the object.
	name, _ := halfChop(argument)
	if name == "" {
		mob.SendMessage("What do you want to appraise?\r\n")
		return
	}
	object, found := w.ResolveObjectInInventory(mob, name)
	if !found {
		mob.SendMessage("You don't seem to have one of those...\r\n")
		return
	}
	cost := object.GetCost()
	if percent > 0 { // GET_SKILL(ch, SKILL_APPRAISE) is dummy_mob's zero value.
		cost += randRange(-cost, cost*2)
	}
	mob.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", cost))
}

// CharacterPour ports do_pour for mobile inventory and room objects. Drink
// condition state is not involved in this handler, so the instance body has
// all state C reads or mutates (src/act.item.c:1159-1325).
func CharacterPour(ctx CharacterCommandContext, commandName string) CharacterCommandResult {
	if ctx.World == nil || ctx.Actor == nil {
		return CharacterCommandUnsupported
	}
	subcmd := scmdPour
	if commandName == "fill" {
		subcmd = scmdFill
	} else if commandName != "pour" {
		return CharacterCommandUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		ctx.World.DoPour(player, nil, commandName, ctx.RawArgs, subcmd)
		return CharacterCommandHandled
	}
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	ctx.World.characterMobPour(mob, ctx.RawArgs, subcmd)
	return CharacterCommandHandled
}

func (w *World) characterMobPour(mob *MobInstance, argument string, subcmd int) {
	arg1, arg2 := twoArguments(argument)
	var fromObj, toObj *ObjectInstance
	if subcmd == scmdPour {
		if arg1 == "" {
			Act(nil, false, mob, nil, nil, nil, "From what do you want to pour?", "", ToChar)
			return
		}
		fromObj = mobObjectInInventory(w, mob, arg1)
		if fromObj == nil {
			Act(nil, false, mob, nil, nil, nil, "You can't find it!", "", ToChar)
			return
		}
		if fromObj.GetTypeFlag() != ITEM_DRINKCON {
			Act(nil, false, mob, nil, nil, nil, "You can't pour from that!", "", ToChar)
			return
		}
	} else {
		if arg1 == "" {
			mob.SendMessage("What do you want to fill?  And what are you filling it from?\r\n")
			return
		}
		toObj = mobObjectInInventory(w, mob, arg1)
		if toObj == nil {
			mob.SendMessage("You can't find it!")
			return
		}
		if toObj.GetTypeFlag() != ITEM_DRINKCON {
			Act(nil, false, mob, nil, toObj, nil, "You can't fill $p!", "", ToChar)
			return
		}
		if arg2 == "" {
			Act(nil, false, mob, nil, toObj, nil, "What do you want to fill $p from?", "", ToChar)
			return
		}
		fromObj = mobObjectInRoom(w, mob, arg2)
		if fromObj == nil {
			mob.SendMessage(fmt.Sprintf("There doesn't seem to be %s %s here.\r\n", an(arg2), arg2))
			return
		}
		if fromObj.GetTypeFlag() != ITEM_FOUNTAIN {
			Act(nil, false, mob, nil, fromObj, nil, "You can't fill something from $p.", "", ToChar)
			return
		}
	}

	if fromObj.GetValue(1) == 0 {
		Act(nil, false, mob, nil, fromObj, nil, "The $p is empty.", "", ToChar)
		return
	}
	if subcmd == scmdPour {
		if arg2 == "" {
			Act(nil, false, mob, nil, nil, nil, "Where do you want it?  Out or in what?", "", ToChar)
			return
		}
		if strings.EqualFold(arg2, "out") {
			w.characterMobPourOut(mob, fromObj)
			return
		}
		toObj = mobObjectInInventory(w, mob, arg2)
		if toObj == nil {
			Act(nil, false, mob, nil, nil, nil, "You can't find it!", "", ToChar)
			return
		}
		if toObj.GetTypeFlag() != ITEM_DRINKCON && toObj.GetTypeFlag() != ITEM_FOUNTAIN {
			Act(nil, false, mob, nil, nil, nil, "You can't pour anything into that.", "", ToChar)
			return
		}
	}
	if toObj == fromObj {
		Act(nil, false, mob, nil, nil, nil, "A most unproductive effort.", "", ToChar)
		return
	}
	if toObj.GetValue(1) != 0 && toObj.GetValue(2) != fromObj.GetValue(2) {
		Act(nil, false, mob, nil, nil, nil, "There is already another liquid in it!", "", ToChar)
		return
	}
	if toObj.GetValue(1) >= toObj.GetValue(0) {
		Act(nil, false, mob, nil, nil, nil, "There is no room for more.", "", ToChar)
		return
	}
	if subcmd == scmdPour {
		mob.SendMessage(fmt.Sprintf("You pour the %s into the %s.", DrinkName(fromObj.GetValue(2)), arg2))
	} else {
		Act(nil, false, mob, nil, toObj, fromObj, "You gently fill $p from $P.", "", ToChar)
		Act(w, true, mob, nil, toObj, fromObj, "$n gently fills $p from $P.", "", ToRoom)
	}
	if toObj.GetValue(1) == 0 {
		setDrinkconName(toObj, fromObj.GetValue(2))
	}
	toObj.SetValue(2, fromObj.GetValue(2))
	amount := toObj.GetValue(0) - toObj.GetValue(1)
	fromObj.SetValue(1, fromObj.GetValue(1)-amount)
	toObj.SetValue(1, toObj.GetValue(0))
	if fromObj.GetValue(1) < 0 {
		toObj.SetValue(1, toObj.GetValue(1)+fromObj.GetValue(1))
		amount += fromObj.GetValue(1)
		fromObj.SetValue(1, 0)
		fromObj.SetValue(2, 0)
		fromObj.SetValue(3, 0)
		clearDrinkconName(fromObj)
	}
	toObj.SetValue(3, boolToInt(toObj.GetValue(3) != 0 || fromObj.GetValue(3) != 0))
	fromObj.SetWeight(fromObj.GetWeight() - amount)
	toObj.SetWeight(toObj.GetWeight() + amount)
}

func mobObjectInInventory(w *World, mob *MobInstance, name string) *ObjectInstance {
	object, _ := w.ResolveObjectInInventory(mob, name)
	return object
}

func mobObjectInRoom(w *World, mob *MobInstance, name string) *ObjectInstance {
	object, _ := w.ResolveObjectInRoom(mob, name)
	return object
}

func (w *World) characterMobPourOut(mob *MobInstance, fromObj *ObjectInstance) {
	Act(w, true, mob, nil, fromObj, nil, "$n empties $p.", "", ToRoom)
	Act(nil, false, mob, nil, fromObj, nil, "You empty $p.", "", ToChar)
	fromObj.SetWeight(fromObj.GetWeight() - fromObj.GetValue(1))
	liquid, poison := fromObj.GetValue(2), fromObj.GetValue(3)
	fromObj.SetValue(1, 0)
	clearDrinkconName(fromObj)
	puddle := w.CreateObject(20, mob.GetRoomVNum())
	if puddle == nil {
		mob.SendMessage("Error, please tell a god.\r\n")
		slog.Error("SYSERR: creating puddle: obj not found")
		return
	}
	puddle.SetValue(2, liquid)
	puddle.SetValue(3, poison)
	puddle.SetTimer(2)
	fromObj.SetValue(2, 0)
	fromObj.SetValue(3, 0)
}

// CharacterPeek ports do_peek for a mobile body. NPC class and skill reads go
// through C's shared dummy_mob player_specials, whose zero values make a
// below-immortal mobile fail the class gate before argument parsing; an
// immortal-level mobile succeeds after the normal percent draw.
func CharacterPeek(ctx CharacterCommandContext) CharacterCommandResult {
	if ctx.World == nil || ctx.Actor == nil {
		return CharacterCommandUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		ctx.World.ExecPeek(player, nil, ctx.RawArgs)
		return CharacterCommandHandled
	}
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	ctx.World.characterMobPeek(mob, ctx.RawArgs)
	return CharacterCommandHandled
}

func (w *World) characterMobPeek(mob *MobInstance, argument string) {
	const npcSpecialClass = ClassMageUser // dummy_mob is zero-initialized.
	if npcSpecialClass != ClassThief && npcSpecialClass != ClassAssassin && mob.GetLevel() < LVL_IMMORT {
		mob.SendMessage("You're not a thief!\r\n")
		return
	}
	name, _ := oneArgument(argument)
	if name == "" {
		mob.SendMessage("Whom do you wish to peek at?\r\n")
		return
	}
	victim, found := w.ResolveCharInRoom(mob, name)
	if !found {
		mob.SendMessage("No-one by that name here.\r\n")
		return
	}
	if victim.Combatant == mob {
		mob.SendMessage("Try the 'inventory' command!\r\n")
		return
	}
	_ = randRange(1, 101) // C draws before the level-conditioned skill check.
	var result ObservationResult
	w.appendCharacterLook(&result, mob, victim)
	w.RenderObservationMessages(result)
}
