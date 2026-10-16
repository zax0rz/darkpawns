package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// Movement-guard ports for NPC actors (DP-1448).
//
// C's command interpreter offers every typed command — and every step a mobile
// takes — to the room's specials (src/interpreter.c:1407-1476). Each guard below
// tests the actor, so its body has to run for mobiles too: a charmed pet walking
// or obeying beside a guard must meet the same gate C applies to any other body.
// The player lane keeps its own callbacks; these registrations add the
// actor-aware lane beside them, exactly as specNoMoveEastActor does.
func init() {
	RegisterActorSpec("no_move_west", specNoMoveWestActor)
	RegisterActorSpec("no_move_south", specNoMoveSouthActor)
	RegisterActorSpec("no_move_north", specNoMoveNorthActor)
	RegisterActorSpec("castle_guard_down", specCastleGuardDownActor)
	RegisterActorSpec("castle_guard_up", specCastleGuardUpActor)
	RegisterActorSpec("castle_guard_north", specCastleGuardNorthActor)
	RegisterActorSpec("chosen_guard", specChosenGuardActor)
	RegisterActorSpec("jailguard", specJailGuardActor)
	RegisterActorSpec("outofjailguard", specOutOfJailGuardActor)
	RegisterActorSpec("guild_guard", specGuildGuardActor)
	RegisterActorSpec("wall_guard_ns", specWallGuardNSActor)
	RegisterActorSpec("newbie_zone_entrance", specNewbieZoneEntranceActor)
}

// noMoveGuardSpec carries the per-direction body of the four C SPECIAL(no_move_*)
// procedures. C duplicates the whole shape four times; only the command name and
// the block text differ.
type noMoveGuardSpec struct {
	direction      string
	blockToNotVict string
	blockToVict    string
	// roomLine is C's third act(): the south and north guards also announce to
	// the room. The east and west guards stop after the victim line.
	roomLine string
}

// specNoMoveGuardActor ports C's no_move_* command-time body for an NPC actor:
// `if (!cmd || mini_mud || !IS_MOVE(cmd) || !AWAKE(mobile)) return FALSE;`, then
// the immortal/hunting gate, then the directional block.
func specNoMoveGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd string, spec noMoveGuardSpec) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	// C: !cmd || mini_mud || !IS_MOVE(cmd) || !AWAKE(mobile). The Go tree has no
	// mini_mud runtime flag, so only the ordinary-mode arms exist.
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
	if cmd != spec.direction {
		return SpecialDeclined
	}
	actingBody := ctx.ActActor()
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, spec.blockToNotVict, "", ToNotVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, spec.blockToVict, "", ToVict, ctx.Actor, ctx.PrivateOutput)
	if spec.roomLine != "" {
		ActWithPrivateSink(w, false, me, nil, nil, nil, spec.roomLine, "", ToRoom, ctx.Actor, ctx.PrivateOutput)
	}
	return SpecialHandled
}

func specNoMoveWestActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// SPECIAL(no_move_west), src/spec_procs2.c:2033-2052.
	return specNoMoveGuardActor(w, ctx, me, cmd, noMoveGuardSpec{
		direction:      "west",
		blockToNotVict: "$n humiliates $N, and blocks $S way.",
		blockToVict:    "$n humiliates you and blocks your way.",
	})
}

func specNoMoveSouthActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// SPECIAL(no_move_south), src/spec_procs2.c:2092-2111.
	return specNoMoveGuardActor(w, ctx, me, cmd, noMoveGuardSpec{
		direction:      "south",
		blockToNotVict: "$n blocks $N's way.",
		blockToVict:    "$n blocks your way.",
		roomLine:       "$n says 'Thou shalt not pass.'",
	})
}

func specNoMoveNorthActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// SPECIAL(no_move_north), src/spec_procs2.c:2054-2073. C declares it
	// (src/spec_assign.c:143) without an ASSIGNMOB row, so no live mobile
	// carries it; the port keeps the class complete for the same reason the
	// player lane does.
	return specNoMoveGuardActor(w, ctx, me, cmd, noMoveGuardSpec{
		direction:      "north",
		blockToNotVict: "$n blocks $N's way.",
		blockToVict:    "$n blocks your way.",
		roomLine:       "$n says 'I cannot let you pass.'",
	})
}

// specCastleGuardActor ports C's command-time body (SPECIAL(castle_guard_down),
// src/spec_procs2.c:2134-2174; ..._up, :2176-2216; ..._north, :2218-2258) for an
// NPC actor. C's `if (!cmd)` second-guard scan is a pulse arm that this lane
// never reaches — every dispatch here carries a command — and the player lane
// keeps it.
func specCastleGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd string, spec castleGuardSpec) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	// C: mini_mud || !AWAKE(mobile)
	if me.GetPosition() <= combat.PosSleeping {
		return SpecialDeclined
	}
	// C: GET_LEVEL(ch) >= LVL_IMMORT
	if ctx.Actor.GetLevel() >= LVL_IMMORT {
		return SpecialDeclined
	}
	if cmd != spec.direction {
		return SpecialDeclined
	}
	if me == ctx.Actor {
		return SpecialDeclined // C's (mobile != ch)
	}
	roomVNum := ctx.Actor.GetRoom()
	// C: !is_owner(ch, world[ch->in_room].number+offset). is_owner returns FALSE
	// for an NPC, so only a switched player body could satisfy it.
	if castleGuardOwnerBody(w, ctx.Actor, roomVNum+spec.ownerOffset) {
		return SpecialDeclined
	}
	// C: ch->master && (ch->master == ch || are_grouped(ch, ch->master)) &&
	// is_owner(ch->master, ...). are_grouped(ch, ch->master) roots the group at
	// ch->master, so it reduces to both bodies carrying AFF_GROUP
	// (src/utils.c:655-674).
	if master := w.combatFollowingBody(ctx.Actor); master != nil &&
		(master == ctx.Actor || (actorHasAffect(ctx.Actor, affGroup) && actorHasAffect(master, affGroup))) &&
		castleGuardOwnerBody(w, master, roomVNum+spec.groupedOwnerOffset) {
		actingBody := ctx.ActActor()
		ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n moves aside and allows you to pass.", "", ToVict, ctx.Actor, ctx.PrivateOutput)
		ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n moves aside and allows $N to pass.", "", ToNotVict, ctx.Actor, ctx.PrivateOutput)
		return SpecialDeclined
	}
	actingBody := ctx.ActActor()
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n blocks your way.", "", ToVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n blocks $N's path.", "", ToNotVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, nil, nil, nil, "$n states, 'Thou shalt not pass.'", "", ToRoom, ctx.Actor, ctx.PrivateOutput)
	return SpecialHandled
}

// castleGuardOwnerBody is C's is_owner() (src/spec_procs2.c:1844-1876) for a
// combat body: FALSE for a mobile, the house lookup for a player.
func castleGuardOwnerBody(w *World, body combat.Combatant, roomVNum int) bool {
	player, ok := body.(*Player)
	if !ok {
		return false
	}
	return isOwner(w, player, roomVNum)
}

// actorHasAffect reports a body's affected_by bit without narrowing to a
// concrete type at the call site.
func actorHasAffect(body combat.Combatant, flag int) bool {
	affected, ok := body.(interface{ IsAffected(int) bool })
	return ok && affected.IsAffected(flag)
}

func specCastleGuardDownActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	return specCastleGuardActor(w, ctx, me, cmd, castleGuardSpec{direction: "down", ownerOffset: 2, groupedOwnerOffset: 2})
}

func specCastleGuardUpActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// C's up guard checks a different house coordinate for its grouped-owner arm.
	return specCastleGuardActor(w, ctx, me, cmd, castleGuardSpec{direction: "up", ownerOffset: 1, groupedOwnerOffset: 0})
}

func specCastleGuardNorthActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	return specCastleGuardActor(w, ctx, me, cmd, castleGuardSpec{direction: "north", ownerOffset: 2, groupedOwnerOffset: 2})
}

// jailGuardSpec carries the room gate and the two messages of C's jailguard and
// outofjailguard pair; only the room, direction and text differ.
type jailGuardSpec struct {
	roomVNum    int
	direction   string
	roomLine    string
	privateLine string
}

// specNewbieZoneEntranceActor ports SPECIAL(newbie_zone_entrance),
// src/spec_procs.c:2262-2274: `if ( !CMD_IS("south") ) return FALSE;` and then
// `if ((GET_LEVEL(ch) >= NEWBIE_LEVEL) && (GET_LEVEL(ch) < LEVEL_IMMORT))`
// writes and returns TRUE. The actor's own level decides, so the body has to run
// for a mobile too: a pet below NEWBIE_LEVEL walks south, one above is turned
// back. The refusal is written to the actor, which only a switched body can see.
func specNewbieZoneEntranceActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	if ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	if cmd != "south" {
		return SpecialDeclined
	}
	if ctx.Actor.GetLevel() >= newbieLevel && ctx.Actor.GetLevel() < LVL_IMMORT {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput("Nah, you're too much of a badass to go in there!\r\n")
		}
		return SpecialHandled
	}
	return SpecialDeclined
}

// specJailGuardBody ports C's jailguard/outofjailguard command-time body
// (src/spec_procs.c:1763-1778 and :1781-1796) for an NPC actor:
// `if (mini_mud || !IS_MOVE(cmd)) return(FALSE);`, the immortal/HUNTING gate,
// then the room-and-direction block. The guard is in ch's room when it is
// offered, and C reads ch->in_room, so the actor's room is the gate.
func specJailGuardBody(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd string, spec jailGuardSpec) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	if cmd == "" || !isMoveCmd(cmd) {
		return SpecialDeclined
	}
	hunter, ok := ctx.Actor.(interface{ IsHunting() bool })
	if !ok {
		return SpecialUnsupported
	}
	if ctx.Actor.GetLevel() >= LVL_IMMORT || hunter.IsHunting() {
		return SpecialDeclined
	}
	if ctx.Actor.GetRoom() != spec.roomVNum || cmd != spec.direction {
		return SpecialDeclined
	}
	actingBody := ctx.ActActor()
	ActWithPrivateSink(w, false, actingBody, nil, nil, nil, spec.roomLine, "", ToRoom, ctx.Actor, ctx.PrivateOutput)
	if ctx.PrivateOutput != nil {
		ctx.PrivateOutput(spec.privateLine + "\r\n")
	}
	return SpecialHandled
}

func specJailGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// SPECIAL(jailguard), src/spec_procs.c:1781-1796: room 8118, north.
	return specJailGuardBody(w, ctx, me, cmd, jailGuardSpec{
		roomVNum:    8118,
		direction:   "north",
		roomLine:    "The guard grabs $n with one hand and throws $m back in the room.",
		privateLine: "The guard stops you from leaving with one flabby hand.",
	})
}

func specOutOfJailGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	// SPECIAL(outofjailguard), src/spec_procs.c:1763-1778: room 8117, south.
	return specJailGuardBody(w, ctx, me, cmd, jailGuardSpec{
		roomVNum:    8117,
		direction:   "south",
		roomLine:    "The guard grabs $n by the collar and blocks $s way.",
		privateLine: "The guard stops you from entering with one quick jerk of your collar.",
	})
}

