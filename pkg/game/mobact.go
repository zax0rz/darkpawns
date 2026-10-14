// mobact.go — Ported from src/mobact.c
//
// Mobile AI: generates intelligent (?) behavior in mobiles.
//
// All rights reserved. See LICENSE for license information.

package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mobactNumber is a test seam for verifying the shared CMWC draw order.
// Production always uses the process-wide deterministic stream.
var mobactNumber = dprng.Number

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hasMobFlag(mob *MobInstance, flag string) bool {
	return mob.HasFlag(flag)
}

func roomHasFlag(room *parser.Room, flag string) bool {
	return roomHasNamedFlag(room, flag)
}

// mobCarriedWeight returns the total weight a mob is currently carrying.
// Mirrors C's IS_CARRYING_W(ch) (utils.h). Used by the scavenger check.
func mobCarriedWeight(m *MobInstance) int {
	weight := 0
	for _, obj := range m.Inventory {
		if obj != nil {
			weight += obj.GetWeight()
		}
	}
	return weight
}

// mobMaxCarryWeight returns the maximum weight a mob can carry.
// Mirrors C's CAN_CARRY_W(ch) = str_app[STRENGTH_APPLY_INDEX(ch)].carry_w
// (utils.h:448), including exceptional strength (18/xx).
func mobMaxCarryWeight(m *MobInstance) int {
	return combat.CarryWeight(m.GetStr(), m.GetStrAdd())
}

// mobMaxCarryCount returns the maximum number of items a mob can carry.
// Mirrors C's CAN_CARRY_N(ch) = 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)
// (utils.h:449).
func mobMaxCarryCount(m *MobInstance) int {
	return 5 + (m.GetDex() >> 1) + (m.GetLevel() >> 1)
}

// callMobSpecSafely invokes a mob's registered spec proc during autonomous
// activity (ch=nil, matching the "no player involved" tick path) and
// recovers from any panic. Spec procs are called for every spec-flagged mob
// on every AI tick, and several were ported from C code where `ch` was the
// acting mob itself during this path (not absent) — a proc that dereferences
// `ch` without a nil-check will panic here. One misbehaving spec proc must
// not crash-loop the whole server; this is a safety net, not a substitute
// for fixing the individual procs (BRIEF-2026-07-04-spec-proc-nil-crash.md).
func callMobSpecSafely(specFn SpecFunc, w *World, mob *MobInstance) (handled bool) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("mob spec proc panicked during autonomous activity — skipping this tick",
				"mob", mob.GetName(), "vnum", mob.Proto().VNum, "panic", r)
			handled = false
		}
	}()
	return specFn(w, nil, mob, "", "")
}

func mobIsEvil(mob *MobInstance) bool {
	if mob == nil || mob.Proto() == nil {
		return false
	}
	return mob.GetAlignment() <= -350
}

func mobIsGood(mob *MobInstance) bool {
	if mob == nil || mob.Proto() == nil {
		return false
	}
	return mob.GetAlignment() >= 350
}

// getMobVNumSpec looks up a mob's registered spec proc by its VNum.
func getMobVNumSpec(vnum int) SpecFunc {
	specName, ok := MobSpecAssign[vnum]
	if !ok || specName == "" {
		return nil
	}
	return GetSpec(specName)
}

// ---------------------------------------------------------------------------
// Main AI Dispatch
// ---------------------------------------------------------------------------

// MobileActivity runs the full mobact.c AI tick for every active mob.
// Faithful port of mobile_activity() in src/mobact.c (lines 24-340).
// Acquires per-mob locks for each mob in the iteration.
//
// C macros translated:
//
//	IS_MOB(mob)   → mob.Proto() != nil
//	FIGHTING(mob) → mob.GetFightingBody() != nil
//	AWAKE(mob)    → mob.GetPosition() >= combat.PosSitting
//	MOB_FLAGGED   → hasMobFlag()
//	hit()         → w.combatEngine.StartCombat()
//	GET_HIT       → mob.GetHP()
//	GET_MAX_HIT   → mob.GetMaxHP()
//	CAN_SEE       → canSeePlayer()
//	PRF_NOHASSLE  → hasPrfNoHassle()
//	GET_MOB_RNUM  → mob.Proto().VNum
//	mob_index[].func → getMobVNumSpec()
func (w *World) MobileActivity() {
	w.mu.RLock()
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
	}
	w.mu.RUnlock()

	for _, ch := range mobs {
		if !ch.IsAlive() {
			continue
		}
		// Do NOT hold ch.mu across mobileActivityForMob. That dispatch runs
		// entirely through self-locking accessors (GetFighting, GetPosition,
		// SetStatus, wanderMob → GetRoom, …); holding the write lock here made
		// the very first accessor self-deadlock, wedging the heartbeat on the
		// first awake mob and hanging every player command that scanned its
		// room. The accessors provide their own synchronization. (DP-590)
		ch.mu.RLock()
		ready := ch.Proto() != nil
		ch.mu.RUnlock()
		if !ready {
			continue
		}
		w.mobileActivityForMob(ch)
	}
}

