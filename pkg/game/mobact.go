// mobact.go — Ported from src/mobact.c
//
// Mobile AI: generates intelligent (?) behavior in mobiles.
//
// All rights reserved. See LICENSE for license information.

package game

import (
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hasMobFlag(mob *MobInstance, flag string) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}
	for _, f := range mob.Prototype.ActionFlags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

func roomHasFlag(room *parser.Room, flag string) bool {
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
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
				"mob", mob.GetName(), "vnum", mob.Prototype.VNum, "panic", r)
			handled = false
		}
	}()
	return specFn(w, nil, mob, "", "")
}

func mobIsEvil(mob *MobInstance) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}
	return mob.Prototype.Alignment <= -350
}

func mobIsGood(mob *MobInstance) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}
	return mob.Prototype.Alignment >= 350
}

// getMobVNumSpec looks up a mob's registered spec proc by its VNum.
func getMobVNumSpec(vnum int) SpecFunc {
	specName, ok := MobSpecAssign[vnum]
	if !ok || specName == "" {
		return nil
	}
	return SpecRegistry[specName]
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
//	IS_MOB(mob)   → mob.Prototype != nil
//	FIGHTING(mob) → mob.GetFighting() != ""
//	AWAKE(mob)    → mob.GetPosition() >= combat.PosSitting
//	MOB_FLAGGED   → hasMobFlag()
//	hit()         → w.combatEngine.StartCombat()
//	GET_HIT       → mob.GetHP()
//	GET_MAX_HIT   → mob.GetMaxHP()
//	CAN_SEE       → canSeePlayer()
//	PRF_NOHASSLE  → hasPrfNoHassle()
//	GET_MOB_RNUM  → mob.Prototype.VNum
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
		ready := ch.Prototype != nil
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
	if ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return
	}
	// C: if (GET_HIT(ch) < GET_MAX_HIT(ch) && MOB_FLAGGED(ch, MOB_CHARMED)) continue;
	if ch.GetHP() < ch.GetMaxHP() && hasMobFlag(ch, "charmed") {
		return
	}

	// -- MOB_SPEC: special procedure dispatch --
	// C: spec proc returns true to skip to next mob.
	if hasMobFlag(ch, "spec") {
		specFn := getMobVNumSpec(ch.Prototype.VNum)
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
	// #nosec G404 — game RNG, not cryptographic
	if hasMobFlag(ch, "scavenger") && rand.IntN(11) == 0 {
		items := w.GetItemsInRoom(ch.RoomVNum)
		if len(items) > 0 {
			best := items[0]
			bestCost := best.GetCost()
			for _, obj := range items[1:] {
				if c := obj.GetCost(); c > bestCost {
					bestCost = c
					best = obj
				}
			}
			w.RemoveItemFromRoom(best, ch.RoomVNum)
			ch.AddToInventory(best)
		}
	}

	// Hunter mobs chase their targets (C: mobact.c)
	if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
		w.huntVictim(ch)
	}

	// -- Mob Movement (wandering) --
	if !hasMobFlag(ch, "sentinel") && ch.GetPosition() >= combat.PosStanding {
		w.wanderMob(ch)
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
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
			// C: MOB_WIMPY && AWAKE(vict)
			if wimpy && vict.GetPosition() > combat.PosSleeping {
				continue
			}
			// C: AFF_PROTECT_EVIL + IS_EVIL(ch) + !number(0,5)
			// #nosec G404 — game RNG, not cryptographic
			if vict.IsAffected(12) && mobIsEvil(ch) && rand.IntN(6) != 0 {
				continue
			}
			// C: AFF_PROTECT_GOOD + IS_GOOD(ch) + !number(0,5)
			// #nosec G404 — game RNG, not cryptographic
			if vict.IsAffected(13) && mobIsGood(ch) && rand.IntN(6) != 0 {
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

	// -- Mob Memory --
	if hasMobFlag(ch, "memory") && len(ch.Memory) > 0 {
		for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
			if vict.IsNPC() {
				continue
			}
			for _, name := range ch.Memory {
				if name == vict.GetName() {
					if w.combatEngine != nil {
						if err := w.combatEngine.StartCombat(ch, vict); err != nil {
							slog.Warn("StartCombat failed in memory mob", "mob", ch.GetName(), "error", err)
						}
					}
					break
				}
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- Helper Mobs --
	if hasMobFlag(ch, "helper") {
		for _, vict := range w.GetMobsInRoom(ch.RoomVNum) {
			if vict == ch {
				continue
			}
			target := vict.GetFighting()
			if target == "" {
				continue
			}
			for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
				if p.GetName() == target {
					if w.combatEngine != nil {
						if err := w.combatEngine.StartCombat(ch, p); err != nil {
							slog.Warn("StartCombat failed in helper mob", "mob", ch.GetName(), "error", err)
						}
					}
					break
				}
			}
		}
		// Rescue any player being attacked by a mob
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if w.NpcRescue(ch, p) {
				break
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- MOB_AGGR24: attack players level 24+ --
	if hasMobFlag(ch, "aggr24") {
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if p.GetLevel() >= 24 {
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, p); err != nil {
						slog.Warn("StartCombat failed in aggr24 mob", "mob", ch.GetName(), "error", err)
					}
				}
				break
			}
		}
	}
	if !ch.IsAlive() || ch.RoomVNum < 0 {
		return
	}

	// -- AGGR24 + AGGRESSIVE + Full HP: attack weaker NPCs --
	if hasMobFlag(ch, "aggr24") && hasMobFlag(ch, "aggressive") && ch.GetHP() >= ch.GetMaxHP() {
		for _, vict := range w.GetMobsInRoom(ch.RoomVNum) {
			if vict == ch {
				continue
			}
			if vict.GetLevel()+3 < ch.GetLevel() {
				if w.combatEngine != nil {
					if err := w.combatEngine.StartCombat(ch, vict); err != nil {
						slog.Warn("StartCombat failed in aggr24 mob vs mob", "mob", ch.GetName(), "error", err)
					}
				}
				break
			}
		}
	}

	// Sound trigger (C: 1-in-16 chance for mp_sound + lua sound)
	// #nosec G404 — game RNG, not cryptographic
	if rand.IntN(16) == 0 {
		w.MpSound(ch)
	}

	// Lua onpulse triggers — fired every tick for mobs with the script
	if ScriptEngine != nil {
		if ch.HasScript("onpulse_all") {
			ctx := ch.CreateScriptContext(nil, nil, "")
			ctx.World = NewWorldScriptableAdapter(w)
			ctx.RoomVNum = ch.RoomVNum
			if _, err := ch.RunScript("onpulse_all", ctx); err != nil {
				slog.Warn("onpulse_all script error", "mob_vnum", ch.GetVNum(), "error", err)
			}
		}
		if ch.HasScript("onpulse_pc") {
			players := w.GetPlayersInRoom(ch.RoomVNum)
			if len(players) > 0 {
				ctx := ch.CreateScriptContext(players[0], nil, "")
				ctx.World = NewWorldScriptableAdapter(w)
				ctx.RoomVNum = ch.RoomVNum
				if _, err := ch.RunScript("onpulse_pc", ctx); err != nil {
					slog.Warn("onpulse_pc script error", "mob_vnum", ch.GetVNum(), "error", err)
				}
			}
		}
	}
}
