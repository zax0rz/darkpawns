package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoMold(ch *Player, objName, newName, newDesc string) SkillResult {
	var obj *ObjectInstance
	var found bool
	if ch != nil && ch.Inventory != nil {
		// C get_obj_in_list_vis scans only ch->carrying and matches the
		// object's keyword list, not its display description or equipment
		// (handler.c:1328-1347; new_cmds.c:69).
		obj, found = resolveVisibleObject(ch, objName, ch.Inventory.FindItems(""), true)
	}
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have one of those."}
	}

	name := strings.ToLower(obj.GetKeywords())
	if !strings.Contains(name, "clay") && !strings.Contains(name, "playdough") && !strings.Contains(name, "halo") {
		return SkillResult{Success: false, MessageToCh: "You do not have anything to mold!"}
	}

	if newName == "" || newDesc == "" {
		return SkillResult{Success: false, MessageToCh: "You must specify a name and a description."}
	}

	// C rewrites the live object's keyword list, short description, and room
	// description (new_cmds.c:88-99). Keep the typed mold fields as well for
	// existing save-state compatibility, but make all player-facing object
	// accessors observe the C mutations immediately.
	moldedName := fmt.Sprintf("%s _%s_ mold_item", newName, ch.GetName())
	obj.Runtime.Name = moldedName
	obj.Runtime.Keywords = moldedName
	obj.Runtime.ShortDesc = newDesc
	obj.Runtime.LongDesc = CapitalizeSentence(newDesc) + " has been left here."
	obj.Runtime.MoldName = newName
	obj.Runtime.MoldDesc = newDesc

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("The material magically hardens when you create %s.", newDesc),
	}
}

// DoBehead implements do_behead() from new_cmds.c.
func DoBehead(ch *Player, targetName string, world *World) SkillResult {
	// C's one_argument() runs before both the character lookup and the
	// position/no-argument gates.  The command wrapper also passes the first
	// token, but keeping the parser here preserves the do_behead call path for
	// direct callers and tests.
	targetName, _ = OneArgument(targetName)
	target, found := world.ResolveCharInRoom(ch, targetName)

	if ch.GetPosition() == combat.PosFighting {
		return SkillResult{Success: false, MessageToCh: "You're a little busy for that!"}
	}
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "Behead who?"}
	}
	if found && target.Combatant == ch {
		return SkillResult{Success: false, MessageToCh: "This MUD doesn't support self-mutilation!"}
	}
	if found {
		return SkillResult{Success: false, MessageToCh: "You kill it first and THEN you behead it!"}
	}

	// C resolves any visible room object by the argument, then accepts only
	// ITEM_CONTAINER objects whose value[3] marks them as corpses.  Keywords do
	// not participate in this predicate (new_cmds.c:251-262).
	room := world.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil {
		return SkillResult{MessageToCh: "You are in a void.\r\n"}
	}

	obj, found := world.ResolveObjectInRoom(ch, targetName)
	if !found {
		return SkillResult{Success: false, MessageToCh: fmt.Sprintf("You can't seem to find a %s to behead!", targetName)}
	}

	if strings.Contains(obj.GetKeywords(), "headless") {
		return SkillResult{Success: false, MessageToCh: "You can't behead something without a head!"}
	}

	if obj.GetTypeFlag() != ITEM_CONTAINER || obj.GetValue(3) == 0 {
		return SkillResult{Success: false, MessageToCh: "You can't behead that!"}
	}

	// Determine weapon type for messaging.  C only examines WEAR_WIELD and
	// treats value[3] == 3 as a slash weapon.
	wielded := false
	slashWeapon := false
	if ch.Equipment != nil {
		if weapon, ok := ch.Equipment.GetItemInSlot(SlotWield); ok {
			wielded = true
			slashWeapon = weapon.GetValue(3) == 3
		}
	}

	return performBehead(world, ch, obj, wielded, slashWeapon,
		func(headObj *ObjectInstance) bool { return world.canTakeObj(ch, headObj) },
		func(headObj *ObjectInstance) error { return world.MoveObjectToPlayerInventory(headObj, ch) },
	)
}

