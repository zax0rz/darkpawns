//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unsafe"
)

// canTakeObj checks if a player can take an object
func (w *World) canTakeObj(ch *Player, obj *ObjectInstance) bool {
	if len(ch.Inventory.Items) >= ch.MaxCarryItems() {
		w.actToChar(ch, "$p: you can't carry that many items.", obj, nil)
		return false
	}
	if ch.CarriedWeight()+obj.GetWeight() > ch.MaxCarryWeight() {
		w.actToChar(ch, "$p: you can't carry that much weight.", obj, nil)
		return false
	}
	// CAN_WEAR(obj, ITEM_WEAR_TAKE): WearFlags stores bitmasks, not one
	// flag value per array element.
	hasTake := obj.GetTypeFlag() == ITEM_MONEY
	if obj.Prototype != nil {
		for _, wf := range obj.Prototype.WearFlags {
			if wf&1 != 0 {
				hasTake = true
				break
			}
		}
	}
	if !hasTake {
		w.actToChar(ch, "$p: you can't take that!", obj, nil)
		return false
	}
	return true
}

// getCheckMoney handles auto-conversion of money items on pickup
func (w *World) getCheckMoney(ch *Player, obj *ObjectInstance) {
	amount := obj.GetValue(0)
	if obj.GetTypeFlag() == ITEM_MONEY && amount > 0 {
		w.ExtractObject(obj, ch.GetRoomVNum())
		if amount > 1 {
			ch.SendMessage(fmt.Sprintf("There were %d coins.\r\n", amount))
		}
		ch.SetGold(ch.GetGold() + amount)
	}
}

// performGetFromContainer gets an item from a container
func (w *World) performGetFromContainer(ch *Player, obj, cont *ObjectInstance, mode int) {
	if mode == findObjInv || w.canTakeObj(ch, obj) {
		if len(ch.Inventory.Items) >= ch.MaxCarryItems() {
			w.actToChar(ch, "$p: you can't hold any more items.", obj, nil)
			return
		}
		// Ensure Location is set so MoveObject's detach can find the container
		obj.Location = LocContainer(cont.ID)
		if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
			w.actToChar(ch, "You can't carry that much.\n", nil, nil)
			// Rollback: move back into container (MoveObject handles re-attach)
			if rbErr := w.MoveObjectToContainer(obj, cont); rbErr != nil {
				slog.Error("rollback after failed get: container restore failed", "player", ch.Name, "obj_vnum", obj.VNum, "error", rbErr)
			}
			return
		}
		w.actToChar(ch, "You get $p from $P.", obj, cont)
		w.actToRoom(ch, "$n gets $p from $P.", obj, cont)
		w.getCheckMoney(ch, obj)
	}
}

// getFromContainer implements C get_from_container() for individual, all,
// and all.<keyword> object modes.
func (w *World) getFromContainer(ch *Player, cont *ObjectInstance, arg string, mode int) {
	if contIsClosed(cont) {
		w.actToChar(ch, "$p is closed.", cont, nil)
		return
	}

	dotmode := findAllDots(arg)
	if dotmode == findIndiv {
		for _, obj := range cont.Contains {
			if isnameWithAbbrevs(arg, obj.GetKeywords()) && canSeeObject(ch, obj) {
				w.performGetFromContainer(ch, obj, cont, mode)
				return
			}
		}
		w.actToChar(ch, fmt.Sprintf("There doesn't seem to be %s %s in $p.", an(arg), arg), cont, nil)
		return
	}

	keyword := strings.TrimPrefix(arg, "all.")
	if dotmode == findAlldot && keyword == "" {
		ch.SendMessage("Get all of what?\r\n")
		return
	}

	items := append([]*ObjectInstance(nil), cont.Contains...)
	found := false
	for _, obj := range items {
		if canSeeObject(ch, obj) && (dotmode == findAll || isnameWithAbbrevs(keyword, obj.GetKeywords())) {
			found = true
			w.performGetFromContainer(ch, obj, cont, mode)
		}
	}
	if found {
		return
	}
	if dotmode == findAll {
		w.actToChar(ch, "$p seems to be empty.", cont, nil)
	} else {
		w.actToChar(ch, fmt.Sprintf("You can't seem to find any %ss in $p.", keyword), cont, nil)
	}
}

