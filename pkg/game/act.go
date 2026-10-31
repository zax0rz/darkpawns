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
var (
	_ Actor = (*Player)(nil)
	_ Actor = (*MobInstance)(nil)
)

// --------------------------------------------------------------------------
// Pronoun helpers — faithful to C utils.h macros.
//
// Go sex values are 0=male, 1=female, 2=neutral. This intentionally does not
// copy C's numeric SEX_* constants (neutral=0, male=1, female=2); the Go Player
// and MobInstance encoding is authoritative at this boundary.
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
	if obj == nil {
		return "a"
	}
	name := obj.GetKeywords()
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
func persName(ch, observer Actor) string {
	if observer == nil {
		return ch.GetName()
	}
	// C's PERS() delegates to CAN_SEE(), whose macro checks blindness, light,
	// and invisibility but deliberately does not treat AFF_HIDE as invisible.
	// Room listing has a separate AFF_HIDE branch; keep that distinction here.
	if !canSeeForPers(observer, ch) {
		return "someone"
	}
	return ch.GetName()
}

// PersName exposes the shared C PERS visibility rule for prompt rendering.
func PersName(subject, observer Actor) string { return persName(subject, observer) }

// canSee returns true if observer can see subject.
// Faithful port of CAN_SEE macro.
func canSee(observer, subject Actor) bool {
	return canSeeWithHide(observer, subject, true)
}

// canSeeForPers mirrors the narrower C CAN_SEE path used by PERS(). C's
// CAN_SEE macro does not inspect AFF_HIDE; callers that need room-listing hide
// semantics use canSee instead.
func canSeeForPers(observer, subject Actor) bool {
	return canSeeWithHide(observer, subject, false)
}

// canSeeCharacterInRoom is canSeeForPers: CAN_SEE's LIGHT_OK arm resolves
// the observer's room light through the body's owning world
// (src/utils.h:515-530), so the shared predicate needs no explicit world.

func canSeeWithHide(observer, subject Actor, hide bool) bool {
	if observer == nil || subject == nil {
		return true
	}
	if observer == subject {
		return true
	}

	// src/utils.h:515-530: visibility does not require AWAKE.

	// Check if observer is a Player or MobInstance (both implement visibilitySubject)
	obsSub, ok := observer.(visibilitySubject)
	if !ok {
		return true
	}
	sbjSub, ok := subject.(visibilitySubject)
	if !ok {
		return true
	}

	// C compares GET_REAL_LEVEL(sub) with GET_INVIS_LEV(obj) first (src/utils.h:529-530).
	if invis, ok := subject.(invisLevelSubject); ok && obsSub.GetLevel() < invis.GetInvisLevel() {
		return false
	}
	// C's IMM_CAN_SEE is MORT_CAN_SEE || PRF_HOLYLIGHT. Level only gates the
	// target's invis level above; it does not reveal AFF_INVISIBLE.
	if hl, ok := obsSub.(holyLightSubject); ok && hl.GetHolyLight() {
		return true
	}

	// 1. LIGHT_OK's blindness arm (src/utils.h:515-517).
	if obsSub.IsAffected(affBlind) {
		return false
	}

	// 2. LIGHT_OK's room-light arm: the observer's room must be lit or the
	// observer has infravision. DP-1295 — every act() $n/$N, PERS and the
	// shared look/target helpers flow through here, so a dark room renders
	// "someone" exactly as C does.
	if w := visibilityWorld(observer); w != nil {
		if w.IsRoomDark(observer.GetRoom()) && !obsSub.IsAffected(affInfravision) {
			return false
		}
	}

	// 3. Invisibility check
	if sbjSub.IsAffected(affInvisible) {
		if !obsSub.IsAffected(affDetectInvisible) {
			return false
		}
	}

	// AFF_HIDE is NOT part of C's CAN_SEE (src/utils.h:521-530): a hidden
	// character is targetable, act()-visible and pursuable — only
	// list_char_to_char suppresses it (src/act.informative.c:637-643), and
	// that filter lives at the listing sites. The hide arm that used to live
	// here made every hidden body untargetable in the port (a hidden
	// prototype mob could not be forced, statted or hit while C's
	// get_char_room_vis found it, src/handler.c:1277-1300).
	_ = hide

	return true
}

