package spells

import (
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// spellAffectMap maps non-damage spell IDs to their corresponding affect types.
var spellAffectMap = map[int]engine.AffectType{
	SpellBlindness: engine.AffectBlind,
	SpellCurse:     engine.AffectCurse,
	SpellPoison:    engine.AffectPoison,
	SpellSleep:     engine.AffectSleep,
	SpellSanctuary: engine.AffectSanctuary,
}

// spellNames maps spell IDs to their human-readable source names.
var spellNames = map[int]string{
	SpellBlindness: "blindness",
	SpellCurse:     "curse",
	SpellPoison:    "poison",
	SpellSleep:     "sleep",
	SpellSanctuary: "sanctuary",
}

// ApplySpellAffects creates an engine.Affect from a non-damage spell and applies
// it to the target via the provided AffectManager.
//
// Parameters:
//   - target: the entity receiving the affect (must implement engine.Affectable)
//   - spellID: one of the Spell* constants (e.g. SpellBlindness)
//   - casterLevel: caster's level, used to scale duration and magnitude
//   - am: the active AffectManager to register the affect with
//
// duration  = casterLevel * 2 (in seconds/ticks)
// magnitude = casterLevel/5 + 1 (min 1)
// source    = spell name string
//
// Returns an error if the spell ID has no mapping (not a non-damage affect spell).
func ApplySpellAffects(target engine.Affectable, spellID int, casterLevel int, am *engine.AffectManager) error {
	affectType, ok := spellAffectMap[spellID]
	if !ok {
		return nil // spell is not a non-damage affect spell; not an error
	}

	duration := casterLevel * 2
	magnitude := casterLevel/5 + 1
	if magnitude < 1 {
		magnitude = 1
	}

	source, ok := spellNames[spellID]
	if !ok {
		source = "unknown spell"
	}

	affect := engine.NewAffect(affectType, duration, magnitude, source)
	am.ApplyAffect(target, affect)

	return nil
}
