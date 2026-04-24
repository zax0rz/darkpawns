package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// cmdAffects lists the player's active spell affects.
// Source: act.informative.c ACMD(do_affects)
func cmdAffects(s *Session, args []string) error {
	if len(s.player.Affects) == 0 {
		s.Send("You are not affected by any spells.")
		return nil
	}

	for _, aff := range s.player.Affects {
		spellName := aff.Source
		if spellName == "" {
			spellName = "unknown"
		}
		s.Send(fmt.Sprintf("Spell: %s  duration: %d  level: %d", spellName, aff.Duration, aff.Magnitude))
	}

	return nil
}

// Ensure engine import is used.
var _ = engine.Affect{}
