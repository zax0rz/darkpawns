package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newSummonerTestWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{
		{
			VNum:  1001,
			Name:  "Cleric Sanctuary",
			Zone:  1,
			Exits: map[string]parser.Exit{"north": {ToRoom: 1002}},
		},
		{
			VNum:  1002,
			Name:  "Far Hall",
			Zone:  1,
			Exits: map[string]parser.Exit{"south": {ToRoom: 1001}},
		},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func TestSpecSummonerUsesDescriptorOrderCanSeeAndNumber0To3(t *testing.T) {
	w := newSummonerTestWorld(t)

	older := NewPlayer(1, "Older", 1002)
	newer := NewPlayer(2, "Newer", 1002)
	older.Flags |= 1 << PrfSummonable
	newer.Flags |= 1 << PrfSummonable
	if err := w.AddPlayer(older); err != nil {
		t.Fatalf("AddPlayer older: %v", err)
	}
	if err := w.AddPlayer(newer); err != nil {
		t.Fatalf("AddPlayer newer: %v", err)
	}
	w.PlayerHasPlayingDescriptor = func(*Player) bool { return true }

	mob := newSpecProcTestMob(t, w, 1001, 35)
	mob.Remember(older.Name)
	mob.Remember(newer.Name)

	seed := uint32(1)
	for dprng.New(seed).Number(0, 3) != 0 {
		seed++
	}
	dprng.ResetStream(seed)

	if !specSummoner(w, nil, mob, "", "") {
		t.Fatal("specSummoner returned false when dprng.Number(0, 3) == 0")
	}
	// C walks descriptor_list (newest connection first), so Newer is summoned
	// into room 1001 and Older stays in room 1002.
	if got := newer.GetRoomVNum(); got != 1001 {
		t.Fatalf("newest remembered player room = %d, want 1001", got)
	}
	if got := older.GetRoomVNum(); got != 1002 {
		t.Fatalf("older remembered player room = %d, want 1002", got)
	}
}

func TestSpecSummonerPrioritizesHuntingOverMemory(t *testing.T) {
	w := newSummonerTestWorld(t)

	hunted := NewPlayer(1, "Hunted", 1002)
	remembered := NewPlayer(2, "Remembered", 1002)
	hunted.Flags |= 1 << PrfSummonable
	remembered.Flags |= 1 << PrfSummonable
	if err := w.AddPlayer(hunted); err != nil {
		t.Fatalf("AddPlayer hunted: %v", err)
	}
	if err := w.AddPlayer(remembered); err != nil {
		t.Fatalf("AddPlayer remembered: %v", err)
	}
	w.PlayerHasPlayingDescriptor = func(*Player) bool { return true }

	mob := newSpecProcTestMob(t, w, 1001, 35)
	mob.Remember(remembered.Name)
	w.SetHuntingTarget(mob, hunted)

	seed := uint32(1)
	for dprng.New(seed).Number(0, 3) != 0 {
		seed++
	}
	dprng.ResetStream(seed)

	if !specSummoner(w, nil, mob, "", "") {
		t.Fatal("specSummoner returned false for hunting target")
	}
	if got := hunted.GetRoomVNum(); got != 1001 {
		t.Fatalf("hunted player room = %d, want 1001", got)
	}
	if got := remembered.GetRoomVNum(); got != 1002 {
		t.Fatalf("remembered player room = %d, want 1002", got)
	}
}
