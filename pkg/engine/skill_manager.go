package engine

import (
	"sort"
	"sync"
)

// SkillManager manages all skills for a character
type SkillManager struct {
	mu     sync.RWMutex
	skills map[string]*Skill // name -> skill
	slots  int               // Maximum number of skill slots
	points int               // Available skill points
}

// NewSkillManager creates a new skill manager with default slots
func NewSkillManager() *SkillManager {
	return &SkillManager{
		skills: make(map[string]*Skill),
		slots:  10, // Default skill slots
		points: 0,
	}
}

// GetSkill returns a skill by name
func (sm *SkillManager) GetSkill(name string) *Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.skills[name]
}

// HasSkill checks if a skill is known
func (sm *SkillManager) HasSkill(name string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	skill, exists := sm.skills[name]
	return exists && skill.Learned
}

// LearnSkill attempts to learn a new skill
func (sm *SkillManager) LearnSkill(skill *Skill, charLevel, stat int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if already learned
	if existing, exists := sm.skills[skill.Name]; exists && existing.Learned {
		return false
	}

	// Check requirements
	if !skill.CanLearn(charLevel, stat) {
		return false
	}

	// Check if we have available slots
	if len(sm.getLearnedSkills()) >= sm.slots {
		return false
	}

	// Check if we have enough skill points
	if sm.points < skill.Difficulty {
		return false
	}

	// Learn the skill
	skill.Learn()
	sm.skills[skill.Name] = skill
	sm.points -= skill.Difficulty

	return true
}

// PracticeSkill attempts to practice a skill
func (sm *SkillManager) PracticeSkill(name string, charLevel, stat int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[name]
	if !exists || !skill.Learned {
		return false
	}

	return skill.PracticeSkill(charLevel, stat)
}

// UseSkill attempts to use a skill
func (sm *SkillManager) UseSkill(name string, charLevel, stat, targetLevel int) (bool, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[name]
	if !exists || !skill.Learned {
		return false, false
	}

	return skill.UseSkill(charLevel, stat, targetLevel)
}

// AddSkillPoints adds skill points to the manager
func (sm *SkillManager) AddSkillPoints(points int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.points += points
}

// GetSkillPoints returns available skill points
func (sm *SkillManager) GetSkillPoints() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.points
}

// GetSlots returns total skill slots
func (sm *SkillManager) GetSlots() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.slots
}

// GetUsedSlots returns number of used skill slots
func (sm *SkillManager) GetUsedSlots() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.getLearnedSkills())
}

// GetAvailableSlots returns number of available skill slots
func (sm *SkillManager) GetAvailableSlots() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.slots - len(sm.getLearnedSkills())
}

// IncreaseSlots increases the number of skill slots
func (sm *SkillManager) IncreaseSlots(additional int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.slots += additional
}

// GetAllSkills returns all skills (learned and unlearned)
func (sm *SkillManager) GetAllSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var skills []*Skill
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}

	// Sort by name
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}

// GetLearnedSkills returns only learned skills
func (sm *SkillManager) GetLearnedSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.getLearnedSkills()
}

// getLearnedSkills is the internal implementation without locking
func (sm *SkillManager) getLearnedSkills() []*Skill {
	var learned []*Skill
	for _, skill := range sm.skills {
		if skill.Learned {
			learned = append(learned, skill)
		}
	}

	// Sort by level (descending), then by name
	sort.Slice(learned, func(i, j int) bool {
		if learned[i].Level != learned[j].Level {
			return learned[i].Level > learned[j].Level
		}
		return learned[i].Name < learned[j].Name
	})

	return learned
}

// GetSkillLevel returns the level of a skill (0 if not learned)
func (sm *SkillManager) GetSkillLevel(name string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[name]
	if !exists || !skill.Learned {
		return 0
	}

	return skill.Level
}

// RegisterSkill adds a skill to the manager without learning it
func (sm *SkillManager) RegisterSkill(skill *Skill) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Don't overwrite if already exists with higher level
	if existing, exists := sm.skills[skill.Name]; exists {
		if existing.Learned && existing.Level > skill.Level {
			return
		}
	}

	sm.skills[skill.Name] = skill
}

// ForgetSkill removes a learned skill and refunds some points
func (sm *SkillManager) ForgetSkill(name string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[name]
	if !exists || !skill.Learned {
		return false
	}

	// Refund half the difficulty points (rounded up)
	refund := (skill.Difficulty + 1) / 2
	sm.points += refund

	// Mark as unlearned but keep in registry
	skill.Learned = false
	skill.Level = 0
	skill.Practice = 0

	return true
}

