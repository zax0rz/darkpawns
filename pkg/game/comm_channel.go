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

// C's raw color bytes and the COLOR_LEV thresholds do_gen_comm gates on
// (src/screen.h:32-46). com_msgs[subcmd][3] carries one per channel.
const (
	ansiReset   = "\x1b[0m"  // KNRM
	ansiYellow  = "\x1b[33m" // KYEL
	ansiGreen   = "\x1b[32m" // KGRN
	ansiMagenta = "\x1b[35m" // KMAG

	colorNormal   = 2 // C_NRM
	colorComplete = 3 // C_CMP
)

// doShout keeps special-procedure callers on the canonical channel path.
func (w *World) doShout(ch *Player, me *MobInstance, arg string) bool {
	w.DoChannel(ch, arg, "shout")
	return true
}

type channelSpec struct {
	verb             string
	blocked          string
	offMessage       string
	color            string
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
		color:            ansiYellow, // KYEL
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
		color:            ansiYellow, // KYEL
		senderOffFlag:    PrfNoGossip,
		recipientOffFlag: PrfNoGossip,
		minimumLevel:     levelCanShout,
	},
	"auction": {
		verb:             "auction",
		blocked:          "You cannot auction!!",
		offMessage:       "You aren't even on the channel!",
		color:            ansiMagenta, // KMAG
		senderOffFlag:    PrfNoAuctions,
		recipientOffFlag: PrfNoAuctions,
		minimumLevel:     levelCanShout,
	},
	"grats": {
		verb:             "congrat",
		blocked:          "You cannot congratulate!",
		offMessage:       "You aren't even on the channel!",
		color:            ansiGreen, // KGRN
		senderOffFlag:    PrfNoGratz,
		recipientOffFlag: PrfNoGratz,
		minimumLevel:     levelCanShout,
	},
	"holler": {
		verb:          "holler",
		blocked:       "You cannot holler!!",
		color:         ansiYellow, // KYEL
		senderOffFlag: -1,
		minimumLevel:  levelCanShout,
		moveCost:      hollerMoveCost,
	},
	"newbie": {
		verb:             "newbie",
		blocked:          "You cannot newbie!",
		offMessage:       "You aren't even on the channel!",
		color:            ansiYellow, // KYEL
		senderOffFlag:    PrfNoNewbie,
		recipientOffFlag: PrfNoNewbie,
	},
}

// channelColorLine applies do_gen_comm's listener color sandwich to one
// already-rendered line: color_on + line + KNRM when the recipient's
// COLOR_LEV reaches openLevel, else the line untouched
// (src/act.comm.c:1290-1295).
func channelColorLine(to Actor, openLevel int, color, line string) string {
	if color == "" || channelRecipientLevel(to) < openLevel {
		return line
	}
	return color + line + ansiReset
}

// channelColorWrap is channelColorLine as a channelActWrapped hook. The
// speaker's own echo sets resetBeforeNewline: C's sprintf folds KNRM into the
// string, so the reset lands before act appends the line ending, while a
// listener's KNRM is a separate send_to_char after act's whole line
// (src/act.comm.c:1262-1295).
func channelColorWrap(openLevel int, color string, resetBeforeNewline bool) func(Actor, string) string {
	return func(to Actor, line string) string {
		if color == "" || channelRecipientLevel(to) < openLevel {
			return line
		}
		if resetBeforeNewline {
			return color + strings.TrimSuffix(line, "\r\n") + ansiReset + "\r\n"
		}
		return color + line + ansiReset
	}
}

// channelRecipientLevel is COLOR_LEV(to). A recipient without a descriptor
// (an NPC) is level 0 and is never colored.
func channelRecipientLevel(to Actor) int {
	if player, ok := to.(*Player); ok && player != nil {
		return colorLevel(player)
	}
	return 0
}

