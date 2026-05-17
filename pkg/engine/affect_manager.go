package engine

import (
	"strconv"
	"sync"
	"time"
)

// Affectable interface defines entities that can have affects applied to them
type Affectable interface {
	GetAffects() []*Affect
	SetAffects([]*Affect)
	GetName() string
	GetID() int
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

// AffectManager manages affects for all affectable entities.
// Single source of truth for all affects — spells, equipment, items.
type AffectManager struct {
	mu        sync.RWMutex
	affects   map[string][]*Affect  // entityID -> flat list of affects
	entityMap map[string]Affectable // entityID -> entity
	flagRefs  map[string]map[uint64]int // entityID -> flag -> reference count
}

// NewAffectManager creates a new affect manager
func NewAffectManager() *AffectManager {
	return &AffectManager{
		affects:   make(map[string][]*Affect),
		entityMap: make(map[string]Affectable),
		flagRefs:  make(map[string]map[uint64]int),
	}
}

// RegisterEntity registers an entity with the affect manager
func (am *AffectManager) RegisterEntity(entity Affectable) {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	am.entityMap[entityID] = entity
	if _, exists := am.affects[entityID]; !exists {
		am.affects[entityID] = make([]*Affect, 0)
	}
	if _, exists := am.flagRefs[entityID]; !exists {
		am.flagRefs[entityID] = make(map[uint64]int)
	}
}

// UnregisterEntity removes an entity from the affect manager
func (am *AffectManager) UnregisterEntity(entity Affectable) {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	delete(am.entityMap, entityID)
	delete(am.affects, entityID)
	delete(am.flagRefs, entityID)
}

// ApplyAffect applies an affect to an entity
func (am *AffectManager) ApplyAffect(entity Affectable, affect *Affect) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)

	// Auto-register if not registered
	if _, exists := am.entityMap[entityID]; !exists {
		am.entityMap[entityID] = entity
		am.affects[entityID] = make([]*Affect, 0)
		am.flagRefs[entityID] = make(map[uint64]int)
	}

	// Stack dedup: if StackID is set, enforce MaxStacks
	if affect.StackID != "" {
		if affect.MaxStacks == 1 {
			am.removeAffectsByStackID(entityID, affect.StackID)
		} else {
			currentStacks := am.countStacks(entityID, affect.StackID)
			if currentStacks >= affect.MaxStacks {
				am.removeOldestStack(entityID, affect.StackID)
			}
		}
	}

	// Apply immediate effects
	am.applyAffectImmediate(entity, affect)

	// Track flag references
	if affect.Flags != 0 {
		if am.flagRefs[entityID] == nil {
			am.flagRefs[entityID] = make(map[uint64]int)
		}
		am.flagRefs[entityID][affect.Flags]++
	}

	// Add to affect list
	am.affects[entityID] = append(am.affects[entityID], affect)

	// Send notification message
	am.sendAffectMessage(entity, affect, true)

	return true
}

// RemoveAffect removes a specific affect by ID
func (am *AffectManager) RemoveAffect(entity Affectable, affectID string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]

	for i, aff := range affects {
		if aff.ID == affectID {
			am.removeAffectImmediate(entity, aff)

			// Decrement flag references
			if aff.Flags != 0 {
				if refs, ok := am.flagRefs[entityID]; ok {
					refs[aff.Flags]--
					if refs[aff.Flags] <= 0 {
						entity.ClearStatusFlag(aff.Flags)
						delete(refs, aff.Flags)
					}
				}
			}

			am.affects[entityID] = append(affects[:i], affects[i+1:]...)
			am.sendAffectMessage(entity, aff, false)
			return true
		}
	}

	return false
}

