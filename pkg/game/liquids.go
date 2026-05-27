package game

// Liquid defines a liquid type with its properties.
// Ported from src/constants.c drink_aff[][3] and drinks[] arrays.
// drink_aff[LIQ_x][0] = DRUNK, [1] = FULL, [2] = THIRST
type Liquid struct {
	Name         string
	Color        string
	DrunkAffect  int // effect on drunkenness (drink_aff[][0])
	FullAffect   int // effect on fullness (drink_aff[][1])
	ThirstAffect int // effect on thirst (drink_aff[][2])
}

// Liquid indices matching LIQ_* defines from src/structs.h
const (
	LiqWater      = 0
	LiqBeer       = 1
	LiqWine       = 2
	LiqAle        = 3
	LiqDarkAle    = 4
	LiqWhisky     = 5
	LiqLemonade   = 6
	LiqFirebrt    = 7
	LiqLocalspc   = 8
	LiqSlime      = 9
	LiqMilk       = 10
	LiqTea        = 11
	LiqCoffee     = 12
	LiqCoffe      = 12 // Deprecated: use LiqCoffee instead
	LiqBlood      = 13
	LiqSaltwater  = 14
	LiqClearwater = 15
)

// Liquids is the indexed list of all liquid types.
// Index by LIQ_x constant to get the matching Liquid.
// Ported from src/constants.c drink_aff[] and drinks[].
var Liquids = []Liquid{
	LiqWater:      {Name: "water", Color: "clear", DrunkAffect: 0, FullAffect: 0, ThirstAffect: 1},
	LiqBeer:       {Name: "beer", Color: "brown", DrunkAffect: 3, FullAffect: 2, ThirstAffect: 2},
	LiqWine:       {Name: "wine", Color: "clear", DrunkAffect: 5, FullAffect: 2, ThirstAffect: 2},
	LiqAle:        {Name: "ale", Color: "brown", DrunkAffect: 2, FullAffect: 2, ThirstAffect: 2},
	LiqDarkAle:    {Name: "dark ale", Color: "dark", DrunkAffect: 1, FullAffect: 2, ThirstAffect: 2},
	LiqWhisky:     {Name: "whiskey", Color: "golden", DrunkAffect: 6, FullAffect: 1, ThirstAffect: 1},
	LiqLemonade:   {Name: "lemonade", Color: "pink", DrunkAffect: 0, FullAffect: 1, ThirstAffect: 8},
	LiqFirebrt:    {Name: "firebreather", Color: "crimson", DrunkAffect: 10, FullAffect: 0, ThirstAffect: 0},
	LiqLocalspc:   {Name: "local speciality", Color: "clear", DrunkAffect: 3, FullAffect: 3, ThirstAffect: 3},
	LiqSlime:      {Name: "slime mold juice", Color: "green", DrunkAffect: 0, FullAffect: 4, ThirstAffect: -8},
	LiqMilk:       {Name: "milk", Color: "white", DrunkAffect: 0, FullAffect: 3, ThirstAffect: 6},
	LiqTea:        {Name: "tea", Color: "brown", DrunkAffect: 0, FullAffect: 1, ThirstAffect: 6},
	LiqCoffee:     {Name: "coffee", Color: "black", DrunkAffect: 0, FullAffect: 1, ThirstAffect: 6},
	LiqBlood:      {Name: "blood", Color: "red", DrunkAffect: 0, FullAffect: 2, ThirstAffect: -1},
	LiqSaltwater:  {Name: "salt water", Color: "clear", DrunkAffect: 0, FullAffect: 1, ThirstAffect: -2},
	LiqClearwater: {Name: "clear water", Color: "clear", DrunkAffect: 0, FullAffect: 0, ThirstAffect: 13},
}

// Liquid names for display — matches drinks[] from src/constants.c
var LiquidNames = []string{
	LiqWater:      "water",
	LiqBeer:       "beer",
	LiqWine:       "wine",
	LiqAle:        "ale",
	LiqDarkAle:    "dark ale",
	LiqWhisky:     "whiskey",
	LiqLemonade:   "lemonade",
	LiqFirebrt:    "firebreather",
	LiqLocalspc:   "local speciality",
	LiqSlime:      "slime mold juice",
	LiqMilk:       "milk",
	LiqTea:        "tea",
	LiqCoffee:     "coffee",
	LiqBlood:      "blood",
	LiqSaltwater:  "salt water",
	LiqClearwater: "clear water",
}

// GetDrinkAffects returns the array-style drink_aff values for a liquid index.
// Returns [drunkAffect, fullAffect, thirstAffect] matching drink_aff[liq] in C.
func GetDrinkAffects(liq int) (drunk, full, thirst int) {
	if liq < 0 || liq >= len(Liquids) {
		return 0, 0, 0
	}
	l := Liquids[liq]
	return l.DrunkAffect, l.FullAffect, l.ThirstAffect
}

// DrinkName returns the display name for a liquid index.
func DrinkName(liq int) string {
	if liq < 0 || liq >= len(LiquidNames) {
		return "unknown"
	}
	return LiquidNames[liq]
}
