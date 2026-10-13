package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSummonerIgnoresNonPlayingMemoryTargetWithoutRNGDraw(t *testing.T) {
	w, _, _ := newSpecProcTestWorld(t)
	target := NewPlayer(2, "Linkless", 1001)
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}
	w.PlayerHasPlayingDescriptor = func(*Player) bool { return false }
	mob := newSpecProcTestMob(t, w, 1001, 20)
	mob.Memory = []string{target.Name}

	seed := uint32(1)
	for dprng.New(seed).Number(0, 4) != 0 {
		seed++
	}
	dprng.ResetStream(seed)
	if specSummoner(w, nil, mob, "", "") {
		t.Fatal("summoner selected a target without a CON_PLAYING descriptor")
	}
	if got, want := dprng.Next(), dprng.New(seed).Next(); got != want {
		t.Fatalf("summoner advanced RNG for a linkless target: next draw %d, want untouched draw %d", got, want)
	}
}

func TestHuntVictimLeavesLinklessHuntingIDUnresolved(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Hunter room", Zone: 1},
		{VNum: 1002, Name: "Prey room", Zone: 1},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	target := NewPlayer(1, "Linkless", 1002)
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}
	w.PlayerHasPlayingDescriptor = func(*Player) bool { return false }
	mob := newSpecProcTestMob(t, w, 1001, 20)
	mob.SetHunting(target.Name)

	w.huntVictim(mob)
	if got, want := mob.GetHunting(), target.Name; got != want {
		t.Fatalf("hunting lookup after missing descriptor = %q, want C's unresolved ID %q retained", got, want)
	}
}
