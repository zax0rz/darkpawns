package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// Subcommand constants for the consumable operations, matching C SCMD_* values.
const (
	scmdDrink = 0
	scmdSip   = 1
	scmdEat   = 0
	scmdTaste = 1
	scmdPour  = 0
	scmdFill  = 1
)

// consumableNumber returns a uniform random integer in [from, to] inclusive,
// using the process-wide C-compatible stream. It remains a variable so unit
// tests can inject exact rolls without adding a second production generator.
var consumableNumber = dprng.Number

// DoDrink implements C do_drink (src/act.item.c:895-1032) for both drink and sip.
// Player-facing text is dispatched through Act(). All object value mutation goes
// through item.GetValue / item.SetValue so prototypes are never touched.
func (w *World) DoDrink(ch *Player, me *MobInstance, cmd, arg string, subcmd int) {
	_ = me

	arg1, _ := oneArgument(arg)
	if arg1 == "" {
		ch.SendMessage("Drink from what?\r\n")
		return
	}

	// Resolve object: inventory first, then room.
	item, found := ch.Inventory.FindItem(arg1)
	onGround := false
	if !found {
		item = w.findObjectInRoomByName(ch.GetRoomVNum(), arg1)
		if item != nil {
			onGround = true
		}
	}
	if item == nil {
		Act(nil, false, ch, nil, nil, nil, "You can't find it!", "", ToChar)
		return
	}

	objType := item.GetTypeFlag()
	if objType != ITEM_DRINKCON && objType != ITEM_FOUNTAIN {
		ch.SendMessage("You can't drink from that!\r\n")
		return
	}
	if onGround && objType == ITEM_DRINKCON {
		ch.SendMessage("You have to be holding that to drink from it.\r\n")
		return
	}

	// Condition checks (GET_COND equivalents).
	if ch.GetCondition(CondDrunk) > 10 && ch.GetCondition(CondThirst) > 0 {
		ch.SendMessage("You can't seem to get close enough to your mouth.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n tries to drink but misses $s mouth!", "", ToRoom)
		return
	}
	if ch.GetCondition(CondFull) > 20 && ch.GetCondition(CondThirst) > 0 {
		ch.SendMessage("Your stomach can't contain anymore!\r\n")
		return
	}
	if ch.GetCondition(CondThirst) > 40 {
		ch.SendMessage("If you drink any more, you'll explode!\r\n")
		return
	}

	if item.GetValue(1) == 0 {
		ch.SendMessage("It's empty.\r\n")
		return
	}

	liqIndex := item.GetValue(2)
	liqName := DrinkName(liqIndex)
	var amount int

	if subcmd == scmdDrink {
		ch.SendMessage(fmt.Sprintf("You drink the %s.\r\n", liqName))
		drunkAff, _, _ := GetDrinkAffects(liqIndex)
		if drunkAff > 0 {
			amount = (25 - ch.GetCondition(CondThirst)) / drunkAff
		} else {
			amount = consumableNumber(3, 8)
		}
	} else {
		Act(w, true, ch, nil, item, nil, "$n sips from $p.", "", ToRoom)
		ch.SendMessage(fmt.Sprintf("It tastes like %s.\r\n", liqName))
		amount = 0
	}

	// Clamp amount to available liquid and weight.
	amount = min(amount, item.GetValue(1))
	weight := min(amount, item.GetWeight())
	if weight > 0 {
		item.SetWeight(item.GetWeight() - weight)
	}

	drunkAff, fullAff, thirstAff := GetDrinkAffects(liqIndex)
	GainCondition(ch, CondDrunk, (drunkAff*amount)/4)

	// Vampire branch: blood is the only sustenance at night.
	if ch.HasPLRFlag(PlrVampire) && liqIndex != LiqBlood && (GetSunlight() == SunSet || GetSunlight() == SunDark) {
		ch.SendMessage(fmt.Sprintf("The vampirism in your body is not satiated by mere %s...\r\n", liqName))
	} else {
		GainCondition(ch, CondFull, (fullAff*amount)/4)
		GainCondition(ch, CondThirst, (thirstAff*amount)/4)

		if ch.GetCondition(CondDrunk) > 10 {
			ch.SendMessage("You feel drunk.\r\n")
		}
		if ch.GetCondition(CondThirst) > 20 {
			ch.SendMessage("You don't feel thirsty any more.\r\n")
		}
	}
	if ch.GetCondition(CondFull) > 20 {
		ch.SendMessage("You are full.\r\n")
	}

	// Poison handling.
	if item.GetValue(3) != 0 {
		ch.SendMessage("Oops, it tasted rather strange!\r\n")
		Act(w, true, ch, nil, nil, nil, "$n chokes and utters some strange sounds.", "", ToRoom)
		applyPoison(ch, amount*3, "poisoned drink")
	}

	// Deplete the container. C decrements val1 unconditionally for both
	// drinkcons and fountains (fountains simply have a very large val1).
	item.SetValue(1, item.GetValue(1)-amount)
	if item.GetValue(1) <= 0 {
		item.SetValue(1, 0)
		item.SetValue(2, 0)
		item.SetValue(3, 0)
		clearDrinkconName(item)
		if item.VNum == 20 {
			w.ExtractObject(item, ch.GetRoomVNum())
		}
	}
}

