package engine

import (
	"sync"
	"time"
)

// Affectable interface defines entities that can have affects applied to them
type Affectable interface {
	GetAffects() []*Affect
	SetAffects([]*Affect)
	GetName() string
	GetID() int
	// IsNPC returns true if this entity is a non-player character
	IsNPC() bool

	// Stat getters/setters for affects to modify
	GetStrength() int
	SetStrength(int)
	GetDexterity() int
	SetDexterity(int)
	GetIntelligence() int
	SetIntelligence(int)
	GetWisdom() int
	SetWisdom(int)
	GetConstitution() int
	SetConstitution(int)
	GetCharisma() int
	SetCharisma(int)

	GetHitRoll() int
	SetHitRoll(int)
	GetDamageRoll() int
	SetDamageRoll(int)
	GetArmorClass() int
	SetArmorClass(int)
	GetTHAC0() int
	SetTHAC0(int)

	GetHP() int
	SetHP(int)
	GetMaxHP() int
	SetMaxHP(int)
	GetMana() int
	SetMana(int)
	GetMaxMana() int
	SetMaxMana(int)
	GetMovement() int
	SetMovement(int)

	// Status flags
	HasStatusFlag(flag uint64) bool
	SetStatusFlag(flag uint64)
	ClearStatusFlag(flag uint64)

	// Messaging
	SendMessage(string)
}

// AffectManager manages affects for all affectable entities
type AffectManager struct {
	mu        sync.RWMutex
	affects   map[string][]*Affect  // entityID -> list of affects
	entityMap map[string]Affectable // entityID -> entity
}

// NewAffectManager creates a new affect manager
func NewAffectManager() *AffectManager {
	return &AffectManager{
		affects:   make(map[string][]*Affect),
		entityMap: make(map[string]Affectable),
	}
}

// RegisterEntity registers an entity with the affect manager
func (am *AffectManager) RegisterEntity(entity Affectable) {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	am.entityMap[entityID] = entity

	// Initialize empty affect list if not exists
	if _, exists := am.affects[entityID]; !exists {
		am.affects[entityID] = make([]*Affect, 0)
	}
}

// UnregisterEntity removes an entity from the affect manager
func (am *AffectManager) UnregisterEntity(entity Affectable) {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	delete(am.entityMap, entityID)
	delete(am.affects, entityID)
}

// ApplyAffect applies an affect to an entity
func (am *AffectManager) ApplyAffect(entity Affectable, affect *Affect) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)

	// Check if entity is registered (call internal version — lock already held)
	if _, exists := am.entityMap[entityID]; !exists {
		am.entityMap[entityID] = entity
		am.affects[entityID] = make([]*Affect, 0)
	}

	// Check stacking rules
	if affect.StackID != "" {
		// Remove existing affects with same StackID if MaxStacks is 1
		if affect.MaxStacks == 1 {
			am.removeAffectsByStackID(entityID, affect.StackID)
		} else {
			// Check if we've reached max stacks
			currentStacks := am.countStacks(entityID, affect.StackID)
			if currentStacks >= affect.MaxStacks {
				// Find oldest affect with this StackID and remove it
				am.removeOldestStack(entityID, affect.StackID)
			}
		}
	}

	// Apply the affect immediately
	am.applyAffectImmediate(entity, affect)

	// Add to affect list
	am.affects[entityID] = append(am.affects[entityID], affect)

	// Send notification message
	am.sendAffectMessage(entity, affect, true)

	return true
}

// RemoveAffect removes a specific affect from an entity
func (am *AffectManager) RemoveAffect(entity Affectable, affectID string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]

	for i, aff := range affects {
		if aff.ID == affectID {
			// Remove affect effects before removing from list
			am.removeAffectImmediate(entity, aff)

			// Remove from slice
			am.affects[entityID] = append(affects[:i], affects[i+1:]...)

			// Send notification message
			am.sendAffectMessage(entity, aff, false)

			return true
		}
	}

	return false
}

