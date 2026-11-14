package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// CharacterCommandContext keeps the command body separate from an optional
// descriptor that controls it. Actor is always the real Player or Mob body.
type CharacterCommandContext struct {
	Actor           combat.Combatant
	DescriptorOwner *Player
	World           *World
	PrivateOutput   func(string)
	RawArgs         string
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
		// C do_follow prints NOPERSON to the actor (src/act.movement.c:893-897);
		// a descriptor-less mobile drops it, a switched body sees it.
		Act(ctx.World, false, mob, nil, nil, nil, "No-one by that name here.", "", ToChar)
		return CharacterCommandHandled
	}
	if target.Combatant == nil {
		return CharacterCommandUnsupported
	}
	followMobBody(ctx.World, mob, target.Combatant)
	return CharacterCommandHandled
}

// CharacterRoll ports do_roll (src/act.other.c:1927-1945) for a mobile actor:
// the private roll line and the room act are the only output, and one
// number(1, max) draw (R3).
func CharacterRoll(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	maxRoll := rollMaximum(extractArg(ctx.RawArgs))
	result := rollNumber(maxRoll, dprng.Number)
	ctx.PrivateOutput(fmt.Sprintf("You roll %d (1-%d).\r\n", result, maxRoll))
	Act(ctx.World, false, mob, nil, nil, nil,
		fmt.Sprintf("With a toss of the dice, $n rolls %d (1-%d).", result, maxRoll), "", ToRoom)
	return CharacterCommandHandled
}

// CharacterVisible ports do_visible (src/act.other.c:557-585) for a mobile
// actor. Immortal mobiles collapse to the "already fully visible" note via the
// level arm's mob bookkeeping (immort invis levels are modeled for players
// only; the low-mob paths carry the witness).
func CharacterVisible(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	out := ctx.PrivateOutput
	if mobHasSpellID(mob, skillNumKkZai) {
		out("You cannot become visible until your zai ends!\r\n")
		return CharacterCommandHandled
	}
	if mob.GetLevel() >= LVL_IMMORT {
		// C perform_immort_vis (src/act.wizard.c:1622-1636); mobs carry no invis
		// level in the port, so the already-visible arm covers the common case.
		if !mob.IsAffected(affHide) || !mob.IsAffected(affInvisible) {
			out("You are already fully visible.\r\n")
			return CharacterCommandHandled
		}
		mobApparition(ctx.World, mob)
		out("You are now fully visible.\r\n")
		return CharacterCommandHandled
	}
	altered := false
	if mob.IsAffected(affInvisible) {
		mobApparition(ctx.World, mob)
		out("You fade into view.\r\n")
		altered = true
	}
	if mob.IsAffected(affSneak) {
		out("You stop sneaking.\r\n")
		mob.RemoveAffectBySpell(skillNumSneak)
		mob.RemoveAffectBySpell(skillNumStealth)
		mob.RemoveAffected(affSneak)
		altered = true
	}
	if !altered {
		out("You are already visible.\r\n")
	}
	return CharacterCommandHandled
}

// mobHasSpellID reports whether the mobile carries a spell record with the
// given spell/skill id, C affected_by_spell's question for NPC bodies.
//
// spell records live in MobInstance.CustomData as *engine.Affect, mirroring
// RemoveAffectBySpell's own walk.
func mobHasSpellID(mob *MobInstance, spellID int) bool {
	mob.mu.RLock()
	defer mob.mu.RUnlock()
	for _, value := range mob.CustomData {
		if aff, ok := value.(*engine.Affect); ok && aff.SpellID == spellID {
			return true
		}
	}
	return false
}

// mobApparition is C appear() (src/fight.c:108-121) for a mobile actor.
func mobApparition(w *World, mob *MobInstance) {
	mob.RemoveAffectBySpell(spellInvisible)
	mob.RemoveAffected(affInvisible)
	mob.RemoveAffected(affHide)
	if mob.GetLevel() < LVL_IMMORT {
		Act(w, false, mob, nil, nil, nil, "$n slowly fades into existence.", "", ToRoom)
	} else {
		Act(w, false, mob, nil, nil, nil, "You feel a strange presence as $n appears, seemingly from nowhere.", "", ToRoom)
	}
}

