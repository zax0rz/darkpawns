package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func testBodyNames(bodies []combat.Combatant) string {
	var names []string
	for _, b := range bodies {
		names = append(names, b.GetName())
	}
	return strings.Join(names, " ")
}
