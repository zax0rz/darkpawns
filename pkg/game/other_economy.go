package game

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
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

	arg, _ = OneArgument(arg)
	amount := 0
	if arg != "" {
		for _, digit := range arg {
			if digit < '0' || digit > '9' {
				ch.SendMessage("How many coins do you wish to split with your group?\r\n")
				return true
			}
		}
		var err error
		amount, err = strconv.Atoi(arg)
		if err != nil {
			slog.Warn("split amount parse failed", "player", ch.Name, "arg", arg, "error", err)
			ch.SendMessage("How many coins do you wish to split with your group?\r\n")
			return true
		}
	}
	if amount <= 0 {
		ch.SendMessage("Sorry, you can't do that.\r\n")
		return true
	}
	grouped := ch.IsAffected(affGroup)
	if amount > ch.GetGold() {
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

	if num == 0 || !grouped {
		ch.SendMessage("With whom do you wish to share your gold?\r\n")
		return true
	}

	share := amount / num
	ch.mu.Lock()
	if amount > ch.Gold {
		ch.mu.Unlock()
		ch.SendMessage("You don't seem to have that much gold to split.\r\n")
		return true
	}
	ch.Gold -= share * (num - 1)
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
		p.Gold += share
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

	// C checks WEAR_HOLD with exact isname before the innate tattoo
	// branch, without CAN_SEE_OBJ (src/act.other.c:906-924). Resolve this
	// tattoo shadow here; other keywords retain their existing item lookup.
	var held *ObjectInstance
	if strings.EqualFold(itemArg, "tattoo") && ch.Equipment != nil {
		candidate, _ := ch.Equipment.GetItemInSlot(SlotHold)
		if candidate != nil && useHeldNameMatches(itemArg, candidate.GetKeywords()) {
			held = candidate
		}
	}
	// Handle tattoo use — from src/tattoo.c use_tattoo().
	if held == nil && strings.EqualFold(itemArg, "tattoo") {
		if ch.TatTimer != 0 {
			suffix := "s"
			if ch.TatTimer <= 1 {
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
				slog.Error("SpawnMob failed for tattoo skull", "error", err)
				break
			}
			AddFollowerQuietMob(mob, ch)
			// Apply charm affect (duration 20)
			mob.AddAffect(&engine.Affect{
				SpellID:   spells.SpellCharm,
				Type:      spells.SpellCharm, // backward compat
				Duration:  20,
				Magnitude: 0,
				Flags:     engine.AFFCharm, // C AFF_CHARM (structs.h:331), translated by AddAffect
			})
			w.tattooRoomAct(ch, mob, "$n's tattoo glows brightly for a second, and $N appears!")
			Act(w, true, ch, mob, nil, nil, "Your tattoo glows brightly for a second, and $N appears!", "", ToChar)
		case TattooEye:
			w.castTattoo(ch, spells.SpellGreatPercept)
		case TattooShip:
			w.castTattoo(ch, spells.SpellChangeDensity)
		case TattooAngel:
			w.castTattoo(ch, spells.SpellBless)
		default:
			ch.SendMessage("Your tattoo can't be 'use'd.\r\n")
			return true
		}
		ch.TatTimer = 24
		return true
	}

	// C do_use (act.other.c:897-936) searches equipped objects only, with
	// WEAR_HOLD is resolved above without a visibility gate; the remaining
	// slots use the existing visible lookup. Inventory and room objects are
	// not valid `use` targets.
	item := held
	if item == nil {
		item = w.FindEquippedVis(ch, itemArg)
	}

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
	// C mag_objectmagic selects its switch branches when equipd == WEAR_HOLD or
	// the item is neither staff nor wand (src/spell_parser.c:576-578); naming
	// the wielded item in the second `use` argument is what puts a held staff or
	// wand onto the tap/point branch. C's equipd scan uses `buf` whole (the
	// half_chop remainder, not its first word) against isname
	// (src/act.other.c:901; src/spell_parser.c:569-572), so a multi-word
	// remainder is passed through unchanged.
	if w.useEquipdHold(ch, useRest) {
		if itemType == ITEM_STAFF {
			w.useStaffTap(ch, item, useRest, spellNum, spellLevel)
		} else {
			w.useWandPoint(ch, item, useRest, spellNum, spellLevel)
		}
		return true
	}
	if itemType == ITEM_WAND {
		w.useWand(ch, item, useRest, spellNum, spellLevel)
	} else {
		w.useStaff(ch, item, useRest, spellNum, spellLevel)
	}

	return true
}

