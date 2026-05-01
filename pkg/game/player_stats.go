package game

import (
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func (p *Player) UpdateActivity() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastActive = time.Now()
}

// SetRoom changes the player's current room.
func (p *Player) SetRoom(vnum int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RoomVNum = vnum
}

// GetRoom returns the player's current room VNum.
func (p *Player) GetRoom() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RoomVNum
}

// Combatant interface implementation

// GetLevel returns the player's level.
func (p *Player) GetLevel() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Level
}

// GetTHAC0 returns the player's THAC0.
func (p *Player) GetTHAC0() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.THAC0
}

// GetAC returns the player's Armor Class including equipment bonuses.
func (p *Player) GetAC() int {
	p.mu.RLock()
	baseAC := p.AC
	p.mu.RUnlock()

	// Add equipment AC bonus
	if p.Equipment != nil {
		baseAC -= p.Equipment.GetArmorClass() // Lower AC is better
	}

	return baseAC
}

// GetHP returns the player's current health.
func (p *Player) GetHP() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

// GetMaxHP returns the player's maximum health.
func (p *Player) GetMaxHP() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MaxHealth
}

// GetMove returns the player's current movement points.
func (p *Player) GetMove() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Move
}

// SetMove sets the player's current movement points.
func (p *Player) SetMove(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Move = v
}

// GetMaxMove returns the player's maximum movement points.
func (p *Player) GetMaxMove() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MaxMove
}

// SetMaxMove sets the player's maximum movement points.
func (p *Player) SetMaxMove(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.MaxMove = v
}

// GetPractices returns the player's practice sessions.
func (p *Player) GetPractices() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Practices
}

// SetPractices sets the player's practice sessions.
func (p *Player) SetPractices(v int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Practices = v
}

// GetDamageRoll returns the player's damage dice including weapon.
func (p *Player) GetDamageRoll() combat.DiceRoll {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if wielding a weapon
	if p.Equipment != nil {
		num, sides := p.Equipment.GetWeaponDamage()
		if num > 0 && sides > 0 {
			return combat.DiceRoll{Num: num, Sides: sides, Plus: 0}
		}
	}

	// No weapon equipped — bare hands
	return combat.DiceRoll{Num: 0, Sides: 0, Plus: 0}
}

// IsNPC returns false for players.
func (p *Player) IsNPC() bool {
	return false
}

// ──── StatModifiable interface implementation ────

// GetMapping returns the stat name → field pointer mapping used by GetStat/SetStat.
func statMapping(p *Player) map[string]*int {
	return map[string]*int{
		"STR":       &p.Stats.Str,
		"DEX":       &p.Stats.Dex,
		"INT":       &p.Stats.Int,
		"WIS":       &p.Stats.Wis,
		"CON":       &p.Stats.Con,
		"CHA":       &p.Stats.Cha,
		"HP":        &p.MaxHealth,
		"Mana":      &p.MaxMana,
		"Move":      &p.MaxMove,
		"Hitroll":   &p.Hitroll,
		"Damroll":   &p.Damroll,
		"AC":        &p.AC,
		"Level":     &p.Level,
		"Alignment": &p.Alignment,
		"StrAdd":    &p.Stats.StrAdd,
	}
}

// GetStat returns the current value of a named stat.
func (p *Player) GetStat(name string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if name == "Age" {
		// Age in MUD years: roughly 100 real seconds = 1 MUD year
		birth := time.Unix(p.Birth, 0)
		age := int(time.Since(birth).Seconds() / 100)
		if age < 1 {
			age = 1
		}
		return age
	}
	if ptr, ok := statMapping(p)[name]; ok {
		return *ptr
	}
	return 0
}

// SetStat sets the value of a named stat.
func (p *Player) SetStat(name string, val int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ptr, ok := statMapping(p)[name]; ok {
		*ptr = val
	}
}

