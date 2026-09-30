package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNonDamageResultKeepsExistingCombatEntry(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 10, "target")
	ktw.world.StopAITicker()
	actor := ktw.addPlayer(t, 1, "Actor", 10, game.ClassThief, false)
	rig := newDrawOrderRig(t, actor.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: actor, world: ktw.world, combatEngine: rig.engine}
	if err := sendSkillResult(sess, actor, ktw.mob, game.SkillResult{StartCombat: true}); err != nil {
		t.Fatal(err)
	}
	if actor.GetFighting() != ktw.mob.GetName() {
		t.Fatalf("non-damage entry suppressed: %q", actor.GetFighting())
	}
}