// performBehead contains the shared object transformation used by do_behead
// and SPECIAL(brain_eater)'s NPC call to do_behead.  C's command accepts both
// players and mobiles; keeping the mutation path shared prevents the NPC
// special from silently skipping the head/corpse/content lifecycle.
func performBehead(world *World, ch Actor, obj *ObjectInstance, wielded, slashWeapon bool, canTake func(*ObjectInstance) bool, moveHead func(*ObjectInstance) error) SkillResult {
	if world == nil || ch == nil || obj == nil {
		return SkillResult{Success: false}
	}

	var msgToCh, msgToRoom string
	if wielded && slashWeapon {
		msgToCh = fmt.Sprintf("You behead %s!", obj.GetShortDesc())
		msgToRoom = fmt.Sprintf("%s beheads %s!", ch.GetName(), obj.GetShortDesc())
	} else {
		msgToCh = fmt.Sprintf("You rip the head off %s with your bare hands!", obj.GetShortDesc())
		if ch.IsNPC() {
			msgToRoom = fmt.Sprintf("%s rips the head off %s!", ch.GetName(), obj.GetShortDesc())
		} else {
			msgToRoom = fmt.Sprintf("%s rips the head off %s with %s bare hands!", ch.GetName(), obj.GetShortDesc(), genderPronoun(ch.GetSex()))
		}
	}

	originalName := obj.GetKeywords()
	originalShortDesc := obj.GetShortDesc()

	// C reads proto 16, rewrites its name/short/long descriptions, and then
	// uses the ordinary can_take_obj gate before placing the head.
	headObj, err := world.SpawnObject(16, -1)
	if err != nil {
		slog.Error("behead head prototype missing", "error", err)
		return SkillResult{Success: false, MessageToCh: "You can't behead that!"}
	}
	headVerb := "ripped"
	if slashWeapon {
		headVerb = "hacked"
	}
	headShortDesc := fmt.Sprintf("a bloody head %s from %s", headVerb, originalShortDesc)
	headObj.Runtime.Keywords = "head"
	headObj.Runtime.ShortDesc = headShortDesc
	headObj.Runtime.LongDesc = CapitalizeSentence(headShortDesc) + " has been left here."
	if canTake(headObj) {
		if err := moveHead(headObj); err != nil {
			slog.Error("behead head inventory placement failed", "error", err)
			if moveErr := world.MoveObjectToRoomFront(headObj, ch.GetRoom()); moveErr != nil {
				slog.Error("behead head room placement failed", "error", moveErr)
			}
		}
	} else if err := world.MoveObjectToRoomFront(headObj, ch.GetRoom()); err != nil {
		slog.Error("behead head room placement failed", "error", err)
	}

	// C reads proto 17, sets the corpse marker values/timer, appends the
	// original object's keywords with "headless beheaded", transfers contents,
	// then extracts the original object.
	headlessCorpseObj, err := world.SpawnObject(17, -1)
	if err != nil {
		slog.Error("behead corpse prototype missing", "error", err)
		return SkillResult{Success: false, MessageToCh: "You can't behead that!"}
	}
	headlessCorpseObj.SetValue(0, 0)
	headlessCorpseObj.SetValue(3, 1)
	headlessCorpseObj.Timer = MaxNPCCorpseTime
	headlessCorpseObj.IsCorpse = true
	headlessCorpseObj.Runtime.Keywords = originalName + " headless beheaded"
	if err := world.MoveObjectToRoomFront(headlessCorpseObj, ch.GetRoom()); err != nil {
		slog.Error("behead corpse room placement failed", "error", err)
	}
	for len(obj.Contains) > 0 {
		content := obj.Contains[0]
		if err := world.MoveObjectToContainer(content, headlessCorpseObj); err != nil {
			slog.Error("behead content transfer failed", "obj_vnum", content.GetVNum(), "error", err)
			break
		}
	}
	world.ExtractObject(obj, ch.GetRoom())

	return SkillResult{
		Success:       true,
		MessageToCh:   msgToCh,
		MessageToRoom: msgToRoom,
	}
}