// GetMaxStat returns the maximum value for a stat (same as GetStat for now).
func (p *Player) GetMaxStat(name string) int {
	return p.GetStat(name)
}

// SetMaxStat sets the maximum value for a stat (same as SetStat for now).
func (p *Player) SetMaxStat(name string, val int) {
	p.SetStat(name, val)
}

// AddStat adds delta to a named stat.
func (p *Player) AddStat(name string, delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ptr, ok := statMapping(p)[name]; ok {
		*ptr += delta
	}
}

// GetSavingThrow returns the saving throw value at the given index (0-4).
func (p *Player) GetSavingThrow(idx int) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if idx < 0 || idx >= 5 {
		return 0
	}
	return p.SavingThrows[idx]
}

// SetSavingThrow sets the saving throw value at the given index (0-4).
func (p *Player) SetSavingThrow(idx int, val int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx < 0 || idx >= 5 {
		return
	}
	p.SavingThrows[idx] = val
}

// GetAffectBitVector returns the affect bitmask.
func (p *Player) GetAffectBitVector() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Affects
}

// SetAffectBitVector sets the affect bitmask.
func (p *Player) SetAffectBitVector(v uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Affects = v
}

// SetAffectBit sets or clears a specific bit in the affect bitmask.
func (p *Player) SetAffectBit(bit uint64, val bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if val {
		p.Affects |= 1 << uint(bit)
	} else {
		p.Affects &^= 1 << uint(bit)
	}
}

// GetMasterAffects returns the list of active master affects.
func (p *Player) GetMasterAffects() []*engine.MasterAffect {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.MasterAffects
}

// SetMasterAffects replaces the list of active master affects.
func (p *Player) SetMasterAffects(affects []*engine.MasterAffect) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.MasterAffects = affects
}

// AddMasterAffect prepends a master affect to the list.
func (p *Player) AddMasterAffect(af *engine.MasterAffect) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.MasterAffects = append([]*engine.MasterAffect{af}, p.MasterAffects...)
}

// RemoveMasterAffect removes a master affect from the list by pointer equality.
func (p *Player) RemoveMasterAffect(af *engine.MasterAffect) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, ma := range p.MasterAffects {
		if ma == af {
			p.MasterAffects = append(p.MasterAffects[:i], p.MasterAffects[i+1:]...)
			return
		}
	}
}

// RemoveAffectBit clears a bit from the player's Affects bitmask.
func (p *Player) RemoveAffectBit(bit int) {
	p.mu.Lock()
	defer p.mu.Unlock()
// #nosec G115
	p.Affects &^= 1 << uint(bit)
}

// GetEquipment returns the player's equipment iterator.
// Returns nil as the nested interface is complex and not yet wired for Go types.
// GetEquipAffects returns all equipment affects for the AffectTotal recalculation.
// Matches C's affect_total() equipment loop.
func (p *Player) GetEquipAffects() []engine.EquipAffectData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.Equipment == nil {
		return nil
	}
	var result []engine.EquipAffectData
	for _, item := range p.Equipment.GetEquippedItems() {
		if item == nil || item.Prototype == nil {
			continue
		}
		// Compute bitvector from ExtraFlags
		var bv uint64
		for i, f := range item.Prototype.ExtraFlags {
// #nosec G115
			bv |= uint64(f) << (uint(i) * 32)
		}
		for _, af := range item.Prototype.Affects {
			result = append(result, engine.EquipAffectData{
				Location:  af.Location,
				Modifier:  af.Modifier,
				Bitvector: bv,
			})
		}
	}
	return result
}

func (p *Player) GetEquipment() interface {
	GetItems() []interface {
		GetAffects() []interface{ GetLocation() int; GetModifier() int }
		GetBitvector() uint64
	}
} {
	return nil
}

// HasSpellAffect checks if the player has an active affect from a specific spell/skill.
// Equivalent to C's affected_by_spell(ch, type) — src/handler.c:460.
