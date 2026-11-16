package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C src/act.offensive.c:794-814,990-996 checks inventory/type order and both
// direction refusals.
func TestShootProjectileBeforeDirection(t *testing.T) {
	for _, c := range []struct{ object, direction, want string }{
		{"rock", "sideways", "You don't seem to have any rocks.\r\n"},
		{"bow", "sideways", "A bow is not a projectile!\r\n"},
		{"arrow", "sideways", "Interesting direction.\r\n"},
		{"arrow", "east", "Interesting Direction.\r\n"},
	} {
		t.Run(c.object, func(t *testing.T) {
			s, arrow := newShootGateSession(t)
			bow, _ := s.player.Equipment.GetItemInSlot(game.SlotWield)
			// Use the real equipped bow's canonical detach into inventory.
			if bow == nil {
				t.Fatal("missing fixture bow")
			}
			if err := s.world.MoveObjectToPlayerInventory(bow, s.player); err != nil {
				t.Fatal(err)
			}
			location, wait := arrow.Location, s.player.GetWaitState()
			dprng.ResetStream(71)
			next := dprng.Next()
			dprng.ResetStream(71)
			if err := CmdShoot(s, []string{c.object, c.direction, "nobody"}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(s.getMessages(), ""); got != c.want {
				t.Fatalf("refusal=%q want %q", got, c.want)
			}
			if arrow.Location != location || s.player.GetWaitState() != wait || dprng.Next() != next {
				t.Fatal("refusal changed projectile, wait or RNG")
			}
		})
	}
}
