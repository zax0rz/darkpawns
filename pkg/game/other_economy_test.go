package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestDoUseWandConsumesInstanceChargeAndUsesDefaultLevel proves the two
// object-magic mechanics that are otherwise not visible in the scenario's
// player-facing transcript: charges mutate on the object instance, and a
// wand with value(0)==0 uses C's DEFAULT_WAND_LVL (12).
func TestDoUseWandConsumesInstanceChargeAndUsesDefaultLevel(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	caster := NewPlayer(1, "Caster", 1001)
	target := NewPlayer(2, "Target", 1001)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatalf("AddPlayer caster: %v", err)
	}
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}

	wand := NewObjectInstance(&parser.Obj{
		VNum:      8053,
		Keywords:  "wand",
		ShortDesc: "a test wand",
		TypeFlag:  ITEM_WAND,
		Values:    [4]int{0, 2, 1, spells.SpellHaste},
	}, -1)
	if err := caster.Inventory.AddItem(wand); err != nil {
		t.Fatalf("AddItem wand: %v", err)
	}
	if err := w.EquipItem(caster, wand, eqWearHold); err != nil {
		t.Fatalf("EquipItem wand: %v", err)
	}

	if !w.DoUse(caster, "wand Target") {
		t.Fatal("DoUse returned false")
	}
	if got := wand.GetValue(2); got != 0 {
		t.Errorf("wand current charges = %d, want 0 after one use", got)
	}
	if !target.IsAffected(affHaste) {
		t.Fatal("wand should apply haste to the target")
	}

	var duration int
	for _, affect := range target.ActiveAffects {
		if affect != nil && affect.SpellID == spells.SpellHaste {
			duration = affect.Duration
			break
		}
	}
	if duration != 12 { // SpellHaste uses the effective cast level directly.
		t.Errorf("haste duration = %d, want 12 from default wand level 12", duration)
	}
}
