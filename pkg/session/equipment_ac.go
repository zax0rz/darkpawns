package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// CalculateAC returns the total AC for a player based on:
// - Base AC from player.AC (set to 10 in NewPlayer)
// - Sum of equipped item AC bonuses
// - Active affects that modify AC (APPLY_AC, location 17)
//
// Note: lower AC is better in DikuMUD/AD&D (negative values = better).
func CalculateAC(p *game.Player) int {
	baseAC := p.AC

	// Equipment AC bonus — armor items have AC in Values[0]
	// Equipment.GetArmorClass() already sums armor AC from worn items
	if p.Equipment != nil {
		acBonus := p.Equipment.GetArmorClass()
		// GetArmorClass returns the positive AC value from armor items.
		// In CircleMUD/DikuMUD, armor AC is subtracted from base AC,
		// and more negative = better protection.
		baseAC -= acBonus
	}

	// Active affects (engine.AffectArmorClass = 8)
	for _, aff := range p.Affects {
		if aff.Type == engine.AffectArmorClass {
			baseAC += aff.Magnitude
		}
	}

	return baseAC
}

// GetEquipmentString returns a formatted multi-line string of all equipped slots.
// Format:
//
//	<worn on finger>     [ring of power]
//	<worn on body>       [leather armor]
//	<wielded>            [short sword]
func GetEquipmentString(p *game.Player) string {
	equipped := p.Equipment.GetEquippedItems()
	if len(equipped) == 0 {
		return "You are not wearing anything."
	}

	var lines []string
	for slot, item := range equipped {
		lines = append(lines, fmt.Sprintf("<%s> [%s]", slot.String(), item.GetShortDesc()))
	}
	return strings.Join(lines, "\n")
}

// GetACString returns a one-line AC summary for display in score/examine output.
func GetACString(p *game.Player) string {
	ac := CalculateAC(p)
	return fmt.Sprintf("Armor Class: %d", ac)
}

// FormatEquipmentDisplay returns a full equipment display block including
// a formatted equipment list and AC summary. Used by examine, score, and
// equipment commands.
func FormatEquipmentDisplay(p *game.Player) string {
	equipStr := GetEquipmentString(p)
	acStr := GetACString(p)
	return fmt.Sprintf("%s\n%s", equipStr, acStr)
}
