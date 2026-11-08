package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func subdueTestWorld(t *testing.T, flags []string) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum: 1001, Name: "Subdue Room", Zone: 1, Flags: flags,
	}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	// Quiesce AI before the test body: background draws must not perturb the
	// draw-parity assertions below (same discipline as kill_payout_test.go).
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	return w
}

func subdueTestPlayer(name string) *Player {
	p := NewPlayer(1, name, 1001)
	p.SetPosition(combat.PosStanding)
	p.SetSkill(SkillSubdue, 100)
	return p
}

// assertZeroDrawsBeyond proves fn consumed no draws at all — C returns before
// the percent roll on these arms (R3a).
func assertZeroDrawsBeyond(t *testing.T, fn func()) {
	t.Helper()
	assertDrawsBeyond(t, 0, fn)
}

// assertDrawsBeyond proves fn consumed exactly n draws: the probe draw lands on
// the same stream position as a reference stream that consumed n draws first.
// Ranges don't matter (one uniform per Number call), only the count does.
func assertDrawsBeyond(t *testing.T, n int, fn func()) {
	t.Helper()
	dprng.ResetStream(1)
	fn()
	got := dprng.Number(0, 999)
	dprng.ResetStream(1)
	for i := 0; i < n; i++ {
		dprng.Number(1, 101)
	}
	want := dprng.Number(0, 999)
	if got != want {
		t.Fatalf("call consumed a divergent draw count: next=%d, want %d after %d draws", got, want, n)
	}
}

func TestDoSubdueDepthGateOrder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*World, *Player, *Player)
		want  string
	}{
		{
			name: "no skill",
			setup: func(_ *World, ch *Player, _ *Player) {
				ch.SetSkill(SkillSubdue, 0)
			},
			want: "You have no idea how!",
		},
		{
			name: "actor fighting",
			setup: func(_ *World, ch *Player, _ *Player) {
				ch.SetFightingBody(NewPlayer(99999, "someone", 1001))
			},
			want: "You're too busy right now!",
		},
		{
			name:  "nil target",
			setup: func(_ *World, _ *Player, _ *Player) {},
			want:  "Subdue who?",
		},
		{
			name:  "self",
			setup: func(_ *World, _ *Player, _ *Player) {},
			want:  "Aren't we funny today...",
		},
		{
			name: "peaceful",
			setup: func(w *World, _ *Player, _ *Player) {
				w.GetRoomInWorld(1001).Flags = []string{"peaceful"}
			},
			want: "You can't contemplate violence in such a place!",
		},
		{
			name: "mounted",
			setup: func(_ *World, ch *Player, _ *Player) {
				ch.MountName = "horse"
			},
			want: "Dismount first!",
		},
		{
			name: "target fighting",
			setup: func(_ *World, _ *Player, target *Player) {
				target.SetFightingBody(NewPlayer(99999, "someone", 1001))
			},
			want: "You can't get close enough!",
		},
		{
			name:  "non-outlaw player",
			setup: func(_ *World, _ *Player, _ *Player) {},
			want:  "You can't subdue them because you are not an Outlaw!",
		},
		{
			name: "stunned target",
			setup: func(_ *World, ch *Player, target *Player) {
				ch.SetPlrFlag(PlrOutlaw, true)
				target.SetPosition(combat.PosStunned)
			},
			want: "What's the point of doing that now?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := subdueTestWorld(t, nil)
			ch := subdueTestPlayer("Subduer")
			target := subdueTestPlayer("Target")
			if tt.name == "nil target" {
				tt.setup(w, ch, nil)
				result := DoSubdue(ch, nil, w)
				if result.MessageToCh != tt.want {
					t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
				}
				return
			}
			if tt.name == "self" {
				tt.setup(w, ch, ch)
				result := DoSubdue(ch, ch, w)
				if result.MessageToCh != tt.want {
					t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
				}
				return
			}
			tt.setup(w, ch, target)
			result := DoSubdue(ch, target, w)
			if result.MessageToCh != tt.want {
				t.Fatalf("message = %q, want %q", result.MessageToCh, tt.want)
			}
		})
	}
}

// C's shopkeeper refusal sits between the Outlaw check and the position gate
// (src/act.offensive.c:1135-1139); the same is_shopkeeper port the other
// offensive skills use decides membership.
func TestDoSubdueShopkeeperGate(t *testing.T) {
	const shopVNum = 990195
	MobSpecAssign[shopVNum] = "shop_keeper"
	t.Cleanup(func() { delete(MobSpecAssign, shopVNum) })

	w := subdueTestWorld(t, nil)
	ch := subdueTestPlayer("Subduer")
	ch.SetPlrFlag(PlrOutlaw, true)
	mob := NewMob(&parser.Mob{
		VNum:      shopVNum,
		Keywords:  "keeper target",
		ShortDesc: "a keeper target",
		Position:  combat.PosStanding,
	}, 1001)
	result := DoSubdue(ch, mob, w)
	if result.MessageToCh != "Haha.. Don't think so." {
		t.Fatalf("shopkeeper gate message = %q, want %q", result.MessageToCh, "Haha.. Don't think so.")
	}
}

