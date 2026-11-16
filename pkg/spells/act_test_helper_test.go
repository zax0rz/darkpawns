package spells

import "strings"

// testActWorld provides the world act() bridge methods for unit tests in
// pkg/spells that do not boot a full game.World.
type testActWorld struct {
	roomChars func(room int) []any
}

func testFormatAct(format string, actor, victim, obj any) string {
	msg := format
	if actor != nil {
		actorName := "someone"
		if n, ok := actor.(interface{ GetName() string }); ok {
			actorName = n.GetName()
		}
		msg = strings.ReplaceAll(msg, "$n", actorName)
		msg = strings.ReplaceAll(msg, "$s", possessivePronoun(actor))
		msg = strings.ReplaceAll(msg, "$m", objectivePronoun(actor))
		msg = strings.ReplaceAll(msg, "$e", testSubjectivePronoun(actor))
	}
	if victim != nil {
		victName := "someone"
		if n, ok := victim.(interface{ GetName() string }); ok {
			victName = n.GetName()
		}
		msg = strings.ReplaceAll(msg, "$N", victName)
		msg = strings.ReplaceAll(msg, "$E", testSubjectivePronoun(victim))
		msg = strings.ReplaceAll(msg, "$S", possessivePronoun(victim))
		msg = strings.ReplaceAll(msg, "$M", objectivePronoun(victim))
	}
	if obj != nil {
		shortDesc := "something"
		if sd, ok := obj.(interface{ GetShortDesc() string }); ok {
			shortDesc = sd.GetShortDesc()
		} else if n, ok := obj.(interface{ GetName() string }); ok {
			shortDesc = n.GetName()
		}
		msg = strings.ReplaceAll(msg, "$p", shortDesc)
	}
	return capitalizeActMessage(msg) + "\r\n"
}

func testSubjectivePronoun(ch any) string {
	if s, ok := ch.(interface{ GetSex() int }); ok {
		switch s.GetSex() {
		case 0:
			return "he"
		case 1:
			return "she"
		}
	}
	return "it"
}

func (w testActWorld) SpellActToChar(caster, victim, object any, format string) {
	sendToCaster(caster, testFormatAct(format, caster, victim, object))
}

func (w testActWorld) SpellObjectAct(caster, target any, message string, toRoom bool) {
	if !toRoom {
		sendToCaster(caster, testFormatAct(message, caster, nil, target))
		return
	}
	w.SpellActToRoom(caster, target, message, false)
}

func (w testActWorld) SpellActToVictim(caster, victim any, format string) bool {
	if s, ok := victim.(interface{ SendMessage(string) }); ok {
		s.SendMessage(testFormatAct(format, caster, victim, nil))
		return true
	}
	return false
}

func (w testActWorld) SpellActToRoom(actor, target any, format string, _ bool) bool {
	return w.deliverRoom(actor, target, nil, format)
}

func (w testActWorld) SpellActNotVict(actor, target any, format string, _ bool) bool {
	return w.deliverRoom(actor, target, target, format)
}

func (w testActWorld) deliverRoom(actor, target, exclude any, format string) bool {
	if w.roomChars == nil {
		return false
	}
	rg, ok := actor.(interface{ GetRoomVNum() int })
	if !ok {
		return false
	}
	msg := testFormatAct(format, actor, target, nil)
	for _, p := range w.roomChars(rg.GetRoomVNum()) {
		if selfTarget(actor, p) || (exclude != nil && selfTarget(exclude, p)) {
			continue
		}
		if !canReceiveAct(p) {
			continue
		}
		if s, ok := p.(interface{ SendMessage(string) }); ok {
			s.SendMessage(msg)
		}
	}
	return true
}

func (w testActWorld) SaySpellDeliver(ch, tch, tobj any, sameClass, otherClass, vict string) bool {
	if w.roomChars == nil {
		return false
	}
	rg, ok := ch.(interface{ GetRoomVNum() int })
	if !ok {
		return false
	}
	casterClass := -1
	if cc, ok := ch.(interface{ GetClass() int }); ok {
		casterClass = cc.GetClass()
	}
	for _, p := range w.roomChars(rg.GetRoomVNum()) {
		if selfTarget(ch, p) || (tch != nil && selfTarget(tch, p)) {
			continue
		}
		if !canReceiveAct(p) {
			continue
		}
		line := otherClass
		if rc, ok := p.(interface{ GetClass() int }); ok && rc.GetClass() == casterClass {
			line = sameClass
		}
		if s, ok := p.(interface{ SendMessage(string) }); ok {
			s.SendMessage(testFormatAct(line, ch, tch, tobj))
		}
	}
	if vict != "" && tch != nil && !selfTarget(ch, tch) && sameRoom(ch, tch) {
		w.SpellActToVictim(ch, tch, vict)
	}
	return true
}
