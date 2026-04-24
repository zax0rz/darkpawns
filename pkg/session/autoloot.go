package session

import (
	"strings"
)

// autoLootState tracks which players have auto-loot enabled.
// Key is the lowercased player name.
var autoLootState = make(map[string]bool)

// IsAutoLootEnabled returns whether the given player has auto-loot toggled on.
func IsAutoLootEnabled(name string) bool {
	return autoLootState[strings.ToLower(name)]
}

// cmdAutoLoot toggles automatic looting after kills.
func cmdAutoLoot(s *Session, args []string) error {
	name := strings.ToLower(s.player.Name)
	current := autoLootState[name]
	autoLootState[name] = !current
	if !current {
		s.Send("Auto-loot enabled.")
	} else {
		s.Send("Auto-loot disabled.")
	}
	return nil
}

func init() {
	cmdRegistry.Register("autoloot", wrapArgs(cmdAutoLoot), "Toggle auto-looting.", 0, 0)
}