const defaultMagicItemLevel = 12 // C spells.h: DEFAULT_WAND_LVL/DEFAULT_STAFF_LVL

// useHeldNameMatches is C isname (src/handler.c:81-112), used by
// do_use's WEAR_HOLD probe. A completed argument ends at a non-alpha
// boundary; prefixes inside an alphabetic name do not match.
func useHeldNameMatches(arg, names string) bool {
	for n := 0; ; {
		for a := 0; ; a, n = a+1, n+1 {
			if a == len(arg) && (n == len(names) || !isASCIIAlpha(names[n])) {
				return true
			}
			if n == len(names) {
				return false
			}
			if a == len(arg) || names[n] == ' ' || !strings.EqualFold(arg[a:a+1], names[n:n+1]) {
				break
			}
		}
		for n < len(names) && isASCIIAlpha(names[n]) {
			n++
		}
		if n == len(names) {
			return false
		}
		n++
	}
}

const tattooRoomNoMagic = 7 // C structs.h: ROOM_NOMAGIC

// castTattoo enters C call_magic directly at DEFAULT_WAND_LVL/CAST_WAND
// (src/tattoo.c:66-74; spell_parser.c:419-439). Keep these direct-entry gates
// local: command-facing and native-special dispatch have separate callers.
func (w *World) castTattoo(ch *Player, spell int) {
	if room := w.GetRoomInWorld(ch.GetRoom()); room != nil && room.HasFlag(tattooRoomNoMagic) && ch.GetLevel() < combat.LVL_IMMORT {
		if ch.GetClass() == ClassPsionic || ch.GetClass() == ClassMystic {
			ch.SendMessage("Your will fades, disturbed by an unseen force.\r\n")
			w.tattooNoMagicRoom(ch, "$n's will fades, disturbed by an unseen force.")
		} else {
			ch.SendMessage("Your magic fizzles out and dies.\r\n")
			w.tattooNoMagicRoom(ch, "$n's magic fizzles out and dies.")
		}
		return
	}
	if ch.GetPosition() == combat.PosSitting {
		ch.SendMessage("You cannot do this sitting!\r\n")
		return
	}
	spells.CastFromTattoo(ch, spell, tattooSpellWorld{World: w, caster: ch})
}

// tattooCanSee applies C CAN_SEE, including world light, without the
// generic room-listing AFF_HIDE extension (src/utils.h:515-530).
func (w *World) tattooCanSee(observer, caster *Player) bool {
	if observer == caster {
		return true
	}
	if realLevel(observer, w) < caster.GetInvisLevel() {
		return false
	}
	lightOK := !observer.IsAffected(affBlind) && (!w.IsRoomDark(observer.GetRoom()) || observer.IsAffected(affInfravision))
	invisOK := !caster.IsAffected(affInvisible) || observer.IsAffected(affDetectInvisible)
	return observer.GetHolyLight() || (lightOK && invisOK)
}

func (w *World) tattooRoomAct(ch *Player, skull *MobInstance, format string) {
	actDeliver(w, false, ch, skull, nil, nil, format, "", ToRoom, func(to Actor, line string) {
		if player, ok := to.(*Player); ok && w.tattooCanSee(player, ch) {
			player.SendMessage(line)
		}
	})
}

// NOMAGIC uses act with hide_invisible FALSE; unseen casters are still
// announced as "someone" (spell_parser.c:424-432; utils.h:515-530).
func (w *World) tattooNoMagicRoom(ch *Player, format string) {
	// C's NOMAGIC path is a plain act(..., TO_ROOM) with hide FALSE
	// (spell_parser.c:419-432): one Act gives the room the same awake
	// audience ("awake gate"), PERS masking ("someone" to observers who
	// cannot see the caster — same bits as tattooCanSee), and capitalised
	// delivery the hand-rolled name-paste skipped (DP-1446).
	Act(w, false, ch, nil, nil, nil, format, "", ToRoom)
}

