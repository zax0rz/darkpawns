package command

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShootProjectileAndZeroWait(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(fmt.Sprint(hit), func(t *testing.T) {
			skill := 1
			if hit {
				skill = 200
			}
			s, arrow, v, _, _ := shootOutcomeFixture(t, false, skill)
			s.player.SetWaitStatePulses(7)
			before := s.player.GetWaitState()
			dprng.ResetStream(71)
			fireOutcome(t, s, v)
			if s.player.GetWaitState() != before {
				t.Fatal("shoot added wait")
			}
			if len(s.player.Inventory.FindItems("arrow")) != 0 {
				t.Fatal("projectile left in inventory")
			}
			items := s.world.GetItemsInRoom(1002)
			if hit {
				if len(items) != 0 || arrow.Location.Kind != game.ObjNowhere || len(s.world.GetAllObjects()) != 1 {
					t.Fatal("hit left projectile")
				}
			} else {
				if len(items) != 1 || items[0] != arrow {
					t.Fatal("miss did not prepend projectile")
				}
			}
		})
	}
}
