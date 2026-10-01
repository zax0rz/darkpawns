package game

import "github.com/zax0rz/darkpawns/pkg/combat"

func (p *Player) StopFighting() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Fighting == "" {
		return
	}
	p.Fighting = ""
	// C's stop_fighting resets POS_FIGHTING to standing, then update_pos
	// re-derives the wounded band from hit points (fight.c). MobInstance does
	// the same; players must not remain command-gated as fighting after their
	// opponent leaves or they are pulled into the void.
	if p.Health > 0 {
		p.Position = combat.PosStanding
	} else if p.Health <= -11 {
		p.Position = combat.PosDead
	} else if p.Health <= -6 {
		p.Position = combat.PosMortally
	} else if p.Health <= -3 {
		p.Position = combat.PosIncap
	} else {
		p.Position = combat.PosStunned
	}
}

// IsFighting returns true if the player is in combat.
func (p *Player) IsFighting() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Fighting != ""
}

// GetClass returns the player's class (Phase 2c addition)
// Source: fight.c uses GET_CLASS(ch) macro
func (p *Player) GetClass() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Class
}

// SetClass sets the player's class.
func (p *Player) SetClass(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Class = v
}

// GetStr returns the player's strength (Phase 2c addition)
// Source: fight.c uses GET_STR(ch) macro
func (p *Player) GetStr() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveAttributesLocked().Str
}

// GetDex returns the player's dexterity (Phase 2c addition)
// Source: fight.c uses GET_DEX(ch) macro
func (p *Player) GetDex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveAttributesLocked().Dex
}

// GetInt returns the player's intelligence (Phase 2c addition)
// Source: fight.c uses GET_INT(ch) macro
func (p *Player) GetInt() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveAttributesLocked().Int
}

// GetWis returns the player's wisdom (Phase 2c addition)
// Source: fight.c uses GET_WIS(ch) macro
func (p *Player) GetWis() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveAttributesLocked().Wis
}

// GetHitroll returns the player's hitroll bonus (Phase 2c addition)
// Source: fight.c uses GET_HITROLL(ch) macro
// Sums APPLY_HITROLL from all equipped items PLUS affect-modified hitroll.
// In the original C code, GET_HITROLL is a field that aggregates both equipment and spell-based modifiers.
func (p *Player) GetHitroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Hitroll + p.sumAffectModsLocked(ApplyHitroll) + p.sumEquipAffectModsLocked(ApplyHitroll)
}

// SetHitroll sets the player's affect-modified hitroll bonus.
func (p *Player) SetHitroll(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Hitroll = v
}

// AdjustHitroll changes the player's base hitroll, matching C's direct
// points.hitroll mutation used by flesh_alter_from/to().
func (p *Player) AdjustHitroll(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Hitroll += delta
}

// GetDamroll returns the player's damroll bonus (Phase 2c addition)
// Source: fight.c uses GET_DAMROLL(ch) macro
// Sums APPLY_DAMROLL from all equipped items PLUS affect-modified damroll.
// In the original C code, GET_DAMROLL is a field that aggregates both equipment and spell-based modifiers.
func (p *Player) GetDamroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Damroll + p.sumAffectModsLocked(ApplyDamroll) + p.sumEquipAffectModsLocked(ApplyDamroll)
}

// SetDamroll sets the player's affect-modified damroll bonus.
func (p *Player) SetDamroll(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Damroll = v
}

// AdjustDamroll changes the player's base damroll, matching C's direct
// points.damroll mutation used by flesh_alter_from/to().
func (p *Player) AdjustDamroll(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Damroll += delta
}

// GetStrAdd returns the player's strength add (exceptional strength)
// Source: utils.h GET_ADD(ch) macro
func (p *Player) GetStrAdd() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveAttributesLocked().StrAdd
}

// GetStrToDam returns the strength to-damage bonus.
// Source: fight.c damage() — str_app[STRENGTH_APPLY_INDEX(ch)].todam.
// Delegates to combat.StrAppToDam so the str_app table lives in one place.
func (p *Player) GetStrToDam() int {
	return combat.StrAppToDam(p)
}

// GetSex returns the player's sex.
