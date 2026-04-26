package game

import (
	"fmt"
	"strings"
)

// doDrink handles the drink command
func (w *World) doDrink(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Drink from what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Find drink container in inventory first, then room
	var temp *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			temp = obj
			break
		}
	}
	onGround := false
	if temp == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, obj.GetKeywords()) {
					temp = obj
					onGround = true
					break
				}
			}
		}
	}
	if temp == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if temp.GetTypeFlag() != ITEM_DRINKCON && temp.GetTypeFlag() != ITEM_FOUNTAIN {
		ch.SendMessage("You can't drink from that!\r\n")
		return true
	}
	if onGround && temp.GetTypeFlag() == ITEM_DRINKCON {
		ch.SendMessage("You have to be holding that to drink from it.\r\n")
		return true
	}

	// Condition checks (simplified)
	liqType := temp.Prototype.Values[2]
	if liqType < 0 || liqType >= len(drinks) {
		liqType = 0
	}

	if temp.Prototype.Values[1] <= 0 {
		ch.SendMessage("It's empty.\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You drink the %s.\r\n", drinks[liqType]))

	drunkAff := drinkAff[liqType][0]
	fullAff := drinkAff[liqType][1]
	thirstAff := drinkAff[liqType][2]

	// Calculate amount to drink
	var amount int
	if drunkAff > 0 {
		amount = 3 + len(drinks)/2 // approximate: number(3,8) or condition-based
		if amount > temp.Prototype.Values[1] {
			amount = temp.Prototype.Values[1]
		}
	} else {
		amount = 3 + len(drinks)/2
		if amount > temp.Prototype.Values[1] {
			amount = temp.Prototype.Values[1]
		}
	}

	w.actToRoom(ch, "$n drinks $p.", temp, nil)

	// Reduce weight
	weightLoss := amount
	if weightLoss > temp.GetWeight() {
		weightLoss = temp.GetWeight()
	}
	_ = weightLoss // weight tracking simplified

	// Update condition
	_ = drunkAff
	_ = fullAff
	_ = thirstAff

	// Empty the container
	temp.Prototype.Values[1] -= amount
	if temp.Prototype.Values[1] <= 0 {
		temp.Prototype.Values[1] = 0
		temp.Prototype.Values[2] = 0
		temp.Prototype.Values[3] = 0
	}

	ch.SendMessage("You feel refreshed.\r\n")
	return true
}

// doEat handles the eat command
func (w *World) doEat(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Eat what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Find food in inventory
	var food *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			food = obj
			break
		}
	}
	if food == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	if food.GetTypeFlag() != ITEM_FOOD && ch.GetLevel() < lvlGod {
		ch.SendMessage("You can't eat THAT!\r\n")
		return true
	}

	foodVal := 0
	if len(food.Prototype.Values) > 0 {
		foodVal = food.Prototype.Values[0]
	}

	w.actToChar(ch, "You eat $p.", food, nil)
	w.actToRoom(ch, "$n eats $p.", food, nil)

	_ = foodVal

	// Consume the food
	w.MoveObjectToNowhere(food)
	ch.SendMessage("That was good!\r\n")
	return true
}

// doPour handles the pour command
func (w *World) doPour(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) == 0 {
		w.actToChar(ch, "From what do you want to pour?", nil, nil)
		return true
	}
	arg1 := strings.TrimSpace(parts[0])
	arg2 := ""
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		w.actToChar(ch, "From what do you want to pour?", nil, nil)
		return true
	}

	// Find from_obj in inventory
	var fromObj *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			fromObj = obj
			break
		}
	}
	if fromObj == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if fromObj.GetTypeFlag() != ITEM_DRINKCON {
		w.actToChar(ch, "You can't pour from that!", nil, nil)
		return true
	}
	if fromObj.Prototype.Values[1] <= 0 {
		w.actToChar(ch, "The $p is empty.", fromObj, nil)
		return true
	}
	if arg2 == "" {
		w.actToChar(ch, "Where do you want it?  Out or in what?", nil, nil)
		return true
	}

	if strings.EqualFold(arg2, "out") {
		// Pour out
		w.actToRoom(ch, "$n empties $p.", fromObj, nil)
		w.actToChar(ch, "You empty $p.", fromObj, nil)
		fromObj.Prototype.Values[1] = 0
		fromObj.Prototype.Values[2] = 0
		fromObj.Prototype.Values[3] = 0
		return true
	}

	// Pour into another container
	var toObj *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			toObj = obj
			break
		}
	}
	if toObj == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if toObj.GetTypeFlag() != ITEM_DRINKCON && toObj.GetTypeFlag() != ITEM_FOUNTAIN {
		w.actToChar(ch, "You can't pour anything into that.", nil, nil)
		return true
	}
	if toObj == fromObj {
		w.actToChar(ch, "A most unproductive effort.", nil, nil)
		return true
	}
	if toObj.Prototype.Values[1] != 0 && toObj.Prototype.Values[2] != fromObj.Prototype.Values[2] {
		w.actToChar(ch, "There is already another liquid in it!", nil, nil)
		return true
	}
	if toObj.Prototype.Values[1] >= toObj.Prototype.Values[0] {
		w.actToChar(ch, "There is no room for more.", nil, nil)
		return true
	}

	w.actToChar(ch, fmt.Sprintf("You pour the %s into the %s.", drinks[fromObj.Prototype.Values[2]], arg2), nil, nil)

	// Perform the pour
	toObj.Prototype.Values[2] = fromObj.Prototype.Values[2]
	amount := toObj.Prototype.Values[0] - toObj.Prototype.Values[1]
	fromObj.Prototype.Values[1] -= amount
	toObj.Prototype.Values[1] = toObj.Prototype.Values[0]

	if fromObj.Prototype.Values[1] < 0 {
		toObj.Prototype.Values[1] += fromObj.Prototype.Values[1]
		fromObj.Prototype.Values[1] = 0
		fromObj.Prototype.Values[2] = 0
		fromObj.Prototype.Values[3] = 0
	}

	// Poison carries over
	if fromObj.Prototype.Values[3] != 0 {
		toObj.Prototype.Values[3] = 1
	}

	return true
}
