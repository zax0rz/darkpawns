package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func sleeperTestWorld(t *testing.T, flags []string) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum: 1001, Name: "Sleeper Room", Zone: 1, Flags: flags,
	}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func sleeperTestPlayer(name string) *Player {
	p := NewPlayer(1, name, 1001)
	p.SetPosition(combat.PosStanding)
	p.SetSkill(SkillSleeper, 100)
	return p
}

func TestDoSleeperDepthGateOrder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*World, *Player, combat.Combatant)
		want  string
	}{
		{
			name: "no skill",
			setup: func(_ *World, ch *Player, _ combat.Combatant) {
				ch.SetSkill(SkillSleeper, 0)
			},
			want: "You have no idea how.",
		},
		{
			name: "actor fighting",
			setup: func(_ *World, ch *Player, _ combat.Combatant) {
				ch.SetFighting("someone")
			},
			want: "You can't do this while fighting!",
		},
		{
			name: "mounted",
			setup: func(_ *World, ch *Player, _ combat.Combatant) {
				ch.MountName = "horse"
			},
			want: "Dismount first!",
		},
		{
			name: "peaceful",
			setup: func(w *World, _ *Player, _ combat.Combatant) {
				w.GetRoomInWorld(1001).Flags = []string{"peaceful"}
			},
			want: "This room just has such a peaceful, easy feeling...",
		},
		{
			name: "wielded",
			setup: func(_ *World, ch *Player, _ combat.Combatant) {
				equipWeapon(t, ch, makeFidelityWeapon(990190, 3))
			},
			want: "You can't get a good grip on them while you are holding that weapon!",
		},
		{
			name:  "nil target",
			setup: func(_ *World, _ *Player, _ combat.Combatant) {},
			want:  "Sleeper who?",
		},
		{
			name:  "self",
			setup: func(_ *World, _ *Player, _ combat.Combatant) {},
			want:  "Can't get to sleep fast enough, huh?",
		},
		{
			name:  "non-outlaw player",
			setup: func(_ *World, _ *Player, _ combat.Combatant) {},
			want:  "You can not sleeper them because you are not an Outlaw!",
		},
		{
			name: "target fighting",
			setup: func(_ *World, ch *Player, target combat.Combatant) {
				ch.SetPlrFlag(PlrOutlaw, true)
				target.SetFighting("someone")
			},
			want: "You can't get a good grip on them while they're fighting!",
		},
		{
			name: "sleeping target",
			setup: func(_ *World, ch *Player, target combat.Combatant) {
				ch.SetPlrFlag(PlrOutlaw, true)
				target.SetPosition(combat.PosSleeping)
			},
			want: "What's the point of doing that now?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := sleeperTestWorld(t, nil)
			ch := sleeperTestPlayer("Sleeper")
			target := sleeperTestPlayer("Target")
			if tt.name == "nil target" {
				tt.setup(w, ch, nil)
				result := DoSleeper(ch, nil, w)
				if result.MessageToCh != tt.want {
					t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
				}
				return
			}
			if tt.name == "self" {
				tt.setup(w, ch, ch)
				result := DoSleeper(ch, ch, w)
				if result.MessageToCh != tt.want {
					t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
				}
				return
			}
			tt.setup(w, ch, target)
			result := DoSleeper(ch, target, w)
			if result.MessageToCh != tt.want {
				t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
			}
		})
	}
}

func TestDoSleeperMobProtectionAndSuccessShape(t *testing.T) {
	ch := sleeperTestPlayer("Sleeper")
	mob := NewMob(&parser.Mob{
		VNum:        990191,
		Keywords:    "aware target",
		ShortDesc:   "an aware target",
		Position:    combat.PosStanding,
		ActionFlags: []string{"AWARE"},
	}, 1001)
	result := DoSleeper(ch, mob, nil)
	if result.Success {
		t.Fatal("aware mob should force the failure arm")
	}
	if !result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Fatalf("failure result = %+v, want post-message retaliation", result)
	}
	if result.WaitCh != 2 {
		t.Fatalf("failure wait = %d, want 2 violence rounds", result.WaitCh)
	}

	noSleepMob := NewMob(&parser.Mob{
		VNum:        990193,
		Keywords:    "nosleep target",
		ShortDesc:   "a nosleep target",
		Position:    combat.PosStanding,
		ActionFlags: []string{"NOSLEEP"},
	}, 1001)
	if result := DoSleeper(sleeperTestPlayer("Sleeper"), noSleepMob, nil); result.Success || !result.RetaliateHit {
		t.Fatalf("nosleep mob result = %+v, want forced failure and retaliation", result)
	}

	dprng.ResetStream(1)
	ch = sleeperTestPlayer("Sleeper")
	mob = NewMob(&parser.Mob{
		VNum:      990192,
		Keywords:  "target",
		ShortDesc: "a target",
		Position:  combat.PosStanding,
	}, 1001)
	result = DoSleeper(ch, mob, nil)
	if !result.Success {
		t.Fatalf("seeded legal sleeper failed: %+v", result)
	}
	if result.MessageToRoomSecond == "" || !result.RoomIncludesTarget || result.WaitCh != 2 || !result.SleepTarget {
		t.Fatalf("success audience shape = %+v", result)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillSleeper || !result.DeferredImproveAfterRoom {
		t.Fatalf("success improvement shape = %+v", result)
	}
}

func TestDoSleeperPlayerLevelWindow(t *testing.T) {
	dprng.ResetStream(1)
	ch := sleeperTestPlayer("Sleeper")
	ch.SetPlrFlag(PlrOutlaw, true)
	target := sleeperTestPlayer("Target")
	target.SetLevel(ch.GetLevel() + 4)

	result := DoSleeper(ch, target, nil)
	if result.Success || !result.RetaliateHit {
		t.Fatalf("out-of-window player result = %+v, want forced failure and retaliation", result)
	}
}
