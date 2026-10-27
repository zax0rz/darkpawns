package game

// SaySpellDeliver is say_spell's audience (src/spell_parser.c:325-343). Every
// occupant of the caster's room except the caster and the spell's target that
// has a descriptor, is awake and is not writing gets perform_act of the line
// for its class: the real spell name when it shares the caster's class, the
// syllable-obfuscated one otherwise. perform_act renders $n through PERS and
// $p through OBJS for that listener, so a caster the listener cannot see is
// "someone" (blind, dark room, invisible caster) and an unseen object is
// "something". The target then receives its own line with act(..., TO_VICT).
//
// sameClass and otherClass carry the spell name already substituted; vict is
// the target's line or "" when C sends none. It reports false when ch is not a
// live body in a room, so the caller can fall back.
func (w *World) SaySpellDeliver(ch, tch, tobj interface{}, sameClass, otherClass, vict string) bool {
	caster, ok := ch.(Actor)
	if !ok || caster == nil || w == nil {
		return false
	}
	room := caster.GetRoom()
	if room < 0 {
		return false
	}
	target, _ := tch.(Actor)
	obj, _ := tobj.(*ObjectInstance)
	casterClass, hasClass := actorClass(caster)

	for _, to := range w.actChar(room) {
		if to == caster || (target != nil && to == target) {
			continue
		}
		// C: !i->desc. Mobiles have no descriptor here.
		listener, isPlayer := to.(*Player)
		if !isPlayer {
			continue
		}
		// C: !AWAKE(i) || PLR_FLAGGED(i, PLR_WRITING).
		if !sendOk(listener, false) || listener.GetFlags()&(1<<uint(PlrWriting)) != 0 {
			continue
		}
		line := otherClass
		if listenerClass, ok := actorClass(listener); ok && hasClass && listenerClass == casterClass {
			line = sameClass
		}
		msg := performAct(line, caster, target, obj, nil, "", "", listener)
		listener.SendMessage(cap(msg) + "\r\n")
	}

	if vict != "" && target != nil && target != caster && target.GetRoom() == room {
		Act(w, false, caster, target, nil, nil, vict, "", ToVict)
	}
	return true
}

func actorClass(a Actor) (int, bool) {
	if c, ok := a.(interface{ GetClass() int }); ok {
		return c.GetClass(), true
	}
	return 0, false
}

// SpellActNotVict is act(format, hideInvisible, ch, 0, vict, TO_NOTVICT) for
// spell code, which cannot import this package. Rendering through Act gives
// each listener PERS for $n/$N and, with hideInvisible, skips a listener who
// cannot see ch, as C does. It reports false when ch is not a body, so the
// caller can fall back.
func (w *World) SpellActNotVict(ch, vict interface{}, format string, hideInvisible bool) bool {
	caster, ok := ch.(Actor)
	if !ok || caster == nil || w == nil {
		return false
	}
	target, _ := vict.(Actor)
	Act(w, hideInvisible, caster, target, nil, nil, format, "", ToNotVict)
	return true
}

// SpellActToRoom is act(format, hideInvisible, ch, 0, vict, TO_ROOM) for spell
// code; see SpellActNotVict.
func (w *World) SpellActToRoom(ch, vict interface{}, format string, hideInvisible bool) bool {
	caster, ok := ch.(Actor)
	if !ok || caster == nil || w == nil {
		return false
	}
	target, _ := vict.(Actor)
	Act(w, hideInvisible, caster, target, nil, nil, format, "", ToRoom)
	return true
}

// SpellCreatedObjectAct is magic.c:1997's act("$n creates $p.", FALSE,
// ch, tobj, 0, TO_ROOM) used by MagCreations. Returning true means the live
// game's Act renderer handled the room line, including SENDOK and per-viewer
// object visibility.
func (w *World) SpellCreatedObjectAct(ch, obj interface{}) bool {
	actor, actorOK := ch.(Actor)
	object, objectOK := obj.(*ObjectInstance)
	if !actorOK || actor == nil || !objectOK || object == nil || w == nil {
		return false
	}
	Act(w, false, actor, nil, object, nil, "$n creates $p.", "", ToRoom)
	return true
}

// SpellActToVictim is act(format, FALSE, caster, 0, victim, TO_VICT) for
// manual spells. The target observes $n through C's PERS visibility rules.
func (w *World) SpellActToVictim(ch, vict interface{}, format string) bool {
	caster, casterOK := ch.(Actor)
	target, targetOK := vict.(Actor)
	if !casterOK || !targetOK || caster == nil || target == nil || w == nil {
		return false
	}
	Act(w, false, caster, target, nil, nil, format, "", ToVict)
	return true
}