// performGetFromRoom picks up an item from the room floor
func (w *World) performGetFromRoom(ch *Player, obj *ObjectInstance) {
	if w.canTakeObj(ch, obj) {
		if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
			w.actToChar(ch, "You can't carry that much.\n", nil, nil)
			// Put item back in room (MoveObject handles rollback internally)
			return
		}
		w.actToChar(ch, "You get $p.", obj, nil)
		w.actToRoom(ch, "$n gets $p.", obj, nil)
		w.getCheckMoney(ch, obj)
	}
}

func (w *World) getFromRoom(ch *Player, arg string) {
	// C get_from_room (act.item.c:297) rejects a mounted actor before any
	// lookup: a rider cannot reach objects on the floor.
	if ch.IsMounted() {
		ch.SendMessage("You can't reach it from your mount.\r\n")
		return
	}
	dotmode := findAllDots(arg)
	items := append([]*ObjectInstance(nil), w.roomItems[ch.GetRoom()]...)
	if dotmode == findIndiv {
		for _, obj := range items {
			if isnameWithAbbrevs(arg, obj.GetKeywords()) && canSeeObject(ch, obj) {
				w.performGetFromRoom(ch, obj)
				return
			}
		}
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg), arg))
		return
	}

	keyword := strings.TrimPrefix(arg, "all.")
	if dotmode == findAlldot && keyword == "" {
		ch.SendMessage("Get all of what?\r\n")
		return
	}
	found := false
	for _, obj := range items {
		if canSeeObject(ch, obj) && (dotmode == findAll || isnameWithAbbrevs(keyword, obj.GetKeywords())) {
			found = true
			w.performGetFromRoom(ch, obj)
		}
	}
	if found {
		return
	}
	if dotmode == findAll {
		ch.SendMessage("There doesn't seem to be anything here.\r\n")
	} else {
		ch.SendMessage(fmt.Sprintf("You don't see any %ss here.\r\n", keyword))
	}
}

// DoGet is the exported wrapper for doGet. Handles player get commands.
func (w *World) DoGet(ch *Player, arg string) {
	w.doGet(ch, nil, "get", arg)
}

// DoGive is the exported wrapper for doGive. Handles player give commands.
func (w *World) DoGive(ch *Player, arg string) {
	w.doGive(ch, nil, "give", arg)
}

// DoDrop is the exported wrapper for doDrop. Handles player drop commands.
func (w *World) DoDrop(ch *Player, arg string) {
	w.doDrop(ch, nil, "drop", arg)
}

