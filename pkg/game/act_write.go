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
		beginNoteWrite(ch, paper)
	}
}

// CharacterWrite routes do_write through the active body while retaining the
// descriptor owner's note-editor state. A switched mobile has no player flags
// of its own; C reaches the immortal's editor through ch->desc, and stores the
// text on the selected object (src/act.comm.c:1023-1100).
func CharacterWrite(ctx CharacterCommandContext) CharacterCommandResult {
	if ctx.Actor == nil || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		ctx.World.DoWrite(player, ctx.RawArgs)
		return CharacterCommandHandled
	}
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	// C returns silently when ch->desc is nil. A live switched body has a
	// descriptor owner; descriptor-less NPC command calls take that same arm.
	if ctx.DescriptorOwner == nil {
		return CharacterCommandHandled
	}
	ctx.World.characterMobWrite(mob, ctx.DescriptorOwner, ctx.RawArgs)
	return CharacterCommandHandled
}

func (w *World) characterMobWrite(mob *MobInstance, descriptorOwner *Player, argument string) {
	paperName, penName := twoArguments(argument)
	if paperName == "" {
		mob.SendMessage("Write?  With what?  ON what?  What are you trying to do?!?\r\n")
		return
	}
	findCarried := func(name string) *ObjectInstance {
		obj, _ := w.ResolveObjectInInventory(mob, name)
		return obj
	}
	var paper, pen *ObjectInstance
	if penName != "" {
		paper = findCarried(paperName)
		if paper == nil {
			mob.SendMessage(fmt.Sprintf("You have no %s.\r\n", paperName))
			return
		}
		pen = findCarried(penName)
		if pen == nil {
			mob.SendMessage(fmt.Sprintf("You have no %s.\r\n", penName))
			return
		}
	} else {
		paper = findCarried(paperName)
		if paper == nil {
			mob.SendMessage(fmt.Sprintf("There is no %s in your inventory.\r\n", paperName))
			return
		}
		if paper.GetTypeFlag() == ITEM_PEN {
			pen = paper
			paper = nil
		} else if paper.GetTypeFlag() != ITEM_NOTE {
			mob.SendMessage("That thing has nothing to do with writing.\r\n")
			return
		}
		held := mob.Equipped(int(SlotHold))
		if held == nil {
			mob.SendMessage(fmt.Sprintf("You can't write with %s %s alone.\r\n", an(paperName), paperName))
			return
		}
		if !CanSeeObject(mob, held) {
			mob.SendMessage("The stuff in your hand is invisible!  Yeech!!\r\n")
			return
		}
		if pen != nil {
			paper = held
		} else {
			pen = held
		}
	}

	switch {
	case pen.GetTypeFlag() != ITEM_PEN:
		Act(nil, false, mob, nil, pen, nil, "$p is no good for writing with.", "", ToChar)
	case paper.GetTypeFlag() != ITEM_NOTE:
		Act(nil, false, mob, nil, paper, nil, "You can't write on $p.", "", ToChar)
	case paper.ActionDescription() != "":
		mob.SendMessage("There's something written on it already.\r\n")
	default:
		mob.SendMessage("Write your note.  End with '@' on a new line.\r\n")
		Act(w, true, mob, nil, nil, nil, "$n begins to jot down a note.", "", ToRoom)
		beginNoteWrite(descriptorOwner, paper)
	}
}

func beginNoteWrite(descriptorOwner *Player, paper *ObjectInstance) {
	if descriptorOwner == nil || paper == nil {
		return
	}
	descriptorOwner.SetPlrFlag(PlrWriting, true)
	noteWriteMu.Lock()
	noteWriteEntries[descriptorOwner.ID] = &noteWriteEntry{obj: paper}
	noteWriteMu.Unlock()
}
