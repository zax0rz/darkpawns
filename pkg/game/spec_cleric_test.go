package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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
	mob.CopyBaseAttributes()
	mob.SetMaxHP(100)
	mob.SetHealth(100)
	mob.SetPosition(combat.PosFighting)
	mob.SetFightingBody(player)
	player.SetPosition(combat.PosFighting)
	player.SetFightingBody(mob)
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
	var commandName, commandArg string
	liveDirect := w.CharacterCommandDirectExecFunc
	w.CharacterCommandDirectExecFunc = func(body combat.Combatant, command string, argument string, subcmdVal int) CharacterCommandResult {
		commandName, commandArg = command, argument
		return liveDirect(body, command, argument, subcmdVal)
	}
	draws := scriptClericRolls(t, 0, 1) // lspell=2, then the always-offensive full-health gate
	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Error("eligible sitting cleric should consume its special tick")
	}
	if got := mob.GetPosition(); got != combat.PosStanding {
		t.Errorf("cleric position after do_stand = %d, want standing", got)
	}
	if commandName != "stand" || commandArg != "" {
		t.Errorf("direct command = %q %q, want stand with empty argument", commandName, commandArg)
	}
	wantDraws := []clericTestDraw{{from: 0, to: 10}, {from: 1, to: 2}}
	if len(*draws) != len(wantDraws) {
		t.Fatalf("cleric draws = %v, want %v", *draws, wantDraws)
	}
	for i, got := range *draws {
		if got != wantDraws[i] {
			t.Errorf("cleric draw %d = %+v, want %+v", i, got, wantDraws[i])
		}
	}
	if got := lastMsg(); !strings.Contains(got, "clambers to its feet") {
		t.Errorf("cleric do_stand room output = %q", got)
	}
}

// TestSpecCleric_Lspell12FallsToDefaultHarm pins C's switch fall-through:
// lspell 12 has no case, so it reaches default and casts harm
// (spec_procs.c:1532-1543).
func TestSpecCleric_Lspell12FallsToDefaultHarm(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 15)
	_ = lastMsg()              // discard the spawn-room line
	scriptClericRolls(t, 9, 1) // min(9+level/5,15) = 12; offense branch
	dprng.ResetStream(1)
	t.Cleanup(func() { dprng.ResetStream(1) })
	initialHP := player.GetHP()

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if got := lastMsg(); !strings.Contains(got, "utters the words") {
		t.Errorf("lspell=12 should fall to default's harm cast, got %q", got)
	}
	if got := player.GetHP(); got >= initialHP {
		t.Errorf("lspell=12 default harm did not damage its target: HP %d -> %d", initialHP, got)
	}
}

// TestSpecCleric_EarthquakeUsesNPCRoomMessage covers the lspell 17-19 arm
// and call_magic's NPC area-spell room narration (spec_procs.c:1527-1531;
// magic.c:1574-1577).
func TestSpecCleric_EarthquakeUsesNPCRoomMessage(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 20)
	_ = lastMsg()               // discard the spawn-room line
	scriptClericRolls(t, 13, 1) // min(13+4,20) = 17; offense branch

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if got := lastMsg(); !strings.Contains(got, "gracefully gestures and the earth begins to shake violently!") {
		t.Errorf("NPC earthquake room output = %q", got)
	}
}

// TestSpecCleric_CallLightningWeatherGate pins the OUTSIDE/sky/level/roll
// chain before the call-lightning cast (spec_procs.c:1491-1497).
func TestSpecCleric_CallLightningWeatherGate(t *testing.T) {
	originalWeather := weatherInfo
	t.Cleanup(func() { weatherInfo = originalWeather })
	for _, tc := range []struct {
		name          string
		level         int
		levelRoll     int
		sky           int
		indoors       bool
		weatherRoll   int
		wantLightning bool
	}{
		{name: "rain outdoors casts", level: 20, levelRoll: 13, sky: SkyRaining, weatherRoll: 0, wantLightning: true},
		{name: "dry outdoors refuses", level: 20, levelRoll: 13, sky: SkyCloudy},
		{name: "indoors refuses", level: 20, levelRoll: 13, sky: SkyRaining, indoors: true},
		{name: "low spell level refuses", level: 10, levelRoll: 10, sky: SkyRaining},
		{name: "weather roll refuses", level: 20, levelRoll: 13, sky: SkyRaining, weatherRoll: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			mob := prepareClericTestCombat(t, w, player, tc.level)
			room := w.GetRoomInWorld(mob.GetRoomVNum())
			room.Sector = SECT_INSIDE
			if tc.indoors {
				room.Flags = []string{"8"}
			}
			weatherInfo.Sky = tc.sky

			rolls := []int{tc.levelRoll, 1} // offense branch
			if tc.levelRoll+tc.level/5 >= 15 && !tc.indoors && tc.sky >= SkyRaining {
				rolls = append(rolls, tc.weatherRoll)
			}
			scriptClericRolls(t, rolls...)
			dprng.ResetStream(1)
			t.Cleanup(func() { dprng.ResetStream(1) })
			initialHP := player.GetHP()
			if got := specCleric(w, nil, mob, "", ""); !got {
				t.Fatal("eligible cleric should consume its special tick")
			}
			got := lastMsg()
			stared := strings.Contains(got, "stares into the sky")
			if stared != tc.wantLightning {
				t.Fatalf("call-lightning narration present = %v, want %v; output %q", stared, tc.wantLightning, got)
			}
			if tc.wantLightning && player.GetHP() >= initialHP {
				t.Fatalf("eligible weather gate did not damage the fighting target: HP %d -> %d; output %q", initialHP, player.GetHP(), got)
			}
		})
	}
}

// TestSpecCleric_BlindnessGateConsumesCDraw pins C's bitwise '&' expression:
// a blind cleric consumes Number(0,3) even when lspell < 4, but does not cast
// Cure Blindness on its blind opponent (spec_procs.c:1553-1557).
func TestSpecCleric_BlindnessGateConsumesCDraw(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := prepareClericTestCombat(t, w, player, 2)
	mob.SetHealth(40)
	player.SetAffect(affBlind, true)
	{
		p := *mob.Proto()
		p.AffectFlags = []string{"blind"}
		mob.SetProto(&p)
	}
	draws := scriptClericRolls(t, 0, 3, 0, 0)

	if got := specCleric(w, nil, mob, "", ""); !got {
		t.Fatal("eligible cleric should consume its special tick")
	}
	if !player.IsAffected(affBlind) {
		t.Fatal("lspell < 4 should consume the blindness draw without curing the blind opponent")
	}
	want := []clericTestDraw{{0, 2}, {1, 9}, {0, 3}, {0, 3}}
	if len(*draws) != len(want) {
		t.Fatalf("cleric draw count = %d, want %d (%v)", len(*draws), len(want), *draws)
	}
	for i, got := range *draws {
		if got != want[i] {
			t.Errorf("cleric draw %d = %+v, want %+v", i, got, want[i])
		}
	}
}
