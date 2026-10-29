package game

// player_physical.go — worn APPLY_AGE / APPLY_CHAR_WEIGHT / APPLY_CHAR_HEIGHT
// (src/handler.c:160-166).
//
// C mutates ch->player.time.birth, player.weight, and player.height in place
// at every affect boundary: equip_char and unequip_char call affect_modify_ar
// per item (src/handler.c:744,788), timed affects go through affect_modify,
// and affect_total's remove+apply passes cancel each other for these fields
// because they live in player data, not aff_abils (src/handler.c:314-345).
// char_to_store unwinds every modifier before saving — unequip all, strip
// affected spells, then write — and reapplies them afterwards
// (src/db.c:2531-2637), so the persisted fields are always the base values.
// The effective value at any instant is therefore base + sum(current
// modifiers), computed here on demand; player.Height/Weight/Birth keep the
// base and stay the serialization contract. Weight and height wrap as C
// ubytes (src/structs.h:885-886); repeated recomputation cannot drift.

import "github.com/zax0rz/darkpawns/pkg/engine"

// physicalDeltaLocked returns the current worn/timed/tattoo modifiers for the
// player. The caller holds p.mu (COrderedWorn reads the equipment under the
// established p.mu → eq.mu order used by affectTotalForItemsLocked).
func (p *Player) physicalDeltaLocked() (weightMod, heightMod int, birthShift int64) {
	for _, item := range COrderedWorn(p) {
		if item == nil {
			continue
		}
		for _, af := range item.GetAffects() {
			// C stores object-affect modifiers as sbyte (structs.h:736-739);
			// wrap like the mobile path before summing.
			mod := mobileSignedPoint(af.Modifier, 8)
			switch af.Location {
			case engine.ApplyCharWeight:
				weightMod += mod
			case engine.ApplyCharHeight:
				heightMod += mod
			case engine.ApplyAge:
				birthShift += int64(mod) * int64(SECS_PER_MUD_YEAR)
			}
		}
	}
	for _, af := range p.ActiveAffects {
		if af == nil {
			continue
		}
		// C's affected_type modifier is also sbyte (structs.h:1050).
		magnitude := mobileSignedPoint(af.Magnitude, 8)
		switch af.Location {
		case engine.ApplyCharWeight:
			weightMod += magnitude
		case engine.ApplyCharHeight:
			heightMod += magnitude
		case engine.ApplyAge:
			birthShift += int64(magnitude) * int64(SECS_PER_MUD_YEAR)
		}
	}
	for _, af := range GetTattooBonuses(p.Tattoo) {
		switch af.Location {
		case engine.ApplyCharWeight:
			weightMod += af.Modifier
		case engine.ApplyCharHeight:
			heightMod += af.Modifier
		case engine.ApplyAge:
			birthShift += int64(af.Modifier) * int64(SECS_PER_MUD_YEAR)
		}
	}
	return weightMod, heightMod, birthShift
}

// ubyteWrap truncates to C's unsigned-byte storage (src/structs.h:885-886).
func ubyteWrap(v int) int {
	return v & 0xFF
}

// EffectiveBirth returns the birth timestamp with worn/timed/tattoo APPLY_AGE
// modifiers applied (C: birth -= mod * SECS_PER_MUD_YEAR).
func (p *Player) EffectiveBirth() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, _, birthShift := p.physicalDeltaLocked()
	return p.Birth - birthShift
}