// tattooSpellWorld restricts benign self-spell room messages to C act's
// awake, visible audience (magic.c:1418-1421; utils.h:515-530). Only the tattoo dispatcher
// receives this adapter; the shared spell helper's other callers are unchanged.
type tattooSpellWorld struct {
	*World
	caster *Player
}

func (w tattooSpellWorld) ForEachPlayerInRoomInterface(room int, fn func(any)) {
	for _, player := range w.GetPlayersInRoom(room) {
		if sendOk(player, false) && w.tattooCanSee(player, w.caster) {
			fn(player)
		}
	}
}

// useWand is the castable-equipment branch of mag_objectmagic
// (src/spell_parser.c:754-783). In C, equipd is compared against the second
// half_chop token, so an ordinary `use wand target` takes this branch even when
// the wand is held; the source call path and its bytes are the authority here.
// takeMagicItemCharge atomically consumes one charge from a wand or staff,
// returning false when the item is out.
//
// The caller checked item.GetValue(2) and then decremented it, so two
// concurrent uses of a one-charge item both fired and the decrement was lost
// — the same charge bought two spells (CWE-362/CWE-367). ObjectInstance has no
// lock of its own and cannot gain one (house_save.go copies it by value), so
// charges serialize on the world lock, which already guards object state.
func (w *World) takeMagicItemCharge(item *ObjectInstance) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if item.GetValue(2) <= 0 {
		return false
	}
	item.SetValue(2, item.GetValue(2)-1)
	return true
}

