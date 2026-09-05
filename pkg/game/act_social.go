// act_social.go — Ported from src/act.social.c
//
// Social action commands: do_action, do_insult, do_dream
// Uses the Socials data from socials.go (parsed from lib/misc/socials).

package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"

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

	// C: PLR_FLAGGED(ch, PLR_NOSHOUT) — mute check blocks socials
	if ch.GetFlags()&(1<<PlrNoshout) != 0 {
		ch.SendMessage("You cannot perform emotes!\r\n")
		return true
	}

	// C treats a missing char_found message as a self-only social: typed
	// arguments are ignored and the no-argument pair is emitted.
	if _, ok := socialMessage(social, socCharFound); !ok {
		if message, ok := socialMessage(social, socCharNoArg); ok {
			Act(nil, false, ch, nil, nil, nil, message, "", ToChar)
		}
		if message, ok := socialMessage(social, socOthersNoArg); ok {
			Act(w, social.hidesInvisibleActor(), ch, nil, nil, nil, message, "", ToRoom)
		}
		return true
	}

	// Extract target name from argument
	targetName := extractArg(argument)

	// No argument supplied — use no_arg messages
	if targetName == "" {
		if message, ok := socialMessage(social, socCharNoArg); ok {
			Act(nil, false, ch, nil, nil, nil, message, "", ToChar)
		}
		if message, ok := socialMessage(social, socOthersNoArg); ok {
			Act(w, social.hidesInvisibleActor(), ch, nil, nil, nil, message, "", ToRoom)
		}
		return true
	}

	// Try to find the target in the room
	target := w.findSocialTarget(ch, targetName)

	if target == nil {
		// Target not found
		if message, ok := socialMessage(social, socNotFound); ok {
			// C sends action->not_found directly with send_to_char(), so
			// literal $-codes remain literal when no victim exists.
			ch.SendMessage(message + "\r\n")
		}
		return true
	}

	// Check if target is self
	targetActor := target.(Actor)
	if target.GetName() == ch.Name {
		if message, ok := socialMessage(social, socCharAuto); ok {
			// C sends char_auto with send_to_char(), not act(). Preserve the
			// authored bytes and any literal $-codes in this actor-only path.
			ch.SendMessage(message + "\r\n")
		}
		if message, ok := socialMessage(social, socOthersAuto); ok {
			Act(w, social.hidesInvisibleActor(), ch, nil, nil, nil, message, "", ToRoom)
		}
		return true
	}

	// Check minimum victim position (DP-411)
	if minimumPosition := social.minimumVictimPosition(); minimumPosition > 0 && targetActor.GetPosition() < minimumPosition {
		Act(nil, false, ch, targetActor, nil, nil, "$N is not in a proper position for that.", "", ToChar|ToSleep)
		return true
	}

	// Target is another character — send messages to actor, room, and target
	// using the new Act() engine which handles $-codes, capitalization, \r\n
	if message, ok := socialMessage(social, socCharFound); ok {
		Act(nil, false, ch, targetActor, nil, nil, message, "", ToChar|ToSleep)
	}

	if message, ok := socialMessage(social, socOthersFound); ok {
		Act(w, social.hidesInvisibleActor(), ch, targetActor, nil, nil, message, "", ToNotVict)
	}

	if message, ok := socialMessage(social, socVictFound); ok {
		Act(nil, social.hidesInvisibleActor(), ch, targetActor, nil, nil, message, "", ToVict)
	}

	return true
}

func socialMessage(social *Social, index int) (string, bool) {
	if social == nil || index < 0 || index >= len(social.Messages) || social.Messages[index] == "#" {
		return "", false
	}
	return social.Messages[index], true
}

// DoInsult implements do_insult() from act.social.c.
func DoInsult(w *World, ch *Player, argument string) {
	targetName := extractArg(argument)

	if targetName == "" {
		ch.SendMessage("I'm sure you don't want to insult *everybody*...\r\n")
		return
	}

	target := w.findSocialTarget(ch, targetName)

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
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	switch dprng.Number(0, 2) {
	case 0:
		if ch.GetSex() == SexMale {
			if target.GetSex() == SexMale {
				insultFormat = "$n accuses you of fighting like a woman!"
			} else {
				insultFormat = "$n says that women can't fight."
			}
		} else { // female or neutral
			if target.GetSex() == SexMale {
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

	// C emits the room act before the actor's private line.  TO_ROOM keeps
	// sleeping recipients out through SENDOK, and hide_invisible=TRUE keeps
	// the source actor hidden from observers who cannot see them.
	Act(w, true, ch, nil, nil, nil, "$n dreams of running naked through a field of tulips.", "", ToRoom)
	ch.SendMessage("You dream of running naked through a field of tulips.\r\n")
}

// extractArg returns the first word of argument, or "" if empty.
func extractArg(argument string) string {
	arg := strings.TrimSpace(argument)
	if arg == "" {
		return ""
	}
	// C do_action parses the target with one_argument (act.social.c): fill
	// words dropped, token lowercased.
	arg1, _ := oneArgument(arg)
	return arg1
}

// findSocialTarget finds a visible character in the room by name, checking
// mobs first then players, matching C get_char_room_vis().
func (w *World) findSocialTarget(observer *Player, name string) socialTarget {
	vnum := observer.GetRoomVNum()
	nameLower := strings.ToLower(name)
	if nameLower == "self" || nameLower == "me" {
		return observer
	}

	// Check mobs in the room
	mobs := w.GetMobsInRoom(vnum)
	for _, m := range mobs {
		if isnameWithAbbrevs(name, charKeywords(m)) && canSeeSocialTarget(observer, m) {
			return m
		}
	}

	// Check players
	players := w.GetPlayersInRoom(vnum)
	for _, p := range players {
		if isnameWithAbbrevs(name, charKeywords(p)) && canSeeSocialTarget(observer, p) {
			return p
		}
	}

	return nil
}

// canSeeSocialTarget matches get_char_room_vis's CAN_SEE checks. Unlike Act's
// delivery gate, C target lookup does not reject an otherwise visible target
// merely because the actor is sleeping; the command's POS_RESTING gate is
// enforced by the interpreter before do_action runs.
func canSeeSocialTarget(observer, subject Actor) bool {
	if observer == nil || subject == nil {
		return true
	}
	obs, obsOK := observer.(visibilitySubject)
	sbj, sbjOK := subject.(visibilitySubject)
	if !obsOK || !sbjOK || obs.GetLevel() >= LVL_IMMORT {
		return true
	}
	if holyLight, ok := obs.(holyLightSubject); ok && holyLight.GetHolyLight() {
		return true
	}
	if obs.IsAffected(affBlind) {
		return false
	}
	if sbj.IsAffected(affInvisible) && !obs.IsAffected(affDetectInvisible) {
		return false
	}
	if sbj.IsAffected(affHide) && !obs.IsAffected(affSenseLife) {
		return false
	}
	return true
}

// roomMessageExcludeTwo sends a message to all players in a room except two named ones.
func (w *World) roomMessageExcludeTwo(vnum int, msg string, exclude1, exclude2 string) {
	for _, p := range w.GetPlayersInRoom(vnum) {
		if p.Name != exclude1 && p.Name != exclude2 {
			p.SendMessage(msg + "\r\n")
		}
	}
}
