package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestStopFollowerMobClearsCharmAndGroup(t *testing.T) {
	for _, spell := range []bool{false, true} {
		t.Run(map[bool]string{false: "ride-bit", true: "spell-record"}[spell], func(t *testing.T) {
			w, master, mob := newRideDepthWorld(t)
			w.StopAITicker()
			mob.SetFollowingBody(master)
			mob.SetAffected(affCharm)
			mob.SetAffected(affGroup)
			if spell {
				mob.AddAffect(engine.NewAffectDirect(7, engine.ApplyNone, 10, 0, engine.AFFCharm, "charm"))
			}
			StopFollowerMob(w, mob)
			if mob.GetFollowing() != "" || w.FollowingBody(mob) != nil || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
				t.Fatalf("stop relation=%q charm=%t group=%t", mob.GetFollowing(), mob.IsAffected(affCharm), mob.IsAffected(affGroup))
			}
			if spell {
				for _, value := range mob.CustomData {
					if aff, ok := value.(*engine.Affect); ok && aff.SpellID == 7 {
						t.Fatal("spell charm record survived stop")
					}
				}
			}
		})
	}
}

func TestDamageBreaksMountCharmAndGroup(t *testing.T) {
	w, master, mob := newRideDepthWorld(t)
	w.StopAITicker()
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(w.WireCombatCallbacks())
	mob.SetFollowingBody(master)
	mob.SetAffected(affCharm)
	mob.SetAffected(affGroup)
	combat.TakeDamageWithDeath(master, mob, 0, combat.TYPE_HIT, func() { t.Fatal("nonlethal attack died") })
	if w.FollowingBody(mob) != nil || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
		t.Fatalf("damage left master=%v charm=%t group=%t", w.FollowingBody(mob), mob.IsAffected(affCharm), mob.IsAffected(affGroup))
	}
}
