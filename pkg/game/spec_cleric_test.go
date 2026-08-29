package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type clericTestDraw struct {
	from int
	to   int
}

func scriptClericRolls(t *testing.T, values ...int) *[]clericTestDraw {
	t.Helper()
	original := clericNumber
	var draws []clericTestDraw
	index := 0
	clericNumber = func(from, to int) int {
		draws = append(draws, clericTestDraw{from: from, to: to})
		if index >= len(values) {
			t.Fatalf("cleric RNG exhausted at Number(%d,%d)", from, to)
		}
		value := values[index]
		index++
		return value
	}
	t.Cleanup(func() { clericNumber = original })
	return &draws
}

func prepareClericTestCombat(t *testing.T, w *World, player *Player, level int) *MobInstance {
	t.Helper()
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), level)
	mob.Intel = 8
	mob.Wis = 8
	mob.SetMaxHP(100)
	mob.SetHealth(100)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.GetName())
	player.SetPosition(combat.PosFighting)
	player.SetFighting(mob.GetName())
	return mob
}

// TestSpecCleric_EntryGatesAndStand covers SPECIAL(cleric)'s AWAKE, command,
// and C do_stand entry path (spec_procs.c:1425-1442). A sleeping cleric must
// not stand; a sitting cleric emits the room-only do_stand line before acting.
func TestSpecCleric_EntryGatesAndStand(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 10)
	_ = lastMsg() // discard the spawn-room line

	mob.SetPosition(combat.PosSleeping)
	if got := specCleric(w, nil, mob, "", ""); got {
		t.Error("sleeping cleric should not consume its special tick")
	}
	if got := mob.GetPosition(); got != combat.PosSleeping {
		t.Errorf("sleeping cleric position = %d, want sleeping", got)
	}

	mob.SetPosition(combat.PosSitting)
	scriptClericRolls(t, 0, 0) // lspell=2, then the always-offensive full-health gate
	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Error("eligible sitting cleric should consume its special tick")
	}
	if got := mob.GetPosition(); got != combat.PosStanding {
		t.Errorf("cleric position after do_stand = %d, want standing", got)
	}
	if got := lastMsg(); !strings.Contains(got, "clambers to its feet") {
		t.Errorf("cleric do_stand room output = %q", got)
	}
}

// TestSpecCleric_Lspell12IsAnOffensiveNoOp pins the intentional hole in C's
// offensive switch: lspell 12 has no spell case (spec_procs.c:1558-1578).
func TestSpecCleric_Lspell12IsAnOffensiveNoOp(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 15)
	_ = lastMsg()              // discard the spawn-room line
	scriptClericRolls(t, 9, 0) // min(9+level/5,15) = 12; offense branch

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if got := lastMsg(); got != "" {
		t.Errorf("lspell=12 offensive no-op emitted player bytes: %q", got)
	}
}

// TestSpecCleric_EarthquakeUsesNPCRoomMessage covers the lspell 17-19 arm
// and call_magic's NPC area-spell room narration (spec_procs.c:1558-1578;
// magic.c:1573-1579).
func TestSpecCleric_EarthquakeUsesNPCRoomMessage(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 20)
	_ = lastMsg()               // discard the spawn-room line
	scriptClericRolls(t, 13, 0) // min(13+4,20) = 17; offense branch

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if got := lastMsg(); !strings.Contains(got, "gracefully gestures and the earth begins to shake violently!") {
		t.Errorf("NPC earthquake room output = %q", got)
	}
}

// TestSpecCleric_CallLightningWeatherGate pins the OUTSIDE/sky/level/roll
// chain before the call-lightning cast (spec_procs.c:1570-1575).
func TestSpecCleric_CallLightningWeatherGate(t *testing.T) {
	originalWeather := weatherInfo
	t.Cleanup(func() { weatherInfo = originalWeather })
	weatherInfo.Sky = SkyRaining

	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 20)
	scriptClericRolls(t, 13, 0, 0) // lspell=17; offense; lightning succeeds

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if got := lastMsg(); !strings.Contains(got, "stares into the sky") {
		t.Errorf("call-lightning pre-cast room output = %q", got)
	}
}

// TestSpecCleric_BlindnessGateConsumesCDraw pins C's bitwise '&' expression:
// an innately blind cleric consumes Number(0,3) even when lspell < 4, then
// proceeds to its ordinary self-heal draw (spec_procs.c:1579-1605).
func TestSpecCleric_BlindnessGateConsumesCDraw(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 10)
	mob.SetHealth(40)
	mob.Prototype.AffectFlags = []string{"blind"}
	draws := scriptClericRolls(t, 0, 2, 1, 0)

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	want := []clericTestDraw{{0, 10}, {0, 8}, {0, 3}, {0, 3}}
	if len(*draws) != len(want) {
		t.Fatalf("cleric draw count = %d, want %d (%v)", len(*draws), len(want), *draws)
	}
	for i, got := range *draws {
		if got != want[i] {
			t.Errorf("cleric draw %d = %+v, want %+v", i, got, want[i])
		}
	}
}
