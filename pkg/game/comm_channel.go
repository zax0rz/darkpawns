//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"strings"
)

// gossipHistory is a circular buffer of the last 25 gossip messages.
// Matches C: struct review_t review[25] in db.c.
type gossipEntry struct {
	Name    string
	Message string
	Invis   int // invis level of the speaker
}

const maxGossipHistory = 25

func (w *World) doSpecComm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	switch strings.ToLower(cmd) {
	case "shout":
		return w.doShout(ch, me, arg)
	case "whisper":
		return w.doWhisper(ch, me, arg)
	case "ask":
		return w.doAsk(ch, me, arg)
	}
	return true
}

// doShout — shout implementation.
func (w *World) doShout(ch *Player, me *MobInstance, arg string) bool {
	arg = skipSpaces(arg)

	if arg == "" {
		sendToChar(ch, "Shout what?\r\n")
		return true
	}
	if ch.GetLevel() < levelCanShout {
		sendToChar(ch, "You must be at least level 5 to shout.\r\n")
		return true
	}
	if ch.GetFlags()&prfNoShout != 0 {
		sendToChar(ch, "You can't shout.\r\n")
		return true
	}
	if w.roomHasFlag(ch.GetRoom(), "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}

	msg := fmt.Sprintf("%s shouts, '%s'\r\n", ch.Name, arg)
	for _, p := range w.AllPlayers() {
		if p.IsNPC() || p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 {
			continue
		}
		if p.GetFlags()&prfNoShout != 0 {
			continue
		}
		if w.roomHasFlag(p.GetRoom(), "soundproof") {
			continue
		}
		p.SendMessage(msg)
	}

	sendToChar(ch, fmt.Sprintf("You shout, '%s'\r\n", arg))
	return true
}

// doWhisper — whisper implementation.
func (w *World) doWhisper(ch *Player, me *MobInstance, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Whisper whom what?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is here.\r\n")
		return true
	}

	vict.SendMessage(fmt.Sprintf("\x1B[1;33m%s whispers, '%s'\033[0m\r\n", ch.Name, msg))
	ch.SendMessage(fmt.Sprintf("You whisper to %s, '%s'\r\n", vict.Name, msg))

	// Broadcast to rest of room that whisper occurred.
	for _, p := range w.GetPlayersInRoom(ch.GetRoom()) {
		if p.Name != ch.Name && p.Name != vict.Name {
			p.SendMessage(fmt.Sprintf("%s whispers something to %s.\r\n", ch.Name, vict.Name))
		}
	}

	return true
}

// doAsk — ask implementation (identical to whisper but broadcasts as ask).
func (w *World) doAsk(ch *Player, me *MobInstance, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Ask whom what?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is here.\r\n")
		return true
	}

	vict.SendMessage(fmt.Sprintf("\x1B[1;36m%s asks, '%s'\033[0m\r\n", ch.Name, msg))
	ch.SendMessage(fmt.Sprintf("You ask %s, '%s'\r\n", vict.Name, msg))

	for _, p := range w.GetPlayersInRoom(ch.GetRoom()) {
		if p.Name != ch.Name && p.Name != vict.Name {
			p.SendMessage(fmt.Sprintf("%s asks %s something.\r\n", ch.Name, vict.Name))
		}
	}

	return true
}

