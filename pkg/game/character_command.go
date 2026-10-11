package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// CharacterCommandContext keeps the command body separate from an optional
// descriptor that controls it. Actor is always the real Player or Mob body.
type CharacterCommandContext struct {
	Actor         combat.Combatant
	World         *World
	PrivateOutput func(string)
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
	actor, ok := ctx.Actor.(Actor)
	if !ok || !actor.IsNPC() || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	switch actor.GetPosition() {
	case combat.PosStanding:
		mob, ok := ctx.Actor.(*MobInstance)
		if !ok {
			return CharacterCommandUnsupported
		}
		if mob.IsAffected(affMounted) {
			// C calls do_dismount here. Go has no generic NPC rider-to-mount
			// relationship, so do not treat a mounted body as unmounted.
			return CharacterCommandUnsupported
		}
		characterCommandAct(ctx, false, "You are already standing.", ToChar)
	case combat.PosSitting:
		characterCommandAct(ctx, false, "You stand up.", ToChar)
		characterCommandAct(ctx, true, "$n clambers to $s feet.", ToRoom)
		ctx.Actor.SetPosition(combat.PosStanding)
	case combat.PosResting:
		characterCommandAct(ctx, false, "You stop resting, and stand up.", ToChar)
		characterCommandAct(ctx, true, "$n stops resting, and clambers on $s feet.", ToRoom)
		ctx.Actor.SetPosition(combat.PosStanding)
	case combat.PosSleeping:
		characterCommandAct(ctx, false, "You have to wake up first!", ToChar)
	case combat.PosFighting:
		characterCommandAct(ctx, false, "Do you not consider fighting as standing?", ToChar)
	default:
		characterCommandAct(ctx, false, "You stop floating around, and put your feet on the ground.", ToChar)
		characterCommandAct(ctx, true, "$n stops floating around, and puts $s feet on the ground.", ToRoom)
		ctx.Actor.SetPosition(combat.PosStanding)
	}
	return CharacterCommandHandled
}

// CharacterSit ports the actor body of do_sit. The command interpreter's
// POS_RESTING gate is applied by its caller; this callback does not emulate it.
func CharacterSit(ctx CharacterCommandContext) CharacterCommandResult {
	actor, ok := ctx.Actor.(Actor)
	if !ok || !actor.IsNPC() || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	switch actor.GetPosition() {
	case combat.PosStanding:
		mob, ok := ctx.Actor.(*MobInstance)
		if !ok {
			return CharacterCommandUnsupported
		}
		if mob.IsAffected(affMounted) {
			characterCommandAct(ctx, false, "You can't rest while mounted.", ToChar)
			return CharacterCommandHandled
		}
		characterCommandAct(ctx, false, "You sit down.", ToChar)
		characterCommandAct(ctx, false, "$n sits down.", ToRoom)
		ctx.Actor.SetPosition(combat.PosSitting)
	case combat.PosSitting:
		characterCommandAct(ctx, false, "You're sitting already.", ToChar)
	case combat.PosResting:
		characterCommandAct(ctx, false, "You stop resting, and sit up.", ToChar)
		characterCommandAct(ctx, true, "$n stops resting.", ToRoom)
		ctx.Actor.SetPosition(combat.PosSitting)
	case combat.PosSleeping:
		characterCommandAct(ctx, false, "You have to wake up first.", ToChar)
	case combat.PosFighting:
		characterCommandAct(ctx, false, "Sit down while fighting? are you MAD?", ToChar)
	default:
		characterCommandAct(ctx, false, "You stop floating around, and sit down.", ToChar)
		// src/act.movement.c:763 passes TRUE for this room act.
		characterCommandAct(ctx, true, "$n stops floating around, and sits down.", ToRoom)
		ctx.Actor.SetPosition(combat.PosSitting)
	}
	return CharacterCommandHandled
}

func characterCommandAct(ctx CharacterCommandContext, hideInvisible bool, format string, actType int) {
	actor := ctx.Actor.(Actor)
	ActWithPrivateSink(ctx.World, hideInvisible, actor, nil, nil, nil, format, "", actType, actor, ctx.PrivateOutput)
}