// RemoveAffectsByType removes all affects of a specific type from an entity
func (am *AffectManager) RemoveAffectsByType(entity Affectable, affectType AffectType) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]
	removed := 0

	var newAffects []*Affect
	for _, aff := range affects {
		if aff.Type == affectType {
			am.removeAffectImmediate(entity, aff)
			am.sendAffectMessage(entity, aff, false)
			removed++
		} else {
			newAffects = append(newAffects, aff)
		}
	}

	am.affects[entityID] = newAffects
	return removed
}

// RemoveAllAffects removes all affects from an entity
func (am *AffectManager) RemoveAllAffects(entity Affectable) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]

	// Remove all affect effects
	for _, aff := range affects {
		am.removeAffectImmediate(entity, aff)
		am.sendAffectMessage(entity, aff, false)
	}

	count := len(affects)
	am.affects[entityID] = make([]*Affect, 0)
	return count
}

// HasAffect checks if an entity has a specific affect type
func (am *AffectManager) HasAffect(entity Affectable, affectType AffectType) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]

	for _, aff := range affects {
		if aff.Type == affectType {
			return true
		}
	}

	return false
}

// GetAffects returns all affects on an entity
func (am *AffectManager) GetAffects(entity Affectable) []*Affect {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entityID := am.getEntityID(entity)
	return am.affects[entityID]
}

// Tick processes all affects for all entities
func (am *AffectManager) Tick() {
	am.mu.Lock()
	defer am.mu.Unlock()

	for entityID, affects := range am.affects {
		entity, exists := am.entityMap[entityID]
		if !exists {
			continue
		}

		var newAffects []*Affect
		for _, aff := range affects {
			expired := aff.Tick()

			if expired {
				// Affect expired, remove its effects
				am.removeAffectImmediate(entity, aff)
				am.sendAffectMessage(entity, aff, false)
			} else {
				newAffects = append(newAffects, aff)

				// Apply periodic effects (like poison damage, regeneration)
				am.applyPeriodicEffect(entity, aff)
			}
		}

		am.affects[entityID] = newAffects
	}
}

// statClampPC clamps an attribute value to the PC-appropriate range [3, 18].
func statClampPC(v int) int {
	if v < 3 {
		return 3
	}
	if v > 18 {
		return 18
	}
	return v
}

// statClampNPC clamps an attribute value to the NPC-appropriate range [3, 25].
func statClampNPC(v int) int {
	if v < 3 {
		return 3
	}
	if v > 25 {
		return 25
	}
	return v
}

// statClamp clamps an attribute based on whether the entity is an NPC.
func statClamp(entity Affectable, v int) int {
	if entity.IsNPC() {
		return statClampNPC(v)
	}
	return statClampPC(v)
}

// hitrollClamp clamps a hitroll bonus to [-100, 100].
func hitrollClamp(v int) int {
	if v < -100 {
		return -100
	}
	if v > 100 {
		return 100
	}
	return v
}

// damrollClamp clamps a damage roll bonus to [-100, 100].
func damrollClamp(v int) int {
	if v < -100 {
		return -100
	}
	if v > 100 {
		return 100
	}
	return v
}

// armorClassClamp clamps an AC value to [-100, 100].
func armorClassClamp(v int) int {
	if v < -100 {
		return -100
	}
	if v > 100 {
		return 100
	}
	return v
}

