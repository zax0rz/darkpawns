package game

import "strings"

func (w *World) showObjToChar(obj *ObjectInstance, ch *Player, mode int) {
	var buf string

	switch {
	case mode == 0 && obj.Prototype.LongDesc != "":
		buf = obj.Prototype.LongDesc
	case mode == 5 || mode == 6:
		w.showObjExamine(obj, ch, mode == 6)
		return
	default:
		if obj.Prototype.ShortDesc != "" {
			buf = obj.Prototype.ShortDesc
		}
	}

	if mode != 3 {
		flags := w.getObjectExtraFlags(obj)
		if flags != "" {
			buf += " " + flags
		}
	}

	buf += "\r\n"
	ch.SendMessage(buf)
}

func (w *World) showObjExamine(obj *ObjectInstance, ch *Player, showExtras bool) {
	switch obj.Prototype.TypeFlag {
	case ITEM_NOTE:
		if obj.Prototype.ActionDesc != "" {
			ch.SendMessage("There is something written upon it:\r\n\r\n" + obj.Prototype.ActionDesc)
		} else {
			ch.SendMessage("It's blank.\r\n")
		}
		return
	case ITEM_DRINKCON:
		ch.SendMessage("A drink container.\r\n")
		return
	case ITEM_FOUNTAIN:
		ch.SendMessage("A fountain.\r\n")
		return
	case ITEM_CONTAINER:
		ch.SendMessage("A container.\r\n")
		return
	}
	if showExtras {
		flags := w.getObjectExtraFlags(obj)
		if flags != "" {
			ch.SendMessage("You see nothing special... " + flags + "\r\n")
			return
		}
	}
	ch.SendMessage("You see nothing special...\r\n")
}

func (w *World) getObjectExtraFlags(obj *ObjectInstance) string {
	var flags []string
	ef := obj.Prototype.ExtraFlags
	if len(ef) > 0 && ef[0]&1 != 0 {
		flags = append(flags, "(invisible)")
	}
	if len(ef) > 0 && ef[0]&4 != 0 {
		flags = append(flags, "(glowing)")
	}
	if len(ef) > 0 && ef[0]&8 != 0 {
		flags = append(flags, "(humming)")
	}
	if len(ef) > 1 && ef[1]&1 != 0 {
		flags = append(flags, "(blessed)")
	}
	return strings.Join(flags, " ")
}

// ---------------------------------------------------------------------------
// doScore — shows player stats (ACMD(do_score) in C)
// ---------------------------------------------------------------------------

