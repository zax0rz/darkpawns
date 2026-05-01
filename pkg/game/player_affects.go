package game

import "github.com/zax0rz/darkpawns/pkg/engine"

func (p *Player) HasSpellAffect(spellID int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, af := range p.ActiveAffects {
		if af != nil && af.SpellID == spellID {
			return true
		}
	}
	return false
}

// GetPosition returns the player's current position.
func (p *Player) GetPosition() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Position
}

// SetPosition sets the player's position.
func (p *Player) SetPosition(pos int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Position = pos
}

// GetFighting returns who the player is fighting.
func (p *Player) GetFighting() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Fighting
}

// SetFighting sets who the player is fighting.
func (p *Player) SetFighting(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Fighting = target
}

// GetWaitState returns the current wait state (PULSE_VIOLENCE ticks).
func (p *Player) GetWaitState() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.WaitState
}

// SetWaitState sets the wait state cooldown.
func (p *Player) SetWaitState(ticks int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.WaitState = ticks
}

// DecrementWaitState reduces wait state by 1 (called each PULSE_VIOLENCE).
func (p *Player) DecrementWaitState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.WaitState > 0 {
		p.WaitState--
	}
}

// IsParrying returns whether parry stance is active.
func (p *Player) IsParrying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Parrying
}

// SetParry toggles parry stance on/off.
func (p *Player) SetParry(active bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Parrying = active
}

// TakeDamage applies damage to the player.
func (p *Player) TakeDamage(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health -= amount
	if p.Health < 0 {
		p.Health = 0
	}
}

// GetAlignment returns the player's alignment score.
func (p *Player) GetAlignment() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Alignment
}

// IsGood returns true if alignment >= 350 (utils.h:454)
func (p *Player) IsGood() bool { return p.GetAlignment() >= 350 }

// IsEvil returns true if alignment <= -350 (utils.h:455)
func (p *Player) IsEvil() bool { return p.GetAlignment() <= -350 }

// IsNeutral returns true if not good and not evil (utils.h:456)
func (p *Player) IsNeutral() bool { return !p.IsGood() && !p.IsEvil() }

// SetSkill sets a skill level (0-100).
func (p *Player) SetSkill(name string, level int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.SkillManager == nil {
		p.SkillManager = engine.NewSkillManager()
	}
	// Create or update skill in manager
	skill := p.SkillManager.GetSkill(name)
	if skill == nil {
		// Create new skill with default values
		skill = engine.NewSkill(name, name, engine.SkillTypeUtility, 3)
		p.SkillManager.RegisterSkill(skill)
	}
	skill.Learned = true
	skill.Level = level
	if level > 0 {
		skill.Learned = true
	}
}

// GetSkill returns a skill level (0 if not set).
func (p *Player) GetSkill(name string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.SkillManager == nil {
		return 0
	}
	return p.SkillManager.GetSkillLevel(name)
}

// LoseExp deducts experience from the player, floored at 0.
func (p *Player) LoseExp(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Exp -= amount
	if p.Exp < 0 {
		p.Exp = 0
	}
}

// Heal restores health to the player.
func (p *Player) Heal(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health += amount
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
}

// Lock acquires the player's mutex. Exported for cross-package atomic
// check-and-modify patterns (e.g., gold transactions in shop_manager).
// Prefer fine-grained Get*/Set* methods when possible.
func (p *Player) Lock()   { p.mu.Lock() }
func (p *Player) Unlock() { p.mu.Unlock() }

// GetName returns the player's name.
func (p *Player) GetName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Name
}

// GetLastDeath returns the last death timestamp (unix time).
// Used by dream.c for nightmare progression.
func (p *Player) GetLastDeath() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.LastDeath
}

// SetLastDeath sets the last death timestamp (unix time).
func (p *Player) SetLastDeath(t int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastDeath = t
}

// SendMessage sends a message to the player through the session layer.
// If the player has no world reference or the world has no MessageSink,
// the message is silently dropped (non-blocking).
func (p *Player) SendMessage(msg string) {
	p.mu.RLock()
	w := p.worldRef
	p.mu.RUnlock()
	if w == nil {
		return
	}
	w.mu.RLock()
	sink := w.MessageSink
	w.mu.RUnlock()
	if sink != nil {
		sink(p.Name, []byte(msg))
	}
}

// StopFighting clears the fighting target.