// applyAffectImmediate applies the immediate effects of an affect
func (am *AffectManager) applyAffectImmediate(entity Affectable, affect *Affect) {
	switch affect.Type {
	case AffectStrength:
		entity.SetStrength(statClamp(entity, entity.GetStrength()+affect.Magnitude))
	case AffectDexterity:
		entity.SetDexterity(statClamp(entity, entity.GetDexterity()+affect.Magnitude))
	case AffectIntelligence:
		entity.SetIntelligence(statClamp(entity, entity.GetIntelligence()+affect.Magnitude))
	case AffectWisdom:
		entity.SetWisdom(statClamp(entity, entity.GetWisdom()+affect.Magnitude))
	case AffectConstitution:
		entity.SetConstitution(statClamp(entity, entity.GetConstitution()+affect.Magnitude))
	case AffectCharisma:
		entity.SetCharisma(statClamp(entity, entity.GetCharisma()+affect.Magnitude))

	case AffectHitRoll:
		entity.SetHitRoll(hitrollClamp(entity.GetHitRoll() + affect.Magnitude))
	case AffectDamageRoll:
		entity.SetDamageRoll(damrollClamp(entity.GetDamageRoll() + affect.Magnitude))
	case AffectArmorClass:
		entity.SetArmorClass(armorClassClamp(entity.GetArmorClass() + affect.Magnitude))
	case AffectTHAC0:
		entity.SetTHAC0(entity.GetTHAC0() + affect.Magnitude)

	case AffectHP:
		entity.SetHP(entity.GetHP() + affect.Magnitude)
	case AffectMaxHP:
		entity.SetMaxHP(entity.GetMaxHP() + affect.Magnitude)
	case AffectMana:
		entity.SetMana(entity.GetMana() + affect.Magnitude)
	case AffectMaxMana:
		entity.SetMaxMana(entity.GetMaxMana() + affect.Magnitude)
	case AffectMovement:
		entity.SetMovement(entity.GetMovement() + affect.Magnitude)

	// Status flags
	case AffectBlind:
		entity.SetStatusFlag(1 << 0)
	case AffectInvisible:
		entity.SetStatusFlag(1 << 1)
	case AffectDetectInvisible:
		entity.SetStatusFlag(1 << 2)
	case AffectDetectMagic:
		entity.SetStatusFlag(1 << 3)
	case AffectSanctuary:
		entity.SetStatusFlag(1 << 4)
	case AffectFlying:
		entity.SetStatusFlag(1 << 5)
	case AffectFloating:
		entity.SetStatusFlag(1 << 6)
	case AffectPassDoor:
		entity.SetStatusFlag(1 << 7)
	case AffectSneak:
		entity.SetStatusFlag(1 << 8)
	case AffectHide:
		entity.SetStatusFlag(1 << 9)
	case AffectCharm:
		entity.SetStatusFlag(1 << 10)
	case AffectPoison:
		entity.SetStatusFlag(1 << 11)
	case AffectSleep:
		entity.SetStatusFlag(1 << 12)
	case AffectStunned:
		entity.SetStatusFlag(1 << 13)
	case AffectParalyzed:
		entity.SetStatusFlag(1 << 14)
	case AffectFlaming:
		entity.SetStatusFlag(1 << 15)
	case AffectHaste:
		entity.SetStatusFlag(1 << 16)
	case AffectSlow:
		entity.SetStatusFlag(1 << 17)
	case AffectProtectionEvil:
		entity.SetStatusFlag(1 << 18)
	case AffectProtectionGood:
		entity.SetStatusFlag(1 << 19)
	case AffectFear:
		entity.SetStatusFlag(1 << 20)
	case AffectCurse:
		entity.SetStatusFlag(1 << 21)
	case AffectSilence:
		entity.SetStatusFlag(1 << 22)
	case AffectWaterBreathing:
		entity.SetStatusFlag(1 << 23)
	case AffectRegeneration:
		entity.SetStatusFlag(1 << 24)
	case AffectInfrared:
		entity.SetStatusFlag(1 << 25)
	case AffectUltraviolet:
		entity.SetStatusFlag(1 << 26)
	}
}