// DoBearhug implements do_bearhug() — bare-handed squeeze attack.
func DoBearhug(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillBearhug) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters.\r\n"}
	}

	// C has no movement-point gate or expenditure in do_bearhug.
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Bear hug who?\r\n"}
	}

	// C rejects a mortal attempt against a non-NPC immortal before the
	// self-target and wielded-weapon checks (new_cmds.c:504-508).
	if !target.IsNPC() && target.GetLevel() >= LVL_IMMORT {
		return SkillResult{Success: false, MessageToCh: "The gods reject your impunity.\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today...\r\n"}
	}

	if ch.Equipment != nil {
		if _, wielded := ch.Equipment.GetItemInSlot(SlotWield); wielded {
			return SkillResult{Success: false, MessageToCh: "You need to be bare handed to get a good grip.\r\n"}
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 150) // 1-150; 101+ is complete failure

	// Sleeping targets and immortal casters force the C failure percent. The
	// later MOB_NOBASH assignment intentionally has the same order as C.
	if target.GetPosition() <= combat.PosSleeping || ch.GetLevel() > LVL_IMMORT {
		percent = 101
	}
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
		percent = 101
	}

	prob := ch.GetSkill(SkillBearhug)

	if percent > prob {
		return SkillResult{
			Success:      false,
			Damage:       0,
			SkillMsgType: SkillBearhugNum,
			StartCombat:  true,
			WaitCh:       2,
		}
	}

	dam := ch.GetLevel() + (ch.GetLevel() / 2) // level * 1.5

	return SkillResult{
		Success:         true,
		Damage:          dam,
		SkillMsgType:    SkillBearhugNum,
		StartCombat:     true,
		WaitCh:          2,
		DeferredImprove: []string{SkillBearhug},
	}
}

