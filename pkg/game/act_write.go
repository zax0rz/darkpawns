package game

import "fmt"

// DoWrite ports do_write (src/act.comm.c:1024-1094). It preserves C's
// two_arguments parsing (papername is the FIRST token, penname the SECOND) and
// every refusal byte, then enters C's note editor.
//
// The session command layer delegates here; the previous hand-written cmdWrite
// parsed the LAST token as the item and printed invented strings (R4), so a
// two-argument "write" reported the pen name and a one-argument "write" fell
// through to the empty-argument message (DP-1516 item 7).
func (w *World) DoWrite(ch *Player, argument string) {
	if ch == nil {
		return
	}

	papername, rest := oneArgument(argument)
	penname, _ := oneArgument(rest)

	if ch.GetFlags()&(1<<uint(PlrNoshout)) != 0 {
		ch.SendMessage("You cannot write anything!\r\n")
		return
	}
	if papername == "" {
		ch.SendMessage("Write?  With what?  ON what?  What are you trying to do?!?\r\n")
		return
	}

	var paper, pen *ObjectInstance
	if penname != "" { // there were two arguments
		if paper = getObjInInvVis(ch, papername); paper == nil {
			ch.SendMessage(fmt.Sprintf("You have no %s.\r\n", papername))
			return
		}
		if pen = getObjInInvVis(ch, penname); pen == nil {
			ch.SendMessage(fmt.Sprintf("You have no %s.\r\n", penname))
			return
		}
	} else { // there was one arg.. let's see what we can find
		if paper = getObjInInvVis(ch, papername); paper == nil {
			ch.SendMessage(fmt.Sprintf("There is no %s in your inventory.\r\n", papername))
			return
		}
		if paper.GetTypeFlag() == ITEM_PEN { // oops, a pen..
			pen = paper
			paper = nil
		} else if paper.GetTypeFlag() != ITEM_NOTE {
			ch.SendMessage("That thing has nothing to do with writing.\r\n")
			return
		}
		// One object was found.. now for the other one.
		held, ok := ch.Equipment.GetItemInSlot(SlotHold)
		if !ok || held == nil {
			ch.SendMessage(fmt.Sprintf("You can't write with %s %s alone.\r\n", an(papername), papername))
			return
		}
		if !CanSeeObject(ch, held) {
			ch.SendMessage("The stuff in your hand is invisible!  Yeech!!\r\n")
			return
		}
		if pen != nil {
			paper = held
		} else {
			pen = held
		}
	}

	// ok.. now let's see what kind of stuff we've found
	switch {
	case pen.GetTypeFlag() != ITEM_PEN:
		Act(nil, false, ch, nil, pen, nil, "$p is no good for writing with.", "", ToChar)
	case paper.GetTypeFlag() != ITEM_NOTE:
		Act(nil, false, ch, nil, paper, nil, "You can't write on $p.", "", ToChar)
	case paper.ActionDescription() != "":
		ch.SendMessage("There's something written on it already.\r\n")
	default:
		// we can write - hooray!
		ch.SendMessage("Write your note.  End with '@' on a new line.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n begins to jot down a note.", "", ToRoom)
		ch.SetPlrFlag(PlrWriting, true)
		noteWriteMu.Lock()
		noteWriteEntries[ch.ID] = &noteWriteEntry{obj: paper}
		noteWriteMu.Unlock()
	}
}
