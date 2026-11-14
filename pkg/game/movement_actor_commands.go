package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CharacterEnter ports do_enter for the body selected by command_interpreter.
func CharacterEnter(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	arg, _ := oneArgument(ctx.RawArgs)
	room := ctx.World.GetRoomInWorld(mob.GetRoomVNum())
	if room == nil {
		return CharacterCommandHandled
	}
	if arg != "" {
		for dir, direction := range dirs {
			exit, exists := room.Exits[direction]
			if exists && exit.Keywords != "" && strings.EqualFold(strings.TrimSpace(exit.Keywords), arg) {
				ctx.World.performMoveMob(mob, dir, true)
				return CharacterCommandHandled
			}
		}
		mobileSend(ctx, mob, "There is no "+arg+" here.\r\n")
		return CharacterCommandHandled
	}
	if movementRoomHasFlag(room, roomFlagIndoors, "indoors") {
		mobileSend(ctx, mob, "You are already indoors.\r\n")
		return CharacterCommandHandled
	}
	for dir, direction := range dirs {
		exit, exists := room.Exits[direction]
		if !exists || exit.ToRoom < 0 || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		if destination := ctx.World.GetRoomInWorld(exit.ToRoom); movementRoomHasFlag(destination, roomFlagIndoors, "indoors") {
			ctx.World.performMoveMob(mob, dir, true)
			return CharacterCommandHandled
		}
	}
	mobileSend(ctx, mob, "You can't seem to find anything to enter.\r\n")
	return CharacterCommandHandled
}

// CharacterLeave ports do_leave for a mobile body.
func CharacterLeave(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	room := ctx.World.GetRoomInWorld(mob.GetRoomVNum())
	if room == nil {
		return CharacterCommandHandled
	}
	if !movementRoomHasFlag(room, roomFlagIndoors, "indoors") {
		mobileSend(ctx, mob, "You are outside.. where do you want to go?\r\n")
		return CharacterCommandHandled
	}
	for dir, direction := range dirs {
		exit, exists := room.Exits[direction]
		if !exists || exit.ToRoom < 0 || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		if destination := ctx.World.GetRoomInWorld(exit.ToRoom); destination != nil && !movementRoomHasFlag(destination, roomFlagIndoors, "indoors") {
			ctx.World.performMoveMob(mob, dir, true)
			return CharacterCommandHandled
		}
	}
	mobileSend(ctx, mob, "I see no obvious exits to the outside.\r\n")
	return CharacterCommandHandled
}

// CharacterRetreat ports the reachable NPC lane of do_retreat. C's
// GET_SKILL(ch, ...) reads dummy_mob's static-zero skill array for mobiles
// (src/db.c:104, 1281; src/utils.h:343). The handler checks position first,
// then with both skills zero C returns Huh? before checking combat or
// consuming a draw (src/act.offensive.c:1019-1024).
func CharacterRetreat(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	if mob.GetPosition() < combat.PosFighting {
		mobileSend(ctx, mob, "Get on your feet first!\r\n")
		return CharacterCommandHandled
	}
	mobileSend(ctx, mob, "Huh?\r\n")
	return CharacterCommandHandled
}

// CharacterTrack ports the mobile refusal in do_track. IS_WARRIOR,
// IS_PALADIN, and IS_RANGER all explicitly exclude NPCs (src/utils.h:408,
// 416, 578), so C exits before reading a skill or drawing RNG.
func CharacterTrack(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}
	mobileSend(ctx, mob, "You have no idea how.\r\n")
	return CharacterCommandHandled
}

