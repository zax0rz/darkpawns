package game

func (p *Player) GetSex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Sex
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

// GetHometown returns the player's hometown index.
