package game

import "testing"

// ──── SpendMove tests ────

func TestSpendMove_Sufficient(t *testing.T) {
	p := &Player{Move: 20}
	if !p.SpendMove(10) {
		t.Error("SpendMove(10) on 20 move should return true")
	}
	if p.Move != 10 {
		t.Errorf("expected 10 remaining, got %d", p.Move)
	}
}

func TestSpendMove_Insufficient(t *testing.T) {
	p := &Player{Move: 5}
	if p.SpendMove(10) {
		t.Error("SpendMove(10) on 5 move should return false")
	}
	if p.Move != 5 {
		t.Errorf("expected 5 remaining (unchanged), got %d", p.Move)
	}
}

func TestSpendMove_Exact(t *testing.T) {
	p := &Player{Move: 10}
	if !p.SpendMove(10) {
		t.Error("SpendMove(10) on 10 move should return true")
	}
	if p.Move != 0 {
		t.Errorf("expected 0 remaining, got %d", p.Move)
	}
}

func TestSpendMove_Zero(t *testing.T) {
	p := &Player{Move: 5}
	if !p.SpendMove(0) {
		t.Error("SpendMove(0) on 5 move should return true")
	}
	if p.Move != 5 {
		t.Errorf("expected 5 remaining (unchanged), got %d", p.Move)
	}
}

func TestSpendMove_Negative(t *testing.T) {
	p := &Player{Move: 5}
	if !p.SpendMove(-1) {
		t.Error("SpendMove(-1) on 5 move should return true (negative = no-op)")
	}
	if p.Move != 5 {
		t.Errorf("expected 5 remaining (unchanged), got %d", p.Move)
	}
}

func TestPlayerGetDamageRoll_BareHandsAreNotWeaponDice(t *testing.T) {
	p := NewPlayer(1, "Barehands", 100)

	got := p.GetDamageRoll()
	if got.Num != 0 || got.Sides != 0 {
		t.Fatalf("bare-hand weapon dice = %dd%d, want 0d0", got.Num, got.Sides)
	}
}
