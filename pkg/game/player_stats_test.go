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

// DP-1516 item 5: C's CAN_CARRY_N is a macro over GET_LEVEL (src/utils.h:449),
// so a level change immediately moves the item-count cap. SetLevel must
// refresh Go's cached Inventory capacity or perform_give's hands-full gate
// (src/act.item.c:687-689) reads a stale cap: demoting an actor below its
// previous level kept the old, larger cap and let transfers C refuses.
func TestSetLevelRefreshesCarryCapacity(t *testing.T) {
	p := NewPlayer(1, "Levelcap", 8162)
	p.Stats = CharStats{Str: 13, StrAdd: 0, Dex: 13, Int: 13, Wis: 13, Con: 13, Cha: 13}
	p.CopyBaseAttributes()
	p.Level = 39
	p.refreshAttributeCapacity()
	if got, want := p.Inventory.GetCapacity(), 5+(13>>1)+(39>>1); got != want {
		t.Fatalf("level 39 capacity=%d want %d", got, want)
	}
	p.SetLevel(1)
	if got, want := p.Inventory.GetCapacity(), 5+(13>>1)+(1>>1); got != want {
		t.Fatalf("after SetLevel(1) capacity=%d want %d (stale cap lets perform_give overfill)", got, want)
	}
}