// visibilityWorld resolves the owning world from a body for CAN_SEE's
// LIGHT_OK arm. Both live bodies carry it; a foreign Actor implementation
// (tests only) sees the pre-DP-1295 light-blind behavior.
func visibilityWorld(observer Actor) *World {
	switch body := observer.(type) {
	case *Player:
		// worldRef is set by World.AddPlayer and read under the player's lock,
		// as Player.SendMessage does.
		body.mu.RLock()
		defer body.mu.RUnlock()
		return body.worldRef
	case *MobInstance:
		return body.world
	}
	return nil
}

// CanSee reports whether observer can see subject using the canonical
// CAN_SEE rules. Session-owned commands such as who use this viewer-aware
// boundary instead of duplicating visibility policy.
func CanSee(observer, subject Actor) bool {
	return canSee(observer, subject)
}

// CanSeeObject reports whether observer can see obj using the canonical
// CAN_SEE_OBJ rules. Session-owned commands use this viewer-aware boundary
// instead of duplicating object visibility policy.
// ChCanSeeObj is the player-typed CAN_SEE_OBJ wrapper for session feeds.
func ChCanSeeObj(ch *Player, obj *ObjectInstance) bool { return chCanSeeObj(ch, obj) }

func CanSeeObject(observer Actor, obj *ObjectInstance) bool {
	return canSeeObject(observer, obj)
}

// sendOk checks whether 'to' can receive a message.
// C: SENDOK(ch, to_sleep) — desc && (AWAKE(ch) || to_sleep) && !PLR_WRITING
func sendOk(to Actor, toSleep bool) bool {
	if !toSleep && to.GetPosition() <= combat.PosSleeping {
		return false
	}
	if player, ok := to.(*Player); ok && player.GetFlags()&(1<<uint(PlrWriting)) != 0 {
		return false
	}
	return true
}

