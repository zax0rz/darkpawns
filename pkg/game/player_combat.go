package game

func (p *Player) StopFighting() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Fighting = ""
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

// GetStr returns the player's strength (Phase 2c addition)
// Source: fight.c uses GET_STR(ch) macro
func (p *Player) GetStr() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Str
}

// GetDex returns the player's dexterity (Phase 2c addition)
// Source: fight.c uses GET_DEX(ch) macro
func (p *Player) GetDex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Dex
}

// GetInt returns the player's intelligence (Phase 2c addition)
// Source: fight.c uses GET_INT(ch) macro
func (p *Player) GetInt() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Int
}

// GetWis returns the player's wisdom (Phase 2c addition)
// Source: fight.c uses GET_WIS(ch) macro
func (p *Player) GetWis() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Wis
}

// GetHitroll returns the player's hitroll bonus (Phase 2c addition)
// Source: fight.c uses GET_HITROLL(ch) macro
// Sums APPLY_HITROLL (location 18) from all equipped items PLUS affect-modified hitroll.
// In the original C code, GET_HITROLL is a field that aggregates both equipment and spell-based modifiers.
func (p *Player) GetHitroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := p.Hitroll
	if p.Equipment != nil {
		for _, item := range p.Equipment.Slots {
			if item == nil || item.Prototype == nil {
				continue
			}
			for _, aff := range item.Prototype.Affects {
				if aff.Location == 18 { // APPLY_HITROLL
					total += aff.Modifier
				}
			}
		}
	}
	return total
}

// SetHitroll sets the player's affect-modified hitroll bonus.
func (p *Player) SetHitroll(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Hitroll = v
}

// GetDamroll returns the player's damroll bonus (Phase 2c addition)
// Source: fight.c uses GET_DAMROLL(ch) macro
// Sums APPLY_DAMROLL (location 19) from all equipped items PLUS affect-modified damroll.
// In the original C code, GET_DAMROLL is a field that aggregates both equipment and spell-based modifiers.
func (p *Player) GetDamroll() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := p.Damroll
	if p.Equipment != nil {
		for _, item := range p.Equipment.Slots {
			if item == nil || item.Prototype == nil {
				continue
			}
			for _, aff := range item.Prototype.Affects {
				if aff.Location == 19 { // APPLY_DAMROLL
					total += aff.Modifier
				}
			}
		}
	}
	return total
}

// SetDamroll sets the player's affect-modified damroll bonus.
func (p *Player) SetDamroll(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Damroll = v
}

// GetStrAdd returns the player's strength add (exceptional strength)
// Source: utils.h GET_ADD(ch) macro
func (p *Player) GetStrAdd() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.StrAdd
}

// GetSex returns the player's sex.
