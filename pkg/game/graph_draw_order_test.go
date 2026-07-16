package game

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestHuntVictimEvasionUsesSequentialSpeechDraws(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Hunter Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
		{VNum: 1002, Name: "Target Room", Zone: 1, Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob := NewMobInstance(&parser.Mob{VNum: 9501, ShortDesc: "a hunter"}, 1001)
	target := NewPlayer(1, "Evasive", 1002)
	target.SetSkill("evasion", 100)
	w.players[target.Name] = target
	mob.SetHunting(target.Name)

	type drawRange struct{ from, to int }
	draws := make([]drawRange, 0, 3)
	results := []int{1, 1, 0} // evade; first speech misses; second speech fires
	previous := huntNumber
	huntNumber = func(from, to int) int {
		draws = append(draws, drawRange{from: from, to: to})
		if len(draws) > len(results) {
			t.Fatalf("unexpected extra hunt draw number(%d,%d)", from, to)
		}
		result := results[len(draws)-1]
		return result
	}
	t.Cleanup(func() { huntNumber = previous })

	w.huntVictim(mob)

	want := []drawRange{{1, 151}, {0, 6}, {0, 6}}
	if !reflect.DeepEqual(draws, want) {
		t.Fatalf("hunt draw sequence = %+v, want %+v", draws, want)
	}
	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("evaded hunter moved to room %d, want 1001", got)
	}
}

func TestHuntVictimSameRoomSkipsEvasionDraw(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Shared Room", Zone: 1}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob := NewMobInstance(&parser.Mob{VNum: 9502, ShortDesc: "a hunter"}, 1001)
	target := NewPlayer(1, "Cornered", 1001)
	target.SetSkill("evasion", 100)
	w.players[target.Name] = target
	mob.SetHunting(target.Name)

	previous := huntNumber
	huntNumber = func(from, to int) int {
		t.Fatalf("same-room hunter unexpectedly drew number(%d,%d)", from, to)
		return 0
	}
	t.Cleanup(func() { huntNumber = previous })

	w.huntVictim(mob)
}
