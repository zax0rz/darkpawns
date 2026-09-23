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
	w.channelActWrapped(channel, hideInvisible, ch, vict, format, actType, nil)
}

func (w *World) channelActWrapped(channel string, hideInvisible bool, ch, vict Actor, format string, actType int, wrap func(Actor, string) string) {
	observer := w.outOfBand()
	if observer == nil && wrap == nil {
		Act(w, hideInvisible, ch, vict, nil, nil, format, "", actType)
		return
	}
	actDeliver(w, hideInvisible, ch, vict, nil, nil, format, "", actType, func(to Actor, line string) {
		if wrap != nil {
			line = wrap(to, line)
		}
		to.SendMessage(line)
		if observer != nil {
			if p, ok := to.(*Player); ok && p != nil {
				observer.ChannelLine(p, channel, channelTalker(ch, to), line)
			}
		}
	})
}

// GroupSay delivers one do_gsay line through act() to the grouped leader and
// followers in the same order as C's prepended follower list.
func (w *World) GroupSay(ch *Player, message string) {
	if ch == nil {
		return
	}
	if !ch.IsAffected(affGroup) {
		ch.SendMessage("But you are not the member of a group!\r\n")
		return
	}
	if message == "" {
		ch.SendMessage("Yes, but WHAT do you want to group-say?\r\n")
		return
	}

	leader := Actor(ch)
	if following := ch.GetFollowing(); following != "" {
		leader = w.followingActor(following)
	}
	if leader != nil {
		leaderName := leader.GetName()
		format := DeleteANSIControls("$n tells the group, '" + message + "'")
		if leader != ch && actorInGroup(leader) {
			w.channelActWrapped("group", false, ch, leader, format, ToVict|ToSleep,
				wrapGroupColor(3, 3, false))
		}
		for _, follower := range w.GetFollowerActors(leaderName) {
			if follower == ch || !actorInGroup(follower) {
				continue
			}
			w.channelActWrapped("group", false, ch, follower, format, ToVict|ToSleep,
				wrapGroupColor(3, 3, false))
		}
	}

	if ch.GetFlags()&(1<<uint(PrfNoRepeat)) != 0 {
		ch.SendMessage("Okay.\r\n")
		return
	}
	echo := DeleteANSIControls("You tell the group, '" + message + "'")
	w.channelActWrapped("group", false, ch, nil, echo, ToChar|ToSleep,
		wrapGroupColor(1, 2, true))
}

func actorInGroup(actor Actor) bool {
	switch member := actor.(type) {
	case *Player:
		return member.IsAffected(affGroup)
	case *MobInstance:
		return member.IsAffected(affGroup)
	default:
		return false
	}
}

func wrapGroupColor(openLevel, resetLevel int, trimLineEnding bool) func(Actor, string) string {
	return func(to Actor, line string) string {
		player, ok := to.(*Player)
		if !ok || player == nil {
			return line
		}
		flags := player.GetFlags()
		level := 0
		if flags&(1<<uint(PrfColor1)) != 0 {
			level++
		}
		if flags&(1<<uint(PrfColor2)) != 0 {
			level += 2
		}
		if trimLineEnding {
			line = strings.TrimSuffix(line, "\r\n")
		}
		if level < openLevel {
			return line
		}
		wrapped := "\x1b[37m" + line
		if level >= resetLevel {
			wrapped += "\x1b[0m"
		}
		return wrapped
	}
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