// DoSlug implements do_slug() — punch attack.
func DoSlug(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSlug) == 0 {
		return SkillResult{Success: false, MessageToCh: SkillUnknownMsg[SkillSlug]}
	}

	// C resolves the target before checking the self, weapon, and mounted
	// gates (new_cmds.c:837-848). Keep those checks here as a second boundary
	// for direct callers; CmdSlug owns the visible-room lookup and fallback.
	if target != nil && target.GetName() == ch.GetName() {
		return SkillResult{Success: false, MessageToCh: "You curl up your fist and slug yourself in the nose! Ouch!"}
	}

	if ch.Equipment != nil {
		if _, wielded := ch.Equipment.GetItemInSlot(SlotWield); wielded {
			return SkillResult{Success: false, MessageToCh: "You can't make a fist while wielding a weapon!"}
		}
	}

	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillSlug)

	if percent > prob {
		return SkillResult{
			Success:         false,
			Damage:          0,
			SkillMsgType:    SkillSlugNum,
			DamageSkill:     SkillSlug,
			StartCombat:     true,
			WaitCh:          2,
			DeferredImprove: []string{SkillSlug},
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	dam := (ch.GetLevel() * dprng.Number(1, 4)) / 2
	return SkillResult{
		Success:      true,
		Damage:       dam,
		SkillMsgType: SkillSlugNum,
		DamageSkill:  SkillSlug,
		StartCombat:  true,
		WaitCh:       2,
	}
}

// DoSmackheads implements do_smackheads() — grab two NPCs and smack them together.
func DoSmackheads(ch *Player, victim1Name, victim2Name string, world *World) SkillResult {
	if ch.GetSkill(SkillSmackheads) == 0 {
		return SkillResult{Success: false, MessageToCh: "The only heads you're gonna smack are yours and Rosie's.\r\n"}
	}

	if victim1Name == victim2Name {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	vill, _, found1 := FindTargetInRoom(world, ch.GetRoomVNum(), victim1Name, ch)
	vil2, _, found2 := FindTargetInRoom(world, ch.GetRoomVNum(), victim2Name, ch)
	if !found1 || !found2 {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	// Check we're not targeting ourselves
	if vill.GetName() == ch.Name || vil2.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "We call that 'headbutt' around here, son...\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You need your hands free to smack some heads!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillSmackheads)

	if percent > prob {
		// Failure — victims duck
		msgToCh := fmt.Sprintf("%s and %s slip out of your hands!", vill.GetName(), vil2.GetName())
		return SkillResult{
			Success:       true,
			MessageToCh:   msgToCh + "\r\n",
			MessageToRoom: fmt.Sprintf("%s and %s duck as %s lunges at them!\r\n", vill.GetName(), vil2.GetName(), ch.Name),
		}
	}

	// Success — smack them together
	dam := 3 * ch.GetLevel()
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   fmt.Sprintf("You grab the heads of %s and %s and bang them together with a sickening *SMACK*.\r\n", vill.GetName(), vil2.GetName()),
		MessageToRoom: fmt.Sprintf("%s grabs the heads of %s and %s and bangs them together with a sickening *SMACK*.\r\n", ch.Name, vill.GetName(), vil2.GetName()),
	}
}

// DoBite implements do_bite() from src/new_cmds.c:1199-1300.
func DoBite(ch *Player, target combat.Combatant) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Bite who?!"}
	}
	if target.GetName() == ch.GetName() {
		return SkillResult{Success: false, MessageToCh: "You bite your tongue and say nothing."}
	}

	chPronouns := GetPronouns(ch.GetName(), ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	actResult := func(chMessage, victMessage, roomMessage string) SkillResult {
		return SkillResult{
			Success:       true,
			MessageToCh:   ActMessage(chMessage, chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage(victMessage, chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage(roomMessage, chPronouns, &victPronouns, ""),
		}
	}

	// Players without a supernatural PLR flag (and NPC callers in C) only
	// produce the ordinary love-bite act trio; they do not deal damage.
	flags := ch.GetFlags()
	werewolf := flags&(1<<PlrWerewolf) != 0
	vampire := flags&(1<<PlrVampire) != 0
	if (!werewolf && !vampire) || ch.IsNPC() {
		return actResult(
			"You give $N a love bite.",
			"$n tries to give you a little love bite.",
			"$n gives $N a love bite.",
		)
	}

	// A flagged player must currently be transformed before bite can reach its
	// supernatural branch. C checks werewolf first when both flags are present.
	if !ch.IsAffected(affWerewolf) && !ch.IsAffected(affVampire) {
		return SkillResult{Success: false, MessageToCh: "You must be transformed to bite!"}
	}

	if target.GetLevel() >= LVL_IMMORT && target.GetLevel() > ch.GetLevel() {
		return SkillResult{Success: false, MessageToCh: "Yeah, right."}
	}

	if victim, ok := target.(*Player); ok {
		victimFlags := victim.GetFlags()
		if victimFlags&(1<<PlrWerewolf) != 0 || victimFlags&(1<<PlrVampire) != 0 {
			return SkillResult{Success: false, MessageToCh: "Your victim is already a creature of the night!"}
		}
	}

	dam := ch.GetLevel()
	if dam > 15 {
		dam = 15
	}

	if ch.IsAffected(affWerewolf) {
		result := actResult(
			"You rip the flesh of $N, and blood pours over your lips!",
			"$n rips your flesh, leaving you bleeding and dazed!",
			"$n rips the flesh of $N, growling with bloodlust!",
		)
		result.Damage = dam
		result.DamageSkill = SkillBite
		result.SkillMsgType = SkillBiteNum
		result.SkillMsgAfterDamage = true
		result.StartCombat = true
		result.WaitCh = 2
		return result
	}

	// Vampire bites draw once to decide whether the bite is sloppy or fighting.
	// C's number(0, GET_LEVEL(ch)/2) is evaluated only on this branch.
	// #nosec G404 — game RNG, not cryptographic
	if dprng.Number(0, ch.GetLevel()/2) == 0 {
		result := actResult(
			"Your fangs sink into the soft flesh of $N, and $S blood pours over your lips.",
			"$n's fangs sink into your flesh, leaving you bleeding and dazed!",
			"$n sinks $s fangs into the flesh of $N, feeding off $S blood!",
		)
		result.MessageToRoom += "\r\n" + ActMessage("$N screams in agony!", chPronouns, &victPronouns, "")
		result.Damage = dam
		result.DamageSkill = SkillBite
		result.SkillMsgType = SkillBiteNum
		result.SkillMsgAfterDamage = true
		result.StartCombat = true
		result.WaitCh = 2
		feedFromBite(ch, target.GetLevel())
		return result
	}

	feedFromBite(ch, target.GetLevel())
	return actResult(
		"Your fangs sink into the soft flesh of $N, and $S blood pours over your lips.",
		"$n's fangs sink into your flesh, leaving you bleeding and dazed!",
		"$n sinks $s fangs into the flesh of $N, feeding off $S blood!",
	)
}

// feedFromBite ports the direct GET_COND updates in do_bite's vampire branch.
// These writes intentionally do not clamp: the C code adds the victim's level
// directly after testing the pre-update value against 40.
func feedFromBite(ch *Player, victimLevel int) {
	if full := ch.GetCondition(CondFull); full < 40 && full >= 0 {
		ch.SetCondition(CondFull, full+victimLevel)
	}
	if thirst := ch.GetCondition(CondThirst); thirst < 40 && thirst >= 0 {
		ch.SetCondition(CondThirst, thirst+victimLevel)
	}
}

// DoTag implements do_tag() — tag someone as "it".
func DoTag(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "Tag who?\r\n"}
	}

	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		return SkillResult{Success: false, MessageToCh: "They don't seem to be here.\r\n"}
	}

	// Self-tag starts the game
	if target.GetName() == ch.Name {
		return SkillResult{Success: true, MessageToCh: "Let the game begin!\r\n"}
	}

	return SkillResult{
		Success:       true,
		MessageToCh:   fmt.Sprintf("You tap %s and scream, 'TAG! You're it!'\r\n", target.GetName()),
		MessageToVict: fmt.Sprintf("%s taps you and screams, 'TAG! You're it!'\r\n", ch.Name),
		MessageToRoom: fmt.Sprintf("%s taps %s and screams, 'TAG! You're it!'\r\n", ch.Name, target.GetName()),
	}
}

