// mobact.go — Ported from src/mobact.c
//
// Mobile AI: generates intelligent (?) behavior in mobiles.
//
// All rights reserved. See LICENSE for license information.

package game

import (
	"math/rand"

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
		if f == flag {
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
		if f == flag {
			return true
		}
	}
	return false
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
//
// C macros translated:
//
//	IS_MOB(mob)   → mob.Prototype != nil
//	FIGHTING(mob) → mob.GetFighting() != ""
//	AWAKE(mob)    → mob.GetPosition() >= combat.PosSitting
//	MOB_FLAGGED   → hasMobFlag()
//	hit()         → aiCombatEngine.StartCombat()
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
		if ch == nil || ch.Prototype == nil {
			continue
		}
		// C: if (!IS_MOB(ch) || FIGHTING(ch) || !AWAKE(ch)) continue;
		if ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
			continue
		}
		// C: if (GET_HIT(ch) < GET_MAX_HIT(ch) && MOB_FLAGGED(ch, MOB_CHARMED)) continue;
		if ch.GetHP() < ch.GetMaxHP() && hasMobFlag(ch, "charmed") {
			continue
		}

		// -- MOB_SPEC: special procedure dispatch --
		// C: spec proc returns true to skip to next mob.
		if hasMobFlag(ch, "spec") {
			specFn := getMobVNumSpec(ch.Prototype.VNum)
			if specFn != nil && specFn(w, nil, ch, "", "") {
				continue
			}
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
		}

		// -- Wake sleeper'd mobs --
		if ch.GetPosition() < combat.PosSitting {
			ch.SetStatus("standing")
		}

		// -- Scavenger (pick up best item, ~1-in-10 chance) --
		if hasMobFlag(ch, "scavenger") && rand.Intn(11) == 0 {
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

		// -- Mob Movement (wandering) --
		if !hasMobFlag(ch, "sentinel") && ch.GetPosition() >= combat.PosStanding {
			w.wanderMob(ch)
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
		}

		// -- Aggressive Mobs --
		isAggressive := hasMobFlag(ch, "aggressive")
		isAggrEvil := hasMobFlag(ch, "aggr_evil")
		isAggrGood := hasMobFlag(ch, "aggr_good")
		isAggrNeutral := hasMobFlag(ch, "aggr_neutral")
		isWimpy := hasMobFlag(ch, "wimpy")
		hasAlignAggr := isAggrEvil || isAggrGood || isAggrNeutral

		if isAggressive || hasAlignAggr {
			for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
				if vict.IsNPC() {
					continue
				}
				// C: MOB_WIMPY && AWAKE(vict)
				if isWimpy && vict.GetPosition() > combat.PosSleeping {
					continue
				}
				// C: AFF_PROTECT_EVIL + IS_EVIL(ch) + !number(0,5)
				if vict.IsAffected(12) && mobIsEvil(ch) && rand.Intn(6) != 0 {
					continue
				}
				// C: AFF_PROTECT_GOOD + IS_GOOD(ch) + !number(0,5)
				if vict.IsAffected(13) && mobIsGood(ch) && rand.Intn(6) != 0 {
					continue
				}
				// Alignment matching faithful to mobact.c:
				// If NONE of the per-alignment flags are set, hit everyone (plain MOB_AGGRESSIVE).
				// If SOME are set, only hit matching alignments.
				shouldHit := false
				if hasAlignAggr {
					vAlign := vict.GetAlignment()
					if isAggrEvil && vAlign <= -350 {
						shouldHit = true
					}
					if isAggrNeutral && vAlign > -350 && vAlign < 350 {
						shouldHit = true
					}
					if isAggrGood && vAlign >= 350 {
						shouldHit = true
					}
				}
				if shouldHit || isAggressive {
					if aiCombatEngine != nil {
						aiCombatEngine.StartCombat(ch, vict)
					}
					break
				}
			}
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
		}

		// -- Mob Memory --
		if hasMobFlag(ch, "memory") && len(ch.Memory) > 0 {
			for _, vict := range w.GetPlayersInRoom(ch.RoomVNum) {
				if vict.IsNPC() {
					continue
				}
				for _, name := range ch.Memory {
					if name == vict.GetName() {
						if aiCombatEngine != nil {
							aiCombatEngine.StartCombat(ch, vict)
						}
						break
					}
				}
			}
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
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
						if aiCombatEngine != nil {
							aiCombatEngine.StartCombat(ch, p)
						}
						break
					}
				}
			}
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
		}

		// -- MOB_AGGR24: attack players level 24+ --
		if hasMobFlag(ch, "aggr24") {
			for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
				if p.GetLevel() >= 24 {
					if aiCombatEngine != nil {
						aiCombatEngine.StartCombat(ch, p)
					}
					break
				}
			}
		}
		if !mobAlive(ch) || ch.RoomVNum < 0 {
			continue
		}

		// -- AGGR24 + AGGRESSIVE + Full HP: attack weaker NPCs --
		if hasMobFlag(ch, "aggr24") && hasMobFlag(ch, "aggressive") && ch.GetHP() >= ch.GetMaxHP() {
			for _, vict := range w.GetMobsInRoom(ch.RoomVNum) {
				if vict == ch {
					continue
				}
				if vict.GetLevel()+3 < ch.GetLevel() {
					if aiCombatEngine != nil {
						aiCombatEngine.StartCombat(ch, vict)
					}
					break
				}
			}
		}
	}
}

// mobAlive returns true if the mob's HP > 0.
func mobAlive(mob *MobInstance) bool {
	return mob != nil && mob.GetHP() > 0
}