// CanReceiveAct reports whether a player recipient passes C's SENDOK gate for
// an ordinary act() message. Session-owned event broadcasters use the same
// position and writing checks as Act; their session membership supplies the
// connected-descriptor part of SENDOK.
func CanReceiveAct(to *Player) bool {
	return to != nil && sendOk(to, false)
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
	return fname(obj.GetKeywords())
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

// visibilitySubject is an interface for entities that can see or be seen.
// Both Player and MobInstance implement this.
type visibilitySubject interface {
	GetPosition() int
	GetLevel() int
	GetRoom() int
	IsAffected(bit int) bool
	IsNPC() bool
}

type invisLevelSubject interface {
	GetInvisLevel() int
}

var (
	_ visibilitySubject = (*Player)(nil)
	_ visibilitySubject = (*MobInstance)(nil)
)

// GetHolyLight returns true if the entity has holy light enabled.
// Player implements this; MobInstance does not.
type holyLightSubject interface {
	GetHolyLight() bool
}

// canSeeObject returns true if 'to' can see 'obj'.
// Ported from C: CAN_SEE_OBJ(sub, obj)
//
//	= MORT_CAN_SEE_OBJ(sub, obj) || PRF_FLAGGED(sub, PRF_HOLYLIGHT)
//	= (LIGHT_OK(sub) && INVIS_OK_OBJ(sub, obj)) || PRF_HOLYLIGHT
//	LIGHT_OK(sub) = !IS_AFFECTED(sub, AFF_BLIND) && (IS_LIGHT(sub->in_room) || IS_AFFECTED(sub, AFF_INFRAVISION))
//	INVIS_OK_OBJ(sub, obj) = !IS_OBJ_STAT(obj, ITEM_INVISIBLE) || IS_AFFECTED(sub, AFF_DETECT_INVIS)
func canSeeObject(to Actor, obj *ObjectInstance) bool {
	if to == nil {
		return true
	}

	// src/utils.h:535-542: object visibility does not require AWAKE.

	// Check if observer is a Player or MobInstance (both implement visibilitySubject)
	sub, ok := to.(visibilitySubject)
	if !ok {
		// Fallback: can't check affects, assume visible
		return true
	}

	// Check PRF_HOLYLIGHT — players with holy light see everything
	if hl, ok := sub.(holyLightSubject); ok && hl.GetHolyLight() {
		return true
	}

	// LIGHT_OK = !AFF_BLIND && (room is light || observer has infravision)
	// (src/utils.h:515-517). DP-1295: the room-light arm resolves through the
	// observer's owning world, so this is the complete CAN_SEE_OBJ predicate.
	if sub.IsAffected(affBlind) {
		return false
	}
	if w := visibilityWorld(to); w != nil {
		if w.IsRoomDark(to.GetRoom()) && !sub.IsAffected(affInfravision) {
			return false
		}
	}

	// INVIS_OK_OBJ = !ITEM_INVISIBLE || observer has detect invis
	if obj != nil && obj.HasExtraFlag(0, extraFlagInvisible) {
		// Object is invisible — need detect invis
		return sub.IsAffected(affDetectInvisible)
	}

	return true
}

// canSeeObjectInWorld is canSeeObject: LIGHT_OK's room-light arm resolves
// through the observer's owning world (src/utils.h:515-542), so the shared
// object predicate needs no explicit world.

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
// Act — the canonical player-facing text primitive.
// C: void act(str, hide_invisible, ch, obj, vict_obj, type)
//
// Game operations author C-style $-code messages and route them exclusively
// through Act so substitution, visibility, pronouns, capitalization, audience,
// and line endings have one implementation. Player.SendMessage crosses the
// transport boundary through World.MessageSink: telnet renders the resulting
// text and WebSocket sessions wrap the same text in an EventData{Type:"text"}
// envelope. Structured MsgState payloads remain separate from Act.
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
	actDeliver(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType, func(to Actor, line string) {
		sendActLineWithWorld(world, to, line)
	})
}

// ActWithPrivateSink preserves the supplied actor's identity and body state
// throughout Act's audience, visibility, and substitution checks while
// routing that actor's delivered line through a descriptor-owned sink. It is
// for switched actors whose body is not itself the connected Player.
func ActWithPrivateSink(world *World, hideInvisible bool, ch, vict Actor, obj, victObj *ObjectInstance, format, arg2 string, actType int, privateActor Actor, privateSink func(string)) {
	if privateActor == nil || privateSink == nil {
		Act(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType)
		return
	}
	actDeliver(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType, func(to Actor, line string) {
		if to == privateActor {
			privateSink(line)
			return
		}
		sendActLineWithWorld(world, to, line)
	})
}

// sendActLine is Act's delivery: the rendered line goes straight to the
// recipient's transport.
func sendActLine(to Actor, line string) { to.SendMessage(line) }

// sendActLineWithWorld routes output to a descriptor attached to an NPC body.
// A descriptorless NPC keeps SendMessage's no-op behavior.
func sendActLineWithWorld(world *World, to Actor, line string) {
	if mob, ok := to.(*MobInstance); ok && world != nil && world.MobileMessageSink != nil && world.MobileDescriptorOwner != nil {
		if world.MobileDescriptorOwner(mob) != nil {
			world.MobileMessageSink(mob, []byte(line))
			return
		}
	}
	sendActLine(to, line)
}