// DoChannel implements C do_gen_comm for player-facing channels. It extends
// directed speech's common eligibility snapshot with channel preference and
// shout-zone gates.
func (w *World) DoChannel(ch *Player, argument, subcmd string) {
	channel := strings.ToLower(subcmd)
	spec, ok := communicationChannels[channel]
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
		// The speaker's own echo gates on C_CMP and folds KNRM into the
		// sprintf, so the reset lands before act appends the line ending
		// (src/act.comm.c:1262-1268).
		w.channelActWrapped(channel, false, ch, nil,
			fmt.Sprintf("You %s, '%s'", spec.verb, argument), ToChar|ToSleep,
			channelColorWrap(colorComplete, spec.color, true))
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
		// A listener's color gates on C_NRM and its KNRM follows act's whole
		// line (src/act.comm.c:1290-1295).
		w.channelActWrapped(channel, false, ch, target,
			fmt.Sprintf("$n %ss, '%s'", spec.verb, argument), ToVict|ToSleep,
			channelColorWrap(colorNormal, spec.color, false))
	}

	if spec.verb == "gossip" {
		w.updateGossipHistory(ch.Name, argument, 0)
		// The grapevine client swaps/clears this callback on its reconnect
		// goroutine; read it under the gossip lock, call it outside (VULN-040).
		w.gossipMu.RLock()
		onGossip := w.OnGossip
		w.gossipMu.RUnlock()
		if onGossip != nil {
			onGossip(ch.Name, argument)
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
	// act("$n gossips, '%s'", ..., TO_VICT) capitalizes the line (act.comm.c:1273, 1293).
	spec := communicationChannels["gossip"]
	message := capitalize(fmt.Sprintf("%s gossips, '%s'\r\n", mobName(me), argument))
	for _, player := range w.GetAllPlayers() {
		if player.GetFlags()&(1<<uint(PrfNoGossip)) != 0 ||
			player.GetFlags()&(1<<uint(PlrWriting)) != 0 ||
			w.communicationRoomSoundproof(player.GetRoom()) {
			continue
		}
		// An NPC author's descriptor loop colors each listener exactly as a
		// player author's does: send_to_char(color_on) / act() / KNRM
		// (src/act.comm.c:1288-1295).
		line := channelColorLine(player, colorNormal, spec.color, message)
		player.SendMessage(line)
		w.mirrorChannelLine(player, "gossip", mobName(me), line)
	}
	w.updateGossipHistory(mobName(me), argument, 0)
}

// doQcomm -- port of do_qcomm() (team/quiz communication).
func (w *World) doQcomm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		ch.SendMessage("What do you want to say?\r\n")
		return true
	}

	msg := fmt.Sprintf("%s says, '%s'\r\n", ch.Name, arg)
	for _, p := range w.GetPlayersInRoom(ch.GetRoom()) {
		if p.Name != ch.Name {
			p.SendMessage(msg)
		}
	}
	ch.SendMessage(fmt.Sprintf("You say, '%s'\r\n", arg))
	return true
}

// doThink -- port of do_think().
func (w *World) doThink(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		ch.SendMessage("What do you want to think?\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You think: '%s'\r\n", arg))
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
			ch.SendMessage("There is no clan with that number.\r\n")
			return true
		}
		arg = remainder
	} else {
		if ch.ClanID == 0 || ch.ClanRank == 0 {
			ch.SendMessage("You're not part of a clan.\r\n")
			return true
		}
		clanNumber = ch.ClanID
	}

	if ch.GetFlags()&(1<<uint(PrfNoCTell)) != 0 {
		ch.SendMessage("You aren't currently on your clan channel.\r\n")
		return true
	}
	if ch.GetFlags()&(1<<uint(PlrNoshout)) != 0 {
		ch.SendMessage("You cannot clan-tell anything!\r\n")
		return true
	}

	arg = skipSpaces(arg)
	if arg == "" {
		ch.SendMessage("What do you want to tell your clan?\r\n")
		return true
	}

	if strings.HasPrefix(arg, "#") {
		rankText, remainder := halfChop(arg[1:])
		if !isClanNumber(rankText) {
			ch.SendMessage("Try entering in a number.\r\n")
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
			ch.SendMessage("No one has a clan rank high enough to hear you!\r\n")
			return true
		}
		arg = skipSpaces(remainder)
		if arg == "" {
			ch.SendMessage("What do you want to tell them?\r\n")
			return true
		}
		levelString = fmt.Sprintf(" (%d) ", minLevel)
	}

	arg = deleteANSIControls(arg)
	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		ch.SendMessage("Okay.\r\n")
	} else {
		echo := fmt.Sprintf("You tell your clan%s, '%s'\r\n", levelString, arg)
		ch.SendMessage(echo)
		w.mirrorChannelLine(ch, "clan", ch.Name, echo)
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
		line := fmt.Sprintf("%s tells your clan%s, '%s'\r\n", senderName, levelString, arg)
		p.SendMessage(line)
		w.mirrorChannelLine(p, "clan", senderName, line)
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

	// gossipHistory is stored newest-first, while C's do_review() walks its
	// fixed array from slot 24 down to slot 0 (newest entries are inserted at
	// slot 0). Preserve the player-visible oldest-first order.
	for index := len(w.gossipHistory) - 1; index >= 0; index-- {
		entry := w.gossipHistory[index]
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
