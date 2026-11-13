package game

import "testing"

// TestLiquidTablesMatchC pins every row of the four C liquid tables against
// literals copied from src/constants.c, so any future regression of one row
// fails here instead of only surfacing as an oracle diff (R5c/R5h, DP-1377).
//
// Cite: src/constants.c:918-936 (drinks[]), :952-965 (drinknames[]),
// :967-984 (drink_aff[][3], columns DRUNK/FULL/THIRST per structs.h:566-568),
// :990-1010 (color_liquid[]).
func TestLiquidTablesMatchC(t *testing.T) {
	for i, want := range []Liquid{
		{"water", "clear", 0, 1, 10},
		{"beer", "brown", 3, 2, 5},
		{"wine", "clear", 5, 2, 5},
		{"ale", "brown", 2, 2, 5},
		{"dark ale", "dark", 1, 2, 5},
		{"whiskey", "golden", 6, 1, 4},
		{"lemonade", "red", 0, 1, 8},
		{"firebreather", "green", 10, 0, 0},
		{"local speciality", "clear", 3, 3, 3},
		{"slime mold juice", "light green", 0, 4, -8},
		{"milk", "white", 0, 3, 6},
		{"tea", "brown", 0, 1, 6},
		{"coffee", "black", 0, 1, 6},
		{"blood", "red", 0, 2, -1},
		{"salt water", "clear", 0, 1, -2},
		{"clear water", "crystal clear", 0, 0, 13},
	} {
		if got := Liquids[i]; got != want {
			t.Errorf("Liquids[%d] = %+v, want C's %+v", i, got, want)
		}
		if got := LiquidNames[i]; got != want.Name {
			t.Errorf("LiquidNames[%d] = %q, want C drinks[] %q", i, got, want.Name)
		}
		if got := DrinkKeyword(i); got != drinknamesC[i] {
			t.Errorf("DrinkKeyword(%d) = %q, want C drinknames[] %q", i, got, drinknamesC[i])
		}
	}
}

// drinknamesC mirrors C drinknames[] (constants.c:952-965): the one-word
// alias linked into a container's keywords, distinct from drinks[].
var drinknamesC = []string{
	"water", "beer", "wine", "ale", "ale", "whiskey", "lemonade",
	"firebreather", "local", "juice", "milk", "tea", "coffee", "blood",
	"salt", "water",
}

// TestGetDrinkAffectsColumnOrder pins the DRUNK/FULL/THIRST return order
// against C's indexes (structs.h:566-568): beer is {3, 2, 5}, so a swapped
// full/thirst pair fails here.
func TestGetDrinkAffectsColumnOrder(t *testing.T) {
	drunk, full, thirst := GetDrinkAffects(LiqBeer)
	if drunk != 3 || full != 2 || thirst != 5 {
		t.Fatalf("GetDrinkAffects(beer) = %d,%d,%d; want C's DRUNK=3, FULL=2, THIRST=5 (drink_aff[1])", drunk, full, thirst)
	}
}