// DoEat implements C do_eat (src/act.item.c:1035-1156) for both eat and taste.
// All object value mutation goes through GetValue / SetValue.
func (w *World) DoEat(ch *Player, me *MobInstance, cmd, arg string, subcmd int) {
	_ = me

	arg1, _ := oneArgument(arg)
	if arg1 == "" {
		ch.SendMessage("Eat what?\r\n")
		return
	}

	item, found := ch.Inventory.FindItem(arg1)
	if !found {
		// Werewolves can eat corpses in the room (low-frequency branch).
		if ch.IsAffected(affWerewolf) {
			item = w.findObjectInRoomByName(ch.GetRoomVNum(), arg1)
		}
		if item == nil {
			ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
			return
		}
	}

	objType := item.GetTypeFlag()

	// Taste on a drink container/fountain delegates to sip.
	if subcmd == scmdTaste && (objType == ITEM_DRINKCON || objType == ITEM_FOUNTAIN) {
		w.DoDrink(ch, nil, "sip", arg, scmdSip)
		return
	}

	// Werewolf corpse-rip branch: preserve structure as a TODO follow-up.
	if !found && ch.IsAffected(affWerewolf) && objType == ITEM_CONTAINER && item.GetValue(3) != 0 {
		// TODO: implement werewolf savage-eat corpse branch including
		// mangled-flesh proto 19 spawn. Fidelity follow-up, not DP-1102.
		Act(nil, false, ch, nil, item, nil, "You savagely rip into $p, feeding your insatiable appetite.", "", ToChar)
		Act(w, true, ch, nil, item, nil, "$n savagely rips into $p, crunching through flesh and bone alike.", "", ToRoom)
		return
	}

	if objType != ITEM_FOOD && ch.GetLevel() < LVL_GOD {
		ch.SendMessage("You can't eat THAT!\r\n")
		return
	}
	if ch.GetCondition(CondFull) > 40 {
		Act(nil, false, ch, nil, nil, nil, "You are too full to eat more!", "", ToChar)
		return
	}

	if subcmd == scmdEat {
		Act(nil, false, ch, nil, item, nil, "You eat $p.", "", ToChar)
		Act(w, true, ch, nil, item, nil, "$n eats $p.", "", ToRoom)
	} else {
		Act(nil, false, ch, nil, item, nil, "You nibble a little bit of $p.", "", ToChar)
		Act(w, true, ch, nil, item, nil, "$n tastes a little bit of $p.", "", ToRoom)
	}

	amount := 0
	if subcmd == scmdEat {
		amount = item.GetValue(0)
	}

	if ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet || GetSunlight() == SunDark) {
		ch.SendMessage("The vampirism in your body is not satiated by mere food...\r\n")
	} else {
		GainCondition(ch, CondFull, amount)
	}

	if ch.GetCondition(CondFull) > 20 {
		Act(nil, false, ch, nil, nil, nil, "You are full.", "", ToChar)
	}

	if item.GetValue(3) != 0 && ch.GetLevel() < LVL_IMMORT {
		ch.SendMessage("Oops, that tasted rather strange!\r\n")
		Act(w, true, ch, nil, nil, nil, "$n coughs and utters some strange sounds.", "", ToRoom)
		applyPoison(ch, amount*2, item.GetShortDesc())
	}

	if subcmd == scmdEat {
		ch.Inventory.RemoveItem(item)
		w.ExtractObject(item, ch.GetRoomVNum())
	} else {
		newVal := item.GetValue(0) - 1
		item.SetValue(0, newVal)
		if newVal <= 0 {
			ch.SendMessage("There's nothing left now.\r\n")
			ch.Inventory.RemoveItem(item)
			w.ExtractObject(item, ch.GetRoomVNum())
		}
	}
}

