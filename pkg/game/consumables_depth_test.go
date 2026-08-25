package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDrinkcon(w *World, ch *Player, vnum, amount, poison int) *ObjectInstance {
	o := NewObjectInstance(&parser.Obj{
		VNum: vnum, ShortDesc: "a water skin", Keywords: "skin",
		TypeFlag: ITEM_DRINKCON, Values: [4]int{5, amount, 0, poison},
	}, -1)
	registerTransferObject(w, o)
	_ = w.MoveObjectToPlayerInventory(o, ch)
	return o
}

func newFood(w *World, ch *Player, vnum, poison int) *ObjectInstance {
	o := NewObjectInstance(&parser.Obj{
		VNum: vnum, ShortDesc: "a loaf of bread", Keywords: "bread",
		TypeFlag: ITEM_FOOD, Values: [4]int{4, 0, 0, poison},
	}, -1)
	registerTransferObject(w, o)
	_ = w.MoveObjectToPlayerInventory(o, ch)
	return o
}

// TestDrinkConditionGates proves do_drink's condition gates (act.item.c:931-951)
// and the empty and poison branches.
func TestDrinkConditionGates(t *testing.T) {
	cases := []struct {
		name                                string
		drunk, full, thirst, amount, poison int
		want                                string
	}{
		{"drunk-gate", 15, 0, 5, 5, 0, "You can't seem to get close enough to your mouth."},
		{"full-gate", 0, 25, 5, 5, 0, "Your stomach can't contain anymore!"},
		{"explode", 0, 0, 45, 5, 0, "If you drink any more, you'll explode!"},
		{"empty", 0, 0, 0, 0, 0, "It's empty."},
		{"poison", 0, 0, 0, 5, 1, "Oops, it tasted rather strange!"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch, lastMsg := newDonateTestWorld(t)
			ch.SetCondition(CondDrunk, tc.drunk)
			ch.SetCondition(CondFull, tc.full)
			ch.SetCondition(CondThirst, tc.thirst)
			newDrinkcon(w, ch, 9600, tc.amount, tc.poison)
			w.DoDrink(ch, nil, "drink", "skin", scmdDrink)
			if got := lastMsg(); !strings.Contains(got, tc.want) {
				t.Fatalf("%s: got %q want substring %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestEatGates proves do_eat's full-stomach and poison branches
// (act.item.c:1103-1130).
func TestEatGates(t *testing.T) {
	t.Run("full-gate", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		ch.SetCondition(CondFull, 45)
		newFood(w, ch, 9700, 0)
		w.DoEat(ch, nil, "eat", "bread", scmdEat)
		if got := lastMsg(); !strings.Contains(got, "You are too full to eat more!") {
			t.Fatalf("full-gate: got %q", got)
		}
	})
	t.Run("poison", func(t *testing.T) {
		w, ch, lastMsg := newDonateTestWorld(t)
		newFood(w, ch, 9701, 1)
		w.DoEat(ch, nil, "eat", "bread", scmdEat)
		if got := lastMsg(); !strings.Contains(got, "Oops, that tasted rather strange!") {
			t.Fatalf("poison: got %q", got)
		}
	})
}

// TestDrinkHoldingGate proves do_drink's on-ground drinkcon gate
// (act.item.c:919): a drink container on the floor can't be drunk from.
func TestDrinkHoldingGate(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	o := NewObjectInstance(&parser.Obj{
		VNum: 9800, ShortDesc: "a water skin", Keywords: "skin",
		TypeFlag: ITEM_DRINKCON, Values: [4]int{5, 5, 0, 0},
	}, -1)
	registerTransferObject(w, o)
	w.AddItemToRoom(o, ch.GetRoomVNum())
	w.DoDrink(ch, nil, "drink", "skin", scmdDrink)
	if got := lastMsg(); !strings.Contains(got, "You have to be holding that to drink from it.") {
		t.Fatalf("holding-gate: got %q", got)
	}
}
