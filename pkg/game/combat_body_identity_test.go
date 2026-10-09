package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCombatBodyIdentityDuplicateEnrollment(t *testing.T) {
	for _, sameRoom := range []bool{true, false} {
		t.Run(map[bool]string{true: "same-room", false: "different-rooms"}[sameRoom], func(t *testing.T) {
			w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 3001, Keywords: "guard", ShortDesc: "a guard", Level: 10}}})
			if err != nil {
				t.Fatal(err)
			}
			w.StopAITicker()
			first, err := w.SpawnMobQuiet(3001, 1001)
			if err != nil {
				t.Fatal(err)
			}
			room := 1001
			if !sameRoom {
				room = 1002
			}
			second, err := w.SpawnMobQuiet(3001, room)
			if err != nil {
				t.Fatal(err)
			}
			if first == second || first.GetID() == second.GetID() || first.GetName() != second.GetName() {
				t.Fatal("invalid duplicate-body fixture")
			}
			one := NewPlayer(first.GetID(), "First", 1001)
			two := NewPlayer(second.GetID(), "Second", room)
			for _, p := range []*Player{one, two} {
				p.Level = 20
				p.Health = 10000
				p.MaxHealth = 10000
				p.SetPosition(combat.PosStanding)
				if err := w.AddPlayer(p); err != nil {
					t.Fatal(err)
				}
			}
			for _, m := range []*MobInstance{first, second} {
				m.mu.Lock()
				m.CurrentHP = 10000
				m.MaxHP = 10000
				m.mu.Unlock()
				m.SetPosition(combat.PosStanding)
			}
			oldCB, oldRoller := combat.GetCallbacks(), combat.GetRoller()
			t.Cleanup(func() { combat.SetCallbacks(oldCB); combat.SetRoller(oldRoller) })
			combat.SetCallbacks(&combat.GameCallbacks{})
			combat.SetRoller(combat.NewSeededRoller(71))
			ce := combat.NewCombatEngine()
			if err := ce.StartCombat(first, one); err != nil {
				t.Fatal(err)
			}
			if err := ce.StartCombat(second, two); err != nil {
				t.Fatalf("distinct duplicate body cannot enroll: %v", err)
			}
			// Reciprocal outgoing pairs have equal numeric IDs across body kinds.
			if err := ce.StartCombat(one, first); err != nil {
				t.Fatalf("player/mob ID collision: %v", err)
			}
			expectedTargets := map[combat.Combatant]combat.Combatant{first: one, second: two, one: first, two: second}
			swings := make(map[combat.Combatant]int)
			var turns []combat.Combatant
			ce.MessageFunc = func(a, b combat.Combatant, _ int, _ int) bool {
				if expectedTargets[a] != b {
					t.Fatalf("swing body %p used target %p, want %p", a, b, expectedTargets[a])
				}
				swings[a]++
				turns = append(turns, a)
				return true
			}
			if err := ce.PerformInitialAttack(first, one); err != nil {
				t.Fatal(err)
			}
			if err := ce.PerformInitialAttack(second, two); err != nil {
				t.Fatal(err)
			}
			if swings[first] != 1 || swings[second] != 1 {
				t.Fatalf("initial swings: %v", swings)
			}
			for body, want := range map[combat.Combatant]combat.Combatant{first: one, second: two, one: first, two: second} {
				if target, ok := ce.GetCombatTarget(body); !ok || target != want {
					t.Fatalf("body %p target=%p/%t, want %p", body, target, ok, want)
				}
			}
			replacement := NewPlayer(one.GetID(), one.GetName(), 1001)
			if ce.IsFighting(replacement) {
				t.Fatal("restored replacement inherited retired body's membership")
			}
			turns = nil
			ce.PerformRound()
			if swings[first] <= 1 || swings[second] <= 1 || swings[one] == 0 || swings[two] == 0 {
				t.Fatalf("later turns lost duplicate or player: %v", swings)
			}
			var order []combat.Combatant
			for _, body := range turns {
				if len(order) == 0 || order[len(order)-1] != body {
					order = append(order, body)
				}
			}
			wantOrder := []combat.Combatant{two, second, one, first}
			if len(order) != len(wantOrder) {
				t.Fatalf("fighter memberships/turns=%v", order)
			}
			for i, body := range order {
				if body != wantOrder[i] {
					t.Fatalf("turn %d body=%p want=%p", i, body, wantOrder[i])
				}
			}
		})
	}
}

// Display access snapshots the reference and releases the body lock before
// querying the opponent; reciprocal fighting must not nest the two locks.
func TestCombatBodyReferencesConcurrentAndReciprocal(t *testing.T) {
	one := NewPlayer(1, "One", 1001)
	two := NewPlayer(2, "Two", 1001)
	mob := NewMob(&parser.Mob{VNum: 3001, ShortDesc: "a guard"}, 1001)
	one.SetFightingBody(two)
	two.SetFightingBody(one)
	done := make(chan struct{}, 3)
	go func() {
		for i := 0; i < 1000; i++ {
			one.SetFightingBody(two)
			_ = one.GetFighting()
			one.StopFighting()
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			two.SetFightingBody(one)
			_ = two.GetFighting()
			two.StopFighting()
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			mob.SetFightingBody(one)
			_ = mob.GetFighting()
			mob.StopFighting()
		}
		done <- struct{}{}
	}()
	for i := 0; i < 3; i++ {
		<-done
	}
}
