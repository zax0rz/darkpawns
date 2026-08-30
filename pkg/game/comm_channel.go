package game

import (
	"fmt"
	"strconv"
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

// doShout keeps special-procedure callers on the canonical channel path.
func (w *World) doShout(ch *Player, me *MobInstance, arg string) bool {
	w.DoChannel(ch, arg, "shout")
	return true
}

type channelSpec struct {
	verb             string
	blocked          string
	offMessage       string
	senderOffFlag    int
	recipientOffFlag int
	minimumLevel     int
	zoneLimited      bool
	minimumHearer    int
	moveCost         int
}

var communicationChannels = map[string]channelSpec{
	"shout": {
		verb:             "shout",
		blocked:          "You cannot shout!!",
		offMessage:       "Turn off your noshout flag first!",
		senderOffFlag:    -1,
		recipientOffFlag: PrfDeaf,
		minimumLevel:     levelCanShout,
		zoneLimited:      true,
		minimumHearer:    combat.PosResting,
	},
	"gossip": {
		verb:             "gossip",
		blocked:          "You cannot gossip!!",
		offMessage:       "You aren't even on the channel!",
		senderOffFlag:    PrfNoGossip,
		recipientOffFlag: PrfNoGossip,
		minimumLevel:     levelCanShout,
	},
	"auction": {
		verb:             "auction",
		blocked:          "You cannot auction!!",
		offMessage:       "You aren't even on the channel!",
		senderOffFlag:    PrfNoAuctions,
		recipientOffFlag: PrfNoAuctions,
		minimumLevel:     levelCanShout,
	},
	"grats": {
		verb:             "congrat",
		blocked:          "You cannot congratulate!",
		offMessage:       "You aren't even on the channel!",
		senderOffFlag:    PrfNoGratz,
		recipientOffFlag: PrfNoGratz,
		minimumLevel:     levelCanShout,
	},
	"holler": {
		verb:          "holler",
		blocked:       "You cannot holler!!",
		senderOffFlag: -1,
		minimumLevel:  levelCanShout,
		moveCost:      hollerMoveCost,
	},
	"newbie": {
		verb:             "newbie",
		blocked:          "You cannot newbie!",
		offMessage:       "You aren't even on the channel!",
		senderOffFlag:    PrfNoNewbie,
		recipientOffFlag: PrfNoNewbie,
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
	if spec.senderOffFlag >= 0 && ch.GetFlags()&(1<<uint(spec.senderOffFlag)) != 0 {
		communicationSend(ch, spec.offMessage)
		return
	}

	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	if argument == "" {
		communicationSend(ch, fmt.Sprintf("Yes, %s, fine, %s we must, but WHAT???", spec.verb, spec.verb))
		return
	}
	argument = deleteANSIControls(argument)
	if spec.moveCost > 0 && !ch.SpendMove(spec.moveCost) {
		communicationSend(ch, "You're too exhausted to holler.")
		return
	}

	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		communicationSend(ch, "Okay.")
	} else {
		communicationSend(ch, fmt.Sprintf("You %s, '%s'", spec.verb, argument))
	}

	senderRoom := w.GetRoomInWorld(ch.GetRoom())
	for _, target := range w.GetAllPlayers() {
		if target == ch {
			continue
		}
		targetState := w.communicationEligibility(ch, target)
		if (spec.recipientOffFlag >= 0 && target.GetFlags()&(1<<uint(spec.recipientOffFlag)) != 0) || targetState.targetWriting || targetState.targetSoundproof {
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

// mobGlobalGossip implements the NPC-authored do_gen_comm(SCMD_GOSSIP) call used by
// quan_lo. C sends this through the global descriptor list, not the room act
// path, and the NPC has no descriptor to receive a self echo.
func (w *World) mobGlobalGossip(me *MobInstance, argument string) {
	if me == nil || w.communicationRoomSoundproof(me.GetRoomVNum()) {
		return
	}
	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	if argument == "" {
		return
	}
	argument = deleteANSIControls(argument)
	message := fmt.Sprintf("%s gossips, '%s'\r\n", mobName(me), argument)
	for _, player := range w.GetAllPlayers() {
		if player.GetFlags()&(1<<uint(PrfNoGossip)) != 0 ||
			player.GetFlags()&(1<<uint(PlrWriting)) != 0 ||
			w.communicationRoomSoundproof(player.GetRoom()) {
			continue
		}
		player.SendMessage(message)
	}
	w.updateGossipHistory(mobName(me), argument, 0)
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
	minLevel := 1
	clanNumber := 0
	levelString := ""

	// C uses a separate clan-number syntax for immortals. Its validation is
	// intentionally before the empty-message gate, and its rank lookup uses
	// clan[c] (the source's one-based command number against a zero-based array).
	if ch.GetLevel() >= LVLImmort {
		first, remainder := halfChop(arg)
		clanNumber, _ = strconv.Atoi(first)
		if clanNumber <= 0 || w.Clans == nil || clanNumber > w.Clans.ClanCount() {
			sendToChar(ch, "There is no clan with that number.\r\n")
			return true
		}
		arg = remainder
	} else {
		if ch.ClanID == 0 || ch.ClanRank == 0 {
			sendToChar(ch, "You're not part of a clan.\r\n")
			return true
		}
		clanNumber = ch.ClanID
	}

	if ch.GetFlags()&(1<<uint(PrfNoCTell)) != 0 {
		sendToChar(ch, "You aren't currently on your clan channel.\r\n")
		return true
	}
	if ch.GetFlags()&(1<<uint(PlrNoshout)) != 0 {
		sendToChar(ch, "You cannot clan-tell anything!\r\n")
		return true
	}

	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to tell your clan?\r\n")
		return true
	}

	if strings.HasPrefix(arg, "#") {
		rankText, remainder := halfChop(arg[1:])
		if !isClanNumber(rankText) {
			sendToChar(ch, "Try entering in a number.\r\n")
			return true
		}
		minLevel, _ = strconv.Atoi(rankText)
		clanForRank := (*Clan)(nil)
		if w.Clans != nil {
			// Match C's clan[c] access. A missing slot is treated as a zero-rank
			// record instead of permitting an out-of-bounds access.
			clanForRank = w.Clans.GetClanByIndex(clanNumber)
		}
		if clanForRank == nil || minLevel > clanForRank.Ranks {
			sendToChar(ch, "No one has a clan rank high enough to hear you!\r\n")
			return true
		}
		arg = skipSpaces(remainder)
		if arg == "" {
			sendToChar(ch, "What do you want to tell them?\r\n")
			return true
		}
		levelString = fmt.Sprintf(" (%d) ", minLevel)
	}

	arg = deleteANSIControls(arg)
	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		sendToChar(ch, "Okay.\r\n")
	} else {
		sendToChar(ch, fmt.Sprintf("You tell your clan%s, '%s'\r\n", levelString, arg))
	}

	for _, p := range w.AllPlayers() {
		if p == ch || p.ClanID != clanNumber || p.ClanRank < minLevel {
			continue
		}
		if p.GetFlags()&(1<<uint(PrfNoCTell)) != 0 {
			continue
		}
		senderName := ch.Name
		if !canSeeSocialTarget(p, ch) {
			senderName = "Someone"
		}
		p.SendMessage(fmt.Sprintf("%s tells your clan%s, '%s'\r\n", senderName, levelString, arg))
	}
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
