package game

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// The eleven class-3 specials of DP-1448 act before they decline, or act on
// every command regardless of the actor, so an NPC actor could not be answered
// by the two decline tables. C's special() (src/interpreter.c:1407-1478) hands
// each of them the real actor and the actor's room; these ports run the same
// C body for a mobile actor and report C's own TRUE/FALSE.
//
// Two C call paths reach them:
//
//   - command_interpreter (src/interpreter.c:947) for every typed command, and
//     for every command an ordered pet is given (do_order,
//     src/act.offensive.c:332); and
//   - do_simple_move(ch, dir, need_specials_check) for a charmed follower
//     (src/act.movement.c:347-349), where special(ch, dir + 1, "") is offered
//     in the room the follower is leaving.
//
// room_activity never runs a room special for a mobile
// (src/comm.c:722-725), so the pulse lane stays player-only.

// cRoomPeople returns the room's occupants in C's world[room].people order.
// char_to_room front-inserts (src/handler.c), so the list is newest-first;
// RoomEntrySequence records that insertion for players and mobiles alike.
func cRoomPeople(w *World, roomVNum int) []Actor {
	if w == nil || roomVNum < 0 {
		return nil
	}
	type occupant struct {
		actor Actor
		seq   uint64
		name  string
	}
	var people []occupant
	for _, mob := range w.GetMobsInRoom(roomVNum) {
		// No MOB_EXTRACT filter: C's room-special bodies walk
		// world[ch->in_room].people unguarded (start_room,
		// src/spec_procs.c:2209; the Elemental Temple loops,
		// src/spec_procs3.c:948, 1012, 1164, 1195, 1254), so a body flagged
		// for extraction earlier in the same pulse is still an occupant and is
		// moved like any other. Only the *mob special* call in special() skips
		// flagged bodies (src/interpreter.c:1453), and that guard lives in
		// pkg/session/character_dispatch.go and pkg/game/flee.go.
		if mob == nil {
			continue
		}
		people = append(people, occupant{actor: mob, seq: mob.GetRoomEntrySequence(), name: mob.GetName()})
	}
	for _, player := range w.GetPlayersInRoom(roomVNum) {
		if player == nil {
			continue
		}
		people = append(people, occupant{actor: player, seq: player.GetRoomEntrySequence(), name: player.GetName()})
	}
	sort.SliceStable(people, func(i, j int) bool {
		if people[i].seq != people[j].seq {
			return people[i].seq > people[j].seq
		}
		return people[i].name < people[j].name
	})
	out := make([]Actor, 0, len(people))
	for _, person := range people {
		out = append(out, person.actor)
	}
	return out
}

// cBody is the part of a character C's special() body reads: the Actor roles
// act() needs plus the combat surface damage(), hit() and level gates read.
type cBody interface {
	Actor
	combat.Combatant
}

// specialActor returns the actor as the special bodies need it, or nil.
func specialActor(ctx CharacterSpecialContext) cBody {
	actor, ok := ctx.Actor.(cBody)
	if !ok {
		return nil
	}
	return actor
}

// elementTeleportPerson runs one iteration of the element rooms' C loop: the
// direct line (TO_CHAR, which a descriptor-less mobile never receives), the
// departure act, the checked char_from_room/char_to_room pair, the optional
// destination look (players only: look_at_room needs a descriptor) and the
// arrival act.
func elementTeleportPerson(w *World, body Actor, destination int, direct, departure, arrival string, look, rawDirect bool, logMsg string) {
	if direct != "" {
		if player, ok := body.(*Player); ok {
			if rawDirect {
				// C: send_to_char(buf, ppl) — the buffer carries its own line
				// endings and nothing appends to it (galeru_alive,
				// src/spec_procs3.c:1204). act() would add one more line ending
				// and print an extra blank line before the destination look.
				player.SendMessage(direct)
			} else {
				Act(w, false, player, nil, nil, nil, direct, "", ToChar)
			}
		}
	}
	Act(w, true, body, nil, nil, nil, departure, "", ToNotVict)

	var err error
	switch subject := body.(type) {
	case *Player:
		err = w.PlayerTransfer(subject, destination)
	case *MobInstance:
		err = w.MobTransfer(subject, destination)
	}
	if err != nil {
		slog.Warn(logMsg, "character", body.GetName(), "room", destination, "error", err)
		return
	}
	if look {
		if player, ok := body.(*Player); ok {
			w.lookAtRoom(player, false)
		}
	}
	Act(w, true, body, nil, nil, nil, arrival, "", ToNotVict)
}

