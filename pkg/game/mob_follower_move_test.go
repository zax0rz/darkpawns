package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// perform_move's follower loop walks ch->followers, players and mobiles alike
// (src/act.movement.c:342-351): a charmed pet follows its leader. Egon's
// playtest report (2026-10-16): bought horses never followed.
func TestNPCFollowerFollowsLeader(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Stable", Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Street", Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
		},
		Mobs: []parser.Mob{{VNum: 2001, Keywords: "horse", ShortDesc: "a horse", Level: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	leader := NewPlayer(1, "Rider", 1001)
	leader.SetMove(100)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatal(err)
	}
	horse, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	horse.SetAffected(affCharm)
	AddFollowerMob(w, horse, leader)

	if r := w.DoMove(leader, "north"); !r.Success {
		t.Fatal("leader could not move north")
	}
	if got := horse.GetRoom(); got != 1002 {
		t.Fatalf("horse room = %d (pos %d, following %q), want 1002", got, horse.GetPosition(), horse.GetFollowing())
	}
}