// doGet handles the get/take command
func (w *World) doGet(ch *Player, me *MobInstance, cmd, arg string) bool {
	// C do_get parses with two_arguments (interpreter.c), which drops fill
	// words ("get in pack" -> "get pack") and lowercases each argument.
	arg1, arg2 := twoArguments(arg)

	if len(ch.Inventory.Items) >= ch.MaxCarryItems() {
		ch.SendMessage("Your arms are already full!\r\n")
		return true
	}
	if arg1 == "" {
		ch.SendMessage("Get what?\r\n")
		return true
	}
	if arg2 == "" {
		w.getFromRoom(ch, arg1)
		return true
	}

	contDotmode := findAllDots(arg2)
	if contDotmode != findIndiv {
		keyword := strings.TrimPrefix(arg2, "all.")
		if contDotmode == findAlldot && keyword == "" {
			ch.SendMessage("Get from all of what?\r\n")
			return true
		}
		found := false
		containers := append([]*ObjectInstance(nil), ch.Inventory.Items...)
		containers = append(containers, w.roomItems[ch.GetRoom()]...)
		for _, cont := range containers {
			if !canSeeObject(ch, cont) || (contDotmode == findAlldot && !isnameWithAbbrevs(keyword, cont.GetKeywords())) {
				continue
			}
			if cont.GetTypeFlag() != ITEM_CONTAINER {
				if contDotmode == findAlldot {
					found = true
					w.actToChar(ch, "$p is not a container.", cont, nil)
				}
				continue
			}
			found = true
			mode := findObjRoom
			if cont.Location.InInventoryOfPlayer(ch.Name) {
				mode = findObjInv
			}
			w.getFromContainer(ch, cont, arg1, mode)
		}
		if !found {
			if contDotmode == findAll {
				ch.SendMessage("You can't seem to find any containers.\r\n")
			} else {
				ch.SendMessage(fmt.Sprintf("You can't seem to find any %ss here.\r\n", keyword))
			}
		}
		return true
	}

	var cont *ObjectInstance
	mode := findObjRoom
	for _, obj := range ch.Inventory.Items {
		if isnameWithAbbrevs(arg2, obj.GetKeywords()) {
			cont = obj
			mode = findObjInv
			break
		}
	}
	if cont == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.GetRoom()] {
				if isnameWithAbbrevs(arg2, obj.GetKeywords()) {
					cont = obj
					break
				}
			}
		}
	}
	if cont == nil {
		ch.SendMessage(fmt.Sprintf("You don't have %s %s.\r\n", an(arg2), arg2))
		return true
	}
	if cont.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "$p is not a container.", cont, nil)
		return true
	}
	w.getFromContainer(ch, cont, arg1, mode)
	return true
}

// performDrop drops an item to the room floor
func (w *World) performDropNamed(ch *Player, obj *ObjectInstance, sname string) {
	if obj.HasExtraFlag(0, extraFlagNoDrop) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, fmt.Sprintf("You can't %s $p, it must be CURSED!", sname), obj, nil)
		return
	}
	if err := w.MoveObjectToRoom(obj, ch.GetRoomVNum()); err != nil {
		slog.Error("drop failed: MoveObjectToRoom", "player", ch.Name, "obj_vnum", obj.VNum, "error", err)
		w.actToChar(ch, "You can't drop that right now.\n", nil, nil)
		return
	}
	w.actToChar(ch, fmt.Sprintf("You %s $p.", sname), obj, nil)
	w.actToRoom(ch, fmt.Sprintf("$n %ss $p.", sname), obj, nil)
}

func (w *World) performDrop(ch *Player, obj *ObjectInstance) {
	w.performDropNamed(ch, obj, "drop")
}

func (w *World) performDropGold(ch *Player, amount int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh.. we are jolly funny today, eh?\r\n")
		return
	}
	if ch.GetGold() < amount {
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}

	money := w.createMoneyObject(amount)
	if err := w.MoveObjectToRoom(money, ch.GetRoomVNum()); err != nil {
		slog.Error("drop gold failed", "player", ch.Name, "amount", amount, "error", err)
		ch.SendMessage("You can't drop that right now.\r\n")
		return
	}
	ch.SetWaitState(1)
	ch.SetGold(ch.GetGold() - amount)
	ch.SendMessage("You drop some gold.\r\n")
	w.actToRoom(ch, fmt.Sprintf("$n drops %s.", createMoneyDesc(amount)), nil, nil)
}