// MobileActivityForMob runs mobact.c AI for a single mob.
// Caller MUST hold mob.mu. Called by runMobAI to avoid O(N²) re-iteration.
func (w *World) MobileActivityForMob(mob *MobInstance) {
	w.mobileActivityForMob(mob)
}

// mobileActivityForMob is the internal per-mob AI dispatch.
// Caller MUST hold mob.mu. Uses direct field access (ch.RoomVNum) instead of
// getter methods to avoid re-entrant lock deadlock.
func (w *World) mobileActivityForMob(ch *MobInstance) {
	// C: if (!IS_MOB(ch) || FIGHTING(ch) || !AWAKE(ch)) continue;
	if ch.GetFightingBody() != nil || ch.GetPosition() <= combat.PosSleeping {
		return
	}

	// Hunter mobs chase their targets two steps at a time. C performs both
	// calls before special procedures and every other per-tick activity.
	if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFightingBody() == nil {
		w.huntVictim(ch)
	}
	if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFightingBody() == nil {
		w.huntVictim(ch)
	}

	// -- MOB_SPEC: special procedure dispatch --
	// C: spec proc returns true to skip to next mob.
	if hasMobFlag(ch, "spec") {
		specFn := getMobVNumSpec(ch.Proto().VNum)
		if specFn != nil && callMobSpecSafely(specFn, w, ch) {
			return
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- Wake sleeper'd mobs --
	if ch.GetPosition() < combat.PosSitting {
		ch.SetStatus("standing")
	}

	// -- Scavenger (pick up best item, ~1-in-10 chance) --
	// C: mobact.c:103-117 — checks CAN_GET_OBJ(ch, obj) (ITEM_WEAR_TAKE,
	// carry weight/count headroom, and CAN_SEE_OBJ) AND GET_OBJ_COST(obj) > max
	// starting with max=1, so items costing 0 or 1 are never picked up.
	if hasMobFlag(ch, "scavenger") {
		items := w.GetItemsInRoom(ch.RoomVNum)
		// C only burns this draw when the room actually has contents.
		// #nosec G404 — game RNG, not cryptographic
		if len(items) > 0 && mobactNumber(0, 10) == 0 {
			maxCost := 1 // C: max = 1; items must cost > 1 to be picked up
			var best *ObjectInstance
			for _, obj := range items {
				if !w.canGetObject(ch, obj) { // C: CAN_GET_OBJ(ch, obj)
					continue
				}
				if obj.GetCost() <= maxCost { // GET_OBJ_COST(obj) > max
					continue
				}
				best = obj
				maxCost = obj.GetCost()
			}
			if best != nil {
				w.RemoveItemFromRoom(best, ch.RoomVNum)
				ch.AddToInventory(best)
				w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s gets %s.", ch.GetName(), best.GetShortDesc()))
			}
		}
	}

	// -- Mob Movement (wandering) --
	// C draws the door unconditionally before checking SENTINEL, position, or
	// whether the exit exists. Every mob that passed the outer gate burns it.
	// #nosec G404 — game RNG, not cryptographic
	door := mobactNumber(0, 18)
	if !hasMobFlag(ch, "sentinel") && ch.GetPosition() == combat.PosStanding {
		if w.wanderMobWithDoor(ch, door) {
			return // C's sector-gate continue skips the rest of this mob's pulse.
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// Sound trigger (C: 1-in-16 chance for mp_sound + lua sound). This draw is
	// unconditional and occurs before all aggression draws.
	// #nosec G404 — game RNG, not cryptographic
	if mobactNumber(0, 15) == 0 {
		w.MpSound(ch)
		// mobact.c:148-158: the sound script, with the mobile as ch and me,
		// only when a player (a non-mobile) is in the room.
		if ch.HasScript("sound") && w.roomHasOtherOccupant(ch, true) {
			w.runPulseScript(ch, "sound")
		}
	}

	// mobact.c:161-175: onpulse_all when anyone else is in the room; a TRUE
	// return ends this mobile's turn.
	if ch.HasScript("onpulse_all") && w.roomHasOtherOccupant(ch, false) {
		if w.runPulseScript(ch, "onpulse_all") {
			return
		}
	}
	if !ch.IsAlive() || ch.HasMobFlag(MobFlagExtract) || ch.RoomVNum < 0 { // utils.c:check_dead
		return
	}

	// mobact.c:180-193: onpulse_pc when a player is in the room.
	if ch.HasScript("onpulse_pc") && w.roomHasOtherOccupant(ch, true) {
		w.runPulseScript(ch, "onpulse_pc")
	}
	if !ch.IsAlive() || ch.HasMobFlag(MobFlagExtract) || ch.RoomVNum < 0 { // utils.c:check_dead
		return
	}

	// -- Aggressive Mobs --
	aggressive := hasMobFlag(ch, "aggressive")
	aggrEvil := hasMobFlag(ch, "aggr_evil")
	aggrGood := hasMobFlag(ch, "aggr_good")
	aggrNeutral := hasMobFlag(ch, "aggr_neutral")
	wimpy := hasMobFlag(ch, "wimpy")
	hasAlignAggr := aggrEvil || aggrGood || aggrNeutral

	if aggressive || hasAlignAggr {
		for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
			if vict.IsNPC() {
				continue
			}
			// C: !CAN_SEE(ch, vict)
			if !w.canSeeCharacterInRoom(ch, vict) {
				continue
			}
			// C: PRF_NOHASSLE
			if vict.GetFlags()&(1<<PrfNohassle) != 0 {
				continue
			}
			// C: MOB_WIMPY && AWAKE(vict)
			if wimpy && vict.GetPosition() > combat.PosSleeping {
				continue
			}
			// C: AFF_PROTECT_EVIL + IS_EVIL(ch) + !number(0,5)
			// #nosec G404 — game RNG, not cryptographic
			if vict.IsAffected(12) && mobIsEvil(ch) && mobactNumber(0, 5) == 0 {
				continue
			}
			// C: AFF_PROTECT_GOOD + IS_GOOD(ch) + !number(0,5)
			// #nosec G404 — game RNG, not cryptographic
			if vict.IsAffected(13) && mobIsGood(ch) && mobactNumber(0, 5) == 0 {
				continue
			}
			// C: AFF_SNEAK + !number(0,3) — 1-in-4 skip
			// #nosec G404 — game RNG, not cryptographic
			if vict.IsAffected(affSneak) && mobactNumber(0, 3) == 0 {
				continue
			}
			// Alignment matching faithful to mobact.c:
			// If NONE of the per-alignment flags are set, hit everyone (plain MOB_AGGRESSIVE).
			// If SOME are set, only hit matching alignments.
			shouldHit := false
			if hasAlignAggr {
				vAlign := vict.GetAlignment()
				if aggrEvil && vAlign <= -350 {
					shouldHit = true
				}
				if aggrNeutral && vAlign > -350 && vAlign < 350 {
					shouldHit = true
				}
				if aggrGood && vAlign >= 350 {
					shouldHit = true
				}
			}
			if shouldHit || aggressive {
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, vict); err != nil {
						slog.Warn("StartCombat failed in aggressive mob", "mob", ch.GetName(), "error", err)
					}
				}
				break
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- Race-hate aggression (src/mobact.c:236-258) --
	// Mobs attack players whose race_hate slots match the mob's race.
	if MobSpecAssign[ch.Proto().VNum] != "shop_keeper" {
		mobRace := ch.Proto().Race
		for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
			if vict.IsNPC() {
				continue
			}
			attacked := false
			for i := 0; i < 5 && !attacked; i++ {
				vict.mu.RLock()
				hates := vict.RaceHates[i] == mobRace
				vict.mu.RUnlock()
				if !hates {
					continue
				}
				if !w.canSeeCharacterInRoom(ch, vict) {
					continue
				}
				if vict.GetFlags()&(1<<PrfNohassle) != 0 {
					continue
				}
				// AFF_PROTECT_EVIL blocks unless the mob is evil AND a 1-in-6 check passes.
				// #nosec G404 — game RNG, not cryptographic
				if vict.IsAffected(affProtectEvil) && (!mobIsEvil(ch) || mobactNumber(0, 5) != 0) {
					continue
				}
				// #nosec G404 — game RNG, not cryptographic
				if mobactNumber(0, 5) == 0 && ch.CanSpeak() {
					w.RoomEcho(ch.RoomVNum, fmt.Sprintf("'Come to destroy my kin? Die!', exclaims %s.", ch.GetName()), ch.GetName())
				}
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, vict); err != nil {
						slog.Warn("StartCombat failed in race-hate mob", "mob", ch.GetName(), "error", err)
					}
				}
				attacked = true
			}
			if attacked {
				break
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- Mob Memory --
	// C: src/mobact.c:261-282. The remembered victim is the first occupant in
	// room order that is not a mobile, is visible to ch, and does not carry
	// PRF_NOHASSLE. The two warning acts are the two halves of one
	// ROOM_PEACEFUL gate: a speaking mobile exclaims, a silent one growls.
	// The attack that follows is unconditional (R1/R4).
	if hasMobFlag(ch, "memory") && len(ch.Memory) > 0 {
		peaceful := w.roomHasFlag(ch.RoomVNum, "peaceful")
		found := false
		for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
			if found {
				break
			}
			if vict.IsNPC() {
				continue
			}
			// C: !CAN_SEE(ch, vict)
			if !w.canSeeCharacterInRoom(ch, vict) {
				continue
			}
			// C: PRF_FLAGGED(vict, PRF_NOHASSLE)
			if vict.GetFlags()&(1<<PrfNohassle) != 0 {
				continue
			}
			for _, name := range ch.Memory {
				if name != vict.GetName() {
					continue
				}
				found = true
				if !peaceful {
					if ch.CanSpeak() {
						Act(w, false, ch, nil, nil, nil,
							"'Hey!  You're the fiend that attacked me!!!', exclaims $n.", "", ToRoom)
					} else {
						Act(w, false, ch, nil, nil, nil,
							"You hear a low growl in the back of $n's throat.", "", ToRoom)
					}
				}
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, vict); err != nil {
						slog.Warn("StartCombat failed in memory mob", "mob", ch.GetName(), "error", err)
					}
				}
				break
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- Helper Mobs --
	// C: src/mobact.c:284-302. A charmed helper whose master is fighting
	// calls do_assist with the master's name and ends its pulse; otherwise it
	// joins the first mobile in room order that is fighting a player it can
	// see. Go has no npc_rescue wiring here: C's npc_rescue (src/mobprog.c:435)
	// is defined but never called, so the port's call was an invention (R4).
	if hasMobFlag(ch, "helper") {
		if ch.IsAffected(affCharm) && ch.GetFollowing() != "" {
			if master := w.followingActor(ch.GetFollowing()); master != nil {
				if c, ok := master.(combat.Combatant); ok && c.GetFightingBody() != nil {
					w.mobAssist(ch, master)
					return
				}
			}
		}
		found := false
		for _, vict := range w.GetMobsInRoom(ch.RoomVNum) {
			if found {
				break
			}
			if vict == ch {
				continue
			}
			target := vict.GetFightingBody()
			if target == nil || target.IsNPC() || target == combat.Combatant(ch) {
				continue
			}
			if !w.canSeeCharacterInRoom(ch, target) {
				continue
			}
			found = true
			Act(w, false, ch, vict, nil, nil, "$n jumps to the aid of $N!", "", ToRoom)
			if w.combatEngine != nil {
				if err := w.combatEngine.StartCombat(ch, target); err != nil {
					slog.Warn("StartCombat failed in helper mob", "mob", ch.GetName(), "error", err)
				}
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- MOB_AGGR24: attack players level 24+ --
	// C: src/mobact.c:304-322. Target selection runs first, then the room
	// grin, the optional speak pair (can_speak), and finally hit().
	if hasMobFlag(ch, "aggr24") {
		var vict *Player
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			// C: !IS_NPC(tmp_ch) && CAN_SEE(ch, tmp_ch) && !PRF_NOHASSLE && !ROOM_PEACEFUL
			if p.IsNPC() {
				continue
			}
			if !w.canSeeCharacterInRoom(ch, p) {
				continue
			}
			if p.GetFlags()&(1<<PrfNohassle) != 0 {
				continue
			}
			if w.roomHasFlag(ch.RoomVNum, "peaceful") {
				continue
			}
			if p.GetLevel() >= 24 {
				vict = p
				break
			}
		}
		if vict != nil {
			Act(w, false, ch, nil, nil, nil, "$n grins evilly.", "", ToRoom)
			if ch.CanSpeak() {
				Act(w, false, ch, nil, nil, nil,
					"'Another dark pawn trying to be a knight, eh?', asks $n.", "", ToRoom)
				Act(w, false, ch, nil, nil, nil,
					"$n says, 'I can fix that.'\n\r", "", ToRoom)
			}
			if w.combatEngine != nil {
				if err := w.combatEngine.StartCombat(ch, vict); err != nil {
					slog.Warn("StartCombat failed in aggr24 mob", "mob", ch.GetName(), "error", err)
				}
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- AGGR24 + AGGRESSIVE + Full HP: attack weaker NPCs --
	// C: src/mobact.c:324-336. The grin act fires at C's point, between
	// selecting the visible weaker mobile and hit().
	if hasMobFlag(ch, "aggr24") && hasMobFlag(ch, "aggressive") && ch.GetHP() >= ch.GetMaxHP() {
		for _, vict := range w.GetMobsInRoom(ch.RoomVNum) {
			if vict == ch {
				continue
			}
			if !w.canSeeCharacterInRoom(ch, vict) {
				continue
			}
			if vict.GetLevel()+3 < ch.GetLevel() {
				Act(w, false, ch, nil, nil, nil, "$n grins evilly.", "", ToRoom)
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, vict); err != nil {
						slog.Warn("StartCombat failed in aggr24 mob vs mob", "mob", ch.GetName(), "error", err)
					}
				}
				break
			}
		}
	}
}

// mobAssist is do_assist() (src/act.offensive.c:54-98) driven by a mobile, as
// mobile_activity's charmed-master arm calls it (src/mobact.c:287-291).
// FIGHTING(ch) is already false — the pulse skips fighting mobiles — and every
// send_to_char line C aims at ch is dropped, because a mobile has no
// descriptor. What remains observable is the helpee line, the room act, and
// the swing. The opponent scan uses the port's room-occupant helper; C's true
// room-list order is DP-1364.
func (w *World) mobAssist(ch *MobInstance, helpee Actor) {
	if ch == nil || helpee == nil {
		return
	}
	// C: if (FIGHTING(ch)) return;
	if ch.GetFightingBody() != nil {
		return
	}
	// C: if (IS_MOUNTED(ch)) { stc("Dismount first!\r\n", ch); return; }
	if ch.IsAffected(affMount) {
		return
	}
	// C: get_char_room_vis(ch, arg) — nothing outside the actor's room.
	if helpee.GetRoom() != ch.GetRoom() {
		return
	}
	// C: helpee == ch
	if Actor(ch) == helpee {
		return
	}
	var opponent Actor
	var opponentCombat combat.Combatant
	for _, to := range w.actChar(ch.GetRoom()) {
		c, ok := to.(combat.Combatant)
		if !ok || to == ch {
			continue
		}
		if any(c.GetFightingBody()) == any(helpee) {
			opponent, opponentCombat = to, c
			break
		}
	}
	// C's "But nobody is fighting $M!" and "You can't see who is fighting $M!"
	// are TO_CHAR lines to a descriptorless mobile, so they are silent.
	if opponent == nil || !w.canSeeCharacterInRoom(ch, opponent) {
		return
	}
	// C: act("$N assists you!", 0, helpee, 0, ch, TO_CHAR) — the helpee hears
	// it, and $N is the assisting mobile.
	Act(w, false, helpee, ch, nil, nil, "$N assists you!", "", ToChar)
	// C: act("$n assists $N.", FALSE, ch, 0, helpee, TO_NOTVICT).
	Act(w, false, ch, helpee, nil, nil, "$n assists $N.", "", ToNotVict)
	if w.combatEngine != nil {
		if err := w.combatEngine.StartCombat(ch, opponentCombat); err != nil {
			slog.Warn("StartCombat failed in charmed helper assist", "mob", ch.GetName(), "error", err)
		}
	}
}

// roomHasOtherOccupant is mobact.c's people scan for the pulse triggers:
// anyone in ch's room other than ch, or with playersOnly, anyone who is not
// a mobile.
func (w *World) roomHasOtherOccupant(ch *MobInstance, playersOnly bool) bool {
	if len(w.GetPlayersInRoom(ch.RoomVNum)) > 0 {
		return true
	}
	if playersOnly {
		return false
	}
	for _, other := range w.GetMobsInRoom(ch.RoomVNum) {
		if other != ch {
			return true
		}
	}
	return false
}

// runPulseScript is run_script(ch, ch, NULL, room, NULL, trigger, LT_MOB)
// for the pulse triggers, and returns the script's result.
func (w *World) runPulseScript(ch *MobInstance, trigger string) bool {
	if ScriptEngine == nil {
		return false
	}
	ctx := ch.CreateSelfScriptContext()
	ctx.World = NewWorldScriptableAdapter(w)
	handled, err := ch.RunScript(trigger, ctx)
	if err != nil {
		slog.Warn("mob pulse script error", "trigger", trigger, "mob_vnum", ch.GetVNum(), "error", err)
	}
	return handled
}