// DoPoint implements do_point() from new_cmds.c: point at a visible character,
// visible room object, direction, or the room itself. targetName has already
// passed through C's one_argument at the session boundary.
func DoPoint(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return pointResult("You point around the room.", fmt.Sprintf("%s points around the room.", ch.Name))
	}

	if target, found := world.ResolveCharInRoom(ch, targetName); found {
		if target.Combatant == ch {
			return pointResult("You point at yourself.", fmt.Sprintf("%s points at %s.", ch.Name, himHer(ch.GetSex())))
		}
		return pointCharacterResult(ch, target.Combatant)
	}

	if object, found := world.ResolveObjectInRoom(ch, targetName); found {
		return pointObjectResult(ch, object)
	}

	if direction := pointDirection(targetName); direction != "" {
		return pointResult("You point "+direction+".", fmt.Sprintf("%s points %s.", ch.Name, direction))
	}

	return pointResult("You point around the room.", fmt.Sprintf("%s points around the room.", ch.Name))
}

func pointResult(toCh, toRoom string) SkillResult {
	return SkillResult{Success: true, MessageToCh: toCh, MessageToRoom: toRoom}
}

func pointCharacterResult(ch *Player, target combat.Combatant) SkillResult {
	targetName := target.GetName()
	weapon := pointWeapon(ch)
	if weapon == nil {
		return SkillResult{
			Success:       true,
			MessageToCh:   fmt.Sprintf("You point at %s.", targetName),
			MessageToVict: fmt.Sprintf("%s points at you.", ch.Name),
			MessageToRoom: fmt.Sprintf("%s points at %s.", ch.Name, targetName),
		}
	}
	weaponName := weapon.GetShortDesc()
	return SkillResult{
		Success:       true,
		MessageToCh:   fmt.Sprintf("You point %s at %s.", weaponName, targetName),
		MessageToVict: fmt.Sprintf("%s points %s at you.", ch.Name, weaponName),
		MessageToRoom: fmt.Sprintf("%s points %s at %s.", ch.Name, weaponName, targetName),
	}
}

func pointObjectResult(ch *Player, target *ObjectInstance) SkillResult {
	targetName := target.GetShortDesc()
	if weapon := pointWeapon(ch); weapon != nil {
		weaponName := weapon.GetShortDesc()
		return pointResult(
			fmt.Sprintf("You point %s at %s.", weaponName, targetName),
			fmt.Sprintf("%s points %s at %s.", ch.Name, weaponName, targetName),
		)
	}
	return pointResult(
		fmt.Sprintf("You point at %s.", targetName),
		fmt.Sprintf("%s points at %s.", ch.Name, targetName),
	)
}