// ================================================================
// whirlpool — src/spec_procs2.c:244-275
// ================================================================

// specWhirlpoolPulls runs SPECIAL(whirlpool) for one room. C reads only
// ch->in_room, then pulls every occupant that is not PRF_NOHASSLE, not the
// whirlpool mobile and not an NPC. `vict != mobile` can never hold here: the
// mobile is an NPC and the loop skips NPCs.
func specWhirlpoolPulls(w *World, roomVNum int) bool {
	if w == nil || roomVNum < 0 {
		return false
	}
	lo, loOK := w.RealRoomIndex(4600)
	hi, hiOK := w.RealRoomIndex(4699)
	if !loOK || !hiOK {
		// C's real_room would yield NOWHERE here; the stock world always has
		// both endpoint rooms, so this is unreachable in practice.
		return false
	}
	occurred := false
	for _, body := range cRoomPeople(w, roomVNum) {
		if body.IsNPC() {
			continue
		}
		player, ok := body.(*Player)
		if !ok || player.GetFlags()&(1<<uint(PrfNohassle)) != 0 {
			continue
		}
		// C: do { to_room = number(real_room(4600), real_room(4699)); }
		//     while (PRIVATE || GODROOM || DEATH || NOMOB);
		// One draw per rejection iteration, over the same absolute rnum
		// bounds C uses, so the shared stream stays aligned (R3a).
		var toRoom int
		for {
			// #nosec G404 — game RNG, not cryptographic
			idx := dprng.Number(lo, hi)
			candidate, ok := w.RoomVNumByIndex(idx)
			if !ok {
				continue
			}
			if w.roomHasFlag(candidate, "private") || w.roomHasFlag(candidate, "godroom") ||
				w.roomHasFlag(candidate, "death") || w.roomHasFlag(candidate, "nomob") {
				continue
			}
			toRoom = candidate
			break
		}
		player.SetRoom(toRoom)
		// C's strings (src/spec_procs2.c:268-269) carry LFCR framing; the
		// telnet transport canonicalizes line endings for both engines, so the
		// CRLF spelling rides the standard pipeline without sendToChar's extra
		// appended terminator (which doubled the blank lines).
		player.SendMessage("A ravaging whirlpool sucks you under!\r\n")
		player.SendMessage("You finally surface, sputtering...\r\n\r\n")
		w.LookAtRoom(player, false)
		occurred = true
	}
	return occurred
}

// specWhirlpoolActor offers SPECIAL(whirlpool) to a mobile command actor.
func specWhirlpoolActor(w *World, ctx CharacterSpecialContext, _ *MobInstance, _, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if specWhirlpoolPulls(w, actor.GetRoom()) {
		return SpecialHandled
	}
	return SpecialDeclined
}

// ================================================================
// quan_lo — src/spec_procs3.c:372-397
// ================================================================

// specQuanLoBody runs SPECIAL(quan_lo). C reads the actor's name in both
// messages and never consumes the command: every arm falls through to
// return(FALSE).
func specQuanLoBody(w *World, actor cBody, me *MobInstance, cmd, arg string) bool {
	if w == nil || actor == nil || me == nil {
		return false
	}
	if cmd != "" && me.GetPosition() > combat.PosSleeping {
		if cmd == "flee" || cmd == "retreat" || cmd == "escape" {
			w.NPCCommandDirect(me, "gossip", "What was that, "+actor.GetName()+
				"? This is not a shawade. Try it again. This time with fewing.")
		}
		arg = strings.TrimSpace(arg)
		if (cmd == "look" || cmd == "examine") && arg != "" && isCName(arg, charKeywords(me)) {
			// C builds the act with $n for the mobile and GET_NAME(ch) for the
			// actor (src/spec_procs3.c:390-392).
			Act(w, true, me, actor, nil, nil, "$n says, 'What is it you seek, "+
				actor.GetName()+"? Tell me and be gone.'", "", ToRoom)
		}
	}
	return false
}

// specQuanLoActor offers SPECIAL(quan_lo) to a mobile command actor.
func specQuanLoActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	specQuanLoBody(w, actor, me, cmd, arg)
	return SpecialDeclined
}

// ================================================================
// elements_minion — src/spec_procs3.c:1217-1240
// ================================================================

