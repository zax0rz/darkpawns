package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// SpecialResult is the result of offering a command to one native special.
// Unsupported is distinct from Declined: the former must keep the NPC command
// path blocked while this special family is unported; the latter allows C's
// ordered dispatcher to offer the command to the next special or handler.
type SpecialResult uint8

const (
	SpecialUnsupported SpecialResult = iota
	SpecialDeclined
	SpecialHandled
)

// SpecEntry carries the old player callback and an optional callback that
// receives the actual command actor. NPCFallback applies only when the entry
// has no actor-aware callback.
type SpecEntry struct {
	Legacy      SpecFunc
	Actor       ActorSpecFunc
	NPCFallback SpecialResult
}

// CharacterSpecialContext keeps the real body separate from the descriptor
// that may control it. PrivateOutput is supplied by the owning session when
// the actor is switched; a descriptor-less mob has no private recipient.
type CharacterSpecialContext struct {
	Actor         combat.Combatant
	PrivateOutput func(string)
}

// ActActor returns the actual actor body so Act retains its visibility,
// position, and identity semantics.
func (ctx CharacterSpecialContext) ActActor() Actor {
	return ctx.Actor
}

// ActorSpecFunc receives the actual C command actor and the special host.
// Returning SpecialUnsupported is not a decline: the shared dispatcher must
// stop and retain an explicit coverage gap instead of silently running the
// command handler.
type ActorSpecFunc func(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult

// DispatchCharacterSpecial offers one registered special to the real actor.
// It preserves the old player callback for *Player bodies, invokes an
// actor-aware callback for NPC bodies, and reports unported legacy-only NPC
// specials explicitly. It does not manufacture a Player or Session.
func DispatchCharacterSpecial(name string, w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult {
	entry, ok := SpecRegistry[name]
	if !ok || (entry.Legacy == nil && entry.Actor == nil) {
		return SpecialUnsupported
	}
	if ctx.Actor == nil {
		return SpecialUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		if entry.Legacy == nil {
			return SpecialUnsupported
		}
		if entry.Legacy(w, player, me, cmd, arg) {
			return SpecialHandled
		}
		return SpecialDeclined
	}
	if !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	if entry.Actor != nil {
		result := entry.Actor(w, ctx, me, cmd, arg)
		if result == SpecialDeclined || result == SpecialHandled || result == SpecialUnsupported {
			return result
		}
		return SpecialUnsupported
	}
	if entry.NPCFallback == SpecialDeclined {
		return npcSpecialFallback(name, cmd)
	}
	return SpecialUnsupported
}

func init() {
	RegisterActorSpec("no_move_east", specNoMoveEastActor)
}

// specNoMoveEastActor ports the NPC command-time branch of C's
// SPECIAL(no_move_east), src/spec_procs2.c:1823-1842. The existing player
// callback remains the player lane and is intentionally unchanged.
func specNoMoveEastActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	if me == nil || ctx.Actor == nil {
		return SpecialUnsupported
	}
	if !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	// C's AWAKE(mobile) is GET_POS(mobile) > POS_SLEEPING. C also rejects
	// mini_mud (src/spec_procs2.c:1827); the Go tree has no mini_mud runtime
	// or startup flag, so this port always runs its ordinary-mode branch.
	if cmd == "" || !isMoveCmd(cmd) || me.GetPosition() <= combat.PosSleeping {
		return SpecialDeclined
	}
	// HUNTING(ch) is an NPC runtime facet; do not infer false for a body whose
	// hunting state is unavailable.
	hunter, ok := ctx.Actor.(interface{ IsHunting() bool })
	if !ok {
		return SpecialUnsupported
	}
	if ctx.Actor.GetLevel() >= LVL_IMMORT || hunter.IsHunting() {
		return SpecialDeclined
	}
	if cmd != "east" || ctx.Actor.GetLevel() >= LVL_IMMORT { // C's second level guard uses LEVEL_IMMORT.
		return SpecialDeclined
	}
	actingBody := ctx.ActActor()
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n humiliates $N, and blocks $S way.", "", ToNotVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n humiliates you and blocks your way.", "", ToVict, ctx.Actor, ctx.PrivateOutput)
	return SpecialHandled
}
