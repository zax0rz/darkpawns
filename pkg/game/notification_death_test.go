package game

import (
	"context"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/events"
)

func TestDeathObservationCoversGuardedAndRawPaths(t *testing.T) {
	for _, kind := range []string{"combat", "environment", "raw"} {
		t.Run(kind, func(t *testing.T) {
			w, _, _, _, p, _ := duplicateCleanupWorld(t)
			count := 0
			w.Events.Subscribe(events.PlayerDiedEvent{}.Type(), func(_ context.Context, e events.BusEvent) error {
				if e.(events.PlayerDiedEvent).PlayerID == p.GetName() {
					count++
				}
				return nil
			})
			switch kind {
			case "combat":
				p.SetHP(-11)
				w.handlePlayerDeath(p, true, combat.TYPE_UNDEFINED, "")
				w.handlePlayerDeath(p, true, combat.TYPE_UNDEFINED, "")
			case "environment":
				p.SetHP(-11)
				w.handlePlayerDeath(p, false, combat.TYPE_UNDEFINED, "")
				w.handlePlayerDeath(p, false, combat.TYPE_UNDEFINED, "")
			case "raw":
				w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
				w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
			}
			if count != 1 {
				t.Fatalf("death observed %d times", count)
			}
		})
	}
}
