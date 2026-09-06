package game

import (
	"fmt"

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
