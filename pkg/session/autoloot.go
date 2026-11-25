package session

import (
	"github.com/zax0rz/darkpawns/pkg/game"
)

// IsAutoLootEnabled returns whether the given player has auto-loot toggled on.
// Auto-loot is stored as the PrfAutoLoot player preference flag so it is
// persisted, race-safe, and kept in sync with the toggle display.
func IsAutoLootEnabled(player *game.Player) bool {
	if player == nil {
		return false
	}
	return player.GetFlags()&(1<<game.PrfAutoLoot) != 0
}
