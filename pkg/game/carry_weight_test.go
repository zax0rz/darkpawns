package game

import "testing"

func TestMaxCarryWeightUsesLiveExceptionalStrength(t *testing.T) {
	p := NewPlayer(1, "Strongarm", MortalStartRoom)
	p.Stats.Str = 18
	p.Stats.StrAdd = 50
	p.Inventory.MaxWeight = 3

	if got := p.MaxCarryWeight(); got != 280 {
		t.Fatalf("MaxCarryWeight() = %d, want 280 for STR 18/50", got)
	}
}

func TestAdvanceLevelRefreshesCarryCapacity(t *testing.T) {
	p := NewPlayer(1, "CarryLevelTest", MortalStartRoom)
	t.Cleanup(func() { _ = DeletePlayer(p.Name) })
	p.Class = ClassWarrior
	p.Level = 20
	p.Stats = CharStats{Str: 18, StrAdd: 100, Dex: 18, Con: 10, Wis: 10}
	p.Inventory.SetCapacity(1, 0, 1, 1)

	p.AdvanceLevel()

	if p.Inventory.MaxWeight != 480 {
		t.Errorf("MaxWeight after AdvanceLevel = %d, want 480", p.Inventory.MaxWeight)
	}
	if p.Inventory.Capacity != 24 {
		t.Errorf("Capacity after AdvanceLevel = %d, want 24", p.Inventory.Capacity)
	}
}
