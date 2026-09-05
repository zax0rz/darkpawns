package game

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func backstabberWeapon() *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      9101,
			Keywords:  "sharp dagger",
			ShortDesc: "a sharp dagger",
			TypeFlag:  ITEM_WEAPON,
			Values:    [4]int{0, 3, 2, 11}, // TYPE_PIERCE - TYPE_HIT
		},
	}
}

func prepareBackstabber(t *testing.T, wield bool) (*World, *Player, *MobInstance, *testCombatEngine) {
	t.Helper()
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	// C parse_simple_mob starts unqualified mob abilities at 11. The generic
	// focused test mob leaves these zero unless the test supplies them.
	mob.Str = 11
	mob.Intel = 11
	mob.Wis = 11
	mob.Dex = 11
	mob.Prototype.Damage = parser.DiceRoll{Num: 1, Sides: 1, Plus: 2}
	if wield {
		mob.EquipItem(backstabberWeapon(), int(SlotWield))
	}
	engine := &testCombatEngine{}
	w.SetCombatEngine(engine)
	return w, player, mob, engine
}

func backstabberSeed(t *testing.T, wantHit bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		rng := dprng.New(seed)
		percent := rng.Number(1, 101)
		prob := rng.Number(50, 100)
		hit := percent <= prob
		if hit != wantHit {
			continue
		}
		if wantHit && rng.Number(1, 20) != 20 {
			continue
		}
		return seed
	}
	t.Fatalf("could not find backstabber seed for wantHit=%t", wantHit)
	return 0
}

func TestSpecBackstabber_EntryGatesAndTargetSelection(t *testing.T) {
	cases := []struct {
		name string
		call func(*World, *Player, *MobInstance) bool
		want bool
	}{
		{
			name: "command gate",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				return specBackstabber(w, player, mob, "look", "")
			},
			want: false,
		},
		{
			name: "fighting mob gate",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.SetFighting(player.GetName())
				return specBackstabber(w, nil, mob, "", "")
			},
			want: false,
		},
		{
			name: "sleeping mob gate",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.SetPosition(combat.PosSleeping)
				return specBackstabber(w, nil, mob, "", "")
			},
			want: false,
		},
		{
			name: "no visible target",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				player.SetAffect(affInvisible, true)
				return specBackstabber(w, nil, mob, "", "")
			},
			want: false,
		},
		{
			name: "no hassle target",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				player.Flags |= 1 << uint(PrfNohassle)
				return specBackstabber(w, nil, mob, "", "")
			},
			want: false,
		},
		{
			name: "weapon gate is handled",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				return specBackstabber(w, nil, mob, "", "")
			},
			want: true,
		},
		{
			name: "fighting target is handled before rolls",
			call: func(w *World, player *Player, mob *MobInstance) bool {
				mob.EquipItem(backstabberWeapon(), int(SlotWield))
				player.SetFighting("another attacker")
				return specBackstabber(w, nil, mob, "", "")
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, player, mob, combatEngine := prepareBackstabber(t, false)
			before := dprng.New(77).Next()
			dprng.ResetStream(77)
			got := tc.call(w, player, mob)
			if got != tc.want {
				t.Fatalf("handled = %t, want %t", got, tc.want)
			}
			if gotNext := dprng.Next(); gotNext != before {
				t.Fatalf("gate consumed a random draw: next=%d, want=%d", gotNext, before)
			}
			if len(combatEngine.starts) != 0 {
				t.Fatalf("gate unexpectedly started combat: %+v", combatEngine.starts)
			}
		})
	}
}

func TestSpecBackstabber_RngArmsAndCombatState(t *testing.T) {
	t.Run("percent miss still enters combat", func(t *testing.T) {
		w, player, mob, combatEngine := prepareBackstabber(t, true)
		player.SetHP(100)
		player.SetPosition(combat.PosStanding)
		seed := backstabberSeed(t, false)
		dprng.ResetStream(seed)

		var skillMessage string
		previous := combat.GetCallbacks()
		combat.SetCallbacks(&combat.GameCallbacks{
			SkillMessage: func(dam int, ch, vict string, attackType, room int) bool {
				skillMessage = ch + "|" + vict + "|" + strconv.Itoa(dam) + "|" + strconv.Itoa(attackType) + "|" + strconv.Itoa(room)
				return true
			},
		})
		t.Cleanup(func() { combat.SetCallbacks(previous) })

		if !specBackstabber(w, nil, mob, "", "") {
			t.Fatal("backstabber miss arm should be handled")
		}
		if got, want := skillMessage, "a test mob|Tester|0|131|1001"; got != want {
			t.Fatalf("skill message = %q, want %q", got, want)
		}
		if got := player.GetHP(); got != 100 {
			t.Fatalf("miss changed victim HP to %d", got)
		}
		if got := player.GetFighting(); got != mob.GetName() {
			t.Fatalf("miss victim fighting target = %q, want %q", got, mob.GetName())
		}
		if got := mob.GetFighting(); got != player.GetName() {
			t.Fatalf("miss mob fighting target = %q, want %q", got, player.GetName())
		}
		if got, want := mob.GetWaitState(), engine.PULSE_VIOLENCE; got != want {
			t.Fatalf("miss mob wait state = %d, want %d", got, want)
		}
		if len(combatEngine.starts) != 1 {
			t.Fatalf("StartCombat calls = %d, want 1", len(combatEngine.starts))
		}
	})

	t.Run("hit uses NPC dice and backstab message", func(t *testing.T) {
		w, player, mob, combatEngine := prepareBackstabber(t, true)
		player.SetHP(100)
		player.SetPosition(combat.PosStanding)
		seed := backstabberSeed(t, true)
		dprng.ResetStream(seed)

		var skillMessage string
		previous := combat.GetCallbacks()
		combat.SetCallbacks(&combat.GameCallbacks{
			SkillMessage: func(dam int, ch, vict string, attackType, room int) bool {
				skillMessage = ch + "|" + vict + "|" + strconv.Itoa(dam) + "|" + strconv.Itoa(attackType) + "|" + strconv.Itoa(room)
				return true
			},
		})
		t.Cleanup(func() { combat.SetCallbacks(previous) })

		expectedRoller := dprng.New(seed)
		expectedRoller.Number(1, 101)
		expectedRoller.Number(50, 100)
		expectedRoller.Number(1, 20)
		weaponDamage := expectedRoller.Dice(1, 1) + 2
		wantDamage := (combat.StrAppToDam(mob) + mob.GetDamroll() + weaponDamage) * int(combat.BackstabMult(mob.GetLevel()))
		wantNext := expectedRoller.Next()

		if !specBackstabber(w, nil, mob, "", "") {
			t.Fatal("backstabber hit arm should be handled")
		}
		if got, want := player.GetHP(), 100-wantDamage; got != want {
			t.Fatalf("victim HP = %d, want %d", got, want)
		}
		if got, want := skillMessage, "a test mob|Tester|"+strconv.Itoa(wantDamage)+"|131|1001"; got != want {
			t.Fatalf("skill message = %q, want %q", got, want)
		}
		if got := dprng.Next(); got != wantNext {
			t.Fatalf("RNG stream after backstab = %d, want %d", got, wantNext)
		}
		if len(combatEngine.starts) != 1 {
			t.Fatalf("StartCombat calls = %d, want 1", len(combatEngine.starts))
		}
		if got, want := mob.GetWaitState(), engine.PULSE_VIOLENCE; got != want {
			t.Fatalf("hit mob wait state = %d, want %d", got, want)
		}
	})
}
