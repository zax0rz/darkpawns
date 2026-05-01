package game

import (
	"fmt"
	"log/slog"
	"strings"
)

// canTakeObj checks if a player can take an object
func (w *World) canTakeObj(ch *Player, obj *ObjectInstance) bool {
	if len(ch.Inventory.Items) >= ch.Inventory.Capacity {
		w.actToChar(ch, "$p: you can't carry that many items.", obj, nil)
		return false
	}
	if ch.Inventory.GetWeight()+obj.GetWeight() > ch.Inventory.Capacity * 10 {
		w.actToChar(ch, "$p: you can't carry that much weight.", obj, nil)
		return false
	}
	// Check ITEM_WEAR_TAKE flag
	hasTake := false
	for _, wf := range obj.Prototype.WearFlags {
		if wf == 1 {
			hasTake = true
			break
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
	if obj.GetTypeFlag() == ITEM_MONEY && obj.Prototype.Values[0] > 0 {
		ch.Inventory.removeItem(obj)
		amount := obj.Prototype.Values[0]
		if amount > 1 {
			ch.SendMessage(fmt.Sprintf("There were %d coins.\r\n", amount))
		}
		ch.Gold += amount
	}
}

// performGetFromContainer gets an item from a container
func (w *World) performGetFromContainer(ch *Player, obj, cont *ObjectInstance, mode int) {
	if mode == findObjInv || w.canTakeObj(ch, obj) {
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

// doGet handles the get/take command
func (w *World) doGet(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		ch.SendMessage("Get what?\r\n")
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		// get all
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil || len(w.roomItems[ch.RoomVNum]) == 0 {
			ch.SendMessage("There doesn't seem to be anything here.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(w.roomItems[ch.RoomVNum]))
		copy(items, w.roomItems[ch.RoomVNum])
		for _, obj := range items {
			w.performGetFromRoom(ch, obj)
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("What do you want to get all of?\r\n")
			return true
		}
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil {
			return true
		}
		items := make([]*ObjectInstance, len(w.roomItems[ch.RoomVNum]))
		copy(items, w.roomItems[ch.RoomVNum])
		found := false
		for _, obj := range items {
			if isname(keyword, obj.GetKeywords()) {
				w.performGetFromRoom(ch, obj)
				found = true
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't see any %ss here.\r\n", keyword))
		}
		return true
	}

	// Individual item
	if arg2 == "" {
		// get <item> from room
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil {
			return true
		}
		var obj *ObjectInstance
		for _, o := range w.roomItems[ch.RoomVNum] {
			if isname(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
			return true
		}
		w.performGetFromRoom(ch, obj)
		return true
	}

	// get <item> <container>
	// Find container and determine mode
	var cont *ObjectInstance
	mode := findObjRoom
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			cont = obj
			mode = findObjInv
			break
		}
	}
	if cont == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.RoomVNum] {
				if isname(arg2, obj.GetKeywords()) {
					cont = obj
					break
				}
			}
		}
	}
	if cont == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg2), arg2))
		return true
	}
	if cont.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "$p is not a container.", cont, nil)
		return true
	}
	if contIsClosed(cont) {
		w.actToChar(ch, "$p is closed.", cont, nil)
		return true
	}

	// Find item inside container
	var obj *ObjectInstance
	for _, o := range cont.Contains {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("There doesn't seem to be %s %s in %s.\r\n", an(arg1), arg1, cont.GetShortDesc()))
		return true
	}

	// Perform the get
	w.performGetFromContainer(ch, obj, cont, mode)
	return true
}

// performDrop drops an item to the room floor
func (w *World) performDrop(ch *Player, obj *ObjectInstance) {
	if objHasFlag(obj, 1<<0) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, "You can't let go of $p!!  Yeech!", obj, nil)
		return
	}
	if err := w.MoveObjectToRoom(obj, ch.GetRoomVNum()); err != nil {
		slog.Error("drop failed: MoveObjectToRoom", "player", ch.Name, "obj_vnum", obj.VNum, "error", err)
		w.actToChar(ch, "You can't drop that right now.\n", nil, nil)
		return
	}
	w.actToChar(ch, "You drop $p.", obj, nil)
	w.actToRoom(ch, "$n drops $p.", obj, nil)
}

// doDrop handles the drop command
func (w *World) doDrop(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Drop what?\r\n")
		return true
	}
	arg1 := parts[0]
	sname := "drop"

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be carrying anything.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		for _, obj := range items {
			w.performDrop(ch, obj)
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
			if isname(keyword, obj.GetKeywords()) {
				w.performDrop(ch, obj)
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
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	w.performDrop(ch, obj)
	return true
}

// performGive gives an object to a player
func (w *World) performGive(ch *Player, vict *Player, obj *ObjectInstance) {
	if objHasFlag(obj, 1<<0) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, "You can't let go of $p!!  Yeech!", obj, nil)
		return
	}
	if len(vict.Inventory.Items) >= vict.Inventory.Capacity {
		w.actToChar(ch, "$N seems to have $S hands full.", vict, obj)
		return
	}
	if obj.GetWeight()+vict.Inventory.GetWeight() > vict.Inventory.Capacity * 10 {
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

// giveFindVict finds the victim for a give command
func (w *World) giveFindVict(ch *Player, arg string) *Player {
	if arg == "" {
		ch.SendMessage("To who?\r\n")
		return nil
	}
	vict := w.FindPlayerInRoom(ch.GetRoomVNum(), arg)
	if vict == nil {
		ch.SendMessage("There doesn't seem to be anyone here by that name.\r\n")
		return nil
	}
	if vict == ch {
		ch.SendMessage("What's the point of that?\r\n")
		return nil
	}
	return vict
}

// performGiveGold gives gold coins to a player
func (w *World) performGiveGold(ch *Player, vict *Player, amount int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh ... we are jolly funny today, eh?\r\n")
		return
	}

	ch.mu.Lock()
	if ch != vict {
		vict.mu.Lock()
	}

	if ch.Gold < amount && ch.GetLevel() < lvlGod {
		if ch != vict {
			vict.mu.Unlock()
		}
		ch.mu.Unlock()
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}

	ch.SendMessage("Ok.\r\n")
	actToVictim(ch, vict, "$n gives you %d gold coins.", nil, nil)
	w.actToRoomExclude(ch, vict, "$n gives %s to $N.", nil, vict)

	if ch.GetLevel() < lvlGod {
		ch.Gold -= amount
	}
	vict.Gold += amount

	if ch != vict {
		vict.mu.Unlock()
	}
	ch.mu.Unlock()
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
	vict := w.giveFindVict(ch, victName)
	if vict == nil {
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findIndiv {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isname(arg1, o.GetKeywords()) {
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
		if dotmode == findAlldot && len(parts) > 0 {
			// Strip all. prefix
			keyword := arg1[4:]
			if keyword == "" && len(parts) > 1 {
				keyword = parts[0]
			}
		}
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be holding anything.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		for _, obj := range items {
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if dotmode == findAll || isname(arg1, obj.GetKeywords()) {
				w.performGive(ch, vict, obj)
			}
		}
	}
	return true
}
