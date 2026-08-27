package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// ---------------------------------------------------------------------------
// do_split — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSplit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		// C do_split (act.other.c): is_number("") is true, so amount = atoi("")
		// = 0, which hits the amount<=0 branch → "Sorry, you can't do that."
		ch.SendMessage("Sorry, you can't do that.\r\n")
		return true
	}

	amount := 0
	if _, err := fmt.Sscanf(arg, "%d", &amount); err != nil {
		ch.SendMessage("That doesn't look like a number.\r\n")
		slog.Warn("split parse failed", "player", ch.Name, "arg", arg, "error", err)
		return true
	}
	if amount <= 0 {
		ch.SendMessage("Sorry, you can't do that.\r\n")
		return true
	}
	ch.mu.Lock()
	if amount > ch.GetGold() {
		ch.mu.Unlock()
		ch.SendMessage("You don't seem to have that much gold to split.\r\n")
		return true
	}

	leaderName := ch.GetFollowing()
	if leaderName == "" {
		leaderName = ch.Name
	}

	// Count group members in same room
	num := 0
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.GetFollowing() != leaderName && p.Name != leaderName {
			continue
		}
		if p.IsAffected(affGroup) {
			num++
		}
	}

	if num <= 1 || !ch.IsAffected(affGroup) {
		ch.mu.Unlock()
		ch.SendMessage("With whom do you wish to share your gold?\r\n")
		return true
	}

	share := amount / num
	ch.SetGold(ch.GetGold() - share*(num-1))
	ch.mu.Unlock()

	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.GetFollowing() != leaderName && p.Name != leaderName {
			continue
		}
		if !p.IsAffected(affGroup) || p.Name == ch.Name {
			continue
		}
		p.mu.Lock()
		p.SetGold(p.GetGold() + share)
		p.mu.Unlock()
		p.SendMessage(fmt.Sprintf("%s splits %d coins; you receive %d.\r\n", ch.Name, amount, share))
	}

	ch.SendMessage(fmt.Sprintf("You split %d coins among %d members -- %d coins each.\r\n", amount, num, share))
	return true
}

// ---------------------------------------------------------------------------
// do_use — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doUse(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// C do_use parses with half_chop (interpreter.c): the first token is
	// lowercased; fill words are not skipped.
	itemArg, useRest := halfChop(arg)

	if itemArg == "" {
		ch.SendMessage(fmt.Sprintf("What do you want to %s?\r\n", cmd))
		return true
	}

	// Handle tattoo use — from src/tattoo.c use_tattoo()
	if strings.EqualFold(itemArg, "tattoo") {
		if ch.TatTimer > 0 {
			suffix := "s"
			if ch.TatTimer == 1 {
				suffix = ""
			}
			ch.SendMessage(fmt.Sprintf("You can't use your tattoo's magick for %d more hour%s.\r\n",
				ch.TatTimer, suffix))
			return true
		}
		switch ch.Tattoo {
		case TattooNone:
			ch.SendMessage("You don't have a tattoo.\r\n")
		case TattooSkull:
			// Summon mob vnum 9 (skull), charm it, make it follow
			mob, err := w.SpawnMob(9, ch.GetRoom())
			if err != nil {
				ch.SendMessage("Your tattoo fizzles...\r\n")
				break
			}
			if err := w.SetFollower(mob.GetName(), ch.GetName(), true); err != nil {
				slog.Error("SetFollower failed for tattoo skull", "mob", mob.GetName(), "leader", ch.GetName(), "error", err)
			}
			// Apply charm affect (duration 20)
			mob.AddAffect(&engine.Affect{
				SpellID:   spells.SpellCharm,
				Type:      spells.SpellCharm, // backward compat
				Duration:  20,
				Magnitude: 0,
				Flags:     1 << 3, // AFF_CHARM
			})
			w.roomMessage(ch.GetRoom(), fmt.Sprintf("%s's tattoo glows brightly for a second, and %s appears!", ch.Name, mob.Prototype.ShortDesc))
			ch.SendMessage(fmt.Sprintf("Your tattoo glows brightly for a second, and %s appears!\r\n", mob.Prototype.ShortDesc))
		case TattooEye:
			spells.Cast(ch, ch, spells.SpellGreatPercept, ch.GetLevel(), w)
		case TattooShip:
			spells.Cast(ch, ch, spells.SpellChangeDensity, ch.GetLevel(), w)
		case TattooAngel:
			spells.Cast(ch, ch, spells.SpellBless, ch.GetLevel(), w)
		default:
			ch.SendMessage("Your tattoo can't be 'use'd.\r\n")
			return true
		}
		ch.TatTimer = 24
		return true
	}

	// C do_use (act.other.c:897-936) searches equipped objects only, with
	// WEAR_HOLD checked before the remaining wear positions. FindEquippedVis
	// preserves that lookup and CAN_SEE_OBJ gate; inventory and room objects
	// are not valid `use` targets.
	item := w.FindEquippedVis(ch, itemArg)

	if item == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(itemArg), itemArg))
		return true
	}

	itemType := item.GetTypeFlag()
	if itemType != ITEM_WAND && itemType != ITEM_STAFF {
		ch.SendMessage("You can't seem to figure out how to use it.\r\nTry holding it.(?)\r\n")
		return true
	}

	spellLevel := item.GetValue(0)
	spellNum := item.GetValue(3)
	if itemType == ITEM_WAND {
		w.useWand(ch, item, useRest, spellNum, spellLevel)
	} else {
		w.useStaff(ch, item, spellNum, spellLevel)
	}

	return true
}