// CharacterShadow ports do_follow's quiet SCMD_SHADOW lane for a mobile body.
// NPC GET_SKILL values read dummy_mob's zero-initialized shared skill array;
// the quiet command still draws before the immortal-level shortcut, as C does.
func CharacterShadow(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	arg, _ := oneArgument(ctx.RawArgs)
	if arg == "" {
		mobileSend(ctx, mob, "Whom do you wish to follow?\r\n")
		return CharacterCommandHandled
	}
	resolved, found := ctx.World.resolveCharInRoomAt(mob, mob.GetRoomVNum(), arg)
	if !found || resolved.Combatant == nil {
		mobileSend(ctx, mob, NoPersonHere)
		return CharacterCommandHandled
	}
	leader := resolved.Combatant
	leaderActor := asActor(leader)
	master := ctx.World.FollowingActor(mob)
	if master == leaderActor {
		ActWithPrivateSink(ctx.World, false, mob, leaderActor, nil, nil,
			"You are already following $M.", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if mob.IsAffected(affCharm) && master != nil {
		ActWithPrivateSink(ctx.World, false, mob, master, nil, nil,
			"But you only feel like following $N!", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if leader == mob {
		if master == nil {
			mobileSend(ctx, mob, "You are already following yourself.\r\n")
		} else {
			StopFollowerMob(ctx.World, mob)
		}
		return CharacterCommandHandled
	}
	if followWouldCycle(ctx.World, mob, leader) {
		ActWithPrivateSink(ctx.World, false, mob, nil, nil, nil,
			"Sorry, but following in loops is not allowed.", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if master != nil {
		StopFollowerMob(ctx.World, mob)
	}
	mob.RemoveAffected(affGroup)
	// The skill value is zero, but C evaluates number(0, 101) before the
	// GET_LEVEL >= LVL_IMMORT alternative, so the draw is unconditional here.
	shadowRoll := dprng.Number(0, 101)
	if 0 > shadowRoll || mob.GetLevel() >= LVL_IMMORT {
		if mobHasSpellID(mob, skillNumShadow) {
			mob.RemoveAffectBySpell(skillNumShadow)
			mob.RemoveAffected(affDodge)
		}
		mob.AddAffect(engine.NewAffectDirect(
			skillNumShadow, engine.ApplyNone, mob.GetLevel(), 0, engine.AFFDodge, SkillShadow,
		))
		mob.SetAffected(affDodge)
		ActWithPrivateSink(ctx.World, false, mob, leaderActor, nil, nil,
			"You now follow $N.", "", ToChar, mob, ctx.PrivateOutput)
		mob.SetFollowingBody(leader)
		return CharacterCommandHandled
	}
	addFollowerMobToBody(ctx.World, mob, leader)
	return CharacterCommandHandled
}

func followWouldCycle(w *World, follower *MobInstance, leader combat.Combatant) bool {
	seen := make(map[combat.Combatant]struct{})
	for current := leader; current != nil; current = w.combatFollowingBody(current) {
		if current == follower {
			return true
		}
		if _, exists := seen[current]; exists {
			return false
		}
		seen[current] = struct{}{}
	}
	return false
}

// CharacterUngroup ports do_ungroup for a mobile leader. The no-argument
// path is descriptor output through the acting body's private sink; follower
// notices and state changes remain on the world/body side.
func CharacterUngroup(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	arg, _ := oneArgument(ctx.RawArgs)
	if arg == "" {
		if ctx.World.FollowingActor(mob) != nil || !GroupedActor(mob) {
			mobileSend(ctx, mob, "But you lead no group!\r\n")
			return CharacterCommandHandled
		}
		disband := fmt.Sprintf("%s has disbanded the group.\r\n", mob.GetName())
		for _, follower := range ctx.World.FollowersOf(mob) {
			if !GroupedActor(follower) {
				continue
			}
			SetGroupedActor(follower, false)
			SendToChar(follower, disband)
			if !actorIsAffected(follower, affCharm) {
				stopFollowingActor(ctx.World, follower)
			}
		}
		SetGroupedActor(mob, false)
		mobileSend(ctx, mob, "You disband the group.\r\n")
		return CharacterCommandHandled
	}
	victim := ctx.World.resolveRoomActor(mob, arg)
	if victim == nil {
		mobileSend(ctx, mob, "There is no such person!\r\n")
		return CharacterCommandHandled
	}
	if !ctx.World.FollowsBody(victim, mob) {
		mobileSend(ctx, mob, "That person is not following you!\r\n")
		return CharacterCommandHandled
	}
	if !GroupedActor(victim) {
		mobileSend(ctx, mob, "That person isn't in your group.\r\n")
		return CharacterCommandHandled
	}
	SetGroupedActor(victim, false)
	ActWithPrivateSink(ctx.World, false, mob, victim, nil, nil,
		"$N is no longer a member of your group.", "", ToChar, mob, ctx.PrivateOutput)
	Act(nil, false, mob, victim, nil, nil, "You have been kicked out of $n's group!", "", ToVict)
	Act(ctx.World, false, mob, victim, nil, nil, "$N has been kicked out of $n's group!", "", ToNotVict)
	if !actorIsAffected(victim, affCharm) {
		stopFollowingActor(ctx.World, victim)
	}
	return CharacterCommandHandled
}

func stopFollowingActor(w *World, actor Actor) {
	switch body := actor.(type) {
	case *Player:
		StopFollower(w, body)
	case *MobInstance:
		StopFollowerMob(w, body)
	}
}
