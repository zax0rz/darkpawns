package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newShootGateSession(t *testing.T) (*killPayoutSession, *game.ObjectInstance) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}}, {VNum: 1002}},
		Objs:  []parser.Obj{{VNum: 2001, Keywords: "arrow", ShortDesc: "an arrow", TypeFlag: int(game.ItemMissile)}, {VNum: 2002, Keywords: "bow", ShortDesc: "a bow", TypeFlag: int(game.ItemFireWeapon), WearFlags: [4]int{1 << 13}}},
		Mobs:  []parser.Mob{{VNum: 3001, Keywords: "guard target", ShortDesc: "a guard", ActionFlags: []string{"sentinel"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	p := game.NewPlayer(1, "Shooter", 1001)
	p.SetSkill(game.SkillShoot, 100)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	arrow, err := w.SpawnObject(2001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(arrow, p); err != nil {
		t.Fatal(err)
	}
	bow, err := w.SpawnObject(2002, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObject(bow, game.LocEquippedPlayer(p.Name, game.SlotWield)); err != nil {
		t.Fatal(err)
	}
	return &killPayoutSession{player: p, world: w}, arrow
}

func assertShootRefusal(t *testing.T, s *killPayoutSession, arrow *game.ObjectInstance, target, want string) {
	t.Helper()
	dprng.ResetStream(71)
	next := dprng.Next()
	dprng.ResetStream(71)
	location := arrow.Location
	wait := s.player.GetWaitState()
	if err := CmdShoot(s, []string{"arrow", "north", target}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.getMessages(), ""); got != want {
		t.Fatalf("shoot refusal=%q want %q", got, want)
	}
	if arrow.Location != location {
		t.Fatalf("refusal moved projectile: %+v", arrow.Location)
	}
	if s.player.GetWaitState() != wait {
		t.Fatal("refusal changed wait")
	}
	if got := dprng.Next(); got != next {
		t.Fatalf("refusal consumed RNG: %d want %d", got, next)
	}
}

func TestShootFallbackReachesSentinelWithoutConsumingProjectile(t *testing.T) {
	s, arrow := newShootGateSession(t)
	if _, err := s.world.SpawnMobQuiet(3001, 1002); err != nil {
		t.Fatal(err)
	}
	assertShootRefusal(t, s, arrow, "nobody", "You cannot see well enough to aim...\r\n")
}
