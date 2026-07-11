package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// TestIsCharmedI verifies the World charm query that pkg/spells delegates to
// (DP-1015). It must use the internal charm bit index (affCharm), NOT the
// engine AFF_CHARM mask. The old spell-layer code passed int(engine.AFFCharm)
// (== 1024) to IsAffected, which treats its argument as a bit index and is
// therefore always false — so area/mass spells never skipped charmed pets.
func TestIsCharmedI(t *testing.T) {
	w := newAnimateTestWorld(t)

	// Player: uncharmed then charmed.
	p := newAnimatePlayer(t, w, "Victim", 12)
	if w.IsCharmedI(p) {
		t.Error("uncharmed player reported as charmed")
	}
	p.SetAffect(affCharm, true)
	if !w.IsCharmedI(p) {
		t.Error("charmed player not reported as charmed")
	}

	// Mob: uncharmed then charmed.
	mob, err := w.SpawnMob(10, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	if w.IsCharmedI(mob) {
		t.Error("freshly spawned mob reported as charmed")
	}
	mob.SetAffected(affCharm)
	if !w.IsCharmedI(mob) {
		t.Error("charmed mob not reported as charmed")
	}

	// The DP-1015 bug: the engine AFF_CHARM *mask* used as a *bit index* never
	// matches. This is exactly what the old spell code did, and why the
	// charm-skip guard was dead code.
	if mob.IsAffected(int(engine.AFFCharm)) {
		t.Error("engine.AFFCharm mask must NOT register as a set bit index — " +
			"that confusion is the DP-1015 bug IsCharmedI fixes")
	}

	// Unknown type: not charmed, no panic.
	if w.IsCharmedI("not a character") {
		t.Error("non-character value reported as charmed")
	}
}