// RemoveAffectsBySpell removes all affects from a specific spell
func (am *AffectManager) RemoveAffectsBySpell(entity Affectable, spellID int) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]
	removed := 0

	var newAffects []*Affect
	for _, aff := range affects {
		if aff.SpellID == spellID {
			am.removeAffectImmediate(entity, aff)
			if aff.Flags != 0 {
				if refs, ok := am.flagRefs[entityID]; ok {
					refs[aff.Flags]--
					if refs[aff.Flags] <= 0 {
						entity.ClearStatusFlag(aff.Flags)
						delete(refs, aff.Flags)
					}
				}
			}
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

	for _, aff := range affects {
		am.removeAffectImmediate(entity, aff)
		am.sendAffectMessage(entity, aff, false)
	}

	// Clear all flag references
	if refs, ok := am.flagRefs[entityID]; ok {
		for flag := range refs {
			entity.ClearStatusFlag(flag)
		}
		am.flagRefs[entityID] = make(map[uint64]int)
	}

	count := len(affects)
	am.affects[entityID] = make([]*Affect, 0)
	return count
}

// HasAffectBySpell checks if an entity has any affect from a specific spell
func (am *AffectManager) HasAffectBySpell(entity Affectable, spellID int) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entityID := am.getEntityID(entity)
	for _, aff := range am.affects[entityID] {
		if aff.SpellID == spellID {
			return true
		}
	}
	return false
}

