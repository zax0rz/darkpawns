package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// CharacterCommandContext keeps the command body separate from an optional
// descriptor that controls it. Actor is always the real Player or Mob body.
type CharacterCommandContext struct {
	Actor         combat.Combatant
	World         *World
	PrivateOutput func(string)
	RawArgs       string
}

// CharacterCommandResult distinguishes an implemented actor callback from a
// command whose C behavior has not been ported for that actor.
type CharacterCommandResult uint8

const (
	CharacterCommandUnsupported CharacterCommandResult = iota
	CharacterCommandHandled
)

// CharacterStand ports the actor body of do_stand. The command interpreter's
// POS_RESTING gate is applied by its caller; this callback does not emulate it.
func CharacterStand(ctx CharacterCommandContext) CharacterCommandResult {
	if _, ok := ctx.Actor.(*MobInstance); !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doPositionCommand(ctx.Actor.(Actor), positionCommandSpecs[0], ctx.PrivateOutput)
	return CharacterCommandHandled
}

// CharacterSit ports the actor body of do_sit.
func CharacterSit(ctx CharacterCommandContext) CharacterCommandResult {
	if _, ok := ctx.Actor.(*MobInstance); !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doPositionCommand(ctx.Actor.(Actor), positionCommandSpecs[1], ctx.PrivateOutput)
	return CharacterCommandHandled
}

// CharacterRest ports the actor body of do_rest (src/act.movement.c:770-806).
func CharacterRest(ctx CharacterCommandContext) CharacterCommandResult {
	if _, ok := ctx.Actor.(*MobInstance); !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doPositionCommand(ctx.Actor.(Actor), positionCommandSpecs[2], ctx.PrivateOutput)
	return CharacterCommandHandled
}

// CharacterSleep ports the actor body of do_sleep (src/act.movement.c:808-839).
func CharacterSleep(ctx CharacterCommandContext) CharacterCommandResult {
	if _, ok := ctx.Actor.(*MobInstance); !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doPositionCommand(ctx.Actor.(Actor), positionCommandSpecs[3], ctx.PrivateOutput)
	return CharacterCommandHandled
}

// CharacterDismount ports do_dismount (src/act.other.c:1597-1619) for a mobile
// rider. The refusal and confirmation lines are descriptorless; the room act
// for a real mount is not.
func CharacterDismount(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doDismountActor(mob)
	return CharacterCommandHandled
}

// CharacterMount ports do_ride's mobile lanes (src/act.other.c:1545-1593). Every
// refusal in do_ride is send_to_char to the rider except the non-mountable
// target's act(...TO_CHAR); CAN_MOUNT(mob) is false (src/utils.h:365-366), so an
// NPC rider has no player-visible output and no state change.
func CharacterMount(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.doRideMob(mob, extractArg(ctx.RawArgs))
	return CharacterCommandHandled
}

// CharacterFollow ports the non-quiet do_follow lane (src/act.movement.c:883-950)
// for a mobile actor. Order/force/lua carry subcmd 0, so C skips the shadow
// skill draw; the state transition and its room/leader notices are shared with
// the mobile follow path.
func CharacterFollow(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	arg := extractArg(ctx.RawArgs)
	if arg == "" {
		return CharacterCommandHandled
	}
	target, found := ctx.World.resolveCharInRoomAt(mob, mob.GetRoomVNum(), arg)
	if !found {
		return CharacterCommandHandled
	}
	if target.Combatant == nil {
		return CharacterCommandUnsupported
	}
	followMobBody(ctx.World, mob, target.Combatant)
	return CharacterCommandHandled
}

// CharacterWake ports do_wake (src/act.movement.c:841-881) for a mobile actor.
func CharacterWake(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	wakeBody(ctx.World, mob, ctx.RawArgs, ctx.PrivateOutput)
	return CharacterCommandHandled
}

