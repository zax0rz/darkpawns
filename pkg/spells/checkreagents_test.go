package spells

import (
	"strings"
	"testing"
)

type mockItem struct {
	shortDesc string
	vnum      int
}

func (m *mockItem) GetShortDesc() string { return m.shortDesc }

type mockInventory struct {
	items []*mockItem
}

func (inv *mockInventory) consumeVNum(vnum int) bool {
	for i, item := range inv.items {
		if item.vnum == vnum {
			inv.items = append(inv.items[:i], inv.items[i+1:]...)
			return true
		}
	}
	return false
}

type mockCaster struct {
	inv      *mockInventory
	messages []string
}

func (c *mockCaster) ConsumeSpellReagentVNum(vnum int) bool { return c.inv.consumeVNum(vnum) }
func (c *mockCaster) SendMessage(msg string)                { c.messages = append(c.messages, msg) }

func TestConsumeReagentMatchesVNumNotName(t *testing.T) {
	tests := []struct {
		name         string
		item         *mockItem
		wantBonus    int
		wantItems    int
		wantMessages int
	}{
		{
			name:      "keyword decoy is not consumed",
			item:      &mockItem{shortDesc: "a shard of obsidian", vnum: 3001},
			wantItems: 1,
		},
		{
			name:         "real reagent with unrelated name is consumed",
			item:         &mockItem{shortDesc: "an unremarkable stone", vnum: 1239},
			wantBonus:    20,
			wantMessages: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caster := &mockCaster{inv: &mockInventory{items: []*mockItem{tc.item}}}
			got := consumeReagent(caster, SpellMagicMissile, 20, "The shard crumbles.")
			if got != tc.wantBonus {
				t.Errorf("consumeReagent bonus = %d, want %d", got, tc.wantBonus)
			}
			if len(caster.inv.items) != tc.wantItems {
				t.Errorf("inventory count = %d, want %d", len(caster.inv.items), tc.wantItems)
			}
			if len(caster.messages) != tc.wantMessages {
				t.Errorf("caster messages = %q, want %d messages", caster.messages, tc.wantMessages)
			}
		})
	}
}

func TestWaterwalkReagentMatchesVNumNotName(t *testing.T) {
	const componentMessage = "A frog leg turns into dust as you mouth the words of the spell...\r\n"
	tests := []struct {
		name          string
		item          *mockItem
		wantItems     int
		wantComponent bool
	}{
		{
			name:      "keyword decoy is not consumed",
			item:      &mockItem{shortDesc: "a frog leg", vnum: 3001},
			wantItems: 1,
		},
		{
			name:          "real reagent with unrelated name is consumed",
			item:          &mockItem{shortDesc: "an unremarkable stone", vnum: 1228},
			wantComponent: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caster := &mockSpellsChar{level: 20, inventory: &mockInventory{items: []*mockItem{tc.item}}}
			MagAffects(20, caster, caster, SpellWaterwalk, int(SaveSpell), nil)
			if len(caster.inventory.items) != tc.wantItems {
				t.Errorf("inventory count = %d, want %d", len(caster.inventory.items), tc.wantItems)
			}
			gotComponent := false
			for _, msg := range caster.messages {
				if msg == componentMessage {
					gotComponent = true
				}
			}
			if gotComponent != tc.wantComponent {
				t.Errorf("component message present = %t, want %t; messages %q", gotComponent, tc.wantComponent, caster.messages)
			}
			if len(caster.activeAffects) != 1 || caster.activeAffects[0].Duration != 20 {
				t.Errorf("waterwalk affects = %+v, want C's always-20 duration", caster.activeAffects)
			}
		})
	}
}

func TestDamageReagentDecoyUsesNoReagentFormula(t *testing.T) {
	caster := &mockSpellsChar{
		name:      "Mage",
		level:     20,
		class:     0,
		inventory: &mockInventory{items: []*mockItem{{shortDesc: "a prism", vnum: 3001}}},
	}
	victim := &mockSpellsChar{name: "Target", level: 10}
	MagDamage(20, caster, victim, SpellColorSpray, int(SaveSpell), nil)
	if len(caster.inventory.items) != 1 {
		t.Fatal("keyword decoy was consumed")
	}
	for _, msg := range caster.messages {
		if strings.Contains(msg, "Pulling a prism") {
			t.Fatalf("decoy produced reagent message %q", msg)
		}
	}
}
