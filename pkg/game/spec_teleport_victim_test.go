package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func prepareTeleportVictim(t *testing.T) (*World, *Player, *MobInstance, func() string) {
	t.Helper()
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.SetPosition(combat.PosStanding)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	lastMsg() // discard the mob-arrival act
	return w, player, mob, lastMsg
}

func addTeleportDestination(t *testing.T, w *World) {
	t.Helper()
	w.mu.Lock()
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Destination", Zone: 1}
	w.roomOrder = append(w.roomOrder, 1002)
	w.mu.Unlock()
}

func TestSpecTeleportVictim_EntryGates(t *testing.T) {
	cases := []struct {
		name string
		call func(*World, *Player, *MobInstance) bool
	}{
		{
			name: "command",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				return specTeleportVictim(w, player, mob, "look", "")
			},
		},
		{
			name: "not fighting",
			call: func(w *World, _ *Player, mob *MobInstance) bool {
				mob.StopFighting()
				return specTeleportVictim(w, nil, mob, "", "")
			},
		},
		{
			name: "sleeping",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.SetPosition(combat.PosSleeping)
				return specTeleportVictim(w, player, mob, "", "")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, player, mob, lastMsg := prepareTeleportVictim(t)
			if tc.call(w, player, mob) {
				t.Fatal("gated invocation was handled")
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("gated invocation emitted output: %q", got)
			}
		})
	}
}

func TestSpecTeleportVictim_ScoffSpeechTeleportAndLandingLook(t *testing.T) {
	w, player, mob, lastMsg := prepareTeleportVictim(t)
	addTeleportDestination(t, w)
	oldRoomObserver := NewPlayer(2, "OldObserver", 1001)
	if err := w.AddPlayer(oldRoomObserver); err != nil {
		t.Fatalf("AddPlayer old observer: %v", err)
	}
	newRoomObserver := NewPlayer(3, "NewObserver", 1002)
	if err := w.AddPlayer(newRoomObserver); err != nil {
		t.Fatalf("AddPlayer new observer: %v", err)
	}
	lastMsg() // discard observer setup output

	// With two rooms, seed 2 selects the second room in C's stable room order.
	dprng.ResetStream(2)
	if !specTeleportVictim(w, nil, mob, "", "") {
		t.Fatal("standing fighting teleport victim should handle")
	}
	if got, want := player.GetRoomVNum(), 1002; got != want {
		t.Fatalf("victim room = %d, want %d", got, want)
	}
	if player.IsFighting() || mob.IsFighting() {
		t.Fatal("teleport should clear both sides of combat")
	}

	got := lastMsg()
	for _, want := range []string{
		"A test mob scoffs at the idea.",
		"A test mob says, 'You can't harm me, mortal. Begone.'",
		"The world around you turns black and you suddenly find yourself..",
		"Tester slowly fades out of existence and is gone.",
		"Tester slowly fades into existence.",
		"Destination",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("teleport victim output missing %q: %q", want, got)
		}
	}
}

func TestSpecTeleportVictim_NonIntelligentMobSkipsSpeech(t *testing.T) {
	w, player, mob, lastMsg := prepareTeleportVictim(t)
	addTeleportDestination(t, w)
	mob.Prototype.Race = 12 // RACE_HORSE is absent from C intelligent_races[].
	dprng.ResetStream(2)

	if !specTeleportVictim(w, nil, mob, "", "") {
		t.Fatal("standing fighting teleport victim should handle")
	}
	got := lastMsg()
	if !strings.Contains(got, "A test mob scoffs at the idea.") {
		t.Fatalf("missing scoff social: %q", got)
	}
	if strings.Contains(got, "You can't harm me, mortal. Begone.") {
		t.Fatalf("non-intelligent mob spoke: %q", got)
	}
	if got, want := player.GetRoomVNum(), 1002; got != want {
		t.Fatalf("victim room = %d, want %d", got, want)
	}
}

func TestSpecTeleportVictim_ResolvesMobCombatTarget(t *testing.T) {
	w, _, mob, lastMsg := prepareTeleportVictim(t)
	target := newSpecProcTestMob(t, w, mob.GetRoomVNum(), 10)
	target.SetFighting(mob.GetName())
	mob.SetTarget(target)
	mob.SetFighting(target.GetName())
	lastMsg()

	resolved := mobFightingTarget(w, mob)
	if resolved != target {
		t.Fatalf("resolved combat target = %v, want target mob %q", resolved, target.GetName())
	}
}

func TestSpecTeleportVictim_SittingStopsAtNativeCallMagicGate(t *testing.T) {
	w, player, mob, lastMsg := prepareTeleportVictim(t)
	addTeleportDestination(t, w)
	mob.SetPosition(combat.PosSitting)
	dprng.ResetStream(2)

	if !specTeleportVictim(w, nil, mob, "", "") {
		t.Fatal("sitting teleport victim procedure should report handled")
	}
	if got, want := player.GetRoomVNum(), 1001; got != want {
		t.Fatalf("sitting call_magic gate moved victim to %d, want %d", got, want)
	}
	if got := lastMsg(); !strings.Contains(got, "A test mob scoffs at the idea.") ||
		!strings.Contains(got, "A test mob says, 'You can't harm me, mortal. Begone.'") ||
		strings.Contains(got, "The world around you turns black") {
		t.Fatalf("sitting call_magic gate output = %q", got)
	}
}