// TeachSkill attempts to teach a skill to another skill manager
func (sm *SkillManager) TeachSkill(skillName string, target *SkillManager, teacherLevel, studentLevel, studentStat int) bool {
	sm.mu.RLock()

	// Get the skill from teacher
	skill, exists := sm.skills[skillName]
	if !exists || !skill.Learned {
		sm.mu.RUnlock()
		return false
	}

	// Check if teacher can teach this skill
	if !skill.CanTeach(teacherLevel) {
		sm.mu.RUnlock()
		return false
	}

	// Create a copy of the skill for teaching (starts at lower level)
	taughtSkill := *skill
	taughtSkill.Level = 1 // Start at level 1 when taught
	taughtSkill.Practice = 0
	taughtSkill.Learned = false // Will be set by LearnSkill

	sm.mu.RUnlock()

	// Try to learn the skill (no locks held, LearnSkill will lock target)
	return target.LearnSkill(&taughtSkill, studentLevel, studentStat)
}

// InitializeDefaultSkills registers common MUD skills
func (sm *SkillManager) InitializeDefaultSkills() {
	// Combat skills
	sm.RegisterSkill(NewSkill("swords", "Swordsmanship", SkillTypeCombat, 3))
	sm.RegisterSkill(NewSkill("axes", "Axe Fighting", SkillTypeCombat, 4))
	sm.RegisterSkill(NewSkill("maces", "Mace Fighting", SkillTypeCombat, 3))
	sm.RegisterSkill(NewSkill("daggers", "Dagger Fighting", SkillTypeCombat, 2))
	sm.RegisterSkill(NewSkill("polearms", "Polearm Fighting", SkillTypeCombat, 5))
	sm.RegisterSkill(NewSkill("archery", "Archery", SkillTypeCombat, 4))
	sm.RegisterSkill(NewSkill("unarmed", "Unarmed Combat", SkillTypeCombat, 1))
	sm.RegisterSkill(NewSkill("shield", "Shield Use", SkillTypeCombat, 2))
	sm.RegisterSkill(NewSkill("parry", "Parrying", SkillTypeCombat, 3))
	sm.RegisterSkill(NewSkill("dodge", "Dodging", SkillTypeCombat, 3))

	// Magic skills
	sm.RegisterSkill(NewSkill("evocation", "Evocation Magic", SkillTypeMagic, 6))
	sm.RegisterSkill(NewSkill("abjuration", "Abjuration Magic", SkillTypeMagic, 5))
	sm.RegisterSkill(NewSkill("conjuration", "Conjuration Magic", SkillTypeMagic, 7))
	sm.RegisterSkill(NewSkill("divination", "Divination Magic", SkillTypeMagic, 4))
	sm.RegisterSkill(NewSkill("enchantment", "Enchantment Magic", SkillTypeMagic, 6))
	sm.RegisterSkill(NewSkill("illusion", "Illusion Magic", SkillTypeMagic, 5))
	sm.RegisterSkill(NewSkill("necromancy", "Necromancy", SkillTypeMagic, 8))
	sm.RegisterSkill(NewSkill("transmutation", "Transmutation Magic", SkillTypeMagic, 7))

	// Utility skills
	sm.RegisterSkill(NewSkill("stealth", "Stealth", SkillTypeUtility, 3))
	sm.RegisterSkill(NewSkill("lockpick", "Lock Picking", SkillTypeUtility, 4))
	sm.RegisterSkill(NewSkill("disarm", "Trap Disarming", SkillTypeUtility, 5))
	sm.RegisterSkill(NewSkill("search", "Searching", SkillTypeUtility, 2))
	sm.RegisterSkill(NewSkill("track", "Tracking", SkillTypeUtility, 3))
	sm.RegisterSkill(NewSkill("heal", "Healing", SkillTypeUtility, 4))
	sm.RegisterSkill(NewSkill("craft", "Crafting", SkillTypeUtility, 3))
	sm.RegisterSkill(NewSkill("appraise", "Appraisal", SkillTypeUtility, 2))
	sm.RegisterSkill(NewSkill("diplomacy", "Diplomacy", SkillTypeUtility, 4))
	sm.RegisterSkill(NewSkill("intimidate", "Intimidation", SkillTypeUtility, 3))
}
