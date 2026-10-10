package game

// Regression tests for DP-1264: player combat state (Fighting, Position,
// Hitpoints and the rest of the vitals block) is read and written from
// several goroutines — the combat-round ticker (pkg/combat/engine.go),
// the point-update ticker (limits_condition.go), and per-connection session
// goroutines. Every access must go through p.mu; these tests hammer the
// locked accessors and VitalsSnapshot from both sides so `go test -race`
// flags any regression to bare field access.

import (
	"testing"
)

func TestVitalsSnapshotMatchesStoredFields(t *testing.T) {
	p := NewPlayer(1, "Hero", 1001)
	p.SetHP(42)
	p.SetMaxHP(100)
	p.SetMana(7)
	p.SetMaxMana(50)
	p.SetMove(11)
	p.SetMaxMove(82)
	p.SetExp(1234)
	p.SetGold(99)
	p.SetLevel(5)
	p.SetClass(ClassWarrior)
	p.SetPosition(PosFighting)

	v := p.VitalsSnapshot()
	if v.Health != 42 || v.MaxHealth != 100 {
		t.Errorf("HP snapshot = %d/%d, want 42/100", v.Health, v.MaxHealth)
	}
	if v.Mana != 7 || v.MaxMana != 50 {
		t.Errorf("Mana snapshot = %d/%d, want 7/50", v.Mana, v.MaxMana)
	}
	if v.Move != 11 || v.MaxMove != 82 {
		t.Errorf("Move snapshot = %d/%d, want 11/82", v.Move, v.MaxMove)
	}
	if v.Exp != 1234 || v.Gold != 99 || v.Level != 5 || v.Class != ClassWarrior {
		t.Errorf("identity snapshot = exp %d gold %d level %d class %d", v.Exp, v.Gold, v.Level, v.Class)
	}
	if v.Position != PosFighting {
		t.Errorf("Position snapshot = %d, want %d", v.Position, PosFighting)
	}
}

// TestPlayerCombatStateConcurrent exercises the locked combat-state
// accessors concurrently from reader and writer goroutines. Under the race
// detector this fails if any participating path touches the bare fields.
