package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShootRealDuplicateMobRetaliation(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 3001, Keywords: "guard", ShortDesc: "a guard", Level: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	first, err := w.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first.ID == second.ID || first.GetName() != second.GetName() {
		t.Fatal("invalid real duplicate fixture")
	}
	other := NewPlayer(1, "Other", 1001)
	shooter := NewPlayer(2, "Shooter", 1001)
	for _, p := range []*Player{other, shooter} {
		p.Level = 20
		p.Health = 1000
		p.MaxHealth = 1000
		p.SetPosition(combat.PosStanding)
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range []*MobInstance{first, second} {
		m.mu.Lock()
		m.CurrentHP = 1000
		m.mu.Unlock()
		m.SetPosition(combat.PosStanding)
	}
	oldCB, oldRoller := combat.GetCallbacks(), combat.GetRoller()
	defer combat.SetCallbacks(oldCB)
	defer combat.SetRoller(oldRoller)
	combat.SetCallbacks(&combat.GameCallbacks{})
	combat.SetRoller(combat.NewScriptedRoller([]int{10, 2, 2, 2, 2, 2, 2, 2}))
	ce := combat.NewCombatEngine()
	if err := ce.StartCombat(first, other); err != nil {
		t.Fatal(err)
	}
	swings := 0
	seen := map[combat.Combatant]int{}
	ce.MessageFunc = func(a, b combat.Combatant, dam, typ int) bool {
		swings++
		seen[a]++
		if a == first && b != other || a == other && b != first || a == second && b != shooter || a == shooter && b != second {
			t.Fatal("attack resolved to another duplicate pair")
		}
		return true
	}
	if err := ce.PerformRangedRetaliation(second, shooter); err != nil {
		t.Fatal(err)
	}
	if swings != 1 {
		t.Fatalf("real distinct surviving shot mob retaliated %d times; want one", swings)
	}
	if first.GetFightingBody() != other || other.GetFightingBody() != first || second.GetFightingBody() != shooter || shooter.GetFightingBody() != second {
		t.Fatal("retaliation crossed duplicate fight identities")
	}
	ce.PerformRound()
	for _, body := range []combat.Combatant{first, other, second, shooter} {
		if seen[body] == 0 {
			t.Fatal("ensuing round lost a fighter")
		}
	}
	ce.RetireCombatant(second)
	if first.GetFightingBody() != other || other.GetFightingBody() != first || second.GetFightingBody() != nil || shooter.GetFightingBody() != nil {
		t.Fatal("retaliation cleanup disturbed other duplicate")
	}
}
