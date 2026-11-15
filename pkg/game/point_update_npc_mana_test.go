package game

import "testing"

// TestPointUpdateRegeneratesNPCNana mirrors C's mana arm in point_update
// (src/limits.c:493-494): a body below max gains mana_gain's NPC value
// (GET_LEVEL), capped by MIN at GET_MAX_MANA, and a body already at max is
// left alone by the < max guard.
func TestPointUpdateRegeneratesNPCNana(t *testing.T) {
	w, _ := newResolverTestWorld(t)
	var mob *MobInstance
	for _, candidate := range w.GetAllMobs() {
		if candidate.GetVNum() == 202 {
			mob = candidate
			break
		}
	}
	if mob == nil {
		t.Fatal("resolver fixture mob 202 not found")
	}
	mob.SetLevel(32)
	mob.SetMaxMana(10)
	mob.SetMana(2)

	w.PointUpdate()
	if got := mob.GetMana(); got != 10 {
		t.Fatalf("PointUpdate mana = %d, want 10 (2 + level 32, capped at 10)", got)
	}

	// The < max guard leaves a full pool untouched.
	mob.SetMana(10)
	w.PointUpdate()
	if got := mob.GetMana(); got != 10 {
		t.Fatalf("PointUpdate full mana = %d, want 10", got)
	}
}
