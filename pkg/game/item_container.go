//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"strings"
)

// performPut puts an object into a container
func (w *World) performPut(ch *Player, obj, cont *ObjectInstance) {
	if cont.GetTotalWeight()+obj.GetWeight() > cont.Prototype.Values[contCapacity] {
		w.actToChar(ch, "$p won't fit in $P.", obj, cont)
		return
	}
	if err := w.MoveObjectToContainer(obj, cont); err != nil {
		w.actToChar(ch, "You can't put that in there.", obj, cont)
		return
	}
	w.actToChar(ch, "You put $p in $P.", obj, cont)
	w.actToRoom(ch, "$n puts $p in $P.", obj, cont)
}

// doPut handles the put command
// DoPut is the exported wrapper for doPut. Handles player put commands.
func (w *World) DoPut(ch *Player, arg string) {
	w.doPut(ch, nil, "put", arg)
}

func (w *World) doPut(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	objDotmode := findAllDots(arg1)
	contDotmode := findAllDots(arg2)

	if arg1 == "" {
		ch.SendMessage("Put what in what?\r\n")
		return true
	}
	if contDotmode != findIndiv {
		ch.SendMessage("You can only put things into one container at a time.\r\n")
		return true
	}
	if arg2 == "" {
		what := "it"
		if objDotmode != findIndiv {
			what = "them"
		}
		ch.SendMessage(fmt.Sprintf("What do you want to put %s in?\r\n", what))
		return true
	}

	// Find container in inventory or room
	var cont *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			cont = obj
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
		ch.SendMessage("You'd better open it first!\r\n")
		return true
	}

	if objDotmode == findIndiv {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isname(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You aren't carrying %s %s.\r\n", an(arg1), arg1))
			return true
		}
		if obj == cont {
			ch.SendMessage("You attempt to fold it into itself, but fail.\r\n")
			return true
		}
		w.performPut(ch, obj, cont)
	} else {
		found := false
		for _, obj := range ch.Inventory.Items {
			if obj == cont {
				continue
			}
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if objDotmode == findAll || isname(arg1, obj.GetKeywords()) {
				found = true
				w.performPut(ch, obj, cont)
			}
		}
		if !found {
			if objDotmode == findAll {
				ch.SendMessage("You don't seem to have anything to put in it.\r\n")
			} else {
				ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", arg1))
			}
		}
	}
	return true
}
