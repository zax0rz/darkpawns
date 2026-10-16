package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// StartCombat exposes the world's canonical combat-entry seam to packages
// that cannot access World.combatEngine directly. Spell damage uses this for
// the C mag_areas -> mag_damage -> damage(0) breath path: damage(0) enrolls
// both awake combatants even though it deals no damage (fight.c:1367-1445).
func (w *World) StartCombat(attacker, defender combat.Combatant) error {
	if w.combatEngine == nil {
		return fmt.Errorf("combat engine is not configured")
	}
	return w.combatEngine.StartCombat(attacker, defender)
}

// MobFirstHit exposes the world's canonical seam for C hit(mob, victim, ...)
// calls that immediately resolve the NPC's opening attack.
func (w *World) MobFirstHit(attacker, defender combat.Combatant) {
	w.mobFirstHit(attacker, defender)
}

// mobFirstHit ports a C mobile's hit() call: hit() consumes its to-hit and
// damage draws, damage()'s protection block runs against the pre-enrolment
// state, and both sides enrol at the damage point only when the damage is
// not refused (fight.c:1318-1445). This is the mobile seam for DP-1438;
// engines without the deferred entry keep the historical eager shape for
// focused tests.
func (w *World) mobFirstHit(attacker, defender combat.Combatant) {
	if w.combatEngine == nil {
		return
	}
	if hitter, ok := w.combatEngine.(interface {
		PerformDeferredInitialAttack(combat.Combatant, combat.Combatant) error
	}); ok {
		if err := hitter.PerformDeferredInitialAttack(attacker, defender); err != nil {
			slog.Warn("deferred initial attack failed", "attacker", attacker.GetName(), "target", defender.GetName(), "error", err)
		}
		return
	}
	if err := w.combatEngine.StartCombat(attacker, defender); err != nil {
		slog.Warn("StartCombat failed in mobile first hit", "attacker", attacker.GetName(), "target", defender.GetName(), "error", err)
	}
}