func pointWeapon(ch *Player) *ObjectInstance {
	if ch == nil || ch.Equipment == nil {
		return nil
	}
	weapon, _ := ch.Equipment.GetItemInSlot(SlotWield)
	return weapon
}

func pointDirection(argument string) string {
	for _, direction := range []string{"east", "west", "up", "down", "north", "south"} {
		if isAbbrev(argument, direction) {
			return direction
		}
	}
	return ""
}

// DoGroinrip implements do_groinrip() — low blow.
func DoGroinrip(ch *Player, target combat.Combatant, world *World) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Groinrip who?"}
	}

	if ch.GetSkill(SkillGroinrip) == 0 {
		return SkillResult{Success: false, MessageToCh: "You're not trained in martial arts!"}
	}

	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "No masochism allowed!"}
	}

	if mob, ok := target.(*MobInstance); ok {
		keeper := false
		if world != nil {
			_, keeper = world.ShopBitvectorForKeeper(mob.GetVNum())
		}
		if keeper || MobSpecAssign[mob.GetVNum()] == "shop_keeper" {
			return SkillResult{Success: false, MessageToCh: "Ha Ha. Don't think so."}
		}
	}

	if !target.IsNPC() && target.GetLevel() >= LVL_IMMORT {
		ch.SetPosition(combat.PosSitting)
		return SkillResult{
			Success:     false,
			MessageToCh: "How dare you try to touch a god!\r\nYou are thrown across the room...",
		}
	}

	if target.GetSex() != SexMale {
		return SkillResult{Success: false, MessageToCh: "Umm, they have nothing there to tug on!"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 121) // 0-120; 101+ is complete failure

	if target.GetPosition() <= combat.PosSleeping || ch.GetLevel() > LVL_IMMORT {
		percent = 0
	}

	prob := ch.GetSkill(SkillGroinrip)

	if percent < prob {
		victimPronouns := GetPronouns(target.GetName(), target.GetSex())
		return SkillResult{
			Success:                  true,
			Damage:                   ch.GetLevel(),
			MessageToRoom:            fmt.Sprintf("%s falls to %s knees, clutching %s groin and throwing up\r\neverywhere!", victimPronouns.Name, victimPronouns.His, victimPronouns.His),
			RoomIncludesActor:        true,
			SkillMsgType:             SkillGroinripNum,
			SkillMsgInDamage:         true,
			DamageSkill:              SkillGroinrip,
			StartCombat:              true,
			WaitCh:                   2,
			DeferredImprove:          []string{SkillGroinrip},
			DeferredImproveAfterRoom: true,
			SpawnPuke:                true,
		}
	}

	// Miss
	return SkillResult{
		Success:          false,
		Damage:           0,
		SkillMsgType:     SkillGroinripNum,
		SkillMsgInDamage: true,
		DamageSkill:      SkillGroinrip,
		StartCombat:      true,
		WaitCh:           2,
	}
}

// DoReview implements do_review() — show recent gossip history.
// Matches C: do_review() in new_cmds.c.
func DoReview(ch *Player, world *World) SkillResult {
	return SkillResult{
		Success:     true,
		MessageToCh: world.ReviewGossip(ch),
	}
}

// DoWhois implements do_whois() — look up player info.
func DoWhois(ch *Player, targetName string) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "For whom do you wish to search?\r\n"}
	}

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("[Looking up %s...]\r\n(Player database lookup not yet connected)\r\n", targetName),
	}
}

