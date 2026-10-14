package game

// modifyRaceHateSlots mirrors src/handler.c:205-238. A negative modifier removes
// the first matching race, zero toggles the single human slot, and a positive
// modifier fills the first empty slot.
func modifyRaceHateSlots(slots *[5]int, modifier int) {
	if slots == nil {
		return
	}
	if modifier < 0 {
		for i, race := range slots {
			if race == -modifier {
				slots[i] = -1
				return
			}
		}
		return
	}
	if modifier == 0 {
		for i, race := range slots {
			if race == 0 {
				slots[i] = -1
				return
			}
		}
	}
	for i, race := range slots {
		if race == -1 {
			slots[i] = modifier
			return
		}
	}
}

// raceHateSlotsForItems rebuilds the C race_hate array from worn items in
// WEAR_* order. C's affect_total revisits equipment in that order, so replaying
// APPLY_RACE_HATE records preserves negative removals and the one-human toggle.
func raceHateSlotsForItems(items []*ObjectInstance) [5]int {
	var slots [5]int
	for i := range slots {
		slots[i] = -1
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		for _, affect := range item.GetAffects() {
			if affect.Location == ApplyRaceHate {
				modifyRaceHateSlots(&slots, affect.Modifier)
			}
		}
	}
	return slots
}
