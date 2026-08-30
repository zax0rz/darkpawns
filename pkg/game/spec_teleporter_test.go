package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func prepareTeleporter(t *testing.T) (*World, *Player, *MobInstance, func() string) {
	t.Helper()
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.SetMaxHP(100)
	mob.SetHealth(1)
	mob.SetPosition(combat.PosStanding)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	return w, player, mob, lastMsg
}

func TestSpecTeleporter_EntryGates(t *testing.T) {
	cases := []struct {
		name string
		call func(*World, *Player, *MobInstance) bool
	}{
		{
			name: "command",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				return specTeleporter(w, player, mob, "look", "")
			},
		},
		{
			name: "not fighting",
			call: func(w *World, _ *Player, mob *MobInstance) bool {
				mob.StopFighting()
				return specTeleporter(w, nil, mob, "", "")
			},
		},
		{
			name: "sleeping",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.SetPosition(combat.PosSleeping)
				return specTeleporter(w, player, mob, "", "")
			},
		},
		{
			name: "full health",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.SetHealth(mob.GetMaxHP())
				return specTeleporter(w, player, mob, "", "")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, player, mob, lastMsg := prepareTeleporter(t)
			lastMsg() // discard the mob-arrival act from SpawnMob
			if got := tc.call(w, player, mob); got {
				t.Fatal("specTeleporter handled a gated invocation")
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("gated invocation emitted player-visible output: %q", got)
			}
		})
	}
}

func TestSpecTeleporter_LowHealthTeleportsMobAndUsesRoomActs(t *testing.T) {
	w, _, mob, lastMsg := prepareTeleporter(t)
	lastMsg() // discard the mob-arrival act from SpawnMob

	w.mu.Lock()
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Destination", Zone: 1}
	w.roomOrder = append(w.roomOrder, 1002)
	w.mu.Unlock()
	observer := NewPlayer(2, "Observer", 1002)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	lastMsg() // keep the assertion scoped to the special-procedure call

	// With two rooms, C's first room-index draw for seed 2 selects RNUM 1,
	// which is VNUM 1002 in this fixture's stable room order.
	dprng.ResetStream(2)
	if !specTeleporter(w, nil, mob, "", "") {
		t.Fatal("low-health awake fighting teleporter should handle")
	}
	if got, want := mob.GetRoomVNum(), 1002; got != want {
		t.Fatalf("teleporter room = %d, want %d", got, want)
	}
	if got := mob.GetFighting(); got != "" {
		t.Fatalf("teleport should stop mob combat, still fighting %q", got)
	}

	got := lastMsg()
	for _, want := range []string{
		"My work here is done.",
		"slowly fades out of existence and is gone.",
		"slowly fades into existence.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("teleporter output missing %q: %q", want, got)
		}
	}
}

func TestSpecTeleporter_RestingMobUsesNativeCallMagicPositionGate(t *testing.T) {
	w, _, mob, lastMsg := prepareTeleporter(t)
	lastMsg() // discard the mob-arrival act from SpawnMob
	mob.SetPosition(combat.PosResting)

	w.mu.Lock()
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Destination", Zone: 1}
	w.roomOrder = append(w.roomOrder, 1002)
	w.mu.Unlock()
	dprng.ResetStream(2)

	if !specTeleporter(w, nil, mob, "", "") {
		t.Fatal("resting awake teleporter should handle")
	}
	if got, want := mob.GetRoomVNum(), 1002; got != want {
		t.Fatalf("resting teleporter room = %d, want %d", got, want)
	}
}

func TestSpecTeleporter_SittingMobStopsAtNativeCallMagicGate(t *testing.T) {
	w, _, mob, lastMsg := prepareTeleporter(t)
	lastMsg() // discard the mob-arrival act from SpawnMob
	mob.SetPosition(combat.PosSitting)

	w.mu.Lock()
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Destination", Zone: 1}
	w.roomOrder = append(w.roomOrder, 1002)
	w.mu.Unlock()
	dprng.ResetStream(2)

	if !specTeleporter(w, nil, mob, "", "") {
		t.Fatal("sitting teleporter procedure should report handled")
	}
	if got, want := mob.GetRoomVNum(), 1001; got != want {
		t.Fatalf("sitting teleporter room = %d, want %d", got, want)
	}
	if got := lastMsg(); !strings.Contains(got, "My work here is done.") || strings.Contains(got, "fades out") {
		t.Fatalf("sitting call_magic gate output = %q", got)
	}
}