func (w *World) useWand(ch *Player, item *ObjectInstance, targetArg string, spellNum, spellLevel int) {
	spellArgument, _ := OneArgument(targetArg)
	target, targetObj, found := w.resolveMagicItemTarget(ch, targetArg, spellNum)
	if !found {
		Act(nil, false, ch, nil, item, nil, "You can't use $p like that.", "", ToChar)
		return
	}

	if target != nil {
		if target == ch {
			Act(nil, false, ch, nil, item, nil,
				"Your $p bathes you in a blinding glow!", "", ToChar)
			roomAct := "$n's $p bathes $m in a blinding glow!"
			hideInvisible := false
			if item.Prototype != nil && item.Prototype.ActionDesc != "" {
				roomAct = item.Prototype.ActionDesc
			}
			Act(w, hideInvisible, ch, target, item, nil, roomAct, "", ToRoom)
		} else {
			Act(nil, false, ch, target, item, nil,
				"Your $p flares up with a blinding glow that surges toward $N!", "", ToChar)
			roomAct := "$n's $p flares up with a blinding glow that surges toward $N!"
			hideInvisible := true
			if item.Prototype != nil && item.Prototype.ActionDesc != "" {
				roomAct = item.Prototype.ActionDesc
				hideInvisible = false
			}
			Act(w, hideInvisible, ch, target, item, nil, roomAct, "", ToRoom)
		}
	} else {
		Act(nil, false, ch, nil, item, targetObj,
			"Your $p flares up with a blinding glow that surges toward $P!", "", ToChar)
		roomAct := "$n's $p flares up with a blinding glow that surges toward $P!"
		hideInvisible := true
		if item.Prototype != nil && item.Prototype.ActionDesc != "" {
			roomAct = item.Prototype.ActionDesc
			hideInvisible = false
		}
		Act(w, hideInvisible, ch, nil, item, targetObj, roomAct, "", ToRoom)
	}

	if !w.takeMagicItemCharge(item) {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	var objectTarget any
	if targetObj != nil {
		objectTarget = targetObj
	}
	spells.CallMagicWithArgument(ch, target, objectTarget, spellNum, level, spells.CastWand, w, spellArgument)
}

// useStaff is the castable-equipment staff branch of mag_objectmagic
// (src/spell_parser.c:785-817). The caster is excluded from the room fan-out;
// area/mass routines receive a nil character target once, matching C.
func (w *World) useStaff(ch *Player, item *ObjectInstance, targetArg string, spellNum, spellLevel int) {
	spellArgument, _ := OneArgument(targetArg)
	roomAct := "$n's $p sparks blindingly, bathing you in its glow."
	hideInvisible := true
	var targetObj *ObjectInstance
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		roomAct = item.Prototype.ActionDesc
		hideInvisible = false
		targetObj = w.resolveUseActionObjectTarget(ch, targetArg)
		if targetObj == nil {
			// C perform_act's CHECK_NULL(vict_obj, OBJS(...)) emits ACTNULL
			// for a missing $P argument (src/comm.c:2392-2435).
			roomAct = strings.ReplaceAll(roomAct, "$P", "<NULL>")
		}
	}
	Act(w, hideInvisible, ch, nil, item, targetObj, roomAct, "", ToRoom)
	Act(nil, false, ch, nil, item, nil,
		"Your $p radiates an ethereal glow that lights the room.", "", ToChar)

	if !w.takeMagicItemCharge(item) {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	si := spells.GetSpellInfo(spellNum)
	if si != nil && (si.HasRoutine(spells.RoutineMasses) || si.HasRoutine(spells.RoutineAreas)) {
		spells.CallMagicWithArgument(ch, nil, nil, spellNum, level, spells.CastStaff, w, spellArgument)
		return
	}
	for _, actor := range w.actChar(ch.GetRoomVNum()) {
		if actor == ch {
			continue
		}
		spells.CallMagicWithArgument(ch, actor, nil, spellNum, level, spells.CastStaff, w, spellArgument)
	}
}

// cWearOrder lists the equipment slots in C WEAR_* order (src/structs.h:390-411)
// so useEquipdHold can reproduce mag_objectmagic's `equipd = wear_pos + 1` scan
// (src/spell_parser.c:569-578).
var cWearOrder = []EquipmentSlot{
	SlotLight, SlotFingerR, SlotFingerL, SlotNeck1, SlotNeck2, SlotBody, SlotHead,
	SlotLegs, SlotFeet, SlotHands, SlotArms, SlotShield, SlotAbout, SlotWaist,
	SlotWristR, SlotWristL, SlotWield, SlotHold, SlotThrow, SlotAblegs, SlotFace, SlotHover,
}

// useEquipdHold reproduces mag_objectmagic's `equipd == WEAR_HOLD` test for a
// held staff or wand (src/spell_parser.c:569-578). C scans wear positions
// 0..NUM_WEARS-1 in order and keeps the LAST name match, so equipd=wear_pos+1
// equals WEAR_HOLD (17) exactly when the last matching slot is WEAR_WIELD (16):
// the second `use` argument names the wielded item. A keyword that also matches
// the held item (slot 17) yields 18 and takes the else branch instead.
func (w *World) useEquipdHold(ch *Player, buf string) bool {
	if buf == "" || ch.Equipment == nil {
		return false
	}
	last := -1
	for i, slot := range cWearOrder {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if ok && item != nil && useHeldNameMatches(buf, item.GetKeywords()) {
			last = i
		}
	}
	return last == 16 // WEAR_WIELD (src/structs.h:406)
}

// useStaffTap is mag_objectmagic's switch ITEM_STAFF tap branch
// (src/spell_parser.c:578-609), reached when useEquipdHold lands on WEAR_HOLD.
// Unlike the held-staff else branch (useStaff), it prints the caster line before
// the room act and fans a MAG_MASSES|MAG_AREAS staff out per room occupant
// instead of casting once.
func (w *World) useStaffTap(ch *Player, item *ObjectInstance, targetArg string, spellNum, spellLevel int) {
	spellArgument, _ := OneArgument(targetArg)
	Act(nil, false, ch, nil, item, nil, "You tap $p three times on the ground.", "", ToChar)
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		Act(w, false, ch, nil, item, nil, item.Prototype.ActionDesc, "", ToRoom)
	} else {
		Act(w, false, ch, nil, item, nil, "$n taps $p three times on the ground.", "", ToRoom)
	}
	if !w.takeMagicItemCharge(item) {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	for _, actor := range w.actChar(ch.GetRoomVNum()) {
		if actor == ch {
			continue
		}
		spells.CallMagicWithArgument(ch, actor, nil, spellNum, level, spells.CastStaff, w, spellArgument)
	}
}

// useWandPoint is mag_objectmagic's switch ITEM_WAND point branch
// (src/spell_parser.c:610-659), reached when useEquipdHold lands on WEAR_HOLD.
// It prints the caster line before the room act and, with no valid target,
// prints "At what should $p be pointed?" instead of the held-wand else branch's
// "You can't use $p like that.".
func (w *World) useWandPoint(ch *Player, item *ObjectInstance, targetArg string, spellNum, spellLevel int) {
	spellArgument, _ := OneArgument(targetArg)
	target, targetObj, found := w.resolveMagicItemTarget(ch, targetArg, spellNum)
	if !found {
		Act(nil, false, ch, nil, item, nil, "At what should $p be pointed?", "", ToChar)
		return
	}
	if target != nil {
		if target == ch {
			// C's tch == ch branch always prints the plain room act; only the
			// at-someone-else branches substitute obj->action_description
			// (src/spell_parser.c:614-624).
			Act(nil, false, ch, nil, item, nil, "You point $p at yourself.", "", ToChar)
			Act(w, false, ch, target, item, nil, "$n points $p at $mself.", "", ToRoom)
		} else {
			Act(nil, false, ch, target, item, nil, "You point $p at $N.", "", ToChar)
			roomAct := "$n points $p at $N."
			hideInvisible := true
			if item.Prototype != nil && item.Prototype.ActionDesc != "" {
				roomAct = item.Prototype.ActionDesc
				hideInvisible = false
			}
			Act(w, hideInvisible, ch, target, item, nil, roomAct, "", ToRoom)
		}
	} else {
		Act(nil, false, ch, nil, item, targetObj, "You point $p at $P.", "", ToChar)
		roomAct := "$n points $p at $P."
		hideInvisible := true
		if item.Prototype != nil && item.Prototype.ActionDesc != "" {
			roomAct = item.Prototype.ActionDesc
			hideInvisible = false
		}
		Act(w, hideInvisible, ch, nil, item, targetObj, roomAct, "", ToRoom)
	}
	if !w.takeMagicItemCharge(item) {
		Act(nil, false, ch, nil, item, nil, "It seems powerless.", "", ToChar)
		Act(w, false, ch, nil, item, nil, "Nothing seems to happen.", "", ToRoom)
		return
	}
	ch.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)
	level := spellLevel
	if level == 0 {
		level = defaultMagicItemLevel
	}
	var objectTarget any
	if targetObj != nil {
		objectTarget = targetObj
	}
	spells.CallMagicWithArgument(ch, target, objectTarget, spellNum, level, spells.CastWand, w, spellArgument)
}

