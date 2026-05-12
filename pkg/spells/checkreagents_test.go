package spells

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Mock types implementing the package-level interfaces
// (ReagentItem, ReagentInventory, InventoryHolder) from affect_spells.go.
//
// Package-level interfaces are required because Go's type assertion with
// locally-defined function-scoped interfaces can never be satisfied by
// types defined outside that function.
// ---------------------------------------------------------------------------

// mockItem satisfies ReagentItem.
type mockItem struct {
	shortDesc string
}

func (m *mockItem) GetShortDesc() string { return m.shortDesc }

// mockInventory satisfies ReagentInventory.
type mockInventory struct {
	items []*mockItem
}

func (inv *mockInventory) FindItem(name string) (ReagentItem, bool) {
	for _, item := range inv.items {
		if item.shortDesc == name {
			return item, true
		}
	}
	return nil, false
}

func (inv *mockInventory) RemoveItem(item ReagentItem) bool {
	for i, it := range inv.items {
		if it == item {
			inv.items = append(inv.items[:i], inv.items[i+1:]...)
			return true
		}
	}
	return false
}

// mockCaster satisfies InventoryHolder and also has SendMessage for
// checkReagents' local messageSender interface.
type mockCaster struct {
	inv      *mockInventory
	messages []string
}

func (c *mockCaster) GetInventory() ReagentInventory { return c.inv }
func (c *mockCaster) SendMessage(msg string)         { c.messages = append(c.messages, msg) }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCheckReagents_Found(t *testing.T) {
	reagent := &mockItem{shortDesc: "pixie dust"}
	inv := &mockInventory{items: []*mockItem{reagent}}
	caster := &mockCaster{inv: inv}

	bonus := checkReagents(caster, 1, 10, "pixie dust")
	if bonus <= 0 {
		t.Fatalf("expected bonus > 0 when reagent found, got %d", bonus)
	}

	// Item should be consumed from inventory
	if len(inv.items) != 0 {
		t.Fatal("reagent should have been consumed from inventory")
	}
}

func TestCheckReagents_NotFound(t *testing.T) {
	inv := &mockInventory{items: []*mockItem{}}
	caster := &mockCaster{inv: inv}

	bonus := checkReagents(caster, 1, 10, "pixie dust")
	if bonus != 0 {
		t.Fatalf("expected bonus 0 when reagent absent, got %d", bonus)
	}
}

func TestCheckReagents_SendsMessage(t *testing.T) {
	reagent := &mockItem{shortDesc: "bat wing"}
	inv := &mockInventory{items: []*mockItem{reagent}}
	caster := &mockCaster{inv: inv}

	bonus := checkReagents(caster, 1, 5, "bat wing", "The bat wing crumbles to ash.")
	if bonus <= 0 {
		t.Fatalf("expected bonus > 0, got %d", bonus)
	}

	if len(caster.messages) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(caster.messages))
	}
	expected := "The bat wing crumbles to ash.\r\n"
	if caster.messages[0] != expected {
		t.Fatalf("expected message %q, got %q", expected, caster.messages[0])
	}
}

func TestCheckReagents_NoReagentsProvided(t *testing.T) {
	caster := &mockCaster{inv: &mockInventory{}}

	bonus := checkReagents(caster, 1, 10)
	if bonus != 0 {
		t.Fatalf("expected bonus 0 with no reagents, got %d", bonus)
	}
}

func TestCheckReagents_BonusScalesWithLevel(t *testing.T) {
	tests := []struct {
		level   int
		wantMin int
		want    int
	}{
		{level: 1, wantMin: 1, want: 1},   // level/2 = 0, clamped to 1
		{level: 2, wantMin: 1, want: 1},   // level/2 = 1
		{level: 10, wantMin: 5, want: 5},  // level/2 = 5
		{level: 30, wantMin: 15, want: 15}, // level/2 = 15
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			reagent := &mockItem{shortDesc: "eye of newt"}
			inv := &mockInventory{items: []*mockItem{reagent}}
			caster := &mockCaster{inv: inv}

			bonus := checkReagents(caster, 1, tt.level, "eye of newt")
			if bonus != tt.want {
				t.Fatalf("level %d: expected bonus %d, got %d", tt.level, tt.want, bonus)
			}
		})
	}
}

func TestCheckReagents_NonHolderCaster(t *testing.T) {
	// A caster that doesn't implement InventoryHolder should return 0
	type plainCaster struct{}
	caster := &plainCaster{}

	bonus := checkReagents(caster, 1, 10, "pixie dust")
	if bonus != 0 {
		t.Fatalf("expected bonus 0 for non-holder caster, got %d", bonus)
	}
}

func TestCheckReagents_ItemConsumedOnExactShortDescMatch(t *testing.T) {
	// Verify the FindItem search: uses exact name match on shortDesc.
	items := []*mockItem{
		{shortDesc: "a pinch of ash"},
		{shortDesc: "shard of obsidian"},
	}
	inv := &mockInventory{items: items}
	caster := &mockCaster{inv: inv}

	bonus := checkReagents(caster, 1, 10, "shard of obsidian")
	if bonus <= 0 {
		t.Fatalf("expected bonus > 0 when reagent found, got %d", bonus)
	}

	// Only the matching item should be removed
	if len(inv.items) != 1 {
		t.Fatalf("expected 1 item remaining, got %d", len(inv.items))
	}
	if inv.items[0].shortDesc != "a pinch of ash" {
		t.Fatalf("expected 'a pinch of ash' to remain, got %q", inv.items[0].shortDesc)
	}
}
