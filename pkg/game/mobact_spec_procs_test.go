package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Regression tests for BRIEF-2026-07-04-spec-proc-nil-crash.md.
//
// mobileActivityForMob calls every spec-flagged mob's registered spec proc
// with ch=nil during autonomous AI ticks (there is no triggering player).
// In the original C (spec_procs2.c), `ch` on that path IS the mob itself, so
// several specs were ported reading `ch.GetX()` when they meant "the mob's
// own state" (me.GetX()) — a real nil-pointer panic risk in production.
// ---------------------------------------------------------------------------

func TestCallMobSpecSafely_RecoversFromPanic(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)

	var panicky SpecFunc = func(w *World, ch *Player, me *MobInstance, cmd, arg string) bool {
		_ = ch.GetPosition() // ch is nil on the autonomous path — this panics
		return true
	}

	handled := callMobSpecSafely(panicky, w, mob)
	if handled {
		t.Error("expected callMobSpecSafely to report unhandled after recovering from a panic")
	}
}

func TestCallMobSpecSafely_PassesThroughNormalResult(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)

	var fn SpecFunc = func(w *World, ch *Player, me *MobInstance, cmd, arg string) bool {
		return true
	}

	if !callMobSpecSafely(fn, w, mob) {
		t.Error("expected callMobSpecSafely to return the spec proc's real result when it doesn't panic")
	}
}

func TestSpecNormalChecker_NilChDoesNotPanic_AttacksNonImmortalPlayer(t *testing.T) {
	w, player := newCombatTestWorld(t)
	player.Level = 10
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)
	startHP := 200
	player.SetHP(startHP)

	handled := specNormalChecker(w, nil, mob, "", "")

	if !handled {
		t.Error("expected specNormalChecker to notice and attack the player")
	}
	if player.GetHP() >= startHP {
		t.Errorf("expected player to take damage from the mob's attack, HP stayed at %d", player.GetHP())
	}
}

func TestSpecNormalChecker_SkipsWhenMobAlreadyFighting(t *testing.T) {
	w, player := newCombatTestWorld(t)
	player.Level = 10
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)
	mob.SetFighting(player.Name)

	if specNormalChecker(w, nil, mob, "", "") {
		t.Error("expected specNormalChecker to skip a mob already fighting")
	}
}

func TestSpecRescuer_NilChDoesNotPanic_DefendsAllyAgainstAttacker(t *testing.T) {
	w, player := newCombatTestWorld(t)
	rescuer, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob rescuer: %v", err)
	}
	ally, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob ally: %v", err)
	}
	rescuer.SetPosition(combat.PosStanding)
	ally.SetPosition(combat.PosStanding)
	ally.SetFighting(player.Name) // player is attacking the ally mob
	startHP := 200
	player.SetHP(startHP)

	handled := specRescuer(w, nil, rescuer, "", "")

	if !handled {
		t.Error("expected specRescuer to defend the ally mob by attacking the player")
	}
	if player.GetHP() >= startHP {
		t.Errorf("expected the rescuer to attack the player defending its ally, HP stayed at %d", player.GetHP())
	}
}

func TestSpecRescuer_SkipsWhenNoAllyBeingAttacked(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	rescuer, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob rescuer: %v", err)
	}
	ally, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob ally: %v", err)
	}
	rescuer.SetPosition(combat.PosStanding)
	ally.SetPosition(combat.PosStanding)
	// ally is not fighting anyone

	if specRescuer(w, nil, rescuer, "", "") {
		t.Error("expected specRescuer to skip when no ally is being attacked")
	}
}

// ---------------------------------------------------------------------------
// Regression tests for BRIEF-2026-07-05-remaining-spec-proc-nil-crashes.md.
// ---------------------------------------------------------------------------

func TestSpecRemorter_NilChDoesNotPanic(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	if specRemorter(w, nil, mob, "", "") {
		t.Error("expected specRemorter to return false on the autonomous (nil ch) path")
	}
}

func TestSpecBreedKiller_NilChDoesNotPanic(t *testing.T) {
	w, player := newCombatTestWorld(t)
	player.Level = 10
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	// specBreedKiller only attacks on a 5% roll; the point of this test is
	// that the nil-ch state checks don't panic, not that the roll lands —
	// so a couple of calls suffice without skewing the shared RNG stream
	// for other unseeded-random tests later in the suite.
	specBreedKiller(w, nil, mob, "", "")
	specBreedKiller(w, nil, mob, "", "")
}

func TestSpecNeverDie_NilChDoesNotPanic_HealsToFull(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)
	mob.SetHealth(1)

	handled := specNeverDie(w, nil, mob, "", "")

	if !handled {
		t.Error("expected specNeverDie to report handled when healing a damaged mob")
	}
	if mob.GetHP() != mob.GetMaxHP() {
		t.Errorf("expected mob to be healed to max HP, got %d/%d", mob.GetHP(), mob.GetMaxHP())
	}
}

func TestSpecNeverDie_SkipsWhenAlreadyFull(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	if specNeverDie(w, nil, mob, "", "") {
		t.Error("expected specNeverDie to skip a mob already at full HP")
	}
}

func TestSpecBrainEater_NilChDoesNotPanic(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	if specBrainEater(w, nil, mob, "", "") {
		t.Error("expected specBrainEater to return false with no corpse in the room")
	}
}

func TestSpecTeleportVictim_NilChDoesNotPanic_SkipsWhenNotFighting(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	if specTeleportVictim(w, nil, mob, "", "") {
		t.Error("expected specTeleportVictim to skip a mob that isn't fighting")
	}
}

func TestSpecCastleGuardDown_NilChDoesNotPanic(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosStanding)

	if specCastleGuardDown(w, nil, mob, "", "") {
		t.Error("expected specCastleGuardDown to return false when no other mob is fighting")
	}
}

func TestSpecRescuer_SkipsWhenRescuerAlreadyFighting(t *testing.T) {
	w, player := newCombatTestWorld(t)
	rescuer, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob rescuer: %v", err)
	}
	ally, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob ally: %v", err)
	}
	rescuer.SetPosition(combat.PosStanding)
	rescuer.SetFighting(player.Name)
	ally.SetPosition(combat.PosStanding)
	ally.SetFighting(player.Name)

	if specRescuer(w, nil, rescuer, "", "") {
		t.Error("expected specRescuer to skip when it's already fighting")
	}
}
