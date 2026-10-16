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
	ActorObject ActorObjectSpecFunc
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

// ActorObjectSpecFunc is the object-host lane for a C object special. The
// actual actor and object are both passed through; a legacy-only object
// special remains Unsupported for NPC dispatch.
type ActorObjectSpecFunc func(w *World, ctx CharacterSpecialContext, obj *ObjectInstance, cmd, arg string) SpecialResult

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
	if npcSpecialDeclinesCommand(name, cmd) {
		return SpecialDeclined
	}
	return SpecialUnsupported
}

// npcSpecialDeclinesCommand reports whether the C body already proves that this
// special returns FALSE, with no side effect, for a command an NPC issued.
// Only two proven shapes qualify: the `if (cmd || ...) return FALSE;` opener
// (cSpecialDeclinesEveryCommand) and the "acts only on the named commands"
// body (cSpecialInterceptsOnly). cmd == "" is never declined: an empty command
// is C's pulse call, a different lane from the typed-command dispatch
// (src/interpreter.c:1407-1456 against src/mobact.c's func(ch, ch, 0, "")), so
// its outcome cannot be inferred from the command-time body.
func npcSpecialDeclinesCommand(name, cmd string) bool {
	if cmd == "" {
		return false
	}
	if cSpecialDeclinesEveryCommand[name] {
		return true
	}
	if intercepted, listed := cSpecialInterceptsOnly[name]; listed && !intercepted[cmd] {
		return true
	}
	return false
}

// DispatchCharacterObjectSpecial preserves legacy player callbacks and
// requires an explicit actor callback for NPCs. A missing assigned handler is
// Unsupported so the shared interpreter cannot silently bypass an applicable
// C special.
func DispatchCharacterObjectSpecial(name string, w *World, ctx CharacterSpecialContext, obj *ObjectInstance, cmd, arg string) SpecialResult {
	if ctx.Actor == nil || obj == nil {
		return SpecialUnsupported
	}
	if player, ok := ctx.Actor.(*Player); ok {
		if fn := GetObjSpecForObject(obj.VNum); fn != nil {
			if fn(w, player, obj, cmd, arg) {
				return SpecialHandled
			}
			return SpecialDeclined
		}
		return SpecialUnsupported
	}
	if !ctx.Actor.IsNPC() {
		return SpecialUnsupported
	}
	if cmd != "" && cSpecialDeclinesEveryCommand[name] {
		return SpecialDeclined
	}
	entry, ok := SpecRegistry[name]
	if ok && entry.ActorObject != nil {
		result := entry.ActorObject(w, ctx, obj, cmd, arg)
		if result == SpecialHandled || result == SpecialDeclined {
			return result
		}
		return SpecialUnsupported
	}
	// An object special with no actor-aware callback still carries the
	// command-time C semantics the two tables prove; consult them so a
	// descriptor-less mobile is not blocked out of every command by an object
	// whose C body already declines it.
	if npcSpecialDeclinesCommand(name, cmd) {
		return SpecialDeclined
	}
	return SpecialUnsupported
}

func init() {
	RegisterActorSpec("no_move_east", specNoMoveEastActor)
	RegisterActorObjectSpec("gen_board", specGenBoardActor)
	RegisterActorSpec("conjured", specConjuredActor)
}

// specConjuredActor ports the command-time branch of C's SPECIAL(conjured),
// src/spec_procs3.c:859-892, for an NPC actor. A charmed conjured creature
// returns FALSE for every command and does nothing at all, which is the state a
// summoned elemental is normally in; without this lane its room blocked every
// command any pet was given. An uncharmed conjured body leaves the world on the
// next call (the act/extract pair at :875-888), and that branch stays an
// explicit gap rather than a silent decline.
func specConjuredActor(_ *World, _ CharacterSpecialContext, me *MobInstance, _, _ string) SpecialResult {
	if me != nil && me.IsAffected(affCharm) {
		return SpecialDeclined
	}
	return SpecialUnsupported
}

