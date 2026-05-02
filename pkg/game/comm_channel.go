//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"strings"
)

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
	if ch.Flags&prfNoShout != 0 {
		sendToChar(ch, "You can't shout.\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}

	msg := fmt.Sprintf("%s shouts, '%s'\r\n", ch.Name, arg)
	for _, p := range w.allPlayers() {
		if p.IsNPC() || p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 {
			continue
		}
		if p.Flags&prfNoShout != 0 {
			continue
		}
		if w.roomHasFlag(p.RoomVNum, "soundproof") {
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
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
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

	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name != ch.Name && p.Name != vict.Name {
			p.SendMessage(fmt.Sprintf("%s asks %s something.\r\n", ch.Name, vict.Name))
		}
	}

	return true
}

// doWrite — port of do_write().
func (w *World) doWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if arg == "" {
		sendToChar(ch, "Write on what?\r\n")
		return true
	}

	// Find a writing surface (tablet, paper, etc.) in inventory or room.
	// Simplified: NPCs check obj list, players check inventory.
	// For now, just say they start writing.
	args := strings.Fields(arg)
	if len(args) == 0 {
		sendToChar(ch, "Write what?\r\n")
		return true
	}
	objName := args[0]
	_ = objName
	sendToChar(ch, "You start writing.\r\n")
	return true
}

// doPage -- port of do_page().
func (w *World) doPage(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	// Format: target msg or multiple targets "target1 target2 msg"
	// Simplified: single target
	target, msg := halfChop(arg)
	if target == "" {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	tch := w.getCharVis(ch, target)
	if tch == nil {
		sendToChar(ch, "No one by that name is playing.\r\n")
		return true
	}

	if msg == "" {
		msg = fmt.Sprintf("%s pages you!\r\n", ch.Name)
	} else {
		tch.SendMessage(fmt.Sprintf("\r\n%s pages: '%s'\r\n", ch.Name, msg))
	}

	sendToChar(ch, fmt.Sprintf("You page %s with '%s'\r\n", tch.Name, msg))
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

	for _, p := range w.allPlayers() {
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
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
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

	// Clan system not yet implemented -- broadcast to all players as a fallback.
	msg := fmt.Sprintf("[Clan] %s tells the clan, '%s'\r\n", ch.Name, arg)
	for _, p := range w.allPlayers() {
		if p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 || p.Flags&prfNoCtell != 0 {
			continue
		}
		p.SendMessage(msg)
	}

	sendToChar(ch, fmt.Sprintf("You tell your clan, '%s'\r\n", arg))
	return true
}
