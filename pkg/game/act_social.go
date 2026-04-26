// act_social.go — Ported from src/act.social.c
//
// Social action commands: do_action, do_insult, do_dream
// Uses the Socials data from socials.go (parsed from lib/misc/socials).

package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Social message indices (matching the socials file loading order).
const (
	socCharNoArg   = 0 // No argument supplied — message to actor
	socOthersNoArg = 1 // No argument supplied — message to room
	socCharFound   = 2 // Target found — message to actor
	socOthersFound = 3 // Target found — message to room (not actor, not target)
	socVictFound   = 4 // Target found — message to target
	socNotFound    = 5 // Argument given but target not found
	socCharAuto    = 6 // Target is self — message to actor
	socOthersAuto  = 7 // Target is self — message to room
)

// socialTarget is anything that can be the target of a social.
type socialTarget interface {
	GetName() string
	SendMessage(msg string)
	GetSex() int
	GetRoom() int
}

// ensure Player satisfies socialTarget
var _ socialTarget = (*Player)(nil)

// ensure MobInstance satisfies socialTarget
var _ socialTarget = (*MobInstance)(nil)

// DoAction implements do_action() from act.social.c.
// cmd is the command name (e.g. "grin", "hug").
// argument is whatever the user typed after the command.
// Returns true if the social was found and handled, false if no social matches cmd.
func DoAction(w *World, ch *Player, cmd string, argument string) bool {
	social, ok := Socials[cmd]
	if !ok {
		ch.SendMessage("That action is not supported.\r\n")
		return true
	}

	// Extract target name from argument
	targetName := extractArg(argument)

	// No argument supplied — use no_arg messages
	if targetName == "" {
		Act(nil, false, ch, nil, nil, nil, social.Messages[socCharNoArg], "", ToChar)
		Act(w, false, ch, nil, nil, nil, social.Messages[socOthersNoArg], "", ToRoom)
		return true
	}

	// Try to find the target in the room
	target := w.findSocialTarget(ch.GetRoomVNum(), targetName)

	if target == nil {
		// Target not found
		if socNotFound < len(social.Messages) {
			Act(nil, false, ch, nil, nil, nil, social.Messages[socNotFound], "", ToChar)
		}
		return true
	}

	// Check if target is self
	targetActor := target.(Actor)
	if target.GetName() == ch.Name {
		if socCharAuto < len(social.Messages) {
			Act(nil, false, ch, nil, nil, nil, social.Messages[socCharAuto], "", ToChar)
		}
		if socOthersAuto < len(social.Messages) {
			Act(w, false, ch, nil, nil, nil, social.Messages[socOthersAuto], "", ToRoom)
		}
		return true
	}

	// Target is another character — send messages to actor, room, and target
	// using the new Act() engine which handles $-codes, capitalization, \r\n
	if socCharFound < len(social.Messages) {
		Act(nil, false, ch, targetActor, nil, nil, social.Messages[socCharFound], "", ToChar)
	}

	if socOthersFound < len(social.Messages) {
		Act(w, false, ch, targetActor, nil, nil, social.Messages[socOthersFound], "", ToNotVict)
	}

	if socVictFound < len(social.Messages) {
		Act(nil, false, ch, targetActor, nil, nil, social.Messages[socVictFound], "", ToVict)
	}

	return true
}

// DoInsult implements do_insult() from act.social.c.
func DoInsult(w *World, ch *Player, argument string) {
	targetName := extractArg(argument)

	if targetName == "" {
		ch.SendMessage("I'm sure you don't want to insult *everybody*...\r\n")
		return
	}

	target := w.findSocialTarget(ch.GetRoomVNum(), targetName)

	if target == nil {
		ch.SendMessage("Can't hear you!\r\n")
		return
	}

	if target.GetName() == ch.Name {
		ch.SendMessage("You feel insulted.\r\n")
		return
	}

	targetActor := target.(Actor)

	ch.SendMessage(fmt.Sprintf("You insult %s.\r\n", target.GetName()))

	// Pick a random insult — send to target via Act()
	var insultFormat string
	switch rand.Intn(3) {
	case 0:
		if ch.GetSex() == 1 { // male
			if target.GetSex() == 1 {
				insultFormat = "$n accuses you of fighting like a woman!"
			} else {
				insultFormat = "$n says that women can't fight."
			}
		} else { // female or neutral
			if target.GetSex() == 1 {
				insultFormat = "$n accuses you of having the smallest... (brain?)"
			} else {
				insultFormat = "$n tells you that you'd lose a beauty contest against a troll."
			}
		}
	case 1:
		insultFormat = "$n calls your mother a bitch!"
	default:
		insultFormat = "$n tells you to get lost!"
	}
	Act(nil, false, ch, targetActor, nil, nil, insultFormat, "", ToVict)

	// Message to everyone else in the room
	Act(w, false, ch, targetActor, nil, nil, "$n insults $N.", "", ToNotVict)
}

// DoDream implements do_dream() from act.social.c.
func DoDream(w *World, ch *Player) {
	if ch.GetPosition() != combat.PosSleeping {
		ch.SendMessage("You daydream about better times.\r\n")
		return
	}

	// Send to self
	ch.SendMessage("You dream of running naked through a field of tulips.\r\n")

	// Send to room (excluding ch), with ToSleep bit so sleeping chars still see it
	Act(w, false, ch, nil, nil, nil, "$n dreams of running naked through a field of tulips.", "", ToRoom|ToSleep)
}

// extractArg returns the first word of argument, or "" if empty.
func extractArg(argument string) string {
	arg := strings.TrimSpace(argument)
	if arg == "" {
		return ""
	}
	parts := strings.SplitN(arg, " ", 2)
	return parts[0]
}

// findSocialTarget finds a character in the room by name, checking mobs first then players.
func (w *World) findSocialTarget(vnum int, name string) socialTarget {
	nameLower := strings.ToLower(name)

	// Check mobs in the room
	mobs := w.GetMobsInRoom(vnum)
	for _, m := range mobs {
		mobNameLower := strings.ToLower(m.GetName())
		if mobNameLower == nameLower || strings.HasPrefix(mobNameLower, nameLower) {
			return m
		}
	}

	// Check players
	players := w.GetPlayersInRoom(vnum)
	for _, p := range players {
		pNameLower := strings.ToLower(p.GetName())
		if pNameLower == nameLower || strings.HasPrefix(pNameLower, nameLower) {
			return p
		}
	}

	return nil
}

// roomMessageExcludeTwo sends a message to all players in a room except two named ones.
func (w *World) roomMessageExcludeTwo(vnum int, msg string, exclude1, exclude2 string) {
	for _, p := range w.GetPlayersInRoom(vnum) {
		if p.Name != exclude1 && p.Name != exclude2 {
			p.SendMessage(msg + "\r\n")
		}
	}
}