// CharacterDream ports do_dream (src/act.social.c:294-304) for a mobile actor.
func CharacterDream(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	if mob.GetPosition() != combat.PosSleeping {
		ctx.PrivateOutput("You daydream about better times.\r\n")
		return CharacterCommandHandled
	}
	Act(ctx.World, true, mob, nil, nil, nil, "$n dreams of running naked through a field of tulips.", "", ToRoom)
	ctx.PrivateOutput("You dream of running naked through a field of tulips.\r\n")
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

// CharacterHit ports the actor body of do_hit (src/act.offensive.c:101-129)
// for a mobile dispatched through the command interpreter — force, order and
// the switched-descriptor lane all reach it, because C's command_interpreter
// runs the same do_* for NPCs (R2e). The gates run in C's order: empty
// argument, room-visible target, self, a charmed follower's master, then the
// standing-and-not-already-fighting swing gate. WAIT_STATE(PULSE_VIOLENCE+2)
// at the end writes ch->wait (utils.h:462-464) — the descriptor-command
// field, not mob_specials.wait_state, which perform_violence reads
// (fight.c:1977) — so it is inert for a forced, descriptorless mobile and is
// not modeled here.
func CharacterHit(ctx CharacterCommandContext) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	send := func(msg string) {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput(msg)
		}
	}
	arg := extractArg(ctx.RawArgs)
	if arg == "" {
		send("Hit who?\r\n")
		return CharacterCommandHandled
	}
	resolved, found := ctx.World.resolveCharInRoomAt(mob, mob.GetRoomVNum(), arg)
	if !found || resolved.Combatant == nil {
		send("They don't seem to be here.\r\n")
		return CharacterCommandHandled
	}
	victim := resolved.Combatant
	if victim == mob {
		ActWithPrivateSink(ctx.World, false, mob, nil, nil, nil,
			"You hit yourself...OUCH!.", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, false, mob, nil, nil, nil,
			"$n hits $mself, and says OUCH!", "", ToRoom)
		return CharacterCommandHandled
	}
	if ctx.World.IsCharmedI(mob) && ctx.World.FollowingBody(mob) == victim {
		ActWithPrivateSink(ctx.World, false, mob, victim, nil, nil,
			"$N is just such a good friend, you simply can't hit $M.", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	// C only starts a hit while standing and when the requested victim is not
	// already FIGHTING(ch); otherwise the refusal line with no swing.
	if mob.GetPosition() != combat.PosStanding || mob.GetFightingBody() == victim {
		send("You do the best you can!\r\n")
		return CharacterCommandHandled
	}
	// C dismounts only on the branch that is about to call hit().
	if mob.IsMountedMob() {
		ctx.World.doDismountActor(mob)
	}
	// hit(ch, vict, TYPE_UNDEFINED): the first swing consumes hit()'s to-hit
	// and damage draws and damage()'s protection block decides, enrolling at
	// the damage point. A body flagged for extraction but still standing —
	// do_kill's raw_kill — swings through the pending-body lane
	// (src/handler.c:1194-1209).
	if ctx.World.combatEngine == nil {
		return CharacterCommandHandled
	}
	if combat.BodyRetired(victim) {
		if swinger, ok := ctx.World.combatEngine.(interface {
			PerformPendingBodySwing(combat.Combatant, combat.Combatant) error
		}); ok {
			if err := swinger.PerformPendingBodySwing(mob, victim); err != nil {
				slog.Error("forced mobile hit failed", "mob", mob.GetName(), "victim", victim.GetName(), "error", err)
			}
		}
		return CharacterCommandHandled
	}
	if hitter, ok := ctx.World.combatEngine.(interface {
		PerformDeferredInitialAttack(combat.Combatant, combat.Combatant) error
	}); ok {
		if err := hitter.PerformDeferredInitialAttack(mob, victim); err != nil {
			slog.Error("forced mobile hit failed", "mob", mob.GetName(), "victim", victim.GetName(), "error", err)
		}
		return CharacterCommandHandled
	}
	if err := ctx.World.combatEngine.StartCombat(mob, victim); err != nil {
		slog.Error("forced mobile hit failed", "mob", mob.GetName(), "victim", victim.GetName(), "error", err)
	}
	return CharacterCommandHandled
}