// The Outlaw refusal is C act.offensive.c:1127-1133: the actor line, then a
// bare send_to_char to the victim naming the actor twice. It returns before
// the percent roll, so it must consume no draws (R3a).
func TestDoSubdueOutlawRefusal(t *testing.T) {
	w := subdueTestWorld(t, nil)
	ch := subdueTestPlayer("Subduer")
	target := subdueTestPlayer("Target")
	assertZeroDrawsBeyond(t, func() {
		result := DoSubdue(ch, target, w)
		if result.Success {
			t.Fatal("non-outlaw actor subduing a player must fail")
		}
		if result.MessageToCh != "You can't subdue them because you are not an Outlaw!" {
			t.Fatalf("actor line = %q", result.MessageToCh)
		}
		want := "Subduer failed to subdue you because Subduer is not an Outlaw."
		if result.MessageToVict != want {
			t.Fatalf("victim line = %q, want %q", result.MessageToVict, want)
		}
	})
}

// C's miss pins (src/act.offensive.c:1146-1168): MOB_AWARE zeroes prob, a
// player victim more than 3 levels apart zeroes prob, and the miss arm ends in
// hit(victim, ch, TYPE_UNDEFINED) after the three acts.
func TestDoSubdueMissPinsAndShape(t *testing.T) {
	ch := subdueTestPlayer("Subduer")
	ch.SetPlrFlag(PlrOutlaw, true)
	awareMob := NewMob(&parser.Mob{
		VNum:        990191,
		Keywords:    "aware target",
		ShortDesc:   "an aware target",
		Position:    combat.PosStanding,
		ActionFlags: []string{"AWARE"},
	}, 1001)
	result := DoSubdue(ch, awareMob, nil)
	if result.Success {
		t.Fatal("aware mob should force the failure arm")
	}
	if !result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Fatalf("failure result = %+v, want post-message retaliation", result)
	}
	if result.WaitCh != 3 {
		t.Fatalf("failure wait = %d, want 3 violence rounds", result.WaitCh)
	}

	ch = subdueTestPlayer("Subduer")
	ch.SetPlrFlag(PlrOutlaw, true)
	target := subdueTestPlayer("Target")
	target.SetLevel(ch.GetLevel() + 4)
	result = DoSubdue(ch, target, nil)
	if result.Success || !result.RetaliateHit {
		t.Fatalf("out-of-window player result = %+v, want forced failure and retaliation", result)
	}
}

// An immortal actor sets percent = 0 (src/act.offensive.c:1149-1150) and is
// excluded from the player level-window miss (:1154), so a far-below player
// victim is subdued deterministically. Success sets PosStunned with no victim
// wait, a 3-round actor wait, and a deferred improvement draw.
func TestDoSubdueImmortalSuccess(t *testing.T) {
	w := subdueTestWorld(t, nil)
	ch := subdueTestPlayer("Subduer")
	ch.SetLevel(LVL_IMMORT)
	ch.SetPlrFlag(PlrOutlaw, true)
	target := subdueTestPlayer("Target")
	assertDrawsBeyond(t, 1, func() {
		result := DoSubdue(ch, target, w)
		if !result.Success {
			t.Fatalf("immortal subdue result = %+v, want success", result)
		}
		if target.GetPosition() != combat.PosStunned {
			t.Fatalf("victim position = %v, want PosStunned", target.GetPosition())
		}
		if result.WaitCh != 3 || result.WaitTarget != 0 {
			t.Fatalf("success waits = ch %d target %d, want 3 and none", result.WaitCh, result.WaitTarget)
		}
		if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillSubdue {
			t.Fatalf("success improvement shape = %+v", result)
		}
		if result.MessageToVict != "Someone sneaks up behind you and knocks you out!" {
			t.Fatalf("victim line = %q", result.MessageToVict)
		}
	})
}

// A seeded legal subdue consumes exactly the C draw (number(1,101+level)) and
// leaves improvement to the command layer's deferred slot, after the acts.
func TestDoSubdueSeededSuccessDrawCount(t *testing.T) {
	ch := subdueTestPlayer("Subduer")
	ch.SetPlrFlag(PlrOutlaw, true)
	mob := NewMob(&parser.Mob{
		VNum:      990192,
		Keywords:  "target",
		ShortDesc: "a target",
		Position:  combat.PosStanding,
	}, 1001)
	assertDrawsBeyond(t, 1, func() {
		result := DoSubdue(ch, mob, nil)
		if !result.Success {
			t.Fatalf("seeded legal subdue failed: %+v", result)
		}
	})
}