const defaultMagicItemLevel = 12 // C spells.h: DEFAULT_WAND_LVL/DEFAULT_STAFF_LVL

// useWand is the castable-equipment branch of mag_objectmagic
// (src/spell_parser.c:754-783). In C, equipd is compared against the second
// half_chop token, so an ordinary `use wand target` takes this branch even when
// the wand is held; the source call path and its bytes are the authority here.
func (w *World) useWand(ch *Player, item *ObjectInstance, targetArg string, spellNum, spellLevel int) {
	target, targetObj, found := w.resolveMagicItemTarget(ch, targetArg, spellNum)
	if !found {
		Act(nil, false, ch, nil, item, nil, "You can't use $p like that.", "", ToChar)
		return
	}

	if target != nil {
		if target == ch {
			Act(nil, false, ch, nil, item, nil,
				"Your $p bathes you in a blinding glow!", "", ToChar)
			Act(w, false, ch, nil, item, nil,
				"$n's $p bathes $m in a blinding glow!", "", ToRoom)
		} else {
			Act(nil, false, ch, target, item, nil,
				"Your $p flares up with a blinding glow that surges toward $N!", "", ToChar)
			Act(w, true, ch, target, item, nil,
				"$n's $p flares up with a blinding glow that surges toward $N!", "", ToRoom)
		}
	} else {
		Act(nil, false, ch, nil, item, targetObj,
			"Your $p flares up with a blinding glow that surges toward $P!", "", ToChar)
		Act(w, true, ch, nil, item, targetObj,
			"$n's $p flares up with a blinding glow that surges toward $P!", "", ToRoom)
	}

	if item.GetValue(2) <= 0 {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	item.SetValue(2, item.GetValue(2)-1)
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	var objectTarget interface{}
	if targetObj != nil {
		objectTarget = targetObj
	}
	spells.CallMagic(ch, target, objectTarget, spellNum, level, spells.CastWand, w)
}

// useStaff is the castable-equipment staff branch of mag_objectmagic
// (src/spell_parser.c:785-817). The caster is excluded from the room fan-out;
// area/mass routines receive a nil character target once, matching C.
func (w *World) useStaff(ch *Player, item *ObjectInstance, spellNum, spellLevel int) {
	Act(w, true, ch, nil, item, nil,
		"$n's $p sparks blindingly, bathing you in its glow.", "", ToRoom)
	Act(nil, false, ch, nil, item, nil,
		"Your $p radiates an ethereal glow that lights the room.", "", ToChar)

	if item.GetValue(2) <= 0 {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	item.SetValue(2, item.GetValue(2)-1)
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	si := spells.GetSpellInfo(spellNum)
	if si != nil && (si.HasRoutine(spells.RoutineMasses) || si.HasRoutine(spells.RoutineAreas)) {
		spells.CallMagic(ch, nil, nil, spellNum, level, spells.CastStaff, w)
		return
	}
	for _, actor := range w.actChar(ch.GetRoomVNum()) {
		if actor == ch {
			continue
		}
		spells.CallMagic(ch, actor, nil, spellNum, level, spells.CastStaff, w)
	}
}

// resolveMagicItemTarget mirrors generic_find's character-first target lookup
// for wand use, followed by the object scopes allowed by the spell template.
func (w *World) resolveMagicItemTarget(ch *Player, targetArg string, spellNum int) (Actor, *ObjectInstance, bool) {
	targetArg, _ = halfChop(targetArg)
	if targetArg == "" {
		return nil, nil, false
	}
	if target, ok := w.ResolveCharInRoom(ch, targetArg); ok {
		actor := asActor(target.Combatant)
		if actor != nil {
			return actor, nil, true
		}
	}
	si := spells.GetSpellInfo(spellNum)
	if si == nil {
		return nil, nil, false
	}
	if si.HasTarget(spells.TarObjInv) {
		if obj, ok := w.ResolveObjectInInventory(ch, targetArg); ok {
			return nil, obj, true
		}
	}
	if si.HasTarget(spells.TarObjRoom) {
		if obj, ok := w.ResolveObjectInRoom(ch, targetArg); ok {
			return nil, obj, true
		}
	}
	if si.HasTarget(spells.TarObjEquip) {
		if obj, ok := w.ResolveObjectInEquipment(ch, targetArg); ok {
			return nil, obj, true
		}
	}
	if si.HasTarget(spells.TarObjWorld) {
		if obj, ok := w.ResolveObjectWorld(ch, targetArg); ok {
			return nil, obj, true
		}
	}
	return nil, nil, false
}

// DoUse is the exported session-level entrypoint for item usage.
func (w *World) DoUse(ch *Player, arg string) bool {
	return w.doUse(ch, nil, "use", arg)
}
