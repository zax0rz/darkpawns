package game

import (
	"fmt"
	"strings"
)

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