// HasAffectByFlag checks if an entity has any affect with a specific flag
func (am *AffectManager) HasAffectByFlag(entity Affectable, flag uint64) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entityID := am.getEntityID(entity)
	for _, aff := range am.affects[entityID] {
		if aff.Flags&flag != 0 {
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

// GetAffectsBySpell returns all affects from a specific spell on an entity
func (am *AffectManager) GetAffectsBySpell(entity Affectable, spellID int) []*Affect {
	am.mu.RLock()
	defer am.mu.RUnlock()

	entityID := am.getEntityID(entity)
	var result []*Affect
	for _, aff := range am.affects[entityID] {
		if aff.SpellID == spellID {
			result = append(result, aff)
		}
	}
	return result
}

// RecalculateStats strips all affect stat modifications and re-applies them.
// This is the unified replacement for AffectTotal from handler.c.
// Must be called under am.mu or externally synchronized.
func (am *AffectManager) RecalculateStats(entity Affectable) {
	am.mu.Lock()
	defer am.mu.Unlock()

	entityID := am.getEntityID(entity)
	affects := am.affects[entityID]

	// Phase 1: Strip all stat modifications from affects
	for _, aff := range affects {
		am.removeAffectImmediate(entity, aff)
	}

	// Phase 2: Strip equipment affects if entity provides them
	if ep, ok := entity.(EquipAffectProvider); ok {
		for _, ea := range ep.GetEquipAffects() {
			eaAffect := &Affect{Location: ea.Location, Magnitude: -ea.Modifier, Flags: ea.Bitvector}
			am.removeAffectImmediate(entity, eaAffect)
		}
	}

	// Phase 3: Re-apply affect stat modifications
	for _, aff := range affects {
		am.applyAffectImmediate(entity, aff)
	}

	// Phase 4: Re-apply equipment affects
	if ep, ok := entity.(EquipAffectProvider); ok {
		for _, ea := range ep.GetEquipAffects() {
			eaAffect := &Affect{Location: ea.Location, Magnitude: ea.Modifier, Flags: ea.Bitvector}
			am.applyAffectImmediate(entity, eaAffect)
		}
	}

	// Phase 5: Clamp values
	am.clampStats(entity)
}

// Tick processes all affects for all entities.
// Lock ordering: collect expired affects under lock, release lock,
// then process removals and messages outside lock (DP-153).
func (am *AffectManager) Tick() {
	type expiredEntry struct {
		entityID string
		entity   Affectable
		aff      *Affect
		spellID  int
	}
	var expiredAffects []expiredEntry

	am.mu.Lock()
	for entityID, affects := range am.affects {
		entity, exists := am.entityMap[entityID]
		if !exists {
			continue
		}

		var newAffects []*Affect
		for _, aff := range affects {
			// Apply periodic effects before ticking
			am.applyPeriodicEffect(entity, aff)

			isExpired := aff.Tick()
			if isExpired {
				expiredAffects = append(expiredAffects, expiredEntry{
					entityID: entityID,
					entity:   entity,
					aff:      aff,
					spellID:  aff.SpellID,
				})
			} else {
				newAffects = append(newAffects, aff)
			}
		}
		am.affects[entityID] = newAffects
	}
	am.mu.Unlock()

	// Phase 2: process removals and messages outside lock
	// Group by entity+spellID so we remove all affects from one spell together
	type spellRemovalKey struct {
		entityID string
		spellID  int
	}
	seen := make(map[spellRemovalKey]bool)
	for _, entry := range expiredAffects {
		key := spellRemovalKey{entry.entityID, entry.spellID}
		if !seen[key] {
			seen[key] = true
			am.RemoveAffectsBySpell(entry.entity, entry.spellID)
		}
	}
}

// --- Internal methods ---

// applyAffectImmediate applies the immediate effects of an affect using the
// table-driven location mapping.
func (am *AffectManager) applyAffectImmediate(entity Affectable, affect *Affect) {
	// Table-driven stat modification
	if statName, ok := applyLocationToStat[affect.Location]; ok {
		addStat(entity, statName, affect.Magnitude)
		return
	}

	// Status flags via bitvector
	if affect.Flags != 0 {
		entity.SetStatusFlag(affect.Flags)
	}

	// Handle special locations
	switch affect.Location {
	case ApplyNone:
		// No stat modification, just flags
	}
}

// removeAffectImmediate removes the effects of an affect
func (am *AffectManager) removeAffectImmediate(entity Affectable, affect *Affect) {
	// Table-driven stat modification (reverse)
	if statName, ok := applyLocationToStat[affect.Location]; ok {
		addStat(entity, statName, -affect.Magnitude)
		return
	}

	// Status flags are handled via flagRefs in the public Remove methods
}

// applyPeriodicEffect applies periodic effects like poison damage or regeneration
func (am *AffectManager) applyPeriodicEffect(entity Affectable, affect *Affect) {
	// Check by flags for backward compatibility with spellStatusFlags
	if affect.Flags&AFFPoison != 0 {
		damage := affect.Magnitude
		if damage < 0 {
			damage = -damage
		}
		if damage < 1 {
			damage = 1
		}
		currentHP := entity.GetHP()
		newHP := currentHP - damage
		if newHP < 0 {
			newHP = 0
		}
		entity.SetHP(newHP)
		entity.SendMessage("The poison courses through your veins!")
	}

	if affect.Flags&AFFRegeneration != 0 {
		heal := affect.Magnitude
		if heal < 1 {
			heal = 1
		}
		currentHP := entity.GetHP()
		maxHP := entity.GetMaxHP()
		newHP := currentHP + heal
		if newHP > maxHP {
			newHP = maxHP
		}
		entity.SetHP(newHP)
		if heal > 0 {
			entity.SendMessage("You feel your wounds healing.")
		}
	}
}

// clampStats ensures all stat values are within valid ranges
func (am *AffectManager) clampStats(entity Affectable) {
	isNPC := entity.IsNPC()
	maxStat := 18
	if isNPC {
		maxStat = 25
	}

	// DEX, INT, WIS, CON, CHA: clamp 3..maxStat
	for _, s := range []string{"DEX", "INT", "WIS", "CON", "CHA"} {
		v := getStat(entity, s)
		if v < 3 {
			setStatDirect(entity, s, 3)
		} else if v > maxStat {
			setStatDirect(entity, s, maxStat)
		}
	}

	// STR: minimum 3; convert excess over 18 to StrAdd for PCs
	str := getStat(entity, "STR")
	if str < 3 {
		setStatDirect(entity, "STR", 3)
	} else if str > 18 && !isNPC {
		// TODO: StrAdd handling when Player implements it
		setStatDirect(entity, "STR", 18)
	} else if str > maxStat {
		setStatDirect(entity, "STR", maxStat)
	}

	// Alignment: clamp -1000..1000
	// (Not stored via AffectManager, but clamp if accessible)
}

// getStat is a helper that reads a stat by name from an entity
func getStat(entity Affectable, name string) int {
	switch name {
	case "STR":
		return entity.GetStrength()
	case "DEX":
		return entity.GetDexterity()
	case "INT":
		return entity.GetIntelligence()
	case "WIS":
		return entity.GetWisdom()
	case "CON":
		return entity.GetConstitution()
	case "CHA":
		return entity.GetCharisma()
	case "HP":
		return entity.GetHP()
	case "MaxHP":
		return entity.GetMaxHP()
	case "Mana":
		return entity.GetMana()
	case "MaxMana":
		return entity.GetMaxMana()
	case "Move":
		return entity.GetMovement()
	case "Hitroll":
		return entity.GetHitRoll()
	case "Damroll":
		return entity.GetDamageRoll()
	case "AC":
		return entity.GetArmorClass()
	case "THAC0":
		return entity.GetTHAC0()
	}
	return 0
}

// setStatDirect sets a stat by name (bypasses clamping)
func setStatDirect(entity Affectable, name string, val int) {
	switch name {
	case "STR":
		entity.SetStrength(val)
	case "DEX":
		entity.SetDexterity(val)
	case "INT":
		entity.SetIntelligence(val)
	case "WIS":
		entity.SetWisdom(val)
	case "CON":
		entity.SetConstitution(val)
	case "CHA":
		entity.SetCharisma(val)
	case "HP":
		entity.SetHP(val)
	case "MaxHP":
		entity.SetMaxHP(val)
	case "Mana":
		entity.SetMana(val)
	case "MaxMana":
		entity.SetMaxMana(val)
	case "Move":
		entity.SetMovement(val)
	case "Hitroll":
		entity.SetHitRoll(val)
	case "Damroll":
		entity.SetDamageRoll(val)
	case "AC":
		entity.SetArmorClass(val)
	case "THAC0":
		entity.SetTHAC0(val)
	}
}

// addStat adds to a stat by name (via StatModifiable or direct)
func addStat(entity Affectable, name string, delta int) {
	if sm, ok := entity.(interface{ AddStat(string, int) }); ok {
		sm.AddStat(name, delta)
		return
	}
	// Fallback: direct get/set
	val := getStat(entity, name)
	setStatDirect(entity, name, val+delta)
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

func (am *AffectManager) getEntityID(entity Affectable) string {
	return entity.GetName() + "_" + strconv.Itoa(entity.GetID())
}

func (am *AffectManager) removeAffectsByStackID(entityID, stackID string) {
	affects := am.affects[entityID]
	var newAffects []*Affect

	for _, aff := range affects {
		if aff.StackID != stackID {
			newAffects = append(newAffects, aff)
		} else {
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
	// Check flags for status-based descriptions
	if affect.Flags&AFFPoison != 0 {
		return "poisoned"
	}
	if affect.Flags&AFFHaste != 0 {
		return "hasted"
	}
	if affect.Flags&AFFSlow != 0 {
		return "slowed"
	}
	if affect.Flags&AFFRegeneration != 0 {
		return "regenerating"
	}
	if affect.Flags&AFFInvisible != 0 {
		return "invisible"
	}
	if affect.Flags&AFFBlind != 0 {
		return "blind"
	}
	if affect.Flags&AFFSleep != 0 {
		return "sleepy"
	}
	if affect.Flags&AFFStunned != 0 {
		return "stunned"
	}
	if affect.Flags&AFFParalyzed != 0 {
		return "paralyzed"
	}
	if affect.Flags&AFFFlaming != 0 {
		return "flaming"
	}
	if affect.Flags&AFFCharm != 0 {
		return "charmed"
	}
	if affect.Flags&AFFFear != 0 {
		return "fearful"
	}
	if affect.Flags&AFFCurse != 0 {
		return "cursed"
	}
	if affect.Flags&AFFSanctuary != 0 {
		return "sanctuary"
	}
	if affect.Flags&AFFFlying != 0 {
		return "flying"
	}
	if affect.Flags&AFFDetectInvisible != 0 {
		return "detecting invisibility"
	}
	if affect.Flags&AFFDetectMagic != 0 {
		return "detecting magic"
	}
	if affect.Flags&AFFInfrared != 0 {
		return "infravision"
	}
	if affect.Flags&AFFWaterBreathing != 0 {
		return "water breathing"
	}
	if affect.Flags&AFFProtectionEvil != 0 || affect.Flags&AFFProtectionGood != 0 {
		return "protected"
	}
	if affect.Flags&AFFSenseLife != 0 {
		return "sensing life"
	}
	if affect.Flags&AFFMindBar != 0 {
		return "mind barred"
	}
	if affect.Flags&AFFWaterwalk != 0 {
		return "water walking"
	}
	if affect.Flags&AFFMetalskin != 0 {
		return "metalskinned"
	}
	if affect.Flags&AFFInvuln != 0 {
		return "invulnerable"
	}

	// Location-based descriptions for stat affects
	if statName, ok := applyLocationToStat[affect.Location]; ok {
		if affect.Magnitude > 0 {
			return "stronger (" + statName + " enhanced)"
		}
		return "weaker (" + statName + " reduced)"
	}

	if applied {
		return "affected"
	}
	return "effect"
}