// specElementsMinionBody runs SPECIAL(elements_minion): a command-independent
// scan of the minion's own visible carrying list, once per keyword, destroying
// the first match of each pass. C returns FALSE whatever it finds, so the
// command it interrupted continues untouched.
func specElementsMinionBody(w *World, me *MobInstance) {
	if w == nil || me == nil {
		return
	}
	for _, keyword := range []string{"talisman", "element", "earth", "fire", "water", "air"} {
		var match *ObjectInstance
		for _, obj := range me.Inventory {
			if obj != nil && canSeeObject(me, obj) && isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				match = obj
				break
			}
		}
		if match == nil {
			continue
		}
		Act(w, true, me, nil, match, nil,
			"$n utters the words 'eradico paratus' and $p disintegrates.", "", ToRoom)
		w.ExtractObject(match, me.GetRoomVNum())
		elementsRemoveCylinders(w, me.GetRoomVNum())
	}
}

// specElementsMinionActor offers SPECIAL(elements_minion) to a mobile command
// actor. The minion acts on its own inventory, never on the actor, and always
// declines.
func specElementsMinionActor(w *World, ctx CharacterSpecialContext, me *MobInstance, _, _ string) SpecialResult {
	if specialActor(ctx) == nil {
		return SpecialUnsupported
	}
	specElementsMinionBody(w, me)
	return SpecialDeclined
}

// ================================================================
// elements_guardian — src/spec_procs3.c:1242-1289
// ================================================================

// specElementsGuardianBody runs SPECIAL(elements_guardian) for one room. C's
// `if (!cmd) return FALSE;` is the opener; every arm below it returns FALSE.
func specElementsGuardianBody(w *World, me *MobInstance, roomVNum int) bool {
	if w == nil || me == nil {
		return false
	}
	people := cRoomPeople(w, roomVNum)
	for i, person := range people {
		ppl, ok := person.(*Player)
		if !ok || ppl.GetLevel() > LVL_IMMORT || ppl.GetFightingBody() != nil {
			continue
		}

		var next Actor
		if i+1 < len(people) {
			next = people[i+1]
		}
		nextPlayer, nextIsPlayer := next.(*Player)
		if !nextIsPlayer || nextPlayer.GetLevel() > LVL_IMMORT || nextPlayer.GetFightingBody() != nil {
			dam := randRange(10, 50)
			elementsGuardianSelfDamage(w, ppl, dam)
			Act(w, true, me, ppl, nil, nil,
				"$n mumbles softly and $N begins screaming loudly, hitting $Mself.", "", ToNotVict)
			Act(nil, true, me, ppl, nil, nil,
				"$n mumbles softly and you begin to scream, involuntarily hitting yourself.", "", ToVict)
			return false
		}

		Act(w, true, ppl, nextPlayer, nil, nil,
			me.GetName()+" mumbles softly and "+ppl.GetName()+" screams loudly, attacking "+nextPlayer.GetName()+"!", "", ToNotVict)
		Act(nil, true, nextPlayer, nil, nil, nil,
			me.GetName()+" mumbles softly and "+ppl.GetName()+" screams loudly, attacking you!", "", ToChar)
		Act(nil, true, ppl, nil, nil, nil,
			me.GetName()+" mumbles softly and you scream loudly, attacking "+nextPlayer.GetName()+"!", "", ToChar)
		elementsGuardianPlayerHit(w, ppl, nextPlayer)
		return false
	}
	return false
}

// specElementsGuardianActor offers SPECIAL(elements_guardian) to a mobile
// command actor. The guardian charms players, never mobiles, and always
// declines the command it interrupted.
func specElementsGuardianActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if cmd == "" {
		return SpecialDeclined
	}
	specElementsGuardianBody(w, me, actor.GetRoom())
	return SpecialDeclined
}

// ================================================================
// elements_platforms — src/spec_procs3.c:1004-1024
// ================================================================

// specElementsPlatformsMove runs SPECIAL(elements_platforms): every occupant is
// sent back to 1314 with the same three acts, and C does not skip mobiles.
func specElementsPlatformsMove(w *World, roomVNum int) {
	for _, body := range cRoomPeople(w, roomVNum) {
		elementTeleportPerson(w, body, 1314,
			"A wave of power surges through you and you feel dizzy.",
			"$n disappears in a brilliant flash of light.",
			"$n appears in a brilliant flash of light.",
			false, false, "elements platforms character transfer failed")
	}
}

func specElementsPlatformsActor(w *World, ctx CharacterSpecialContext, _ *MobInstance, _, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	specElementsPlatformsMove(w, actor.GetRoom())
	return SpecialHandled
}

