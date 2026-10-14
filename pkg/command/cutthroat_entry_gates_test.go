package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1438: the failed cutthroat arm ends in hit(ch, victim,
// SKILL_CUTTHROAT) (new_cmds.c:652), so damage()'s gates must see the
// pre-enrolment state: in a peaceful room the swing refuses after its
// draws and enrols nobody. The old eager StartCombat enrolled both sides
// first, the gate then read FIGHTING(victim)==ch and the damage landed.
// This test fails on that shape (R5h).
func TestCutthroatTailRefusedInPeacefulRoom(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 10, "target")
	ktw.world.StopAITicker()
	ktw.world.GetRoomInWorld(ktw.mob.GetRoom()).Flags = []string{"peaceful"}
	actor := ktw.addPlayer(t, 1, "Cutter", 12, game.ClassWarrior, false)

	oldCB := combat.GetCallbacks()
	ce := combat.NewCombatEngine()
	ce.SetCallbacks(ktw.world.WireCombatCallbacks())
	t.Cleanup(func() { ce.Stop(); combat.SetCallbacks(oldCB) })
	session := &killPayoutSession{player: actor, world: ktw.world, combatEngine: ce}

	// Pin a natural-1 miss so the tail consumes exactly the to-hit draw.
	var seed uint32
	for s := uint32(1); s < 8192; s++ {
		dprng.ResetStream(s)
		if dprng.Number(1, 20) == 1 {
			seed = s
			break
		}
	}
	if seed == 0 {
		t.Fatal("no seed found whose first number(1,20) is 1")
	}

	dprng.ResetStream(seed)
	result := game.SkillResult{InitialAttack: true}
	if err := sendSkillResult(session, actor, ktw.mob, result); err != nil {
		t.Fatal(err)
	}
	gotNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("refused cutthroat swing changed the draw stream: next=%d, want %d (one number(1,20) consumed)", gotNext, wantNext)
	}
	if ce.IsFighting(actor) || actor.GetFighting() != "" || ktw.mob.GetFighting() != "" {
		t.Fatalf("refused cutthroat swing enrolled: actor %q, mob %q", actor.GetFighting(), ktw.mob.GetFighting())
	}
	if hp := ktw.mob.GetHP(); hp != 500 {
		t.Fatalf("refused cutthroat swing damaged the victim: hp=%d", hp)
	}
}
