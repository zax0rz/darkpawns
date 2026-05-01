package game

import "strings"

func (p *Player) GetHometown() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Hometown
}

// GetMountName returns the name of the mount the player is riding (empty if none).
func (p *Player) GetMountName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MountName
}

// GetCon returns the player's constitution.
func (p *Player) GetCon() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Con
}

// IsInGroup returns whether the player is in a group.
func (p *Player) IsInGroup() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.InGroup
}

// GetFollowing returns the name of the player's group leader (empty if leading).
func (p *Player) GetFollowing() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Following
}

// GetCha returns the player's charisma.
func (p *Player) GetCha() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stats.Cha
}

// GetCondition returns the value of condition cond (CondDrunk=0, CondFull=1, CondThirst=2).
// Source: structs.h:566-568, utils.h GET_COND() macro.
func (p *Player) GetCondition(cond int) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if cond < 0 || cond >= len(p.Conditions) {
		return 0
	}
	return p.Conditions[cond]
}

// SetCondition sets condition cond to val, clamped to [0,48], or -1 for immortal.
func (p *Player) SetCondition(cond, val int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cond < 0 || cond >= len(p.Conditions) {
		return
	}
	p.Conditions[cond] = val
}

// HasPLRFlag returns true if PLR flag bit n is set.
func (p *Player) HasPLRFlag(bit int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if bit < 0 || bit >= 64 {
		return false
	}
	return p.PlayerFlags&(1<<uint(bit)) != 0
}

// SetPLRFlag sets PLR flag bit n.
func (p *Player) SetPLRFlag(bit int) {
	p.mu.Lock()
	defer p.mu.Unlock()
// #nosec G115
	p.PlayerFlags |= 1 << uint(bit)
}

// ClearPLRFlag clears PLR flag bit n.
func (p *Player) ClearPLRFlag(bit int) {
	p.mu.Lock()
	defer p.mu.Unlock()
// #nosec G115
	p.PlayerFlags &= ^(1 << uint(bit))
}

// IsAffected returns true if AFF flag bit n is set.
// Source: structs.h AFF_* constants, utils.h IS_AFFECTED() macro.
func (p *Player) IsAffected(affBit int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if affBit < 0 || affBit >= 64 {
		return false
	}
	return p.Affects&(1<<uint(affBit)) != 0
}

// SetAffect sets or clears AFF flag bit n.
func (p *Player) SetAffect(affBit int, val bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if affBit < 0 || affBit >= 64 {
		return
	}
	if val {
		p.Affects |= 1 << uint(affBit)
	} else {
		p.Affects &^= 1 << uint(affBit)
	}
}

// GetFlags returns the raw PLR flags bitmask.
// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
func (p *Player) GetFlags() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Flags
}

// SetPlrFlag sets or clears PLR flag bit N on this player.
// Source: utils.h PLR_FLAGS() macro.
func (p *Player) SetPlrFlag(bit int, val bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if bit < 0 || bit >= 64 {
		return
	}
	if val {
		p.Flags |= 1 << uint(bit)
	} else {
		p.Flags &^= 1 << uint(bit)
	}
}

// IsIgnoring returns true if this player is ignoring the given player name.
func (p *Player) IsIgnoring(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.IgnoredPlayers == nil {
		return false
	}
	return p.IgnoredPlayers[strings.ToLower(name)]
}

// AddIgnore adds a player to the ignore list.
func (p *Player) AddIgnore(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.IgnoredPlayers == nil {
		p.IgnoredPlayers = make(map[string]bool)
	}
	p.IgnoredPlayers[strings.ToLower(name)] = true
}

// RemoveIgnore removes a player from the ignore list.
func (p *Player) RemoveIgnore(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.IgnoredPlayers == nil {
		return
	}
	delete(p.IgnoredPlayers, strings.ToLower(name))
}

// GetIgnoredPlayers returns a list of all ignored player names.
func (p *Player) GetIgnoredPlayers() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.IgnoredPlayers == nil {
		return nil
	}
	var names []string
	for name := range p.IgnoredPlayers {
		names = append(names, name)
	}
	return names
}
