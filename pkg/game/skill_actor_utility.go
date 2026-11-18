package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// CharacterUtilitySkill routes the mobile body of the utility/stealth skills
// registered via wrapSkill in pkg/session/commands.go. C runs these same
// do_* handlers against ch (the switched mobile) after the interpreter gates.
func CharacterUtilitySkill(ctx CharacterCommandContext, commandName string, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	rawArg := strings.TrimSpace(ctx.RawArgs)
	if rawArg == "" && len(args) > 0 {
		rawArg = strings.Join(args, " ")
	}
	switch strings.ToLower(commandName) {
	case "sneak":
		return characterSneak(ctx, mob, SkillSneak, skillNumSneak)
	case "stealth":
		return characterSneak(ctx, mob, SkillStealth, skillNumStealth)
	case "hide":
		return characterHide(ctx, mob, false)
	case "kabuki":
		return characterHide(ctx, mob, true)
	case "mindlink":
		return characterMindlink(ctx, mob, rawArg)
	case "turn":
		return characterTurn(ctx, mob)
	case "palm":
		return characterPalm(ctx, mob, rawArg)
	case "point":
		return characterPoint(ctx, mob, rawArg)
	case "compare":
		return characterCompare(ctx, mob, rawArg)
	case "sharpen":
		return characterSharpen(ctx, mob, rawArg)
	case "carve":
		return characterCarve(ctx, mob, rawArg)
	default:
		return CharacterCommandUnsupported
	}
}

