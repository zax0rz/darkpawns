package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRaceHateEquipmentRebuildAndRemoval(t *testing.T) {
	w, combatEngine := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9408, 18, 0, "sentinel")
	player := newMobactTestPlayer(w, "RaceHateWearer")
	item := NewObjectInstance(&parser.Obj{
		VNum:      5200,
		Keywords:  "race dagger",
		ShortDesc: "a race dagger",
		TypeFlag:  ITEM_WEAPON,
		WearFlags: [4]int{1<<0 | 1<<13},
		Affects:   []parser.ObjAffect{{Location: ApplyRaceHate, Modifier: 18}, {Location: ApplyRaceHate, Modifier: 20}},
	}, -1)
	if err := player.Inventory.AddItem(item); err != nil {
		t.Fatalf("add item to inventory: %v", err)
	}

	w.DoWield(player, "dagger")
	if got, want := player.RaceHates, [5]int{18, 20, -1, -1, -1}; got != want {
		t.Fatalf("race hates after equip = %v, want %v", got, want)
	}

	w.mobileActivityForMob(mob)
	if len(combatEngine.starts) != 1 || combatEngine.starts[0].defender != player.Name {
		t.Fatalf("race-hate mobile dispatch did not attack wearer: %+v", combatEngine.starts)
	}

	w.DoRemove(player, "dagger")
	if got, want := player.RaceHates, [5]int{-1, -1, -1, -1, -1}; got != want {
		t.Fatalf("race hates after unequip = %v, want %v", got, want)
	}
}

func TestRaceHateModifierSlotSemantics(t *testing.T) {
	slots := [5]int{-1, -1, -1, -1, -1}
	modifyRaceHateSlots(&slots, 7)
	modifyRaceHateSlots(&slots, 7)
	modifyRaceHateSlots(&slots, -7)
	if got, want := slots, [5]int{-1, 7, -1, -1, -1}; got != want {
		t.Fatalf("race-hate insert/remove slots = %v, want %v", got, want)
	}
	modifyRaceHateSlots(&slots, 0)
	if got, want := slots, [5]int{0, 7, -1, -1, -1}; got != want {
		t.Fatalf("human slot insert = %v, want %v", got, want)
	}
	modifyRaceHateSlots(&slots, 0)
	if got, want := slots, [5]int{-1, 7, -1, -1, -1}; got != want {
		t.Fatalf("human slot removal = %v, want %v", got, want)
	}
}