// ================================================================
// elements_galeru_alive — src/spec_procs3.c:1184-1215
// ================================================================

// specElementsGaleruAliveMoves runs SPECIAL(elements_galeru_alive) for one room
// once C's mob-1315 gate passes. C walks every occupant, not just players.
func specElementsGaleruAliveMoves(w *World, roomVNum int) bool {
	if w == nil {
		return false
	}
	for _, mob := range w.GetMobsInRoom(roomVNum) {
		if mob != nil && mob.GetVNum() == 1315 {
			return false
		}
	}
	for _, body := range cRoomPeople(w, roomVNum) {
		elementTeleportPerson(w, body, 1395,
			"You begin to feel very dizzy and the world around you fades...\r\n\n",
			"$n disappears in a brilliant flash of light.",
			"$n appears in a brilliant flash of light.",
			true, true, "elements galeru alive character transfer failed")
	}
	return true
}

// specElementsGaleruAliveActor offers SPECIAL(elements_galeru_alive) to a mobile
// command actor. C's `if (!cmd) return FALSE;` is the opener, so only a typed
// command reaches the transfer.
func specElementsGaleruAliveActor(w *World, ctx CharacterSpecialContext, _ *MobInstance, cmd, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if cmd == "" {
		return SpecialDeclined
	}
	if specElementsGaleruAliveMoves(w, actor.GetRoom()) {
		return SpecialHandled
	}
	return SpecialDeclined
}

// ================================================================
// carrion — src/spec_procs2.c:1726-1754
// ================================================================

// carrionStalkerVnum is the mobile C's carrion loads (src/spec_procs2.c:1742).
const carrionStalkerVnum = 14308

// specCarrionBody runs SPECIAL(carrion) for the actor in its room. C reads the
// actor's awake state, level and room; the stalker it loads takes the actor's
// level as its own level and damage roll, and then attacks the actor itself.
func specCarrionBody(w *World, actor cBody, cmd, arg string) bool {
	if w == nil || actor == nil {
		return false
	}
	// C: if ((!AWAKE(ch) && GET_HIT(ch)>0) || mini_mud) return (FALSE);
	if actor.GetPosition() <= combat.PosSleeping && actor.GetHP() > 0 {
		return false
	}
	if cmd == "" {
		return false
	}
	// C: skip_spaces(&argument), then a substring test for corpse/corpses/pile.
	arg = strings.TrimLeft(arg, " \t\r\n\v\f")
	if arg == "" || (!strings.Contains(arg, "corpse") && !strings.Contains(arg, "corpses") && !strings.Contains(arg, "pile")) {
		return false
	}
	stalker, err := w.spawnMobQuiet(carrionStalkerVnum, actor.GetRoom())
	if err != nil {
		slog.Warn("carrion stalker load failed", "room", actor.GetRoom(), "error", err)
		return false
	}
	stalker.SetLevel(actor.GetLevel())
	stalker.SetDamroll(actor.GetLevel())
	Act(w, true, stalker, nil, nil, nil,
		"Suddenly $n skitters from out of a corpse!", "", ToRoom)
	if err := w.mobHit(stalker, actor); err != nil {
		slog.Warn("carrion stalker attack failed", "mob", stalker.GetName(), "error", err)
	}
	return true
}

// specCarrionActor offers SPECIAL(carrion) to a mobile command actor.
func specCarrionActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult {
	// carrion is a room special (src/spec_assign.c:587); C passes no mobile
	// receiver, and a non-nil one would mean the assignment moved.
	if me != nil {
		return SpecialUnsupported
	}
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if specCarrionBody(w, actor, cmd, arg) {
		return SpecialHandled
	}
	return SpecialDeclined
}

// ================================================================
// dump — src/spec_procs.c:254-293
// ================================================================

// specDumpActor offers SPECIAL(dump) to a mobile command actor. C destroys every
// object in the actor's room before it inspects the command at all, then acts
// on a drop. The room-clearing arm is ported; the drop arm calls do_drop for the
// actor, and the port has no mobile do_drop yet (the drop command itself has no
// NPC actor handler), so that one command stays an explicit gap rather than a
// silent partial run. Every other command C answers 0 and the port declines.
func specDumpActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult {
	if me != nil {
		return SpecialUnsupported // dump is a room special (src/spec_assign.c:573)
	}
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	_ = w.roomCleanup(actor.GetRoom())
	if cmd != "drop" {
		return SpecialDeclined
	}
	return SpecialUnsupported
}