func characterSneak(ctx CharacterCommandContext, mob *MobInstance, skill string, spell int) CharacterCommandResult {
	if mob.IsAffected(affMounted) {
		sendBodyMessage(ctx.World, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	sendBodyMessage(ctx.World, mob, "Okay, you'll try to move silently for a while.\r\n")
	if mob.IsAffected(affSneak) {
		mob.RemoveAffectBySpell(skillNumSneak)
		mob.RemoveAffectBySpell(skillNumStealth)
		mob.RemoveAffected(affSneak)
	}
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := dexAppSkill(mob.GetDex()).Sneak
	if percent > prob {
		return CharacterCommandHandled
	}
	mob.AddAffect(engine.NewAffectDirect(
		spell,
		engine.ApplyNone,
		mob.GetLevel(),
		0,
		engine.AFFSneak,
		skill,
	))
	return CharacterCommandHandled
}

func characterHide(ctx CharacterCommandContext, mob *MobInstance, kabuki bool) CharacterCommandResult {
	if mob.IsAffected(affMounted) {
		sendBodyMessage(ctx.World, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if msg := hideDaytimeSectorMessage(ctx.World, mob.GetRoom()); msg != "" {
		sendBodyMessage(ctx.World, mob, msg+"\r\n")
		return CharacterCommandHandled
	}
	if kabuki {
		sendBodyMessage(ctx.World, mob, "You attempt to practice the art of kabuki.\r\n")
	} else {
		sendBodyMessage(ctx.World, mob, "You attempt to hide yourself.\r\n")
	}
	if mob.IsAffected(affHide) {
		mob.RemoveAffected(affHide)
	}
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := dexAppSkill(mob.GetDex()).Hide
	if percent <= prob {
		mob.SetAffected(affHide)
	}
	return CharacterCommandHandled
}

func characterMindlink(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	targetName, _ := OneArgument(rawArg)
	if targetName == "" {
		sendBodyMessage(ctx.World, mob, "Link your mind to whose?\r\n")
		return CharacterCommandHandled
	}
	target, found := ctx.World.ResolveCharInRoom(mob, targetName)
	if !found {
		sendBodyMessage(ctx.World, mob, "They don't seem to be here.\r\n")
		return CharacterCommandHandled
	}
	if target.Combatant == mob {
		sendBodyMessage(ctx.World, mob, "You wish you could.\r\n")
		return CharacterCommandHandled
	}
	// Mobiles have GET_SKILL(ch, SKILL_MINDLINK) == 0 (src/new_cmds2.c:267).
	sendBodyMessage(ctx.World, mob, "Yeah, right.\r\n")
	return CharacterCommandHandled
}

func characterTurn(ctx CharacterCommandContext, mob *MobInstance) CharacterCommandResult {
	// Mobiles have GET_SKILL(ch, SKILL_TURN) == 0 (src/new_cmds2.c:965).
	sendBodyMessage(ctx.World, mob, "Huh?!?\r\n")
	return CharacterCommandHandled
}

func characterPalm(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	// C do_palm draws number(1, 101) at declaration before checking carry/arg
	// (src/new_cmds.c:2112).
	// #nosec G404 — game RNG, not cryptographic
	_ = dprng.Number(1, 101)
	arg1, _ := OneArgument(rawArg)
	if mobCarriedItemCount(mob) >= mobMaxCarryCount(mob) {
		sendBodyMessage(ctx.World, mob, "Your arms are already full!\r\n")
		return CharacterCommandHandled
	}
	if arg1 == "" {
		sendBodyMessage(ctx.World, mob, "Palm what?\r\n")
		return CharacterCommandHandled
	}
	// With prob == 0, prob > percent is always false, so C calls get_from_room(ch, arg1, FALSE).
	palmCtx := ctx
	palmCtx.RawArgs = arg1
	getForMobile(palmCtx, mob)
	return CharacterCommandHandled
}

func characterPoint(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	targetName, _ := OneArgument(rawArg)
	if targetName == "" {
		sendBodyMessage(ctx.World, mob, "You point around the room.\r\n")
		Act(ctx.World, false, mob, nil, nil, nil, "$n points around the room.", "", ToRoom)
		return CharacterCommandHandled
	}
	if target, found := ctx.World.ResolveCharInRoom(mob, targetName); found {
		if target.Combatant == mob {
			sendBodyMessage(ctx.World, mob, "You point at yourself.\r\n")
			Act(ctx.World, false, mob, nil, nil, nil, "$n points at $mself.", "", ToRoom)
			return CharacterCommandHandled
		}
		weapon := mob.Equipped(mobWearWield)
		victActor := asActor(target.Combatant)
		if weapon == nil {
			ActWithPrivateSink(ctx.World, false, mob, victActor, nil, nil, "You point at $N.", "", ToChar, mob, ctx.PrivateOutput)
			Act(ctx.World, false, mob, victActor, nil, nil, "$n points at you.", "", ToVict)
			Act(ctx.World, false, mob, victActor, nil, nil, "$n points at $N.", "", ToNotVict)
		} else {
			ActWithPrivateSink(ctx.World, false, mob, victActor, weapon, nil, "You point $p at $N.", "", ToChar, mob, ctx.PrivateOutput)
			Act(ctx.World, false, mob, victActor, weapon, nil, "$n points $p at you.", "", ToVict)
			Act(ctx.World, false, mob, victActor, weapon, nil, "$n points $p at $N.", "", ToNotVict)
		}
		return CharacterCommandHandled
	}
	if obj, found := ctx.World.ResolveObjectInRoom(mob, targetName); found {
		weapon := mob.Equipped(mobWearWield)
		if weapon == nil {
			sendBodyMessage(ctx.World, mob, fmt.Sprintf("You point at %s.\r\n", obj.GetShortDesc()))
			Act(ctx.World, false, mob, nil, obj, nil, "$n points at $p.", "", ToRoom)
		} else {
			sendBodyMessage(ctx.World, mob, fmt.Sprintf("You point %s at %s.\r\n", weapon.GetShortDesc(), obj.GetShortDesc()))
			Act(ctx.World, false, mob, nil, weapon, obj, "$n points $p at $P.", "", ToRoom)
		}
		return CharacterCommandHandled
	}
	if dir := pointDirection(targetName); dir != "" {
		sendBodyMessage(ctx.World, mob, fmt.Sprintf("You point %s.\r\n", dir))
		Act(ctx.World, false, mob, nil, nil, nil, fmt.Sprintf("$n points %s.", dir), "", ToRoom)
		return CharacterCommandHandled
	}
	sendBodyMessage(ctx.World, mob, "You point around the room.\r\n")
	Act(ctx.World, false, mob, nil, nil, nil, "$n points around the room.", "", ToRoom)
	return CharacterCommandHandled
}

func characterCompare(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	if mob.IsAffected(affBlind) {
		sendBodyMessage(ctx.World, mob, "You can't see a damned thing!\r\n")
		return CharacterCommandHandled
	}
	arg1, rest := OneArgument(rawArg)
	arg2, _ := OneArgument(rest)
	var obj1, obj2 *ObjectInstance
	if arg1 != "" {
		obj1, _ = resolveVisibleObject(mob, arg1, mob.InventorySnapshot(), true)
	}
	if arg2 != "" {
		obj2, _ = resolveVisibleObject(mob, arg2, mob.InventorySnapshot(), true)
	}
	if mob.GetFightingBody() != nil {
		sendBodyMessage(ctx.World, mob, "You're pretty busy right now!\n\r")
		return CharacterCommandHandled
	}
	if obj1 == nil || obj2 == nil {
		sendBodyMessage(ctx.World, mob, "Looks like you don't have those objects..\n\r")
		return CharacterCommandHandled
	}
	if obj1 == obj2 {
		sendBodyMessage(ctx.World, mob, "They're the same thing!\n\r")
		return CharacterCommandHandled
	}
	if obj1.GetTypeFlag() != obj2.GetTypeFlag() {
		sendBodyMessage(ctx.World, mob, "You can't compare those things!\r\n")
		return CharacterCommandHandled
	}
	if obj1.GetTypeFlag() != ITEM_WEAPON && obj1.GetTypeFlag() != ITEM_ARMOR {
		sendBodyMessage(ctx.World, mob, "Compare is only for weapons and armor.\r\n")
		return CharacterCommandHandled
	}
	// Mobiles have skill 0 -> prob = 20 + level (src/new_cmds.c:1964).
	prob := 20 + mob.GetLevel()
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	if percent > prob {
		sendBodyMessage(ctx.World, mob, "You can't quite tell which is better.\r\n")
		return CharacterCommandHandled
	}
	diff := compareObjectScore(obj1) - compareObjectScore(obj2)
	switch {
	case diff == 0:
		sendBodyMessage(ctx.World, mob, fmt.Sprintf("%s and %s look about the same.\r\n", CapitalizeSentence(obj1.GetShortDesc()), obj2.GetShortDesc()))
	case diff > 0:
		sendBodyMessage(ctx.World, mob, fmt.Sprintf("%s looks better than %s.\r\n", CapitalizeSentence(obj1.GetShortDesc()), obj2.GetShortDesc()))
	default:
		sendBodyMessage(ctx.World, mob, fmt.Sprintf("%s looks better than %s.\r\n", CapitalizeSentence(obj2.GetShortDesc()), obj1.GetShortDesc()))
	}
	return CharacterCommandHandled
}

func compareObjectScore(obj *ObjectInstance) int {
	if obj.GetTypeFlag() == ITEM_ARMOR {
		return obj.GetValue(0)
	}
	return (obj.GetValue(1) * (obj.GetValue(2) + 1)) / 2
}

func characterSharpen(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	arg1, _ := OneArgument(rawArg)
	if arg1 == "" {
		sendBodyMessage(ctx.World, mob, "Sharpen what?\r\n")
		return CharacterCommandHandled
	}
	obj, found := resolveVisibleObject(mob, arg1, mob.InventorySnapshot(), true)
	if !found {
		sendBodyMessage(ctx.World, mob, "Sharpen what?\r\n")
		return CharacterCommandHandled
	}
	if obj.GetTypeFlag() != ITEM_WEAPON || obj.GetValue(3) != 3 {
		sendBodyMessage(ctx.World, mob, "This weapon can not be sharpened.\n\r")
		return CharacterCommandHandled
	}
	if obj.HasExtraFlag(0, itemExtraMagic) {
		sendBodyMessage(ctx.World, mob, "This weapon can not be sharpened any further.\r\n")
		return CharacterCommandHandled
	}
	for _, aff := range obj.GetAffects() {
		if aff.Location != 0 {
			sendBodyMessage(ctx.World, mob, "This weapon can not be sharpened any further.\r\n")
			return CharacterCommandHandled
		}
	}
	if mob.GetFightingBody() != nil {
		sendBodyMessage(ctx.World, mob, "You're too busy to be sharpening anything!\r\n")
		return CharacterCommandHandled
	}
	// #nosec G404 — game RNG, not cryptographic
	_ = dprng.Number(1, 101)
	sendBodyMessage(ctx.World, mob, "You fail to sharpen it.\r\n")
	return CharacterCommandHandled
}

func characterCarve(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	if mob.GetPosition() == combat.PosFighting || mob.GetFightingBody() != nil {
		sendBodyMessage(ctx.World, mob, "How can you think of food at a time like this?!?\r\n")
		return CharacterCommandHandled
	}
	targetName, _ := OneArgument(rawArg)
	if targetName == "" {
		sendBodyMessage(ctx.World, mob, "You want to carve what?!?\r\n")
		return CharacterCommandHandled
	}
	if target, found := ctx.World.ResolveCharInRoom(mob, targetName); found {
		if target.Combatant == mob {
			sendBodyMessage(ctx.World, mob, "This game doesn't support self-mutilation!\r\n")
		} else {
			sendBodyMessage(ctx.World, mob, "You kill it first and THEN you can eat it!\r\n")
		}
		return CharacterCommandHandled
	}
	corpse, found := ctx.World.ResolveObjectInRoom(mob, targetName)
	if !found {
		sendBodyMessage(ctx.World, mob, fmt.Sprintf("You can't seem to find a %s to carve!\r\n", targetName))
		return CharacterCommandHandled
	}
	if !strings.Contains(corpse.GetKeywords(), "corpse") {
		sendBodyMessage(ctx.World, mob, "Your initials are about all you can carve in that!\r\n")
		return CharacterCommandHandled
	}
	if !strings.Contains(corpse.GetKeywords(), "carve_") {
		sendBodyMessage(ctx.World, mob, "There's no way you could ever eat THAT!!!\r\n")
		return CharacterCommandHandled
	}
	if mobCarriedItemCount(mob) >= mobMaxCarryCount(mob) {
		sendBodyMessage(ctx.World, mob, "Your arms are already full!\r\n")
		return CharacterCommandHandled
	}
	foodVNum := 8015
	switch {
	case strings.Contains(corpse.GetKeywords(), "carve_beef"):
		foodVNum = 8000
	case strings.Contains(corpse.GetKeywords(), "carve_chicken"):
		foodVNum = 8002
	case strings.Contains(corpse.GetKeywords(), "carve_pork"):
		foodVNum = 8001
	}
	proto, ok := ctx.World.GetObjPrototype(foodVNum)
	if !ok {
		return CharacterCommandHandled
	}
	meat := NewObjectInstance(proto, -1)
	if err := ctx.World.MoveObject(meat, LocInventoryMob(mob.ID)); err != nil {
		return CharacterCommandHandled
	}
	sendBodyMessage(ctx.World, mob, fmt.Sprintf("You carve %s from %s.\r\n", meat.GetShortDesc(), corpse.GetShortDesc()))
	Act(ctx.World, true, mob, nil, meat, corpse, "$n carves $p from $P.", "", ToRoom)
	return CharacterCommandHandled
}
