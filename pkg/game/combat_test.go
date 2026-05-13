package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newCombatTestWorld builds a minimal World with a room and a mob for combat tests.
func newCombatTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "a training dummy"},
		},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	player.Level = 10
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	t.Cleanup(func() {
		w.StopAITicker()
	})

	return w, player
}

// spawnTargetMob spawns a training dummy in room 1001 for combat tests.
func spawnTargetMob(t *testing.T, w *World) *MobInstance {
	t.Helper()
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	return mob
}

// ---------------------------------------------------------------------------
// TestDoBackstab_NotThief — non-thief tries backstab, verify class check
// ---------------------------------------------------------------------------

func TestDoBackstab_NotThief(t *testing.T) {
	w, player := newCombatTestWorld(t)

	// Player has no backstab skill (default)
	mob := spawnTargetMob(t, w)

	result := DoBackstab(player, mob, w)
	if result.Success {
		t.Error("DoBackstab should fail for player without backstab skill")
	}
	if result.MessageToCh != "You have no idea how." {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
	if result.MessageToCh != "You have no idea how." {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

// ---------------------------------------------------------------------------
// TestDoBackstab_InCombat — can't backstab while fighting
// ---------------------------------------------------------------------------

func TestDoBackstab_InCombat(t *testing.T) {
	w, player := newCombatTestWorld(t)

	// Give player backstab skill
	player.SetSkill(SkillBackstab, 50)

	// Equip a weapon so the weapon check passes
	weapon, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Skip("weapon vnum 3002 not available, skipping")
	}
	_ = w.MoveObject(weapon, LocEquippedPlayer(player.Name, SlotWield))

	// Spawn a target mob
	target := spawnTargetMob(t, w)

	// Set player as already fighting
	player.SetFighting("SomeOtherGuy")

	result := DoBackstab(player, target, w)
	if result.Success {
		t.Error("DoBackstab should fail when player is already fighting")
	}
	if result.MessageToCh == "" {
		t.Error("expected non-empty failure message")
	}
}

// ---------------------------------------------------------------------------
// TestDoBackstab_TargetFighting — can't backstab a fighting target
// ---------------------------------------------------------------------------

func TestDoBackstab_TargetFighting(t *testing.T) {
	w, player := newCombatTestWorld(t)

	// Give player backstab skill
	player.SetSkill(SkillBackstab, 50)

	// Equip a weapon
	weapon, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Skip("weapon vnum 3002 not available, skipping")
	}
	_ = w.MoveObject(weapon, LocEquippedPlayer(player.Name, SlotWield))

	// Spawn target that is fighting
	target := spawnTargetMob(t, w)
	target.SetFighting("SomeoneElse")

	result := DoBackstab(player, target, w)
	if result.Success {
		t.Error("DoBackstab should fail when target is fighting")
	}
}

// ---------------------------------------------------------------------------
// TestDoBackstab_BareHands — backstab with bare hands uses default (1,4) damage
// ---------------------------------------------------------------------------

func TestDoBackstab_BareHands(t *testing.T) {
	w, player := newCombatTestWorld(t)

	// Give player backstab skill
	player.SetSkill(SkillBackstab, 50)

	// Don't equip any weapon — bare hands (1,4) still count
	target := spawnTargetMob(t, w)

	result := DoBackstab(player, target, w)
	// In CircleMUD, bare-handed backstab uses default damage (1,4)
	// Should NOT fail with "no idea how" or "need a weapon" message
	if result.MessageToCh == "You have no idea how." {
		t.Error("bare-handed backstab should not fail for missing skill")
	}
	if result.MessageToCh == "You need to wield a weapon to make it a success." {
		t.Error("bare-handed backstab should not fail for missing weapon (CircleMUD default is 1,4)")
	}
}

// ---------------------------------------------------------------------------
// TestDoBash_NotFighter — non-fighter tries bash
// ---------------------------------------------------------------------------

func TestDoBash_NotFighter(t *testing.T) {
	w, player := newCombatTestWorld(t)

	target := spawnTargetMob(t, w)

	result := DoBash(player, target)
	if result.Success {
		t.Error("DoBash should fail for player without bash skill")
	}
}

// ---------------------------------------------------------------------------
// TestDoBash_NoMovePoints — player with insufficient move
// ---------------------------------------------------------------------------

func TestDoBash_NoMovePoints(t *testing.T) {
	w, player := newCombatTestWorld(t)

	player.SetSkill(SkillBash, 50)
	player.Move = 0

	target := spawnTargetMob(t, w)

	result := DoBash(player, target)
	if result.Success {
		t.Error("DoBash should fail with no move points")
	}
	if result.MessageToCh != "You haven't the energy!" {
		t.Errorf("unexpected message: %q", result.MessageToCh)
	}
}

// ---------------------------------------------------------------------------
// TestDoBash_TargetSitting — can't bash someone already sitting
// ---------------------------------------------------------------------------

func TestDoBash_TargetSitting(t *testing.T) {
	w, player := newCombatTestWorld(t)

	player.SetSkill(SkillBash, 50)
	player.Move = 100

	target := spawnTargetMob(t, w)
	target.mu.Lock()
	target.Status = "sitting"
	target.mu.Unlock()

	result := DoBash(player, target)
	if result.Success {
		t.Error("DoBash should fail when target is sitting")
	}
}

// ---------------------------------------------------------------------------
// TestDoKick_NotMonk — non-monk tries kick
// ---------------------------------------------------------------------------

func TestDoKick_NotMonk(t *testing.T) {
	w, player := newCombatTestWorld(t)

	target := spawnTargetMob(t, w)

	result := DoKick(player, target)
	if result.Success {
		t.Error("DoKick should fail for player without kick skill")
	}
}

// ---------------------------------------------------------------------------
// TestDoTrip_NotThief — non-thief tries trip
// ---------------------------------------------------------------------------

func TestDoTrip_NotThief(t *testing.T) {
	w, player := newCombatTestWorld(t)

	target := spawnTargetMob(t, w)

	result := DoTrip(player, target)
	if result.Success {
		t.Error("DoTrip should fail for player without trip skill")
	}
}

// ---------------------------------------------------------------------------
// TestBackstabMult — verify multiplier calculation
// ---------------------------------------------------------------------------

func TestBackstabMult(t *testing.T) {
	tests := []struct {
		level int
		want  float64
	}{
		{0, 1.0},
		{1, 1.2},   // 1*0.2 + 1
		{5, 2.0},   // 5*0.2 + 1
		{10, 3.0},  // 10*0.2 + 1
		{20, 5.0},  // 20*0.2 + 1
		{30, 7.0},  // 30*0.2 + 1
		{31, 20.0}, // cap at level >= 31
		{50, 20.0}, // cap
		{100, 20.0},
	}
	for _, tt := range tests {
		got := backstabMult(tt.level)
		if got != tt.want {
			t.Errorf("backstabMult(%d) = %f, want %f", tt.level, got, tt.want)
		}
	}
}
