package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// grantClassSpells maintains the legacy SpellMap catalog with every castable
// spell the player's class qualifies for at their current level. The cast
// command uses the canonical class-level table plus practiced proficiency;
// SpellMap remains populated for save compatibility and older consumers.
//
// It is additive and idempotent: safe to call on character creation, on login,
// and after leveling up. Entries whose number is not a castable spell (i.e. the
// class's combat/utility skills, which have no spellDB entry) are skipped here —
// those are gated separately by CanUseSkill / the guild practice proc.
//
// The class→skill catalog now lives in game.ClassSpells (canonical layer).
func grantClassSpells(p *game.Player) {
	if p == nil {
		return
	}
	class := p.Class
	if class < 0 || class >= len(game.ClassSpells) {
		return
	}
	if p.SpellMap == nil {
		p.SpellMap = make(map[string]int)
	}
	for _, entry := range game.ClassSpells[class] {
		if entry.Level > p.Level {
			continue
		}
		sd, ok := spellDB[entry.Num]
		if !ok {
			continue // a skill, not a castable spell
		}
		name := strings.ToLower(sd.Name)
		if _, known := p.SpellMap[name]; !known {
			p.SpellMap[name] = entry.Level
		}
	}
}
