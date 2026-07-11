package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestTakeDamage_FloorsAtNegative11 is the DP-1021 regression: HP is allowed
// into the wounded band instead of clamping at 0, but a single massive blow
// floors at -11 (POS_DEAD threshold, fight.c update_pos) rather than going
// arbitrarily negative.
func TestTakeDamage_FloorsAtNegative11(t *testing.T) {
	p := NewPlayer(1, "Bleeder", 1001)
	p.SetHP(5)
	p.TakeDamage(100)
	if got := p.GetHP(); got != -11 {
		t.Errorf("player HP after massive hit = %d, want -11 (floor)", got)
	}

	// Sub-lethal damage lands in the wounded band, not clamped to 0.
	p.SetHP(5)
	p.TakeDamage(9) // 5 - 9 = -4
	if got := p.GetHP(); got != -4 {
		t.Errorf("player HP = %d, want -4 (wounded band, not clamped to 0)", got)
	}

	m := &MobInstance{CurrentHP: 5, MaxHP: 30}
	m.TakeDamage(100)
	if got := m.GetHP(); got != -11 {
		t.Errorf("mob HP after massive hit = %d, want -11 (floor)", got)
	}
}

// TestDoDamage_WoundedBandThreshold verifies a mob dropped to 0 (or into the
// wounded band) is NOT dead — only crossing -11 (POS_DEAD) kills it. DP-1021.
func TestDoDamage_WoundedBandThreshold(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)

	// Drop to exactly 0 HP → POS_STUNNED, still alive.
	hp := mob.GetHP()
	w.doDamage(player, mob, hp, "bash")
	if !mob.IsAlive() {
		t.Fatal("mob at 0 HP should be stunned, not dead")
	}
	if mob.GetPosition() != combat.PosStunned {
		t.Errorf("mob position at 0 HP = %d, want PosStunned(%d)", mob.GetPosition(), combat.PosStunned)
	}

	// Bleed into the mortally-wounded band (HP -8), still alive.
	w.doDamage(player, mob, 8, "bash")
	if !mob.IsAlive() {
		t.Fatal("mob at -8 HP should be mortally wounded, not dead")
	}
	if mob.GetPosition() != combat.PosMortally {
		t.Errorf("mob position at -8 HP = %d, want PosMortally(%d)", mob.GetPosition(), combat.PosMortally)
	}

	// Cross the POS_DEAD threshold (HP <= -11) → dead.
	w.doDamage(player, mob, 5, "bash")
	if mob.IsAlive() {
		t.Error("mob at -11 HP should be dead")
	}
}

// TestPointUpdateBleedOut drives the point-update tick: an incapacitated
// character bleeds instead of regenerating, and worsens through the wounded
// band (incap → mortally). DP-1021 / limits.c:510-513. The final death at
// HP <= -11 is covered by TestDoDamage_WoundedBandThreshold (crossing -11
// here would immediately respawn the player, hiding the transient death).
func TestPointUpdateBleedOut(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	p := NewPlayer(42, "Dying", 1001)
	p.SetHP(-4) // incapacitated
	p.SetPosition(combat.PosIncap)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	// Tick 1: incapacitated bleeds 1/tick — must NOT regenerate to positive HP.
	w.PointUpdate()
	if p.GetHP() != -5 {
		t.Fatalf("incap HP after 1 tick = %d, want -5 (bled 1, not regenerated)", p.GetHP())
	}
	if p.GetPosition() != combat.PosIncap {
		t.Errorf("position after 1 tick = %d, want PosIncap(%d)", p.GetPosition(), combat.PosIncap)
	}

	// Tick 2: crosses into the mortally-wounded band (HP <= -6).
	w.PointUpdate()
	if p.GetHP() != -6 {
		t.Fatalf("HP after 2 ticks = %d, want -6", p.GetHP())
	}
	if p.GetPosition() != combat.PosMortally {
		t.Errorf("position at HP -6 = %d, want PosMortally(%d)", p.GetPosition(), combat.PosMortally)
	}
}
