// Package game — act() message substitution engine
// Ported from C src/comm.c:2392-2555
//
// act() is the most-used function in the entire MUD codebase. It performs
// $-code substitution on format strings and dispatches the result to the
// appropriate audience based on the act type (TO_CHAR, TO_VICT, TO_ROOM,
// TO_NOTVICT).

package game

import (
	"log"
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Act type constants — matching C comm.h
const (
	ToRoom    = 1
	ToVict    = 2
	ToNotVict = 3
	ToChar    = 4
	ToSleep   = 128
)

// Actor is the interface that both Player and MobInstance satisfy for act().
type Actor interface {
	GetName() string
	GetSex() int
	GetPosition() int
	SendMessage(msg string)
	GetRoom() int
	IsNPC() bool
}

// Ensure compliance at compile time.
var _ Actor = (*Player)(nil)
var _ Actor = (*MobInstance)(nil)

// --------------------------------------------------------------------------
// Pronoun helpers — faithful to C utils.h macros.
//
// Go sex values match C: 0=male, 1=female, 2=neutral (confirmed by
// MobInstance.GetSex comment and C SEX_* constants).
// --------------------------------------------------------------------------

// hmhr returns the objective pronoun for ch ("him"/"her"/"it").
// C: HMHR(ch) — GET_SEX(ch)==SEX_MALE ? "him" : GET_SEX(ch)==SEX_FEMALE ? "her" : "it"
func hmhr(a Actor) string {
	switch a.GetSex() {
	case 0:
		return "him"
	case 1:
		return "her"
	default:
		return "it"
	}
}

// hshr returns the possessive pronoun for ch ("his"/"her"/"its").
// C: HSHR(ch)
func hshr(a Actor) string {
	switch a.GetSex() {
	case 0:
		return "his"
	case 1:
		return "her"
	default:
		return "its"
	}
}

// hssh returns the subjective pronoun for ch ("he"/"she"/"it").
// C: HSSH(ch)
func hssh(a Actor) string {
	switch a.GetSex() {
	case 0:
		return "he"
	case 1:
		return "she"
	default:
		return "it"
	}
}

// sana returns "an" or "a" based on the first letter of the object's keywords.
// C: SANA(obj) — strchr("aeiouyAEIOUY", *(obj)->name) ? "an" : "a"
func sana(obj *ObjectInstance) string {
	if obj == nil || obj.Prototype == nil {
		return "a"
	}
	name := obj.Prototype.Keywords
	if name == "" {
		return "a"
	}
	first := rune(name[0])
	switch unicode.ToLower(first) {
	case 'a', 'e', 'i', 'o', 'u', 'y':
		return "an"
	default:
		return "a"
	}
}

// persName returns the name of ch as seen by observer.
// C: PERS(ch, vict) — GET_NAME(ch) if CAN_SEE(vict, ch), else "someone".
// Uses simplified CAN_SEE (awake check only) for now.
func persName(ch, observer Actor) string {
	if observer == nil {
		return ch.GetName()
	}
	if observer.GetPosition() <= combat.PosSleeping {
		return "someone"
	}
	return ch.GetName()
}

// canSee is a simplified CAN_SEE using the AWAKE check.
// Full CAN_SEE checks AFF_BLIND, AFF_INVISIBLE, AFF_HIDE, etc.
func canSee(observer, subject Actor) bool {
	if observer == nil {
		return true
	}
	return observer.GetPosition() > combat.PosSleeping
}

// sendOk checks whether 'to' can receive a message.
// C: SENDOK(ch, to_sleep) — desc && (AWAKE(ch) || to_sleep) && !PLR_WRITING
func sendOk(to Actor, toSleep bool) bool {
	if !toSleep && to.GetPosition() <= combat.PosSleeping {
		return false
	}
	return true
}

// fname returns the first word of a keywords string.
// C: fname() from utils.c
func fname(keywords string) string {
	idx := strings.IndexByte(keywords, ' ')
	if idx == -1 {
		return keywords
	}
	return keywords[:idx]
}

// objName returns the object's first keyword or "something" if not visible.
// C: OBJN(obj, vict) — fname(obj->name) if CAN_SEE_OBJ, else "something"
func objName(obj *ObjectInstance, to Actor) string {
	if obj == nil {
		return "something"
	}
	if !canSeeObject(to, obj) {
		return "something"
	}
	if obj.Prototype != nil {
		return fname(obj.Prototype.Keywords)
	}
	return "something"
}

// objShortDesc returns the object's short description or "something".
// C: OBJS(obj, vict) — obj->short_description if CAN_SEE_OBJ, else "something"
func objShortDesc(obj *ObjectInstance, to Actor) string {
	if obj == nil {
		return "something"
	}
	if !canSeeObject(to, obj) {
		return "something"
	}
	return obj.GetShortDesc()
}

// canSeeObject returns true if 'to' can see 'obj'.
// Simplified CAN_SEE_OBJ — uses awake check.
func canSeeObject(to Actor, obj *ObjectInstance) bool {
	if to == nil {
		return true
	}
	return to.GetPosition() > combat.PosSleeping
}

// cap capitalizes the first rune of s.
// C: CAP(st) — *st = UPPER(*st)
func cap(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// --------------------------------------------------------------------------
// performAct — the $-code substitution engine.
// C: perform_act(orig, ch, obj, vict_obj, to)
//
// Returns the fully formatted string with all $-codes resolved.
// Does NOT send — the caller sends.
//
// In C, vict_obj is void* and can be a char_data*, obj_data*, or char*.
// In Go we split this into:
//   - vict Actor        — for $N, $E, $S, $M (character tokens)
//   - victObj *ObjectInstance — for $O, $P, $A (object tokens)
//   - arg string        — for $t, $r, $q (first string arg)
//   - arg2 string       — for $T, $R, $Q (second string arg)
// --------------------------------------------------------------------------

func performAct(format string, ch, vict Actor, obj, victObj *ObjectInstance, arg, arg2 string, to Actor) string {
	var buf strings.Builder
	buf.Grow(len(format) + 64)

	i := 0
	for i < len(format) {
		if format[i] != '$' {
			buf.WriteByte(format[i])
			i++
			continue
		}
		i++ // skip $
		if i >= len(format) {
			break
		}
		code := format[i]
		i++

		var sub string
		switch code {
		case 'n':
			sub = persName(ch, to)
		case 'N':
			if vict != nil {
				sub = persName(vict, to)
			} else {
				sub = "someone"
			}
		case 'm':
			sub = hmhr(ch)
		case 'M':
			if vict != nil {
				sub = hmhr(vict)
			} else {
				sub = "someone"
			}
		case 's':
			sub = hshr(ch)
		case 'S':
			if vict != nil {
				sub = hshr(vict)
			} else {
				sub = "someone"
			}
		case 'e':
			sub = hssh(ch)
		case 'E':
			if vict != nil {
				sub = hssh(vict)
			} else {
				sub = "someone"
			}
		case 'o':
			if obj != nil {
				sub = objName(obj, to)
			} else {
				sub = "something"
			}
		case 'O':
			if victObj != nil {
				sub = objName(victObj, to)
			} else {
				sub = "something"
			}
		case 'p':
			if obj != nil {
				sub = objShortDesc(obj, to)
			} else {
				sub = "something"
			}
		case 'P':
			if victObj != nil {
				sub = objShortDesc(victObj, to)
			} else {
				sub = "something"
			}
		case 'a':
			if obj != nil {
				sub = sana(obj)
			} else {
				sub = "a"
			}
		case 'A':
			if victObj != nil {
				sub = sana(victObj)
			} else {
				sub = "a"
			}
		case 't':
			sub = arg
		case 'T':
			sub = arg2
		case 'r':
			if len(arg) > 0 {
				sub = strings.ToUpper(string(arg[0])) + arg[1:]
			}
		case 'R':
			if len(arg2) > 0 {
				sub = strings.ToUpper(string(arg2[0])) + arg2[1:]
			}
		case 'q':
			sub = strings.ToLower(arg)
		case 'Q':
			sub = strings.ToLower(arg2)
		case 'F':
			if arg2 != "" {
				sub = fname(arg2)
			} else {
				sub = "someone"
			}
		case '$':
			sub = "$"
		default:
			log.Printf("SYSERR: Illegal $-code to act(): %c", code)
		}

		buf.WriteString(sub)
	}

	return buf.String()
}

// --------------------------------------------------------------------------
// Act — the dispatch function.
// C: void act(str, hide_invisible, ch, obj, vict_obj, type)
//
// Parameters:
//   - world:       used only for TO_ROOM/TO_NOTVICT to iterate room occupants
//   - hideInvisible: if true, observers unable to see ch are skipped
//   - ch:          the actor character (can be nil)
//   - vict:        the victim/target character (can be nil) — $N/$E/$S/$M
//   - obj:         primary object (can be nil) — $o/$p/$a
//   - victObj:     secondary object (can be nil) — $O/$P/$A
//   - format:      the format string with $-codes
//   - arg2:        string argument — $T/$F/$R/$Q
//   - actType:     one of ToChar, ToVict, ToRoom, ToNotVict, optionally OR'd with ToSleep
// --------------------------------------------------------------------------

func Act(world *World, hideInvisible bool, ch, vict Actor, obj, victObj *ObjectInstance, format, arg2 string, actType int) {
	if format == "" {
		return
	}

	// Strip ToSleep bit
	toSleep := false
	if actType&ToSleep != 0 {
		toSleep = true
		actType &^= ToSleep
	}

	// TO_CHAR: send to ch only
	if actType == ToChar {
		if ch != nil && sendOk(ch, toSleep) {
			msg := performAct(format, ch, vict, obj, victObj, "", arg2, ch)
			ch.SendMessage(cap(msg) + "\r\n")
		}
		return
	}

	// TO_VICT: send to vict only
	if actType == ToVict {
		if vict != nil && sendOk(vict, toSleep) {
			msg := performAct(format, ch, vict, obj, victObj, "", arg2, vict)
			vict.SendMessage(cap(msg) + "\r\n")
		}
		return
	}

	// TO_ROOM or TO_NOTVICT: iterate room occupants
	if world == nil {
		log.Println("SYSERR: no valid target to act()!")
		return
	}

	// Determine room VNum from ch or obj
	roomVNum := -1
	if ch != nil {
		roomVNum = ch.GetRoom()
	} else if obj != nil {
		roomVNum = obj.RoomVNum
	}
	if roomVNum < 0 {
		log.Println("SYSERR: no valid target to act()!")
		return
	}

	// Get all actors in the room
	actors := world.actChar(roomVNum)
	for _, to := range actors {
		if !sendOk(to, toSleep) {
			continue
		}
		if hideInvisible && ch != nil && !canSee(to, ch) {
			continue
		}
		// Skip ch for both TO_ROOM and TO_NOTVICT
		if to == ch {
			continue
		}
		// For TO_NOTVICT, also skip vict
		if actType == ToNotVict && vict != nil && to == vict {
			continue
		}
		msg := performAct(format, ch, vict, obj, victObj, "", arg2, to)
		to.SendMessage(cap(msg) + "\r\n")
	}
}

// actChar returns all actors (players + mobs) in a room.
func (w *World) actChar(roomVNum int) []Actor {
	var actors []Actor
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		actors = append(actors, p)
	}
	for _, m := range w.GetMobsInRoom(roomVNum) {
		actors = append(actors, m)
	}
	return actors
}

// --------------------------------------------------------------------------
// Convenience wrappers
// --------------------------------------------------------------------------

// SendToChar sends a formatted message to just ch.
func SendToChar(ch Actor, format string) {
	Act(nil, false, ch, nil, nil, nil, format, "", ToChar)
}

// SendToVict sends a formatted message to just vict.
func SendToVict(ch, vict Actor, format string) {
	Act(nil, false, ch, vict, nil, nil, format, "", ToVict)
}

// SendToRoom sends a formatted message to everyone in ch's room except ch.
func SendToRoom(world *World, ch Actor, format string) {
	Act(world, false, ch, nil, nil, nil, format, "", ToRoom)
}
