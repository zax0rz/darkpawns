package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
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

// doShout keeps special-procedure callers on the canonical channel path.
func (w *World) doShout(ch *Player, me *MobInstance, arg string) bool {
	w.DoChannel(ch, arg, "shout")
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

type channelSpec struct {
	verb          string
	blocked       string
	offMessage    string
	offFlag       int
	minimumLevel  int
	zoneLimited   bool
	minimumHearer int
}

var communicationChannels = map[string]channelSpec{
	"shout": {
		verb:          "shout",
		blocked:       "You cannot shout!!",
		offMessage:    "Turn off your noshout flag first!",
		offFlag:       PrfDeaf,
		minimumLevel:  levelCanShout,
		zoneLimited:   true,
		minimumHearer: combat.PosResting,
	},
	"gossip": {
		verb:         "gossip",
		blocked:      "You cannot gossip!!",
		offMessage:   "You aren't even on the channel!",
		offFlag:      PrfNoGossip,
		minimumLevel: levelCanShout,
	},
	"auction": {
		verb:         "auction",
		blocked:      "You cannot auction!!",
		offMessage:   "You aren't even on the channel!",
		offFlag:      PrfNoAuctions,
		minimumLevel: levelCanShout,
	},
	"gratz": {
		verb:         "congrat",
		blocked:      "You cannot congratulate!",
		offMessage:   "You aren't even on the channel!",
		offFlag:      PrfNoGratz,
		minimumLevel: levelCanShout,
	},
	"newbie": {
		verb:       "newbie",
		blocked:    "You cannot newbie!",
		offMessage: "You aren't even on the channel!",
		offFlag:    PrfNoNewbie,
	},
}

// DoChannel implements C do_gen_comm for player-facing channels. It extends
// directed speech's common eligibility snapshot with channel preference and
// shout-zone gates.
func (w *World) DoChannel(ch *Player, argument, subcmd string) {
	spec, ok := communicationChannels[strings.ToLower(subcmd)]
	if !ok {
		communicationSend(ch, "Unknown channel.")
		return
	}

	state := w.communicationEligibility(ch, nil)
	if state.senderNoShout {
		communicationSend(ch, spec.blocked)
		return
	}
	if state.senderSoundproof {
		communicationSend(ch, "The walls seem to absorb your words.")
		return
	}
	if ch.GetLevel() < spec.minimumLevel {
		communicationSend(ch, fmt.Sprintf("You must be at least level %d before you can %s.", spec.minimumLevel, spec.verb))
		return
	}
	if checkStupid(ch) {
		communicationSend(ch, "You are too stupid to communicate with language!")
		return
	}
	if ch.GetFlags()&(1<<uint(spec.offFlag)) != 0 {
		communicationSend(ch, spec.offMessage)
		return
	}

	argument = strings.TrimSpace(argument)
	if argument == "" {
		communicationSend(ch, fmt.Sprintf("Yes, %s, fine, %s we must, but WHAT???", spec.verb, spec.verb))
		return
	}
	argument = deleteANSIControls(argument)

	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		communicationSend(ch, "Ok.")
	} else {
		communicationSend(ch, fmt.Sprintf("You %s, '%s'", spec.verb, argument))
	}

	senderRoom := w.GetRoomInWorld(ch.GetRoom())
	for _, target := range w.GetAllPlayers() {
		if target == ch {
			continue
		}
		targetState := w.communicationEligibility(ch, target)
		if target.GetFlags()&(1<<uint(spec.offFlag)) != 0 || targetState.targetWriting || targetState.targetSoundproof {
			continue
		}
		if spec.zoneLimited {
			targetRoom := w.GetRoomInWorld(target.GetRoom())
			if senderRoom == nil || targetRoom == nil || senderRoom.Zone != targetRoom.Zone || target.GetPosition() < spec.minimumHearer {
				continue
			}
		}
		Act(nil, false, ch, target, nil, nil, fmt.Sprintf("$n %ss, '%s'", spec.verb, argument), "", ToVict|ToSleep)
	}

	if spec.verb == "gossip" {
		w.updateGossipHistory(ch.Name, argument, 0)
		if w.OnGossip != nil {
			w.OnGossip(ch.Name, argument)
		}
	}
}

// doGenComm keeps scripts and special procedures on the canonical channel path.
func (w *World) doGenComm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	w.DoChannel(ch, arg, cmd)
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
