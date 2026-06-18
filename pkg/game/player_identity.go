package game

func (p *Player) GetSex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Sex
}

// SetSex sets the player's sex (0=male, 1=female, 2=neutral).
func (p *Player) SetSex(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Sex = v
}

// Scripting interface implementations

func (p *Player) GetID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ID
}

func (p *Player) GetHealth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

func (p *Player) SetHealth(health int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health = health
}

func (p *Player) GetMaxHealth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MaxHealth
}

func (p *Player) GetGold() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Gold
}

// GetInventory returns the player's inventory items.
func (p *Player) GetInventory() []*ObjectInstance {
	if p.Inventory == nil {
		return nil
	}
	return p.Inventory.Items
}

func (p *Player) SetGold(gold int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Gold = gold
}

// GetBankGold returns the player's banked gold (GET_BANK_GOLD in the C source).
func (p *Player) GetBankGold() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.BankGold
}

// SetBankGold sets the player's banked gold.
func (p *Player) SetBankGold(gold int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.BankGold = gold
}

// GetExp returns the player's experience points.
func (p *Player) GetExp() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Exp
}

// SetExp sets the player's experience points.
func (p *Player) SetExp(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Exp = v
}

// AddExp adds delta to the player's experience, flooring at 0.
func (p *Player) AddExp(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Exp += delta
	if p.Exp < 0 {
		p.Exp = 0
	}
}

// GetStrength returns the player's strength.
func (p *Player) GetStrength() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Str
}

// SetStrength sets the player's strength.
func (p *Player) SetStrength(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Stats.Str = v
}

func (p *Player) GetRace() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Race
}

func (p *Player) GetRoomVNum() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RoomVNum
}

// GetTitle returns the player's title.
func (p *Player) GetTitle() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Title
}

// GetDescription returns the player's description.
func (p *Player) GetDescription() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Description
}

// GetHometown returns the player's hometown index.