// actDeliver is act()'s audience and rendering logic with the final write
// supplied by the caller. Act delivers with sendActLine; channelAct also
// mirrors the identical line out of band. Keeping one body means the text a
// player receives cannot depend on which of the two called it.
func actDeliver(world *World, hideInvisible bool, ch, vict Actor, obj, victObj *ObjectInstance, format, arg2 string, actType int, deliver func(to Actor, line string)) {
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
			deliver(ch, cap(msg)+"\r\n")
		}
		return
	}

	// TO_VICT: send to vict only
	if actType == ToVict {
		if vict != nil && sendOk(vict, toSleep) {
			msg := performAct(format, ch, vict, obj, victObj, "", arg2, vict)
			deliver(vict, cap(msg)+"\r\n")
		}
		return
	}

	// TO_ROOM or TO_NOTVICT: iterate room occupants
	if world == nil {
		// Go-defensive arm. C's world is a global array (comm.c:2400) and
		// cannot be nil, so this arm has no C counterpart; the message and
		// its type/level/file contract are still C's (comm.c:2525).
		MudLog("SYSERR: no valid target to act()!", MudlogComplete, LVL_IMMORT, true)
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
		// comm.c:2523-2526: neither ch nor obj has a valid room, so act()
		// has no audience. C logs this before returning; the file side comes
		// from the same MudLog call (toFile TRUE).
		MudLog("SYSERR: no valid target to act()!", MudlogComplete, LVL_IMMORT, true)
		return
	}

	// src/comm.c:2529-2530: arena broadcast precedes the normal room act.
	if ch != nil && world.RoomHasFlag(ch.GetRoom(), "arena") {
		actArenaBroadcast(world, hideInvisible, ch, vict, obj, victObj, format, arg2, deliver)
	}

	// Get all actors in the room
	actors := world.actChar(roomVNum)
	detachedOwners := world.mobileDescriptorOwners(roomVNum)
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
		if player, ok := to.(*Player); ok && detachedOwners[player] {
			continue
		}
		// For TO_NOTVICT, also skip vict
		if actType == ToNotVict && vict != nil && to == vict {
			continue
		}
		msg := performAct(format, ch, vict, obj, victObj, "", arg2, to)
		deliver(to, cap(msg)+"\r\n")
	}
}

func (w *World) mobileDescriptorOwners(roomVNum int) map[*Player]bool {
	owners := make(map[*Player]bool)
	if w == nil || w.MobileDescriptorOwner == nil {
		return owners
	}
	for _, mob := range w.GetMobsInRoom(roomVNum) {
		if owner := w.MobileDescriptorOwner(mob); owner != nil {
			owners[owner] = true
		}
	}
	return owners
}

// actArenaBroadcast ports do_broadcast (src/comm.c:2539-2556). Its format
// remains raw &R/&n text until the recipient's terminal color funnel expands it.
func actArenaBroadcast(world *World, hideInvisible bool, ch, vict Actor, obj, victObj *ObjectInstance, format, arg2 string, deliver func(Actor, string)) {
	broadcast := "&RBroadcast: " + format + "&n"
	var actors []Actor
	for _, p := range world.GetAllPlayers() {
		actors = append(actors, p)
	}
	for _, m := range world.GetAllMobs() {
		actors = append(actors, m)
	}
	detachedOwners := make(map[*Player]bool)
	if world.MobileDescriptorOwner != nil {
		for _, mob := range world.GetAllMobs() {
			if owner := world.MobileDescriptorOwner(mob); owner != nil {
				detachedOwners[owner] = true
			}
		}
	}
	for _, to := range actors {
		if to == ch || !sendOk(to, true) {
			continue
		}
		if player, ok := to.(*Player); ok && detachedOwners[player] {
			continue
		}
		// The live nobroadcast toggle writes PrfNoBroad, not NoBroadcast.
		if p, ok := to.(*Player); ok && p.GetFlags()&(1<<uint(PrfNoBroad)) != 0 {
			continue
		}
		if hideInvisible && !canSee(to, ch) {
			continue
		}
		msg := performAct(broadcast, ch, vict, obj, victObj, "", arg2, to)
		deliver(to, cap(msg)+"\r\n")
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

// SendToChar is C's send_to_char (src/comm.c:2335-2339): a direct write to the
// character's descriptor, with no act() substitution and no SENDOK gate, so it
// reaches a sleeping or writing recipient. Callers that want C's act() gate use
// Act(...ToChar) instead.
func SendToChar(ch Actor, messg string) {
	if ch == nil || messg == "" {
		return
	}
	ch.SendMessage(messg)
}