// removeAffectImmediate removes the effects of an affect
func (am *AffectManager) removeAffectImmediate(entity Affectable, affect *Affect) {
	switch affect.Type {
	case AffectStrength:
		entity.SetStrength(statClamp(entity, entity.GetStrength()-affect.Magnitude))
	case AffectDexterity:
		entity.SetDexterity(statClamp(entity, entity.GetDexterity()-affect.Magnitude))
	case AffectIntelligence:
		entity.SetIntelligence(statClamp(entity, entity.GetIntelligence()-affect.Magnitude))
	case AffectWisdom:
		entity.SetWisdom(statClamp(entity, entity.GetWisdom()-affect.Magnitude))
	case AffectConstitution:
		entity.SetConstitution(statClamp(entity, entity.GetConstitution()-affect.Magnitude))
	case AffectCharisma:
		entity.SetCharisma(statClamp(entity, entity.GetCharisma()-affect.Magnitude))

	case AffectHitRoll:
		entity.SetHitRoll(hitrollClamp(entity.GetHitRoll() - affect.Magnitude))
	case AffectDamageRoll:
		entity.SetDamageRoll(damrollClamp(entity.GetDamageRoll() - affect.Magnitude))
	case AffectArmorClass:
		entity.SetArmorClass(armorClassClamp(entity.GetArmorClass() - affect.Magnitude))
	case AffectTHAC0:
		entity.SetTHAC0(entity.GetTHAC0() - affect.Magnitude)

	case AffectHP:
		// Don't reduce current HP when affect is removed
		// Only adjust if it was a max HP affect
	case AffectMaxHP:
		entity.SetMaxHP(entity.GetMaxHP() - affect.Magnitude)
		// Ensure current HP doesn't exceed new max
		if entity.GetHP() > entity.GetMaxHP() {
			entity.SetHP(entity.GetMaxHP())
		}
	case AffectMana:
		// Don't reduce current mana when affect is removed
	case AffectMaxMana:
		entity.SetMaxMana(entity.GetMaxMana() - affect.Magnitude)
		// Ensure current mana doesn't exceed new max
		if entity.GetMana() > entity.GetMaxMana() {
			entity.SetMana(entity.GetMaxMana())
		}
	case AffectMovement:
		entity.SetMovement(entity.GetMovement() - affect.Magnitude)

	// Clear status flags
	case AffectBlind:
		entity.ClearStatusFlag(1 << 0)
	case AffectInvisible:
		entity.ClearStatusFlag(1 << 1)
	case AffectDetectInvisible:
		entity.ClearStatusFlag(1 << 2)
	case AffectDetectMagic:
		entity.ClearStatusFlag(1 << 3)
	case AffectSanctuary:
		entity.ClearStatusFlag(1 << 4)
	case AffectFlying:
		entity.ClearStatusFlag(1 << 5)
	case AffectFloating:
		entity.ClearStatusFlag(1 << 6)
	case AffectPassDoor:
		entity.ClearStatusFlag(1 << 7)
	case AffectSneak:
		entity.ClearStatusFlag(1 << 8)
	case AffectHide:
		entity.ClearStatusFlag(1 << 9)
	case AffectCharm:
		entity.ClearStatusFlag(1 << 10)
	case AffectPoison:
		entity.ClearStatusFlag(1 << 11)
	case AffectSleep:
		entity.ClearStatusFlag(1 << 12)
	case AffectStunned:
		entity.ClearStatusFlag(1 << 13)
	case AffectParalyzed:
		entity.ClearStatusFlag(1 << 14)
	case AffectFlaming:
		entity.ClearStatusFlag(1 << 15)
	case AffectHaste:
		entity.ClearStatusFlag(1 << 16)
	case AffectSlow:
		entity.ClearStatusFlag(1 << 17)
	case AffectProtectionEvil:
		entity.ClearStatusFlag(1 << 18)
	case AffectProtectionGood:
		entity.ClearStatusFlag(1 << 19)
	case AffectFear:
		entity.ClearStatusFlag(1 << 20)
	case AffectCurse:
		entity.ClearStatusFlag(1 << 21)
	case AffectSilence:
		entity.ClearStatusFlag(1 << 22)
	case AffectWaterBreathing:
		entity.ClearStatusFlag(1 << 23)
	case AffectRegeneration:
		entity.ClearStatusFlag(1 << 24)
	case AffectInfrared:
		entity.ClearStatusFlag(1 << 25)
	case AffectUltraviolet:
		entity.ClearStatusFlag(1 << 26)
	}
}

