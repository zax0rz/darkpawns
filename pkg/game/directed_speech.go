package game

import (
	"fmt"
	"sort"
	"strings"
)

type communicationEligibility struct {
	senderNoShout    bool
	senderSoundproof bool
	senderNoTell     bool
	targetNoTell     bool
	targetSoundproof bool
	targetWriting    bool
	targetVisible    bool
	self             bool
}

// communicationEligibility is the common gate snapshot for directed speech.
// Channel commands extend these same sender/room predicates in domain 6b.
func (w *World) communicationEligibility(ch *Player, target Actor) communicationEligibility {
	self := target == ch
	state := communicationEligibility{
		senderNoShout:    ch.GetFlags()&(1<<uint(PlrNoshout)) != 0,
		senderSoundproof: w.communicationRoomSoundproof(ch.GetRoom()),
		senderNoTell:     ch.GetFlags()&(1<<uint(PrfNotell)) != 0,
		targetVisible:    target != nil && (self || canSee(ch, target)),
		self:             self,
	}
	if player, ok := target.(*Player); ok {
		flags := player.GetFlags()
		state.targetNoTell = flags&(1<<uint(PrfNotell)) != 0
		state.targetWriting = flags&(1<<uint(PlrWriting)) != 0
		state.targetSoundproof = w.communicationRoomSoundproof(player.GetRoom())
	}
	return state
}

func (w *World) communicationRoomSoundproof(roomVNum int) bool {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}
	if roomHasFlagBit(room.Flags, 5) {
		return true
	}
	for _, flag := range room.Flags {
		if strings.EqualFold(flag, "soundproof") {
			return true
		}
	}
	return false
}

func communicationSend(ch Actor, message string) {
	Act(nil, false, ch, nil, nil, nil, strings.TrimSuffix(message, "\r\n"), "", ToChar|ToSleep)
}

func deleteANSIControls(message string) string {
	return strings.ReplaceAll(message, "&", "")
}

func splitDirectedSpeech(argument string) (string, string) {
	argument = strings.TrimSpace(argument)
	if argument == "" {
		return "", ""
	}
	if split := strings.IndexAny(argument, " \t\r\n"); split >= 0 {
		return argument[:split], strings.TrimSpace(argument[split:])
	}
	return argument, ""
}

func (w *World) visibleTellTarget(ch *Player, name string) *Player {
	players := w.GetAllPlayers()
	sort.Slice(players, func(i, j int) bool { return players[i].GetName() < players[j].GetName() })
	for _, player := range players {
		if isnameWithAbbrevs(name, player.GetName()) && (player == ch || canSee(ch, player)) {
			return player
		}
	}
	return nil
}

// DoSay implements C do_say, including mute, drunken speech, punctuation,
// ANSI-control deletion, and norepeat behavior.
func (w *World) DoSay(ch *Player, argument string) {
	argument = strings.TrimSpace(argument)
	state := w.communicationEligibility(ch, nil)
	if checkStupid(ch) {
		communicationSend(ch, "You are too stupid to communicate with language!")
		return
	}
	if state.senderNoShout {
		communicationSend(ch, "You cannot speak!")
		return
	}
	if argument == "" {
		communicationSend(ch, "Yes, but WHAT do you want to say?")
		return
	}

	roomMessage := argument
	if ch.GetCondition(CondDrunk) > 10 {
		roomMessage = speakDrunk(argument)
	}
	roomMessage = deleteANSIControls(roomMessage)
	actorMessage := deleteANSIControls(argument)

	roomVerb, actorVerb := "says", "say"
	switch argument[len(argument)-1] {
	case '!':
		roomVerb, actorVerb = "exclaims", "exclaim"
	case '?':
		roomVerb, actorVerb = "asks", "ask"
	case '.':
		roomVerb, actorVerb = "states", "state"
	}
	Act(w, false, ch, nil, nil, nil, fmt.Sprintf("$n %s, '%s'", roomVerb, roomMessage), "", ToRoom)
	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		communicationSend(ch, "Ok.")
		return
	}
	communicationSend(ch, fmt.Sprintf("You %s '%s'", actorVerb, actorMessage))
}

var drunkSyllables = []syllable{
	{" ", " "},
	{"are", "arsh"},
	{"and", "andsh"},
	{"how", "howsh"},
	{"what", "wha'"},
	{"is", "ish"},
	{"where", "whersh"},
	{"kill", "murderize"},
	{"ck", "shkin"},
	{"the ", "th' "},
	{"S", "SH"},
	{"T", "Th"},
	{"U", "u"},
	{"s", "sh"},
	{"t", "th"},
}

func speakDrunk(message string) string {
	return applySyllableSubstitution(message, drunkSyllables)
}