// ================================================================
// start_room — src/spec_procs.c:2204-2258
// ================================================================

// specStartRoomBody runs SPECIAL(start_room) for one room. C walks the room's
// people and returns FALSE the moment it meets an immortal — after any mortal
// it already moved — then relocates every other body to its hometown altar
// (a mobile has no hometown: read_mobile leaves player.hometown at 0, so the
// switch takes its default arm, real_room(8004)).
func specStartRoomBody(w *World, roomVNum int) bool {
	if w == nil {
		return false
	}
	for _, occupant := range cRoomPeople(w, roomVNum) {
		body, ok := occupant.(cBody)
		if !ok {
			continue
		}
		if body.GetLevel() >= lvlImmort {
			return false
		}
		if player, ok := body.(*Player); ok {
			player.SendMessage(startRoomBirthMessage(player.GetName()))
			if err := w.PlayerTransfer(player, NewbieHometownRoom(player.GetHometown())); err != nil {
				slog.Warn("start_room player transfer failed", "player", player.GetName(), "error", err)
				continue
			}
			// C's do_look renders the fresh mortal's default PRF_AUTOEXIT state,
			// which is on — the oracle's hometown look carries "[ Exits: … ]".
			w.lookAtRoom(player, false)
			continue
		}
		mob, ok := body.(*MobInstance)
		if !ok {
			continue
		}
		// C's send_to_char and do_look write nothing to a descriptor-less
		// mobile; preserve the command call after the same room transfer.
		if err := w.MobTransfer(mob, NewbieHometownRoom(0)); err != nil {
			slog.Warn("start_room mobile transfer failed", "mob_vnum", mob.GetVNum(), "error", err)
			continue
		}
		w.NPCCommandDirect(mob, "look", "")
	}
	return true
}

// specStartRoomActor offers SPECIAL(start_room) to a mobile command actor.
func specStartRoomActor(w *World, ctx CharacterSpecialContext, me *MobInstance, _, _ string) SpecialResult {
	if me != nil {
		return SpecialUnsupported // start_room is a room special
	}
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if specStartRoomBody(w, actor.GetRoom()) {
		return SpecialHandled
	}
	return SpecialDeclined
}

// ================================================================
// elements_master_column — src/spec_procs3.c:936-1002
// ================================================================

// specElementsMasterColumnMoves runs SPECIAL(elements_master_column) for one
// room. Its has_object array is declared once and reset inside the else arm, so
// C's per-person scan carries one quirk worth naming: a carried talisman at
// index i leaves has_object[i] set for the next person only when an earlier
// index was found first. The loop below keeps C's exact reset order.
func specElementsMasterColumnMoves(w *World, roomVNum int) {
	newLocs := []int{1320, 1331, 1342, 1353, 1372}
	objNames := []string{"earth", "air", "fire", "water"}
	hasObject := [4]bool{}

	for _, body := range cRoomPeople(w, roomVNum) {
		var carried []*ObjectInstance
		switch subject := body.(type) {
		case *Player:
			carried = subject.GetInventory()
		case *MobInstance:
			carried = subject.Inventory
		}
		for _, obj := range carried {
			if obj == nil {
				continue
			}
			switch obj.GetVNum() {
			case 1300:
				hasObject[0] = true
			case 1301:
				hasObject[1] = true
			case 1302:
				hasObject[2] = true
			case 1303:
				hasObject[3] = true
			}
		}

		found := 0
		for i := range hasObject {
			if !hasObject[i] {
				break
			}
			found++
			hasObject[i] = false
		}

		var message string
		switch found {
		case 0:
			message = "You feel a tingling sensation and your vision fades. When you wake...\r\n"
		case len(objNames):
			message = "The four talismans glow softly and your vision fades. When you wake...\r\n"
		default:
			message = "The talisman of " + objNames[found-1] + " glows softly and your vision fades. When you wake...\r\n"
		}

		// C's act format has no leading space (src/spec_procs3.c:998): the space
		// that appears before this line is do_description's trailing space,
		// written as the first byte of the next line when the destination room
		// lists extra descriptions (src/act.informative.c:2779). The port's
		// ObservationResult carries that byte itself and now emits it with the
		// bracket write, so the act must not add a second one.
		elementTeleportPerson(w, body, newLocs[found], message,
			"$n vanishes in a brilliant flash of light.",
			"$n appears in a brilliant flash of light.",
			true, false, "elements master column character transfer failed")
	}
}

