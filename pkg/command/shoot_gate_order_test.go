package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C src/act.offensive.c:794-814 checks inventory and missile type before direction.
func TestShootProjectileBeforeDirection(t *testing.T) {
	for _, c := range []struct{ object, want string }{
		{"rock", "You don't seem to have any rocks.\r\n"},
		{"bow", "A bow is not a projectile!\r\n"},
		{"arrow", "Interesting direction.\r\n"},
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
			if err := CmdShoot(s, []string{c.object, "sideways", "nobody"}); err != nil {
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
