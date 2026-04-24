package game

// Stubs for functions referenced by pkg/session that live in C files not yet ported.

// EatFood handles consuming a food item (from act.item.c).
// Returns the food value amount and any error.
func EatFood(p *Player, item *ObjectInstance) (int, error) {
	return 1, nil
}

// DrinkLiquid handles drinking from a drink container (from act.item.c).
// Returns amount consumed, liquid type, and any error.
func DrinkLiquid(p *Player, item *ObjectInstance) (int, int, error) {
	return 1, 0, nil
}

// MakeCorpse creates a corpse object when a character dies (from act.other.c).
func MakeCorpse(ch *Player) *ObjectInstance {
	return nil
}