// specElementsMasterColumnActor offers SPECIAL(elements_master_column) to a
// mobile command actor. C always returns TRUE on this path.
func specElementsMasterColumnActor(w *World, ctx CharacterSpecialContext, _ *MobInstance, _, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	specElementsMasterColumnMoves(w, actor.GetRoom())
	return SpecialHandled
}

// ================================================================
// elements_galeru_column — src/spec_procs3.c:1137-1182
// ================================================================

// specElementsGaleruColumnMoves runs SPECIAL(elements_galeru_column) for one
// room. C returns FALSE until all four talismans stand in their rooms, then
// moves every non-NPC occupant (the comment says "Do not teleport NPCs" and
// the IS_NPC continue enforces it).
func specElementsGaleruColumnMoves(w *World, roomVNum int) bool {
	if w == nil {
		return false
	}
	roomVnums := []int{1360, 1364, 1380, 1384}
	talVnums := []int{1300, 1301, 1302, 1303}
	found := 0
	for i := 0; i < 4; i++ {
		for _, item := range w.GetItemsInRoom(roomVnums[i]) {
			if item != nil && item.GetVNum() == talVnums[i] {
				found++
				break
			}
		}
	}
	if found != 4 {
		return false
	}

	for _, body := range cRoomPeople(w, roomVNum) {
		player, ok := body.(*Player)
		if !ok {
			continue // C: if (IS_NPC(ppl)) continue;
		}
		// C's send_to_char buffer already contains its unusual CRLF/LF
		// spacing; send the bytes directly instead of letting Act append
		// another line end.
		player.SendMessage("Four beams of colored light from the corners of the chamber converge around you.\r\n\n")
		Act(w, true, player, nil, nil, nil,
			"$n is struck by four beams of colored light and slowly vanishes!", "", ToNotVict)
		if err := w.PlayerTransfer(player, 1389); err != nil {
			slog.Warn("elements galeru column player transfer failed", "player", player.GetName(), "error", err)
			continue
		}
		w.lookAtRoom(player, false)
		// C: act("$n materialises from nowhere in a swirl of colors.", ...)
		// with no leading space (src/spec_procs3.c:1181); the space in front of
		// it is do_description's trailing space arriving as the first byte of
		// this line, which the look above now emits itself.
		Act(w, true, player, nil, nil, nil,
			"$n materialises from nowhere in a swirl of colors.", "", ToNotVict)
	}
	return true
}

// specElementsGaleruColumnActor offers SPECIAL(elements_galeru_column) to a
// mobile command actor.
func specElementsGaleruColumnActor(w *World, ctx CharacterSpecialContext, _ *MobInstance, _, _ string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil {
		return SpecialUnsupported
	}
	if specElementsGaleruColumnMoves(w, actor.GetRoom()) {
		return SpecialHandled
	}
	return SpecialDeclined
}

func init() {
	// SPECIAL(carrion), src/spec_procs2.c:1726-1754 — room 14305.
	RegisterActorSpec("carrion", specCarrionActor)
	// SPECIAL(dump), src/spec_procs.c:254-293 — rooms 8085 and 21223.
	RegisterActorSpec("dump", specDumpActor)
	// SPECIAL(start_room), src/spec_procs.c:2204-2258 — room 8099.
	RegisterActorSpec("start_room", specStartRoomActor)
	// SPECIAL(whirlpool), src/spec_procs2.c:244-275 — mob 12200.
	RegisterActorSpec("whirlpool", specWhirlpoolActor)
	// SPECIAL(quan_lo), src/spec_procs3.c:372-397 — mob 19405.
	RegisterActorSpec("quan_lo", specQuanLoActor)
	// Elemental Temple: rooms 1315, 1326/1337/1348/1359, 1372, 1394 and mobs
	// 1313, 1314 (src/spec_assign.c:167-174, 594-598).
	RegisterActorSpec("elements_master_column", specElementsMasterColumnActor)
	RegisterActorSpec("elements_platforms", specElementsPlatformsActor)
	RegisterActorSpec("elements_galeru_column", specElementsGaleruColumnActor)
	RegisterActorSpec("elements_galeru_alive", specElementsGaleruAliveActor)
	RegisterActorSpec("elements_guardian", specElementsGuardianActor)
	RegisterActorSpec("elements_minion", specElementsMinionActor)
}