// CharacterMove ports do_move's perform_move call (src/act.movement.c:361-366)
// for a mobile dispatched through the command interpreter. needSpecialsCheck
// is false because command_interpreter already offered the command to
// special() (src/interpreter.c:947) — C's double-spec-proc guard.
func CharacterMove(ctx CharacterCommandContext, dir int) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil || dir < 0 || dir >= len(dirs) {
		return CharacterCommandUnsupported
	}
	ctx.World.performMoveMob(mob, dir, false)
	return CharacterCommandHandled
}

// CharacterFlee ports do_flee (src/act.offensive.c:360-415) for a mobile
// dispatched through the command interpreter.
func CharacterFlee(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	ctx.World.DoFlee(mob)
	return CharacterCommandHandled
}

// wakeBody is do_wake's body shared by both character types. Every message
// addressed to the waker is act(...TO_CHAR) and therefore descriptorless for a
// mobile; the room and victim notices reach players.
func wakeBody(w *World, body combat.Combatant, argument string, private func(string)) {
	actor, ok := body.(Actor)
	if !ok {
		return
	}
	argument = firstWord(strings.TrimSpace(argument))
	if argument != "" {
		if actor.GetPosition() == combat.PosSleeping {
			wakeSelfLine(w, actor, "Maybe you should wake yourself up first.", private)
			return
		}
		var target Actor
		if strings.EqualFold(argument, actor.GetName()) || strings.EqualFold(argument, "self") || strings.EqualFold(argument, "me") {
			target = actor
		} else if resolved, ok := w.resolveCharInRoomAt(body, actor.GetRoom(), argument); ok {
			target = asActor(resolved.Combatant)
		}
		if target == nil {
			wakeSelfLine(w, actor, NoPersonHere, private)
			return
		}
		if target != actor {
			switch {
			case target.GetPosition() > combat.PosSleeping:
				ActWithPrivateSink(w, false, actor, target, nil, nil, "$E is already awake.", "", ToChar, actor, private)
			case actorIsAffected(target, affSleep):
				ActWithPrivateSink(w, false, actor, target, nil, nil, "You can't wake $M up!", "", ToChar, actor, private)
			case target.GetPosition() < combat.PosSleeping:
				ActWithPrivateSink(w, false, actor, target, nil, nil, "$E's in pretty bad shape!", "", ToChar, actor, private)
			default:
				ActWithPrivateSink(w, false, actor, target, nil, nil, "You wake $M up.", "", ToChar, actor, private)
				Act(w, false, actor, target, nil, nil, "$n wakes up $N.", "", ToNotVict)
				// Make the victim awake before rendering their message so
				// PERS($n) resolves the waker as C's TO_SLEEP delivery does.
				setActorPosition(target, combat.PosSitting)
				Act(w, false, actor, target, nil, nil, "You are awakened by $n.", "", ToVict|ToSleep)
			}
			return
		}
	}

	if actorIsAffected(actor, affSleep) {
		wakeSelfLine(w, actor, "You can't wake up!", private)
		Act(w, true, actor, nil, nil, nil, "$n tosses and turns uncomfortably.", "", ToRoom)
	} else if actor.GetPosition() > combat.PosSleeping {
		wakeSelfLine(w, actor, "You are already awake...", private)
	} else {
		wakeSelfLine(w, actor, "You awaken, and sit up.", private)
		Act(w, true, actor, nil, nil, nil, "$n awakens.", "", ToRoom)
		setActorPosition(actor, combat.PosSitting)
	}
}

// wakeSelfLine keeps the player path's ToSleep audience and drops the line for
// a descriptorless mobile, as C's send_to_char does.
func wakeSelfLine(w *World, actor Actor, message string, private func(string)) {
	if player, ok := actor.(*Player); ok {
		movementSendToChar(player, message)
		return
	}
	ActWithPrivateSink(w, false, actor, nil, nil, nil, message, "", ToChar|ToSleep, actor, private)
}