// specGuildGuardActor ports SPECIAL(guild_guard), src/spec_procs.c:569-608, for an
// NPC actor. C has no IS_NPC guard on the guild_info loop, so a mobile of any
// class is blocked at the guild's own doorway: the "wrong class" test is
// `(IS_NPC(ch) || GET_CLASS(ch) != guild_info[i][0])`.
func specGuildGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	actingBody := ctx.ActActor()
	if cmd == "flee" || cmd == "escape" || cmd == "retreat" {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput("You try to flee inside the guild but the guard stops you!\r\n")
		}
		ActWithPrivateSink(w, false, actingBody, nil, nil, nil,
			// C's act format ends with its own \r\n (spec_procs.c:580); act()
			// appends the second — the room's blank line.
			"$n tries to flee inside the guild but the guard block $s way!\r\n", "", ToRoom, ctx.Actor, ctx.PrivateOutput)
		return SpecialHandled
	}
	direction, moving := guildGuardDirection(cmd)
	if !moving || me.IsAffected(affBlind) {
		// C hands a fighting guard to fighter(guard, guard, 0, NULL)
		// (src/spec_procs.c:587): a commandless pulse for the guard itself.
		if me.IsFighting() && specFighter(w, nil, me, "", "") {
			return SpecialHandled
		}
		return SpecialDeclined
	}
	if ctx.Actor.GetLevel() >= LVL_IMMORT || isRemortOnlyClass(ctx.Actor.GetClass()) {
		return SpecialDeclined
	}
	if hunter, ok := ctx.Actor.(interface{ IsHunting() bool }); ok && hunter.IsHunting() {
		return SpecialDeclined
	}
	roomVNum := ctx.Actor.GetRoom()
	for _, e := range GuildInfo {
		if roomVNum != e.Room || direction != e.Direction {
			continue
		}
		ActWithPrivateSink(w, false, actingBody, nil, nil, nil, "The guard humiliates $n, and blocks $s way.", "", ToRoom, ctx.Actor, ctx.PrivateOutput)
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput("The guard humiliates you, and blocks your way.\r\n")
		}
		return SpecialHandled
	}
	return SpecialDeclined
}

// specWallGuardNSActor ports SPECIAL(wall_guard_ns), src/spec_procs2.c:2260-2300,
// for an NPC actor. Every typed command — and a sleeping or fighting guard on
// the pulse — takes C's `return (fighter(ch, ch, 0, NULL))` arm, which runs the
// fighter body with the ACTOR as both the character and the mobile. That is why
// the delegation passes the actor as me and no Player.
func specWallGuardNSActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	actor, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return SpecialUnsupported
	}
	if cmd != "" && me.GetPosition() > combat.PosSleeping && !me.IsFighting() {
		return SpecialDeclined
	}
	if specFighter(w, nil, actor, "", "") {
		return SpecialHandled
	}
	return SpecialDeclined
}

// specChosenGuardActor ports SPECIAL(chosen_guard), src/spec_procs2.c:2113-2132:
// `if (!cmd || !IS_MOVE(cmd) || !AWAKE(mobile)) return FALSE;`, the immortal and
// chosen gate, then the south block. C's IS_CHOSEN is `!IS_NPC(ch) &&
// PLR_FLAGGED(ch, PLR_CHOSEN)` (src/utils.h:422), so an NPC actor never takes
// that arm.
func specChosenGuardActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	if me == nil || ctx.Actor == nil || !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	if cmd == "" || !isMoveCmd(cmd) || me.GetPosition() <= combat.PosSleeping {
		return SpecialDeclined
	}
	if ctx.Actor.GetLevel() >= LVL_IMMORT {
		return SpecialDeclined
	}
	if cmd != "south" {
		return SpecialDeclined
	}
	actingBody := ctx.ActActor()
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n blocks $N's way.", "", ToNotVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, actingBody, nil, nil, "$n blocks your way.", "", ToVict, ctx.Actor, ctx.PrivateOutput)
	ActWithPrivateSink(w, false, me, nil, nil, nil, "$n says 'Thou shalt not pass.'", "", ToRoom, ctx.Actor, ctx.PrivateOutput)
	return SpecialHandled
}
