package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
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

func TestShootPCLevelWindow(t *testing.T) {
	for _, level := range []int{9, 10, 30, 31} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			s, arrow := newShootGateSession(t)
			target := game.NewPlayer(2, "Victim", 1002)
			target.SetLevel(level)
			if err := s.world.AddPlayer(target); err != nil {
				t.Fatal(err)
			}
			if level == 9 || level == 31 {
				assertShootRefusal(t, s, arrow, "Victim", "Maybe that isn't such a great idea...\r\n")
			} else {
				// Only the gate is claimed here; outcomes stay blocked for Train 2.
				dprng.ResetStream(71)
				if err := CmdShoot(s, []string{"arrow", "north", "Victim"}); err != nil {
					t.Fatal(err)
				}
				output := strings.Join(s.getMessages(), "")
				if output == "Maybe that isn't such a great idea...\r\n" || output == "Twang...\r\n" || output == "" {
					t.Fatalf("allowed boundary %d did not reach the target outcome: %q", level, output)
				}
			}
		})
	}
}

func TestShootTargetFightingRefusalAndOrder(t *testing.T) {
	for _, kind := range []string{"pc", "mob", "sentinel", "level-before-fighting"} {
		t.Run(kind, func(t *testing.T) {
			s, arrow := newShootGateSession(t)
			var target combat.Combatant
			name := "Victim"
			if kind == "pc" || kind == "level-before-fighting" {
				pc := game.NewPlayer(2, name, 1002)
				pc.SetLevel(10)
				if kind == "level-before-fighting" {
					pc.SetLevel(9)
				}
				if err := s.world.AddPlayer(pc); err != nil {
					t.Fatal(err)
				}
				target = pc
			} else {
				mob, err := s.world.SpawnMobQuiet(3001, 1002)
				if err != nil {
					t.Fatal(err)
				}
				if kind == "mob" {
					mob.ClearMobFlag(1)
				} // C MOB_SENTINEL, src/structs.h:248
				target = mob
				name = "guard"
			}
			target.SetFighting("Other")
			want := "It looks like they are fighting, you can't aim properly.\r\n"
			if kind == "level-before-fighting" {
				want = "Maybe that isn't such a great idea...\r\n"
			}
			hp, room, position := target.GetHP(), target.GetRoom(), target.GetPosition()
			assertShootRefusal(t, s, arrow, name, want)
			if target.GetHP() != hp || target.GetRoom() != room || target.GetPosition() != position || target.GetFighting() != "Other" {
				t.Fatal("refusal changed target state")
			}
		})
	}
}
