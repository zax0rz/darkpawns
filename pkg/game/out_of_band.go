package game

import "strings"

// OutOfBandObserver receives structured copies of game moments that already
// produce player-facing text, so a transport can mirror them out of band
// (GMCP for Mudlet and other rich telnet clients).
//
// Every call is made after — and never instead of — the C-faithful text for
// that moment, and implementations must not write player text or draw from
// the game RNG. The observer may only restate what the player was just
// shown; it is not a channel for information the text withheld (R4).
type OutOfBandObserver interface {
	// RoomShown fires after a room render reaches the player: the same moment
	// C's look_at_room() prints the room name. Dark and blind renders print no
	// room, so they do not fire.
	RoomShown(p *Player, roomVNum int)
	// ChannelLine fires once per delivered communication line, with the exact
	// line the player received and the speaker as that player saw them ("$n"
	// resolved through the same visibility rules, so an invisible speaker is
	// "Someone").
	ChannelLine(p *Player, channel, talker, line string)
	// PointUpdated fires once after each point_update() tick, when silent
	// regeneration has changed vitals without printing anything.
	PointUpdated()
}

// channelAct is Act for a line that belongs to a named communication
// channel. Delivery is Act's own (actDeliver), so the text bytes are exactly
// Act's; each player who received the line is then reported to the
// out-of-band observer with that same line.
func (w *World) channelAct(channel string, hideInvisible bool, ch, vict Actor, format string, actType int) {
	observer := w.outOfBand()
	if observer == nil {
		Act(w, hideInvisible, ch, vict, nil, nil, format, "", actType)
		return
	}
	actDeliver(w, hideInvisible, ch, vict, nil, nil, format, "", actType, func(to Actor, line string) {
		to.SendMessage(line)
		if p, ok := to.(*Player); ok && p != nil {
			observer.ChannelLine(p, channel, channelTalker(ch, to), line)
		}
	})
}

// channelSend is communicationSend for a sender's own echo of a channel line
// ("You gossip, '...'"). The speaker of an echo is always the sender.
func (w *World) channelSend(channel string, ch *Player, message string) {
	w.channelAct(channel, false, ch, nil, strings.TrimSuffix(message, "\r\n"), ToChar|ToSleep)
}

// MirrorChannelLine reports a channel line that was delivered by a direct
// SendMessage rather than through Act (group tell, which the session layer
// renders).
func (w *World) MirrorChannelLine(p *Player, channel, talker, line string) {
	w.mirrorChannelLine(p, channel, talker, line)
}

// mirrorChannelLine reports a channel line that was delivered by a direct
// SendMessage rather than through Act (NPC gossip, clan tell).
func (w *World) mirrorChannelLine(p *Player, channel, talker, line string) {
	if observer := w.outOfBand(); observer != nil && p != nil {
		observer.ChannelLine(p, channel, talker, line)
	}
}

// channelTalker renders the speaker exactly as act()'s "$n" does for this
// viewer, capitalized the way it leads the delivered line.
func channelTalker(ch, to Actor) string {
	if ch == nil {
		return ""
	}
	return cap(performAct("$n", ch, nil, nil, nil, "", "", to))
}

func (w *World) outOfBand() OutOfBandObserver {
	if w == nil {
		return nil
	}
	return w.OutOfBand
}