// applyPeriodicEffect applies periodic effects like poison damage or regeneration
func (am *AffectManager) applyPeriodicEffect(entity Affectable, affect *Affect) {
	switch affect.Type {
	case AffectPoison:
		// Poison does damage each tick
		damage := affect.Magnitude
		if damage < 0 {
			damage = -damage // Ensure positive damage
		}
		if damage == 0 {
			damage = 1 // Default poison damage
		}

		currentHP := entity.GetHP()
		newHP := currentHP - damage
		if newHP < 0 {
			newHP = 0
		}
		entity.SetHP(newHP)

		// Send poison damage message
		entity.SendMessage("The poison courses through your veins!")

	case AffectRegeneration:
		// Regeneration heals each tick
		heal := affect.Magnitude
		if heal < 0 {
			heal = -heal // Ensure positive healing
		}
		if heal == 0 {
			heal = 1 // Default regeneration
		}

		currentHP := entity.GetHP()
		maxHP := entity.GetMaxHP()
		newHP := currentHP + heal
		if newHP > maxHP {
			newHP = maxHP
		}
		entity.SetHP(newHP)

		// Send regeneration message
		if heal > 0 {
			entity.SendMessage("You feel your wounds healing.")
		}

	case AffectHaste:
		// Haste might give extra attacks, handled elsewhere
		// For now, just a placeholder

	case AffectSlow:
		// Slow might reduce attacks, handled elsewhere
		// For now, just a placeholder
	}
}

// Helper methods
func (am *AffectManager) getEntityID(entity Affectable) string {
	return entity.GetName() + "_" + string(rune(entity.GetID()))
}

func (am *AffectManager) removeAffectsByStackID(entityID, stackID string) {
	affects := am.affects[entityID]
	var newAffects []*Affect

	for _, aff := range affects {
		if aff.StackID != stackID {
			newAffects = append(newAffects, aff)
		} else {
			// Remove affect effects
			if entity, exists := am.entityMap[entityID]; exists {
				am.removeAffectImmediate(entity, aff)
				am.sendAffectMessage(entity, aff, false)
			}
		}
	}

	am.affects[entityID] = newAffects
}

func (am *AffectManager) countStacks(entityID, stackID string) int {
	count := 0
	for _, aff := range am.affects[entityID] {
		if aff.StackID == stackID {
			count++
		}
	}
	return count
}

func (am *AffectManager) removeOldestStack(entityID, stackID string) {
	affects := am.affects[entityID]
	var oldestIndex = -1
	var oldestTime time.Time

	for i, aff := range affects {
		if aff.StackID == stackID {
			if oldestIndex == -1 || aff.AppliedAt.Before(oldestTime) {
				oldestIndex = i
				oldestTime = aff.AppliedAt
			}
		}
	}

	if oldestIndex != -1 {
		// Remove oldest affect
		oldestAffect := affects[oldestIndex]
		if entity, exists := am.entityMap[entityID]; exists {
			am.removeAffectImmediate(entity, oldestAffect)
			am.sendAffectMessage(entity, oldestAffect, false)
		}

		am.affects[entityID] = append(affects[:oldestIndex], affects[oldestIndex+1:]...)
	}
}

func (am *AffectManager) sendAffectMessage(entity Affectable, affect *Affect, applied bool) {
	if applied {
		entity.SendMessage("You feel " + getAffectDescription(affect, true))
	} else {
		entity.SendMessage("The " + getAffectDescription(affect, false) + " wears off.")
	}
}

func getAffectDescription(affect *Affect, applied bool) string {
	switch affect.Type {
	case AffectStrength:
		if affect.Magnitude > 0 {
			return "stronger"
		} else {
			return "weaker"
		}
	case AffectHaste:
		return "hasted"
	case AffectSlow:
		return "slowed"
	case AffectPoison:
		return "poisoned"
	case AffectRegeneration:
		return "regenerating"
	case AffectInvisible:
		return "invisible"
	case AffectBlind:
		return "blind"
	case AffectSleep:
		return "sleepy"
	case AffectStunned:
		return "stunned"
	case AffectParalyzed:
		return "paralyzed"
	case AffectFlaming:
		return "flaming"
	case AffectCharm:
		return "charmed"
	case AffectFear:
		return "fearful"
	case AffectCurse:
		return "cursed"
	default:
		if applied {
			return "affected"
		} else {
			return "effect"
		}
	}
}