// doDrop handles the drop command
func (w *World) doDrop(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("What do you want to drop?\r\n")
		return true
	}
	arg1 := parts[0]
	sname := "drop"

	if amount, err := strconv.Atoi(arg1); err == nil {
		if len(parts) > 1 && (strings.EqualFold(parts[1], "coin") || strings.EqualFold(parts[1], "coins")) {
			w.performDropGold(ch, amount)
		} else {
			ch.SendMessage("Sorry, you can't do that to more than one item at a time.\r\n")
		}
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be carrying anything.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		for _, obj := range items {
			w.performDropNamed(ch, obj, sname)
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage(fmt.Sprintf("What do you want to %s all of?\r\n", sname))
			return true
		}
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		found := false
		for _, obj := range items {
			if isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				w.performDropNamed(ch, obj, sname)
				found = true
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual drop
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isnameWithAbbrevs(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	w.performDropNamed(ch, obj, sname)
	return true
}

// performGive gives an object to a player
func (w *World) performGive(ch *Player, vict *Player, obj *ObjectInstance) {
	if obj.HasExtraFlag(0, extraFlagNoDrop) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, "You can't let go of $p!!  Yeech!", obj, nil)
		return
	}
	if vict.Inventory.IsFull() {
		w.actToChar(ch, "$N seems to have $S hands full.", vict, obj)
		return
	}
	if obj.GetWeight()+vict.Inventory.GetWeight() > vict.Inventory.GetCapacity()*10 {
		w.actToChar(ch, "$E can't carry that much weight.", vict, nil)
		return
	}
	if err := w.MoveObjectToPlayerInventory(obj, vict); err != nil {
		w.actToChar(ch, "$E can't carry any more.\n", vict, nil)
		// Give item back to giver
		if rbErr := w.MoveObjectToPlayerInventory(obj, ch); rbErr != nil {
			slog.Error("rollback after failed give: restore to giver failed", "player", ch.Name, "obj_vnum", obj.VNum, "error", rbErr)
		}
		return
	}
	w.actToChar(ch, "You give $p to $N.", obj, vict)
	actToVictim(ch, vict, "$n gives you $p.", obj, nil)
	w.actToRoomExclude(ch, vict, "$n gives $p to $N.", obj, vict)
}

func (w *World) performGiveToMob(ch *Player, vict *MobInstance, obj *ObjectInstance) {
	if !vict.HasFlag("OKGIVE") && ch.GetLevel() < lvlImmort {
		Act(nil, false, ch, vict, obj, nil, "$N doesn't seem to be interested in $p.", "", ToChar)
		return
	}
	if obj.HasExtraFlag(0, extraFlagNoDrop) && ch.GetLevel() < lvlImmort {
		Act(nil, false, ch, nil, obj, nil, "You can't let go of $p!!  Yeech!", "", ToChar)
		return
	}
	if obj.GetWeight()+mobCarriedWeight(vict) > mobMaxCarryWeight(vict) {
		Act(nil, false, ch, vict, nil, nil, "$E can't carry that much weight.", "", ToChar)
		return
	}
	if err := w.MoveObjectToMobInventory(obj, vict); err != nil {
		slog.Error("give to mob failed", "player", ch.Name, "mob_vnum", vict.GetVNum(), "obj_vnum", obj.VNum, "error", err)
		Act(nil, false, ch, vict, nil, nil, "$E can't carry that much weight.", "", ToChar)
		return
	}

	Act(nil, false, ch, vict, obj, nil, "You give $p to $N.", "", ToChar)
	Act(nil, false, ch, vict, obj, nil, "$n gives you $p.", "", ToVict)
	Act(w, true, ch, vict, obj, nil, "$n gives $p to $N.", "", ToNotVict)

	// C runs the mob's ongive behavior only after the object has moved.
	if ScriptEngine != nil && vict.HasScript("ongive") {
		ctx := vict.CreateScriptContext(ch, obj, "")
		ctx.World = NewWorldScriptableAdapter(w)
		ctx.RoomVNum = vict.GetRoom()
		if _, err := vict.RunScript("ongive", ctx); err != nil {
			slog.Warn("ongive script error", "mob_vnum", vict.GetVNum(), "error", err)
		}
	}
}

// giveFindVict finds the victim for a give command
func (w *World) giveFindVict(ch *Player, arg string) *Player {
	if arg == "" {
		ch.SendMessage("To who?\r\n")
		return nil
	}
	vict := w.FindPlayerInRoom(ch.GetRoomVNum(), arg)
	if vict == nil {
		ch.SendMessage(NoPersonHere)
		return nil
	}
	if vict == ch {
		ch.SendMessage("What's the point of that?\r\n")
		return nil
	}
	return vict
}