// DoPour implements C do_pour (src/act.item.c:1159-1335) for both pour and fill.
// All object value mutation goes through GetValue / SetValue.
func (w *World) DoPour(ch *Player, me *MobInstance, cmd, arg string, subcmd int) {
	_ = me

	arg1, arg2 := twoArguments(arg)

	var fromObj, toObj *ObjectInstance

	if subcmd == scmdPour {
		if arg1 == "" {
			Act(nil, false, ch, nil, nil, nil, "From what do you want to pour?", "", ToChar)
			return
		}
		fromObj, _ = ch.Inventory.FindItem(arg1)
		if fromObj == nil {
			Act(nil, false, ch, nil, nil, nil, "You can't find it!", "", ToChar)
			return
		}
		if fromObj.GetTypeFlag() != ITEM_DRINKCON {
			Act(nil, false, ch, nil, nil, nil, "You can't pour from that!", "", ToChar)
			return
		}
	} else { // scmdFill
		if arg1 == "" {
			ch.SendMessage("What do you want to fill?  And what are you filling it from?\r\n")
			return
		}
		toObj, _ = ch.Inventory.FindItem(arg1)
		if toObj == nil {
			ch.SendMessage("You can't find it!\r\n")
			return
		}
		if toObj.GetTypeFlag() != ITEM_DRINKCON {
			Act(nil, false, ch, nil, toObj, nil, "You can't fill $p!", "", ToChar)
			return
		}
		if arg2 == "" {
			Act(nil, false, ch, nil, toObj, nil, "What do you want to fill $p from?", "", ToChar)
			return
		}
		fromObj = w.findObjectInRoomByName(ch.GetRoomVNum(), arg2)
		if fromObj == nil {
			ch.SendMessage(fmt.Sprintf("There doesn't seem to be %s %s here.\r\n", an(arg2), arg2))
			return
		}
		if fromObj.GetTypeFlag() != ITEM_FOUNTAIN {
			Act(nil, false, ch, nil, fromObj, nil, "You can't fill something from $p.", "", ToChar)
			return
		}
	}

	if fromObj.GetValue(1) == 0 {
		Act(nil, false, ch, nil, fromObj, nil, "The $p is empty.", "", ToChar)
		return
	}

	if subcmd == scmdPour {
		if arg2 == "" {
			Act(nil, false, ch, nil, nil, nil, "Where do you want it?  Out or in what?", "", ToChar)
			return
		}
		if strings.EqualFold(arg2, "out") {
			Act(w, true, ch, nil, fromObj, nil, "$n empties $p.", "", ToRoom)
			Act(nil, false, ch, nil, fromObj, nil, "You empty $p.", "", ToChar)

			// Empty weight.
			fromObj.SetWeight(fromObj.GetWeight() - fromObj.GetValue(1))

			liq := fromObj.GetValue(2)
			poison := fromObj.GetValue(3)
			fromObj.SetValue(1, 0)
			clearDrinkconName(fromObj)

			// Spawn puddle (vnum 20) in the room.
			puddle := w.CreateObject(20, ch.GetRoomVNum())
			if puddle == nil {
				ch.SendMessage("Error, please tell a god.\r\n")
				slog.Error("SYSERR: creating puddle: obj not found")
				return
			}
			puddle.SetValue(2, liq)
			puddle.SetValue(3, poison)
			puddle.SetTimer(2)

			fromObj.SetValue(2, 0)
			fromObj.SetValue(3, 0)
			return
		}

		toObj, _ = ch.Inventory.FindItem(arg2)
		if toObj == nil {
			Act(nil, false, ch, nil, nil, nil, "You can't find it!", "", ToChar)
			return
		}
		if toObj.GetTypeFlag() != ITEM_DRINKCON && toObj.GetTypeFlag() != ITEM_FOUNTAIN {
			Act(nil, false, ch, nil, nil, nil, "You can't pour anything into that.", "", ToChar)
			return
		}
	}

	if toObj == fromObj {
		Act(nil, false, ch, nil, nil, nil, "A most unproductive effort.", "", ToChar)
		return
	}
	if toObj.GetValue(1) != 0 && toObj.GetValue(2) != fromObj.GetValue(2) {
		Act(nil, false, ch, nil, nil, nil, "There is already another liquid in it!", "", ToChar)
		return
	}
	if toObj.GetValue(1) >= toObj.GetValue(0) {
		Act(nil, false, ch, nil, nil, nil, "There is no room for more.", "", ToChar)
		return
	}

	if subcmd == scmdPour {
		ch.SendMessage(fmt.Sprintf("You pour the %s into the %s.", DrinkName(fromObj.GetValue(2)), arg2))
	}
	if subcmd == scmdFill {
		Act(nil, false, ch, nil, toObj, fromObj, "You gently fill $p from $P.", "", ToChar)
		Act(w, true, ch, nil, toObj, fromObj, "$n gently fills $p from $P.", "", ToRoom)
	}

	// New alias: name the target drinkcon after the liquid if it was empty.
	if toObj.GetValue(1) == 0 {
		setDrinkconName(toObj, fromObj.GetValue(2))
	}

	// First same type liquid.
	toObj.SetValue(2, fromObj.GetValue(2))

	// Then how much to pour.
	amount := toObj.GetValue(0) - toObj.GetValue(1)
	fromObj.SetValue(1, fromObj.GetValue(1)-amount)
	toObj.SetValue(1, toObj.GetValue(0))

	if fromObj.GetValue(1) < 0 {
		// There was too little; clamp target and from.
		toObj.SetValue(1, toObj.GetValue(1)+fromObj.GetValue(1))
		amount += fromObj.GetValue(1)
		fromObj.SetValue(1, 0)
		fromObj.SetValue(2, 0)
		fromObj.SetValue(3, 0)
		clearDrinkconName(fromObj)
	}

	// Poison carries over.
	toObj.SetValue(3, boolToInt(toObj.GetValue(3) != 0 || fromObj.GetValue(3) != 0))

	// Weight changes.
	fromObj.SetWeight(fromObj.GetWeight() - amount)
	toObj.SetWeight(toObj.GetWeight() + amount)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// twoArguments splits s into the first two whitespace-delimited words.
func twoArguments(s string) (string, string) {
	first, rest := oneArgument(s)
	second, _ := oneArgument(rest)
	return first, second
}

// findObjectInRoomByName locates an object in the room by keyword, mirroring
// C get_obj_in_list_vis (isname_with_abbrevs). C does not match short
// descriptions here, so neither do we (R4).
func (w *World) findObjectInRoomByName(roomVNum int, name string) *ObjectInstance {
	for _, obj := range w.GetItemsInRoom(roomVNum) {
		if isnameWithAbbrevs(name, obj.GetKeywords()) {
			return obj
		}
	}
	return nil
}

// setDrinkconName prefixes the object's short description with the liquid name,
// reproducing C name_to_drinkcon().
func setDrinkconName(obj *ObjectInstance, liq int) {
	prefix := DrinkName(liq)
	if prefix == "" || prefix == "unknown" {
		return
	}
	base := obj.GetShortDesc()
	// Avoid double-prefixing.
	if strings.HasPrefix(strings.ToLower(base), strings.ToLower(prefix)+" ") {
		return
	}
	obj.Runtime.ShortDescOverride = fmt.Sprintf("%s %s", prefix, base)
}

// clearDrinkconName strips the leading liquid-name prefix, reproducing C
// name_from_drinkcon().
func clearDrinkconName(obj *ObjectInstance) {
	if obj.Runtime.ShortDescOverride == "" {
		return
	}
	fields := strings.Fields(obj.Runtime.ShortDescOverride)
	if len(fields) > 1 {
		obj.Runtime.ShortDescOverride = strings.Join(fields[1:], " ")
	} else {
		obj.Runtime.ShortDescOverride = ""
	}
}

// applyPoison adds a poison affect to the player and sets the AFF_POISON bit.
func applyPoison(ch *Player, duration int, source string) {
	if duration <= 0 {
		return
	}
	ch.AddAffect(engine.NewAffectDirect(0, engine.ApplyNone, duration, 0, engine.AFFPoison, source))
	ch.SetAffect(affPoison, true)
}

// boolToInt returns 1 if b is true, 0 otherwise.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