// doWrite — port of do_write().
// Source: act.comm.c:1024 do_write() — full logic ported from C.
//
// C logic summary:
//
//	two_arguments(argument, papername, penname)
//	- No args: print usage error
//	- PLR_NOSHOUT: block writing
//	- Two args: look up both paper and pen in inventory
//	- One arg: find it in inventory; if it's a pen swap pen/paper, else it must
//	  be ITEM_NOTE. Check held slot for the other object.
//	- Validate pen is ITEM_PEN and paper is ITEM_NOTE
//	- If paper already has text: reject ("already written on")
//	- Otherwise: set PLR_WRITING, put player into string-editor mode
func (w *World) doWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Source: act.comm.c:1024
	arg = skipSpaces(arg)

	// NPCs can't write — no descriptor
	if ch.IsNPC() {
		return true
	}

	if arg == "" {
		sendToChar(ch, "Write?  With what?  ON what?  What are you trying to do?!?\r\n")
		return true
	}

	// C: PLR_FLAGGED(ch, PLR_NOSHOUT)
	if ch.GetFlags()&(1<<PlrNoshout) != 0 {
		sendToChar(ch, "You cannot write anything!\r\n")
		return true
	}

	// Already composing something
	if ch.GetFlags()&(1<<PlrWriting) != 0 {
		sendToChar(ch, "You are already writing something.\r\n")
		return true
	}

	// Parse up to two arguments: papername and (optional) penname
	// C: two_arguments(argument, papername, penname)
	papername, rest := halfChop(arg)
	penname, _ := halfChop(rest)

	var paper, pen *ObjectInstance

	if penname != "" {
		// Two arguments: look up paper then pen explicitly
		var found bool
		paper, found = ch.Inventory.FindItem(papername)
		if !found {
			sendToChar(ch, fmt.Sprintf("You have no %s.\r\n", papername))
			return true
		}
		pen, found = ch.Inventory.FindItem(penname)
		if !found {
			sendToChar(ch, fmt.Sprintf("You have no %s.\r\n", penname))
			return true
		}
	} else {
		// One argument — figure out what we found and check held slot for the other
		var found bool
		paper, found = ch.Inventory.FindItem(papername)
		if !found {
			sendToChar(ch, fmt.Sprintf("There is no %s in your inventory.\r\n", papername))
			return true
		}

		if paper.GetObjType() == ITEM_PEN {
			// Oops — they named the pen first; swap
			pen = paper
			paper = nil
		} else if paper.GetObjType() != ITEM_NOTE {
			sendToChar(ch, "That thing has nothing to do with writing.\r\n")
			return true
		}

		// One object found; look for the other in the hold slot
		held := w.GetEquipped(ch, eqWearHold)
		if held == nil {
			sendToChar(ch, fmt.Sprintf("You can't write with %s %s alone.\r\n", an(papername), papername))
			return true
		}

		if pen != nil {
			// We have pen, held must be the paper
			paper = held
		} else {
			// We have paper, held must be the pen
			pen = held
		}
	}

	// Validate types — C: GET_OBJ_TYPE checks
	if pen.GetObjType() != ITEM_PEN {
		sendToChar(ch, fmt.Sprintf("%s is no good for writing with.\r\n", pen.GetShortDesc()))
		return true
	}
	if paper.GetObjType() != ITEM_NOTE {
		sendToChar(ch, fmt.Sprintf("You can't write on %s.\r\n", paper.GetShortDesc()))
		return true
	}

	// C: if (paper->action_description) — already has text
	if paper.Runtime.NoteText != "" {
		sendToChar(ch, "There's something written on it already.\r\n")
		return true
	}

	// All checks pass — enter string-editor mode
	// C: send_to_char("Write your note.  End with '@' on a new line.\r\n", ch)
	// C: act("$n begins to jot down a note.", TRUE, ch, 0, 0, TO_ROOM)
	sendToChar(ch, "Write your note.  End with '@' on a new line.\r\n")
	w.roomMessageExcludeTwo(ch.GetRoom(),
		fmt.Sprintf("%s begins to jot down a note.", ch.Name), ch.Name, "")

	// Engage note-write mode — PLR_WRITING set, input routed to HandleNoteInput
	StartNoteWriting(ch, paper)
	return true
}

// doPage -- port of do_page().
// Extended: supports multiple targets as "page target1 target2 ... msg".
// Source: act.comm.c lines 1107-1136 do_page() — extended for multi-target.
func (w *World) doPage(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	// Parse targets: repeatedly halfChop to extract target names,
	// last remaining word(s) is the message.
	// Extended: C do_page() used half_chop for single target only;
	// this Go version iterates for multi-target support.
	targets := make([]string, 0)
	remaining := arg
	for {
		tname, rest := halfChop(remaining)
		if tname == "" {
			break
		}
		// Check if there's more after this word
		nextWord, _ := halfChop(rest)
		if nextWord == "" {
			// This is the last word — it's the message, not a target
			break
		}
		targets = append(targets, tname)
		remaining = rest
	}

	if len(targets) == 0 {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	msg := remaining
	if msg == "" {
		msg = fmt.Sprintf("%s pages you!", ch.Name)
	}

	matched := make([]string, 0)
	for _, tname := range targets {
		tch := w.getCharVis(ch, tname)
		if tch == nil {
			sendToChar(ch, "No one by that name is playing.\r\n")
			continue
		}

		tch.SendMessage(fmt.Sprintf("\r\n%s pages: '%s'\r\n", ch.Name, msg))
		matched = append(matched, tch.Name)
	}

	if len(matched) > 0 {
		sendToChar(ch, fmt.Sprintf("You page %s with '%s'\r\n",
			strings.Join(matched, ", "), msg))
	}

	return true
}

// doGenComm -- port of do_gen_comm() (gossip, chat, auction, gratz, newbie).
func (w *World) doGenComm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		// Determine channel name from cmd / subcmd
		switch strings.ToLower(cmd) {
		case "gossip":
			sendToChar(ch, "Gossip what?\r\n")
		case "auction":
			sendToChar(ch, "Auction what?\r\n")
		case "gratz":
			sendToChar(ch, "Gratz whom?\r\n")
		case "newbie":
			sendToChar(ch, "Newbie what?\r\n")
		default:
			sendToChar(ch, "Say what?\r\n")
		}
		return true
	}

	// Build channel header
	var header string
	var flag uint64
	var minLevel int
	var channelName string

	switch strings.ToLower(cmd) {
	case "gossip":
		header = fmt.Sprintf("%s gossips, '%s'\r\n", ch.Name, arg)
		flag = prfNoGossip
		minLevel = levelCanGossip
		channelName = "gossip"
	case "auction":
		header = fmt.Sprintf("%s auctions, '%s'\r\n", ch.Name, arg)
		flag = prfNoAuct
		channelName = "auction"
	case "gratz":
		header = fmt.Sprintf("%s congratulates, '%s'\r\n", ch.Name, arg)
		flag = prfNoGratz
		channelName = "gratz"
	case "newbie":
		header = fmt.Sprintf("%s says, '%s'\r\n", ch.Name, arg)
		flag = prfNoNewbie
		channelName = "newbie"
	default:
		sendToChar(ch, "Unknown channel.\r\n")
		return true
	}

	if ch.GetLevel() < minLevel {
		sendToChar(ch, fmt.Sprintf("You need to be level %d to use that channel.\r\n", minLevel))
		return true
	}

	for _, p := range w.AllPlayers() {
		if p.IsNPC() || p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 {
			continue
		}
		if p.Flags&flag != 0 {
			continue
		}
		p.SendMessage(header)
	}

	sendToChar(ch, fmt.Sprintf("You %s, '%s'\r\n", channelName, arg))

	// Record gossip for review command (matches C: update_review in act.comm.c)
	if channelName == "gossip" {
		w.updateGossipHistory(ch.Name, arg, 0)
		if w.OnGossip != nil {
			w.OnGossip(ch.Name, arg)
		}
	}

	return true
}

