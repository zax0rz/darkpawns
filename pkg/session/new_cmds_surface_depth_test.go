package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestTripOutputGatesMatchCAtRealDispatch(t *testing.T) {
	tests := []struct {
		name       string
		peaceful   bool
		skill      int
		mount      bool
		targetKind string
		flying     bool
		position   int
		args       []string
		want       string
	}{
		{
			name: "no skill",
			want: "You'd better leave the sneaky stuff to the thieves.\r\n",
		},
		{
			name:     "peaceful room",
			peaceful: true,
			skill:    1,
			want:     "This room just has such a peaceful, easy feeling...\r\n",
		},
		{
			name:  "missing target",
			skill: 1,
			want:  "Trip who?\r\n",
		},
		{
			name:       "mounted",
			skill:      1,
			mount:      true,
			targetKind: "mob",
			args:       []string{"target"},
			want:       "Dismount first!\r\n",
		},
		{
			name:       "self target",
			skill:      1,
			targetKind: "self",
			args:       []string{"hero"},
			want:       "You trip over your shoe laces...\n\r",
		},
		{
			name:       "flying target",
			skill:      1,
			targetKind: "player",
			flying:     true,
			args:       []string{"flyer"},
			want:       "You can't trip something that's FLYING!\r\n",
		},
		{
			name:       "sleeping target",
			skill:      1,
			targetKind: "player",
			position:   combat.PosSleeping,
			args:       []string{"sleeper"},
			want:       "Whats the point of doing that now?\r\n",
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := makeGateTestManager(t, test.peaceful)
			actor := makeGateSession(t, m, index+1, "Hero", 20)
			if test.skill > 0 {
				actor.player.SetSkill(game.SkillTrip, test.skill)
			}
			if test.mount {
				actor.player.MountName = "a pony"
			}

			switch test.targetKind {
			case "mob":
				if _, err := m.world.SpawnMob(5000, 1001); err != nil {
					t.Fatalf("SpawnMob: %v", err)
				}
			case "player":
				victim := game.NewPlayer(index+100, test.args[0], 1001)
				victim.SetLevel(20)
				if test.position != 0 {
					victim.SetPosition(test.position)
				}
				if test.flying {
					var foundFlyBit bool
					for bit, flag := range game.AffBitToEngineFlag {
						if flag&engine.AFFFlying != 0 {
							victim.SetAffect(bit, true)
							foundFlyBit = true
							break
						}
					}
					if !foundFlyBit {
						t.Fatal("C AFF_FLYING bit is not mapped")
					}
				}
				if err := m.world.AddPlayer(victim); err != nil {
					t.Fatalf("AddPlayer victim: %v", err)
				}
			case "self", "":
			default:
				t.Fatalf("unknown target kind %q", test.targetKind)
			}

			if err := ExecuteCommand(actor, "trip", test.args); err != nil {
				t.Fatalf("ExecuteCommand(trip): %v", err)
			}
			if got := readSendText(t, actor); got != test.want {
				t.Fatalf("trip output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPalmEarlyOutputGatesMatchCAtRealDispatch(t *testing.T) {
	t.Run("no argument", func(t *testing.T) {
		dprng.ResetStream(163)
		m := makeGateTestManager(t, false)
		actor := makeGateSession(t, m, 1, "Hero", 20)

		if err := ExecuteCommand(actor, "palm", nil); err != nil {
			t.Fatalf("ExecuteCommand(palm): %v", err)
		}
		if got, want := readSendText(t, actor), "Palm what?\r\n"; got != want {
			t.Fatalf("palm no-argument output = %q, want %q", got, want)
		}
	})

	t.Run("full carry count precedes no-argument refusal", func(t *testing.T) {
		dprng.ResetStream(163)
		m := makeGateTestManager(t, false)
		actor := makeGateSession(t, m, 1, "Hero", 20)
		actor.player.Stats.Dex = 10
		actor.player.Inventory.SetCapacity(actor.player.Stats.Str, actor.player.Stats.StrAdd, actor.player.Stats.Dex, actor.player.GetLevel())
		for index := 0; index < actor.player.MaxCarryItems(); index++ {
			obj := m.world.NewObjectFromProto(&parser.Obj{
				VNum:      6000 + index,
				Keywords:  "test object",
				ShortDesc: "a test object",
			}, 1001)
			if err := m.world.MoveObjectToPlayerInventory(obj, actor.player); err != nil {
				t.Fatalf("fill inventory at item %d: %v", index, err)
			}
		}

		if err := ExecuteCommand(actor, "palm", nil); err != nil {
			t.Fatalf("ExecuteCommand(palm): %v", err)
		}
		if got, want := readSendText(t, actor), "Your arms are already full!\r\n"; got != want {
			t.Fatalf("palm full-inventory output = %q, want %q", got, want)
		}
	})
}

func TestCircleBusyGateMatchesCAtRealDispatch(t *testing.T) {
	tests := []struct {
		name         string
		victimFights bool
		want         string
	}{
		{
			name:         "reciprocal fighting blocks a different target",
			victimFights: true,
			want:         "You're a little too busy right now!\r\n",
		},
		{
			name: "one-way fighting reaches the weapon gate",
			want: "You need to wield a weapon to make it a success.\r\n",
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := makeGateTestManager(t, false)
			actor := makeGateSession(t, m, index+1, "Hero", 20)
			victim := game.NewPlayer(index+100, "Opponent", 1001)
			victim.SetLevel(20)
			requested := game.NewPlayer(index+200, "Requested", 1001)
			requested.SetLevel(20)
			if err := m.world.AddPlayer(victim); err != nil {
				t.Fatalf("AddPlayer opponent: %v", err)
			}
			if err := m.world.AddPlayer(requested); err != nil {
				t.Fatalf("AddPlayer requested target: %v", err)
			}
			actor.player.SetFightingBody(victim)
			if test.victimFights {
				victim.SetFightingBody(actor.player)
			}

			if err := ExecuteCommand(actor, "circle", []string{"requested"}); err != nil {
				t.Fatalf("ExecuteCommand(circle): %v", err)
			}
			if got := readSendText(t, actor); got != test.want {
				t.Fatalf("circle output = %q, want %q", got, test.want)
			}
		})
	}
}