// DoTell implements C do_tell and records the reply target on the recipient.
func (w *World) DoTell(ch *Player, argument string) {
	targetName, message := splitDirectedSpeech(argument)
	if targetName == "" || message == "" {
		communicationSend(ch, "Who do you wish to tell what??")
		return
	}
	target := w.visibleTellTarget(ch, targetName)
	if target == nil {
		communicationSend(ch, "No-one by that name here.")
		return
	}
	state := w.communicationEligibility(ch, target)
	if state.self {
		communicationSend(ch, "You try to tell yourself something.")
		return
	}
	if state.senderNoTell && ch.GetLevel() < lvlImmort {
		communicationSend(ch, "You can't tell other people while you have notell on.")
		return
	}
	if state.senderNoShout {
		communicationSend(ch, "You cannot tell anyone anything!")
		return
	}
	if state.senderSoundproof {
		communicationSend(ch, "The walls seem to absorb your words.")
		return
	}
	if state.targetWriting {
		Act(nil, false, ch, target, nil, nil, "$E's writing a message right now; try again later.", "", ToChar|ToSleep)
		return
	}
	if (state.targetNoTell || state.targetSoundproof) && ch.GetLevel() < lvlImmort {
		Act(nil, false, ch, target, nil, nil, "$E can't hear you.", "", ToChar|ToSleep)
		return
	}
	if target.IsIgnoring(ch.Name) {
		communicationSend(ch, target.Name+" is ignoring you.")
		return
	}
	w.performTell(ch, target, message)
}

// DoReply implements C do_reply using the game-owned last-teller identity.
func (w *World) DoReply(ch *Player, argument string) {
	lastTeller := ch.GetLastTeller()
	if lastTeller == "" {
		communicationSend(ch, "You have no-one to reply to!")
		return
	}
	argument = strings.TrimSpace(argument)
	if argument == "" {
		communicationSend(ch, "What is your reply?")
		return
	}
	target, ok := w.GetPlayer(lastTeller)
	if !ok {
		for _, player := range w.GetAllPlayers() {
			if strings.EqualFold(player.Name, lastTeller) {
				target = player
				ok = true
				break
			}
		}
	}
	if !ok || target == nil {
		communicationSend(ch, "They are no longer playing.")
		return
	}
	state := w.communicationEligibility(ch, target)
	if state.targetWriting {
		communicationSend(ch, "They are writing now, try later.")
		return
	}
	if state.senderNoShout {
		communicationSend(ch, "You cannot tell anyone anything!")
		return
	}
	if state.targetNoTell {
		communicationSend(ch, "They can't hear you.")
		return
	}
	if state.senderSoundproof {
		communicationSend(ch, "The walls seem to absorb your words.")
		return
	}
	if state.targetSoundproof {
		Act(nil, false, ch, target, nil, nil, "$E can't hear you.", "", ToChar|ToSleep)
		return
	}
	if target.IsIgnoring(ch.Name) {
		communicationSend(ch, target.Name+" is ignoring you.")
		return
	}
	w.performTell(ch, target, argument)
}

func (w *World) performTell(ch, target *Player, message string) {
	message = deleteANSIControls(message)
	Act(nil, false, ch, target, nil, nil, fmt.Sprintf("$n tells you, '%s'", message), "", ToVict|ToSleep)
	if target.GetAFK() {
		Act(nil, false, ch, target, nil, nil, "$E is AFK right now, $E may not hear you.", "", ToChar|ToSleep)
	}
	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		communicationSend(ch, "Okay.")
	} else {
		Act(nil, false, ch, target, nil, nil, fmt.Sprintf("You tell $N, '%s'", message), "", ToChar|ToSleep)
	}
	target.SetLastTeller(ch.Name)
}

// DoSpecComm implements C do_spec_comm for room-local whisper and ask.
func (w *World) DoSpecComm(ch *Player, argument string, ask bool) {
	state := w.communicationEligibility(ch, nil)
	if state.senderNoShout {
		communicationSend(ch, "Sorry, you cannot do that.")
		return
	}
	actionSing, actionPlur := "whisper to", "whispers to"
	actionOthers := "$n whispers something to $N."
	if ask {
		actionSing, actionPlur = "ask", "asks"
		actionOthers = "$n asks $N a question."
	}
	targetName, message := splitDirectedSpeech(argument)
	if targetName == "" || message == "" {
		communicationSend(ch, fmt.Sprintf("Whom do you want to %s.. and what??", actionSing))
		return
	}

	var victim Actor
	if strings.EqualFold(targetName, ch.Name) || strings.EqualFold(targetName, "self") || strings.EqualFold(targetName, "me") {
		victim = ch
	} else if target, ok := w.ResolveCharInRoom(ch, targetName); ok {
		victim = asActor(target.Combatant)
	}
	if victim == nil {
		communicationSend(ch, "No-one by that name here.")
		return
	}
	state = w.communicationEligibility(ch, victim)
	if !state.targetVisible {
		communicationSend(ch, "No-one by that name here.")
		return
	}
	if state.self {
		communicationSend(ch, "You can't get your mouth close enough to your ear...")
		return
	}

	message = deleteANSIControls(message)
	Act(nil, false, ch, victim, nil, nil, fmt.Sprintf("$n %s you, '%s'", actionPlur, message), "", ToVict)
	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		communicationSend(ch, "Okay.")
	} else {
		Act(nil, false, ch, victim, nil, nil, fmt.Sprintf("You %s $N, '%s'", actionSing, message), "", ToChar)
	}
	Act(w, false, ch, victim, nil, nil, actionOthers, "", ToNotVict)
}