// doQcomm -- port of do_qcomm() (team/quiz communication).
func (w *World) doQcomm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to say?\r\n")
		return true
	}

	msg := fmt.Sprintf("%s says, '%s'\r\n", ch.Name, arg)
	for _, p := range w.GetPlayersInRoom(ch.GetRoom()) {
		if p.Name != ch.Name {
			p.SendMessage(msg)
		}
	}
	sendToChar(ch, fmt.Sprintf("You say, '%s'\r\n", arg))
	return true
}

// doThink -- port of do_think().
func (w *World) doThink(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to think?\r\n")
		return true
	}

	sendToChar(ch, fmt.Sprintf("You think: '%s'\r\n", arg))
	return true
}

// doCTell -- port of do_ctell() (clan tell).
func (w *World) doCTell(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to tell your clan?\r\n")
		return true
	}

	// Broadcast to clan members only.
	msg := fmt.Sprintf("[Clan] %s tells the clan, '%s'\r\n", ch.Name, arg)
	for _, p := range w.AllPlayers() {
		if p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 || p.Flags&prfNoCtell != 0 {
			continue
		}
		// Filter: only clan members with the same ClanID
		if ch.ClanID == 0 || p.ClanID != ch.ClanID {
			continue
		}
		p.SendMessage(msg)
	}

	sendToChar(ch, fmt.Sprintf("You tell your clan, '%s'\r\n", arg))
	return true
}

// updateGossipHistory adds a gossip entry to the ring buffer.
// Matches C: update_review() in new_cmds.c — shifts all entries up, inserts at [0].
func (w *World) updateGossipHistory(name, message string, invisLevel int) {
	w.gossipMu.Lock()
	defer w.gossipMu.Unlock()

	// Shift all entries up by one (drop the oldest if at capacity)
	if len(w.gossipHistory) >= maxGossipHistory {
		w.gossipHistory = w.gossipHistory[:maxGossipHistory-1]
	}
	// Prepend new entry at front
	w.gossipHistory = append([]gossipEntry{{Name: name, Message: message, Invis: invisLevel}}, w.gossipHistory...)
}

// ReviewGossip returns the formatted gossip history for the review command.
// Matches C: do_review() in new_cmds.c.
func (w *World) ReviewGossip(ch *Player) string {
	w.gossipMu.RLock()
	defer w.gossipMu.RUnlock()

	var buf strings.Builder
	buf.WriteString("Last Gossips:\r\n-------------\r\n")

	for _, entry := range w.gossipHistory {
		// Hide invisible players below viewer's level
		if entry.Invis > ch.GetLevel() {
			buf.WriteString("Someone invisible: ")
		} else {
			buf.WriteString(entry.Name)
			buf.WriteString(": ")
		}
		buf.WriteString(entry.Message)
		buf.WriteString("\r\n")
	}

	return buf.String()
}