// resolveUseActionObjectTarget mirrors mag_objectmagic's generic_find lookup
// for the object argument used only by a staff's action description
// (src/spell_parser.c:560-565,785-790). Character matches take precedence;
// otherwise C searches equipment, inventory, then room before using the
// object as $P in the room act (src/handler.c:1537-1556; DP-1518).
func (w *World) resolveUseActionObjectTarget(ch *Player, targetArg string) *ObjectInstance {
	targetArg, _ = halfChop(targetArg)
	if targetArg == "" {
		return nil
	}
	if _, ok := w.ResolveCharInRoom(ch, targetArg); ok {
		return nil
	}
	object, _ := w.GenericFindObject(ch, targetArg)
	return object
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
	if !si.HasTarget(spells.TarObjInv) && !si.HasTarget(spells.TarObjRoom) &&
		!si.HasTarget(spells.TarObjEquip) && !si.HasTarget(spells.TarObjWorld) {
		return nil, nil, false
	}
	// C generic_find searches equipment, inventory, and room in that order
	// before mag_objectmagic checks whether the spell accepts any object target
	// (src/spell_parser.c:560-565,628-637; src/handler.c:1537-1556; DP-1518).
	// The individual TAR_OBJ_* bits do not restrict where generic_find may
	// find that object.
	if object, _ := w.GenericFindObject(ch, targetArg); object != nil {
		return nil, object, true
	}
	return nil, nil, false
}

// DoUse is the exported session-level entrypoint for item usage.
func (w *World) DoUse(ch *Player, arg string) bool {
	return w.doUse(ch, nil, "use", arg)
}
