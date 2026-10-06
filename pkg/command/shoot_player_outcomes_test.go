package command

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShootPlayerOutcomes(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(fmt.Sprint(hit), func(t *testing.T) {
			skill := 1
			if hit {
				skill = 200
			}
			s, _, v, e, _ := shootOutcomeFixture(t, false, skill)
			dprng.ResetStream(71)
			_ = dprng.Number(1, 101)
			damage := 0
			if hit {
				damage = 7 + dprng.Dice(2, 7) + dprng.Dice(3, 4)
				_ = dprng.Number(1, 200)
			}
			next := dprng.Next()
			dprng.ResetStream(71)
			fireOutcome(t, s, v)
			if v.GetHP() != 1000-damage || v.GetRoom() != 1002 || v.GetFighting() != "" || s.player.GetFighting() != "" || e.calls != 0 {
				t.Fatalf("PC outcome HP=%d room=%d fighting=%q calls=%d", v.GetHP(), v.GetRoom(), v.GetFighting(), e.calls)
			}
			if got := dprng.Next(); got != next {
				t.Fatalf("next draw=%d want %d", got, next)
			}
		})
	}
}

func TestShootLethalCommandBoundary(t *testing.T) {
	s, arrow, v, _, _ := shootOutcomeFixture(t, false, 200)
	p := v.(*game.Player)
	p.SetHP(1)
	p.SetExp(903)
	p.Deaths = 6
	s.player.SetExp(5000)
	s.player.Kills = 7
	s.player.PKs = 4
	s.player.SetDamroll(100)
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(s.world.WireCombatCallbacks())
	dprng.ResetStream(71)
	fireOutcome(t, s, v)
	if p.GetExp() != 602 || p.Deaths != 6 || !p.HasPLRFlag(game.PlrExtract) || s.player.GetExp() != 5000 || s.player.Kills != 7 || s.player.PKs != 4 || s.player.HasPLRFlag(game.PlrOutlaw) {
		t.Fatal("shoot command bypassed legacy die or added killer bookkeeping")
	}
	if arrow.Location.Kind != game.ObjNowhere || len(s.world.GetItemsInRoom(1002)) != 1 {
		t.Fatal("lethal command failed projectile/corpse ownership")
	}
}