// DoPalm implements do_palm() — conceal a small object up your sleeve.
func DoPalm(ch *Player, objName string, world *World) SkillResult {
	if objName == "" {
		return SkillResult{Success: false, MessageToCh: "Palm what?\r\n"}
	}

	// Find item in room
	items := world.GetItemsInRoom(ch.GetRoomVNum())
	var targetItem *ObjectInstance
	targetLower := strings.ToLower(objName)
	for _, item := range items {
		iname := strings.ToLower(item.GetKeywords())
		if strings.Contains(iname, targetLower) {
			targetItem = item
			break
		}
	}

	if targetItem == nil {
		return SkillResult{Success: false, MessageToCh: "You don't see that here.\r\n"}
	}

	// Check weight <= 1 (small object)
	if targetItem.GetWeight() > 1 {
		return SkillResult{Success: false, MessageToCh: "That's too big to palm!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillPalm)

	if prob > percent {
		// Success — move to inventory
		if err := world.MoveObjectToPlayerInventory(targetItem, ch); err != nil {
			return SkillResult{
				Success:     false,
				MessageToCh: "You can't carry that much.\r\n",
			}
		}
		return SkillResult{
			Success:       true,
			MessageToCh:   "You palm the item skillfully.\r\n",
			MessageToRoom: fmt.Sprintf("%s deftly palms something.\r\n", ch.Name),
		}
	}

	// Failure — item stays on ground
	return SkillResult{
		Success:       true,
		MessageToCh:   fmt.Sprintf("You try to palm %s but fumble it!\r\n", targetItem.GetShortDesc()),
		MessageToRoom: fmt.Sprintf("%s fumbles with %s!\r\n", ch.Name, targetItem.GetShortDesc()),
	}
}

// fleshAlterWeapon mirrors flesh_alter_weapon() in src/new_cmds.c:1836-1870.
func fleshAlterWeapon(level int) string {
	switch {
	case level <= 3:
		return "studded wooden club"
	case level <= 6:
		return "razor-sharp dagger"
	case level <= 9:
		return "steel-shafted axe"
	case level <= 12:
		return "studded steel mace"
	case level <= 15:
		return "battle flail"
	case level <= 18:
		return "steel-shafted battle axe"
	case level <= 21:
		return "double-headed battle axe"
	case level <= 24:
		return "studded morning-star"
	case level <= 27:
		return "gleaming broad sword"
	case level <= 29:
		return "gleaming long sword"
	default:
		return "gleaming scythe"
	}
}

// DoFleshAlter implements do_flesh_alter() — transform your hand into a weapon.
func DoFleshAlter(ch *Player) SkillResult {
	if ch.GetSkill(SkillFleshAlter) == 0 {
		return SkillResult{Success: false, MessageToCh: "You know nothing of altering your flesh!\n\r"}
	}

	// C: number(0, 101 + (FIGHTING(ch) ? 10 : 0)).
	rollMax := 101
	if ch.GetFighting() != "" {
		rollMax += 10
	}
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(0, rollMax)
	prob := ch.GetSkill(SkillFleshAlter)

	if percent > prob {
		return SkillResult{
			Success:         false,
			MessageToCh:     "You lose your concentration!",
			WaitCh:          2,
			DeferredImprove: []string{SkillFleshAlter},
		}
	}

	weapon := fleshAlterWeapon(ch.GetLevel())
	if ch.IsAffected(affFleshAlter) {
		ch.SetAffect(affFleshAlter, false)
		ch.AdjustHitroll(-((ch.GetLevel() / 3) + 1))
		ch.AdjustDamroll(-((ch.GetLevel() / 2) + 1))
		return SkillResult{
			Success:       true,
			MessageToCh:   "You shift your molecules back to normal.\r\n" + fmt.Sprintf("Your hand reverts from a %s.", weapon),
			MessageToRoom: fmt.Sprintf("%s's hand reverts from a %s!", ch.Name, weapon),
		}
	}

	ch.SetAffect(affFleshAlter, true)
	ch.AdjustHitroll((ch.GetLevel() / 3) + 1)
	ch.AdjustDamroll((ch.GetLevel() / 2) + 1)
	message := fmt.Sprintf("Your hand turns into a %s!", weapon)
	roomMessage := fmt.Sprintf("%s's hand turns into a %s!", ch.Name, weapon)
	if wielded, ok := ch.Equipment.GetItemInSlot(SlotWield); ok && wielded != nil {
		if err := ch.Equipment.Unequip(SlotWield, ch.Inventory); err != nil {
			slog.Error("flesh alter could not unequip wielded item", "player", ch.Name, "item", wielded.GetShortDesc(), "error", err)
		} else {
			message = fmt.Sprintf("You stop using %s.\r\n%s", wielded.GetShortDesc(), message)
			roomMessage = fmt.Sprintf("%s stops using %s.\r\n%s", ch.Name, wielded.GetShortDesc(), roomMessage)
		}
	}
	return SkillResult{
		Success:       true,
		MessageToCh:   message,
		MessageToRoom: roomMessage,
	}
}

// heShe returns "he" / "she" / "it" based on sex.