// performGiveGold gives gold coins to a player
// Lock ordering: always lock by pointer address to prevent ABBA deadlock.
func (w *World) performGiveGold(ch *Player, vict *Player, amount int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh ... we are jolly funny today, eh?\r\n")
		return
	}

	// Consistent lock ordering by pointer address — prevents deadlock
	// when two players exchange gold simultaneously.
	first, second := ch, vict
	if ch != vict && uintptr(unsafe.Pointer(&ch.mu)) > uintptr(unsafe.Pointer(&vict.mu)) {
		first, second = vict, ch
	}
	first.mu.Lock()
	if first != second {
		second.mu.Lock()
	}

	// Always operate on ch/vict regardless of lock order
	if ch.Gold < amount && ch.GetLevel() < lvlGod {
		if first != second {
			second.mu.Unlock()
		}
		first.mu.Unlock()
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}

	ch.SendMessage("Ok.\r\n")
	actToVictim(ch, vict, "$n gives you %d gold coins.", nil, nil)
	w.actToRoomExclude(ch, vict, "$n gives %s to $N.", nil, vict)

	if ch.GetLevel() < lvlGod {
		ch.SetGold(ch.GetGold() - amount)
	}
	vict.SetGold(vict.GetGold() + amount)

	if first != second {
		second.mu.Unlock()
	}
	first.mu.Unlock()
}

// doGive handles the give command
func (w *World) doGive(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Give what to who?\r\n")
		return true
	}

	arg1 := parts[0]

	// Check if first arg is a number (gold)
	if isNumber(arg1) {
		amount := atoi(arg1)
		if len(parts) < 2 {
			ch.SendMessage("Give what to who?\r\n")
			return true
		}
		// Check for "coins" or "coin" keyword
		arg2 := parts[1]
		if strings.EqualFold(arg2, "coins") || strings.EqualFold(arg2, "coin") {
			victName := ""
			if len(parts) > 2 {
				victName = parts[2]
			}
			// Check mob before player — bribe path
			mob := w.FindMobInRoom(ch.GetRoomVNum(), victName)
			if mob != nil {
				if ch.Gold < amount {
					ch.SendMessage("You don't have that many coins!\r\n")
					return true
				}
				ch.SetGold(ch.GetGold() - amount)
				w.MpGive(mob, ch, amount)
				if ScriptEngine != nil && mob.HasScript("bribe") {
					ctx := mob.CreateScriptContext(ch, nil, fmt.Sprintf("%d", amount))
					ctx.World = NewWorldScriptableAdapter(w)
					ctx.RoomVNum = mob.GetRoom()
					if _, err := mob.RunScript("bribe", ctx); err != nil {
						slog.Warn("bribe script error", "mob_vnum", mob.GetVNum(), "error", err)
					}
				}
				return true
			}
			vict := w.giveFindVict(ch, victName)
			if vict != nil {
				w.performGiveGold(ch, vict, amount)
			}
			return true
		}
		// Just a number wasn't coins
		ch.SendMessage("You can't give more than one item at a time.\r\n")
		return true
	}

	// Give object
	victName := ""
	if len(parts) > 1 {
		victName = parts[1]
	}

	// Check mob before player — ongive/item path
	mob := w.FindMobInRoom(ch.GetRoomVNum(), victName)
	if mob != nil {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isnameWithAbbrevs(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
			return true
		}
		w.performGiveToMob(ch, mob, obj)
		return true
	}

	vict := w.giveFindVict(ch, victName)
	if vict == nil {
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findIndiv {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isnameWithAbbrevs(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
			return true
		}
		w.performGive(ch, vict, obj)
	} else {
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be holding anything.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		for _, obj := range items {
			// CAN_SEE_OBJ(ch, obj) — act.item.c:824. The C give-all loop skips
			// items the giver can't see; it has no extra-flags bit check.
			if !canSeeObject(ch, obj) {
				continue
			}
			if dotmode == findAll || isnameWithAbbrevs(arg1, obj.GetKeywords()) {
				w.performGive(ch, vict, obj)
			}
		}
	}
	return true
}
