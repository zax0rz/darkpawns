package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// C npc_steal refuses any victim at LVL_IMMORT or above (src/spec_procs.c:307).
// specThief's outer gate normally hides the helper's own check, so call it
// directly: a level-40 victim must keep every coin.
// TestNpcStealRejectsImmortalLevelVictim fails while the port compares against
// the foreign 50 (a level-40 victim falls through to the steal branch).
func TestNpcStealRejectsImmortalLevelVictim(t *testing.T) {
	w, victim, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	victim.SetLevel(40)
	victim.SetPosition(combat.PosSleeping)
	victim.SetGold(1000)

	npcSteal(w, mob, victim)

	if got := victim.GetGold(); got != 1000 {
		t.Fatalf("npcSteal robbed an immortal victim: gold = %d, want 1000", got)
	}
}
