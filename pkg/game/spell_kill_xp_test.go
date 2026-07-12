package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestSpellKill_AwardsXPAndKillCredit is the DP-1022 (F2) end-to-end regression:
// a spell kill must drive World.HandleDeath — the same DeathFunc melee uses — so
// the caster earns XP and kill-credit. Before the fix, mag_damage() routed spell
// death through HandleNonCombatDeath(killer=nil), awarding zero XP and never
// advancing the caster's kill counter ("casters can't level from kills").
func TestSpellKill_AwardsXPAndKillCredit(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)

	// ApplyDamageModifiers resolves affects via the production callbacks.
	orig := combat.GetCallbacks()
	defer combat.SetCallbacks(orig)
	combat.SetCallbacks(w.WireCombatCallbacks())

	// Give the victim an XP bounty and a level equal to the killer's so the
	// level-difference scaling doesn't zero the award (robust to the F3/F4
	// share-formula rework, which only changes the magnitude).
	mob.Prototype.Exp = 5000
	mob.Prototype.Level = player.GetLevel()
	mob.MaxHP = 100
	mob.CurrentHP = 1

	expBefore := player.GetExp()
	killsBefore := player.Kills

	// Cast Magic Missile until the training dummy crosses POS_DEAD (HP <= -11).
	for i := 0; i < 50 && mob.IsAlive(); i++ {
		spells.MagDamage(player.GetLevel(), player, mob, spells.SpellMagicMissile, int(spells.SaveSpell), w)
	}

	if mob.IsAlive() {
		t.Fatal("mob still alive after 50 Magic Missiles; spell damage never killed it")
	}
	if got := player.GetExp(); got <= expBefore {
		t.Errorf("caster exp = %d, want > %d (spell kill must award XP)", got, expBefore)
	}
	if got := player.Kills; got != killsBefore+1 {
		t.Errorf("caster kills = %d, want %d (spell kill must credit exactly one kill)", got, killsBefore+1)
	}
}
