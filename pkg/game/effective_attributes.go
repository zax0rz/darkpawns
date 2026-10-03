package game

import "github.com/zax0rz/darkpawns/pkg/engine"

// boundEffectiveAttributes is affect_total's final ability pass. Base values
// remain intact for the next recomputation (src/handler.c:337-372).
func boundEffectiveAttributes(stats CharStats, npc bool) CharStats {
	limit := 18
	if npc {
		limit = 25
	}
	stats.Dex = max(0, min(stats.Dex, limit))
	stats.Int = max(0, min(stats.Int, limit))
	stats.Wis = max(0, min(stats.Wis, limit))
	stats.Con = max(0, min(stats.Con, limit))
	stats.Str = max(0, stats.Str)
	if npc {
		stats.Str = min(stats.Str, 25)
	} else if stats.Str > 18 {
		stats.StrAdd = min(100, stats.StrAdd+(stats.Str-18)*10)
		stats.Str = 18
	}
	return stats
}

func addAttributeModifier(stats *CharStats, location, modifier int) {
	switch location {
	case ApplyStr:
		stats.Str += modifier
	case ApplyDex:
		stats.Dex += modifier
	case ApplyInt:
		stats.Int += modifier
	case ApplyWis:
		stats.Wis += modifier
	case ApplyCon:
		stats.Con += modifier
	case ApplyCha:
		stats.Cha += modifier
	}
}

// effectiveAttributesLocked returns aff_abils. Before an actor's initial copy
// it has only its base constructor state; production initialization explicitly
// copies that state before the first equipment/affect boundary.
func (p *Player) effectiveAttributesLocked() CharStats {
	if p.effectiveAttributes != nil {
		return *p.effectiveAttributes
	}
	return p.Stats
}

// CopyBaseAttributes is C's aff_abils = real_abils, without affect_total's
// limits. In particular restore can copy 25 into a PC (act.wizard.c:1612).
func (p *Player) CopyBaseAttributes() {
	p.mu.Lock()
	stats := p.Stats
	p.effectiveAttributes = &stats
	p.mu.Unlock()
	p.refreshAttributeCapacity()
}

func (p *Player) affectTotalLocked() {
	stats := p.Stats
	for _, item := range COrderedWorn(p) {
		for _, af := range item.GetAffects() {
			addAttributeModifier(&stats, af.Location, af.Modifier)
		}
	}
	for _, af := range p.ActiveAffects {
		if af != nil {
			addAttributeModifier(&stats, af.Location, af.Magnitude)
		}
	}
	for _, af := range GetTattooBonuses(p.Tattoo) {
		addAttributeModifier(&stats, af.Location, af.Modifier)
	}
	stats = boundEffectiveAttributes(stats, false)
	p.effectiveAttributes = &stats
	p.Alignment = max(-1000, min(p.Alignment, 1000))
}

// AffectTotal recomputes abilities at C's equip, affect and Lua boundaries.
// It never clips base attributes or reuses an already-converted STR_ADD.
func (p *Player) AffectTotal() {
	p.mu.Lock()
	p.affectTotalLocked()
	p.mu.Unlock()
	p.refreshAttributeCapacity()
}

func (p *Player) refreshAttributeCapacity() {
	p.mu.RLock()
	stats, level := p.effectiveAttributesLocked(), p.Level
	p.mu.RUnlock()
	if p.Inventory != nil {
		p.Inventory.SetCapacity(stats.Str, stats.StrAdd, stats.Dex, level)
	}
}

func (m *MobInstance) effectiveAttributesLocked() CharStats {
	if m.effectiveAttributes != nil {
		return *m.effectiveAttributes
	}
	stats := CharStats{Str: m.Str, Dex: m.Dex, Int: m.Intel, Wis: m.Wis, Con: m.Con, Cha: m.Cha}
	if m.Runtime.StrAddOverride != nil {
		stats.StrAdd = *m.Runtime.StrAddOverride
	}
	return stats
}

func (m *MobInstance) CopyBaseAttributes() {
	m.mu.Lock()
	m.effectiveAttributes = nil
	stats := m.effectiveAttributesLocked()
	m.effectiveAttributes = &stats
	m.mu.Unlock()
}

func (m *MobInstance) AffectTotal() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.affectTotalLocked()
}

func (m *MobInstance) affectTotalLocked() {
	stats := CharStats{Str: m.Str, Dex: m.Dex, Int: m.Intel, Wis: m.Wis, Con: m.Con, Cha: m.Cha}
	if m.Runtime.StrAddOverride != nil {
		stats.StrAdd = *m.Runtime.StrAddOverride
	}
	for _, item := range m.Equipment {
		if item != nil {
			for _, af := range item.GetAffects() {
				addAttributeModifier(&stats, af.Location, af.Modifier)
			}
		}
	}
	for _, value := range m.CustomData {
		if af, ok := value.(*engine.Affect); ok {
			addAttributeModifier(&stats, af.Location, af.Magnitude)
		}
	}
	stats = boundEffectiveAttributes(stats, true)
	m.effectiveAttributes = &stats
	alignment := m.alignmentLocked()
	alignment = max(-1000, min(alignment, 1000))
	m.Runtime.AlignmentOverride = &alignment
}

// RestoreEffectiveAttributes mirrors store_to_char's base copy, unbounded
// tattoo application, then affect/equipment totals (src/db.c:2441-2481).
// The no-modifier load path retains the saved base values without clipping.
func (p *Player) RestoreEffectiveAttributes() {
	p.mu.Lock()
	stats := p.Stats
	for _, af := range GetTattooBonuses(p.Tattoo) {
		addAttributeModifier(&stats, af.Location, af.Modifier)
	}
	p.effectiveAttributes = &stats
	if len(p.ActiveAffects) > 0 || len(COrderedWorn(p)) > 0 {
		p.affectTotalLocked()
	}
	p.mu.Unlock()
	p.refreshAttributeCapacity()
}

// LoginConstitution is CON_MENU's pre-Crash_load check. Go restores objects
// before its menu; C has only store_to_char's tattoos/spells at this point.
// src/db.c:2441-2481; src/interpreter.c:2174-2184.
func (p *Player) LoginConstitution() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	con := p.Stats.Con
	for _, af := range GetTattooBonuses(p.Tattoo) {
		if af.Location == ApplyCon {
			con += af.Modifier
		}
	}
	for _, af := range p.ActiveAffects {
		if af != nil && af.Location == ApplyCon {
			con += af.Magnitude
		}
	}
	if len(p.ActiveAffects) > 0 {
		con = max(0, min(con, 18))
	}
	return con
}
