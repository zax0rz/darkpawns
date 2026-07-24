package command

// DP-1213: positive-damage skill hits must enroll BOTH combatants in engine
// combat (C: damage() calls set_fighting unconditionally), and a killing hit
// must not enroll a corpse. These tests drive the REAL sendSkillResult path.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// findSuccessfulTrip returns a seed whose DoTrip succeeds against the rig's
// mob, plus the result. Deterministic via dprng.ResetStream.
func findSuccessfulTrip(t *testing.T, p *game.Player, ktw *killTestWorld) (uint32, game.SkillResult) {
	t.Helper()
	for s := uint32(1); s < 200; s++ {
		ktw.mob.SetPosition(combat.PosStanding)
		dprng.ResetStream(s)
		res := game.DoTrip(p, ktw.mob, ktw.world)
		if res.Success {
			return s, res
		}
	}
	t.Skip("no trip success in 199 seeds (RNG)")
	return 0, game.SkillResult{}
}

// TestSendSkillResult_TripSuccess_EnrollsBothCombatants — a positive-damage
// trip hit must put BOTH combatants in the engine's combatOrder with mutual
// FIGHTING, so combat rounds actually fire (DP-1213: previously only the
// victim's field was set by DoSpellDamage and PerformRound had nothing to
// run). The post-round assertions also exercise the downed-mob stand-up:
// the tripped (POS_SITTING, wait 1) mob is IN combatOrder, so the next round
// decrements its wait and stands it up ("scrambles to his feet!") instead of
// ending combat (fight.c:1977-1998).
func TestSendSkillResult_TripSuccess_EnrollsBothCombatants(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker() // quiesce the shared stream
	p := ktw.addPlayer(t, 1, "Tripper", 10, game.ClassThief, false)
	p.SetSkill(game.SkillTrip, 100)

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	seed, _ := findSuccessfulTrip(t, p, ktw)

	ktw.mob.SetPosition(combat.PosStanding)
	dprng.ResetStream(seed)
	res := game.DoTrip(p, ktw.mob, ktw.world)
	if !res.Success || res.Damage <= 0 {
		t.Fatalf("seed %d: expected positive-damage trip success, got success=%v dam=%d", seed, res.Success, res.Damage)
	}
	if err := sendSkillResult(sess, p, ktw.mob, res); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}

	// Mutual FIGHTING (C: set_fighting is mutual).
	if p.GetFighting() != ktw.mob.GetName() {
		t.Errorf("attacker should be fighting the mob after a positive-damage hit, got %q", p.GetFighting())
	}
	if ktw.mob.GetFighting() != p.Name {
		t.Errorf("victim should be fighting the attacker, got %q", ktw.mob.GetFighting())
	}
	if !rig.engine.IsFighting(p.Name) || !rig.engine.IsFighting(ktw.mob.GetName()) {
		t.Errorf("both combatants should be enrolled: IsFighting(%s)=%v IsFighting(%s)=%v",
			p.Name, rig.engine.IsFighting(p.Name), ktw.mob.GetName(), rig.engine.IsFighting(ktw.mob.GetName()))
	}

	// The mob must be in combatOrder, not just the pair map: a PerformRound
	// processes it. The trip sat it down with wait 1, so this round it
	// decrements to 0 and STANDS UP (C's two-step perform_violence) rather
	// than being skipped (not enrolled) or stopped (the DP-1213 stand-up bug).
	var broadcasts []string
	rig.engine.BroadcastFunc = func(_ int, msg, _ string) { broadcasts = append(broadcasts, msg) }
	dprng.ResetStream(7)
	rig.engine.PerformRound()

	if ktw.mob.GetPosition() == combat.PosSitting {
		t.Error("tripped mob was never processed by PerformRound — it is not in combatOrder (still sitting)")
	}
	if ktw.mob.GetPosition() != combat.PosFighting {
		t.Errorf("tripped mob should stand up on its first round, got position %d", ktw.mob.GetPosition())
	}
	scrambled := false
	for _, b := range broadcasts {
		if strings.Contains(b, "scrambles") {
			scrambled = true
		}
	}
	if !scrambled {
		t.Errorf("expected 'scrambles to his feet!' broadcast when the downed mob stands, got %v", broadcasts)
	}
	if ktw.mob.GetFighting() != p.Name {
		t.Errorf("combat must NOT stop when the downed mob stands (DP-1213 stand-up bug), fighting=%q",
			ktw.mob.GetFighting())
	}
}

// TestSendSkillResult_KillHit_DoesNotEnrollCorpse — when the skill hit KILLS
// the target, C does not set_fighting a corpse: neither combatant is enrolled
// and the attacker's FIGHTING stays clear. The tripper is level 30 so the hit
// (level/2+1 = 16) drives the 1-HP mob to HP <= -11 — Go's death pipeline
// only runs at POS_DEAD (DP-1021), so a smaller hit would leave the mob in a
// wounded band instead of killing it.
func TestSendSkillResult_KillHit_DoesNotEnrollCorpse(t *testing.T) {
	ktw := newKillTestWorld(t, 1, 500, 100, 3, "rat") // 1 HP — a 16-dam hit kills
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Tripper", 30, game.ClassThief, false)
	p.SetSkill(game.SkillTrip, 100)

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	seed, _ := findSuccessfulTrip(t, p, ktw)

	ktw.mob.SetPosition(combat.PosStanding)
	dprng.ResetStream(seed)
	res := game.DoTrip(p, ktw.mob, ktw.world)
	if !res.Success {
		t.Fatalf("seed %d: expected trip success on re-run", seed)
	}
	if err := sendSkillResult(sess, p, ktw.mob, res); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}

	if ktw.findMob() != nil {
		t.Fatalf("1-HP mob should be dead and removed after the hit")
	}
	if p.GetFighting() != "" {
		t.Errorf("killing hit must not enroll the attacker (C enrolls no corpse), fighting=%q", p.GetFighting())
	}
	if rig.engine.IsFighting(p.Name) || rig.engine.IsFighting(ktw.mob.GetName()) {
		t.Error("killing hit must not enroll either combatant in engine combat")
	}
}