// specGenBoardActor ports the NPC command-time branch of C's SPECIAL(gen_board),
// src/boards.c:179-210. C's first statement is `if (!ch->desc) return 0;`
// (src/boards.c:186-187): a descriptor-less body — a pet, a charmed follower, or
// any mobile driven without a session — declines every command before the
// command gate. A switched mobile keeps its descriptor, so the CMD_WRITE /
// CMD_LOOK / CMD_EXAMINE / CMD_READ / CMD_REMOVE gate still applies and every
// other command returns 0. The board body itself (write/read/remove and the
// long-display editor) stays in the player lane, so the named commands report
// Unsupported for an NPC actor rather than running half of it.
func specGenBoardActor(_ *World, ctx CharacterSpecialContext, _ *ObjectInstance, cmd, _ string) SpecialResult {
	if ctx.PrivateOutput == nil {
		return SpecialDeclined
	}
	switch cmd {
	case "write", "look", "examine", "read", "remove":
		return SpecialUnsupported
	}
	return SpecialDeclined
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

// cSpecialDeclinesEveryCommand lists the C specials whose first statement is
// `if (cmd || ...) return FALSE;`, with no side effect before it: they act only
// on the pulse (cmd == 0) and decline every typed command, whoever types it.
// For an NPC issuing a command (an ordered pet, a follower's perform_move), the
// faithful answer is therefore "declined", not "unsupported". Without this, a
// charmed follower could neither move nor obey orders in any room holding one
// of these mobiles (Egon's playtest report, 2026-10-16).
//
// Derived mechanically from src/*.c: the SPECIAL's first non-declaration
// statement is an early return whose || operands include a bare `cmd`
// (butler's initializers call only get_obj_in_list_vis). Specials that test
// particular commands (CMD_IS, IS_MOVE) are not here; they need per-command
// ports (DP-1448).
var cSpecialDeclinesEveryCommand = map[string]bool{
	"backstabber":         true,
	"black_undead_knight": true,
	"brain_eater":         true,
	"breed_killer":        true,
	"butler":              true,
	"citizen":             true,
	"cityguard":           true,
	"cleric":              true,
	"conductor":           true,
	"dragon_breath":       true,
	"elemental_room":      true,
	"eq_thief":            true,
	"fearface":            true,
	"fido":                true,
	"field_object":        true,
	"fighter":             true,
	"ira":                 true,
	"jail":                true,
	"janitor":             true,
	"little_boy":          true,
	"magic_user":          true,
	"mallory":             true,
	"mickey":              true,
	"mindflayer":          true,
	"never_die":           true,
	"normal_checker":      true,
	"paladin":             true,
	"pissedalchemist":     true,
	"puff":                true,
	"red_undead_knight":   true,
	"rescuer":             true,
	// roach, src/spec_procs3.c:707-805: `if (cmd || !AWAKE(ch)) return (FALSE);`.
	// The mechanical derivation in PR #2069 anchored its scan to a line holding
	// only `SPECIAL(name)`, so this same-line definition (`SPECIAL(roach) {`) was
	// invisible. conjured, src/spec_procs3.c:859-892, is the other one, and it is
	// an actor port instead: it acts for a body that is not charmed.
	"roach":           true,
	"snake":           true,
	"take_to_jail":    true,
	"teleport_victim": true,
	"teleporter":      true,
	"thief":           true,
	"tipster":         true,
	"troll":           true,
	"zen_master":      true,
}

// cSpecialInterceptsOnly lists C specials that act only on named commands and
// return FALSE, with no side effect before it, for every other command: no
// output, no state change, no number() draw, no act(). An NPC issuing any other
// command is therefore declined; the named commands from an NPC stay
// unsupported until they are ported for NPC actors. An empty set means the C
// body declines every typed command (it acts only on the pulse call).
//
// Entry shape, verified by reading each C function: a leading command gate
// whose failure path returns FALSE immediately, or a command dispatch whose
// fall-through reaches `return FALSE` with nothing written or drawn on the way.
// skip_spaces/one_argument and buffer setup are harmless; anything that writes,
// moves, hits or draws is not.
//
// Classified in DP-1448 (round 8) from the C source; see the round-8 handoff in
// docs/fidelity/depth/handoff/ for the specials that need an NPC port instead.
var cSpecialInterceptsOnly = map[string]map[string]bool{
	// SPECIAL(stableboy), src/spec_procs2.c:315-461: buy, list, stable and
	// collect branches, then return(FALSE).
	"stableboy": {"buy": true, "list": true, "stable": true, "collect": true},

	// --- Town services and shops (DP-1448, town first) ---

	// SPECIAL(assassin), src/spec_procs2.c:845-924: CMD_IS("list"), else
	// CMD_IS("hire"), else return(FALSE). as_loc = ch->in_room + 1 is setup.
	"assassin": {"list": true, "hire": true},
	// SPECIAL(bank), src/spec_procs.c:2345-2399: balance, deposit, withdraw,
	// else return 0. The KD and Kir-Oshi ATMs are object hosts (8034, 18224).
	"bank": {"balance": true, "deposit": true, "withdraw": true},
	// SPECIAL(pet_shops), src/spec_procs.c:1844-1908: CMD_IS("list"), else
	// CMD_IS("buy"), then `/* All commands except list and buy */ return 0;`.
	"pet_shops": {"list": true, "buy": true},
	// SPECIAL(recharger), src/new_cmds2.c:415-486: the opener
	// `if (!(CMD_IS("list") || CMD_IS("help") || CMD_IS("recharge"))) return
	// FALSE;` is the whole gate; every later branch is one of those three.
	"recharger": {"list": true, "help": true, "recharge": true},
	// SPECIAL(clerk), src/spec_procs3.c:75-142: the zone switch runs first and
	// its default arm writes to ch only for a zone where no clerk stands (mob
	// 18228 is zone 182; the switch's other arms are 80 and 212). The command
	// gate is `if (!(CMD_IS("list") || CMD_IS("buy"))) return(FALSE);`.
	"clerk": {"list": true, "buy": true},
	// SPECIAL(con_seller), src/spec_procs3.c:239-314: the CAN_SEE act fires
	// only inside the CMD_IS("buy")||CMD_IS("list") arm; without them the body
	// returns FALSE before any write.
	"con_seller": {"list": true, "buy": true},
	// SPECIAL(prostitute), src/spec_procs3.c:670-705: same shape as con_seller.
	"prostitute": {"list": true, "buy": true},
	// SPECIAL(tattoo1), src/spec_procs2.c:945-1008, SPECIAL(tattoo2),
	// src/spec_procs2.c:1010-1072, SPECIAL(tattoo3), src/spec_procs2.c:1075-1137,
	// SPECIAL(tattoo4), src/spec_procs2.c:1282-1340: CMD_IS("list"), else
	// CMD_IS("buy"), then return(FALSE). tattoo1's buy arm has an IS_NPC guard
	// that returns FALSE; tattoo2-4 have no guard, but none of the four reaches
	// a write or a draw for any other command.
	"tattoo1": {"list": true, "buy": true},
	"tattoo2": {"list": true, "buy": true},
	"tattoo3": {"list": true, "buy": true},
	"tattoo4": {"list": true, "buy": true},
	// SPECIAL(identifier), src/spec_procs2.c:1193-1280: list, value, give, then
	// return(FALSE).
	"identifier": {"list": true, "value": true, "give": true},
	// SPECIAL(mortician), src/spec_procs3.c:807-856: `if(!cmd) return FALSE;`,
	// then CMD_IS("list") and CMD_IS("retrieve"), then return FALSE. cost and
	// skip_spaces are pure setup.
	"mortician": {"list": true, "retrieve": true},
	// SPECIAL(horn), src/spec_procs.c:2401-2420: only CMD_IS("use") can reach a
	// write; a non-use command falls straight through to return(FALSE). The
	// Silent Tower horn is object 14415.
	"horn": {"use": true},
	// SPECIAL(moon_gate), src/gate.c:242-326: `if (!CMD_IS("enter") && !
	// CMD_IS("look")) return FALSE;`. Both arms are command-specific; the rest
	// of the body is setup.
	"moon_gate": {"enter": true, "look": true},

	// --- Town and city mobiles (DP-1448) ---

	// SPECIAL(beholder), src/new_cmds2.c:328-398: `if (cmd) { ... }` where the
	// only TRUE arm is CAST/RECITE and everything else returns FALSE. The whole
	// pulse body (the number() draws and casts) lives behind cmd == 0.
	"beholder": {"cast": true, "recite": true},
	// SPECIAL(cuchi), src/spec_procs.c:1034-1071: `if (!CMD_IS("pat")) return
	// FALSE ;`.
	"cuchi": {"pat": true},
	// SPECIAL(dracula), src/spec_procs.c:1798-1834: the opener
	// `if (!cmd || !CMD_IS("look"))` returns FALSE (delegating to magic_user
	// only on the pulse), so only "look" is intercepted.
	"dracula": {"look": true},
	// SPECIAL(medusa), src/spec_procs2.c:1552-1581: the petrify gate is
	// `!((CMD_IS("look") || CMD_IS("examine")) && isname(argument, ...))` and its
	// failing path returns FALSE before any output.
	"medusa": {"look": true, "examine": true},
	// SPECIAL(no_get), src/new_cmds2.c:552-574: get, palm and take reach the
	// strike; every other command returns FALSE after the AWAKE/HIT gate.
	"no_get": {"get": true, "palm": true, "take": true},
	// SPECIAL(pray_for_items), src/spec_procs.c:2068-2168:
	// `if (!CMD_IS("pray")) return FALSE;`.
	"pray_for_items": {"pray": true},
	// SPECIAL(portal_to_temple), src/spec_procs3.c:515-540: `if(mini_mud ||
	// (!CMD_IS("say") && !CMD_IS("'"))) return(FALSE);`, then a strcasecmp of the
	// argument against "setchswayno"; the mismatch path returns FALSE before
	// do_say.
	"portal_to_temple": {"say": true, "'": true},
	// SPECIAL(recruiter), src/spec_procs3.c:905-930: `if (!cmd) return FALSE;`,
	// then CMD_IS("kill")||CMD_IS("hit") and CMD_IS("cast")||CMD_IS("will"),
	// then return FALSE.
	"recruiter": {"kill": true, "hit": true, "cast": true, "will": true},
	// SPECIAL(suck_in), src/spec_procs.c:2279-2299: `if (mini_mud ||
	// !CMD_IS("look")) return FALSE;`.
	"suck_in": {"look": true},

	// --- Movement guards whose only unnamed-command path is a plain decline
	// (DP-1448) ---

	// SPECIAL(oro_quarters_room), src/spec_procs.c:2302-2318 and
	// SPECIAL(oro_study_room), src/spec_procs.c:2321-2337:
	// `if ( IS_MOB(ch) || !CMD_IS("south") ) return FALSE ;` — an NPC actor is
	// declined unconditionally, before the name comparison and the hit halving.
	"oro_quarters_room": {},
	"oro_study_room":    {},
	// SPECIAL(werewolf), src/spec_procs3.c:427-454: every arm is behind
	// `if (!cmd && FIGHTING(mob) && mob)`, so a typed command always reaches
	// return FALSE without acting.
	"werewolf": {},

	// --- Elemental Temple (DP-1448) ---

	// SPECIAL(elements_load_cylinders), src/spec_procs3.c:1058-1135: only the
	// GET arm (do_get + elements_remove_cylinders) and the DROP arm act; every
	// other command returns FALSE before touching the room contents.
	"elements_load_cylinders": {"get": true, "drop": true},
}
