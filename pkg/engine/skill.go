package engine

import (
	"math/rand"
	"time"
)

// SkillType represents the category of a skill
type SkillType int

const (
	SkillTypeCombat SkillType = iota
	SkillTypeMagic
	SkillTypeUtility
)

// Skill represents a learnable ability with progression
type Skill struct {
	Name        string    // Unique identifier for the skill
	DisplayName string    // Display name for players
	Type        SkillType // Combat, Magic, or Utility
	Level       int       // Current proficiency level (0-100)
	Practice    int       // Practice points accumulated (0-100)
	Difficulty  int       // Learning difficulty (1-10, higher = harder)
	MaxLevel    int       // Maximum achievable level (usually 100)
	Learned     bool      // Whether the character has learned this skill
	LastUsed    time.Time // When the skill was last used (for improvement checks)
}

// NewSkill creates a new skill with default values
func NewSkill(name, displayName string, skillType SkillType, difficulty int) *Skill {
	return &Skill{
		Name:        name,
		DisplayName: displayName,
		Type:        skillType,
		Level:       0,
		Practice:    0,
		Difficulty:  difficulty,
		MaxLevel:    100,
		Learned:     false,
		LastUsed:    time.Time{},
	}
}

// CanLearn checks if a skill can be learned based on character stats
func (s *Skill) CanLearn(charLevel, stat int) bool {
	// Base requirement: character level >= difficulty
	if charLevel < s.Difficulty {
		return false
	}

	// Stat requirements based on skill type
	switch s.Type {
	case SkillTypeCombat:
		// Combat skills require strength or dexterity
		return stat >= 10
	case SkillTypeMagic:
		// Magic skills require intelligence or wisdom
		return stat >= 12
	case SkillTypeUtility:
		// Utility skills have lower requirements
		return stat >= 8
	default:
		return true
	}
}

// Learn marks the skill as learned and sets initial level
func (s *Skill) Learn() {
	s.Learned = true
	s.Level = 1 // Start at level 1 when learned
	s.Practice = 0
	s.LastUsed = time.Now()
}

// PracticeSkill attempts to improve the skill through practice
// Returns true if practice was successful, false otherwise
func (s *Skill) PracticeSkill(charLevel, stat int) bool {
	if !s.Learned {
		return false
	}

	// Can't practice beyond max level
	if s.Level >= s.MaxLevel {
		return false
	}

	// Practice points accumulate
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	s.Practice += 10 + rand.Intn(20) // 10-30 practice points

	// Check if we can level up
	if s.Practice >= 100 {
		// Calculate success chance based on difficulty and stats
		successChance := 50 + (stat * 2) - (s.Difficulty * 5) + (charLevel - s.Level)

		if successChance < 10 {
			successChance = 10 // Minimum 10% chance
		}
		if successChance > 90 {
			successChance = 90 // Maximum 90% chance
		}

		// Roll for success
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(100) < successChance {
			s.Level++
			s.Practice = 0
			return true
		} else {
			// Failed practice, lose some points
			s.Practice -= 30
			if s.Practice < 0 {
				s.Practice = 0
			}
		}
	}

	return false
}

// UseSkill attempts to use the skill and may improve it
// Returns success boolean and improvement boolean
func (s *Skill) UseSkill(charLevel, stat int, targetLevel int) (bool, bool) {
	if !s.Learned {
		return false, false
	}

	now := time.Now()
	improved := false

	// Check if we can attempt improvement (once per minute minimum)
	if now.Sub(s.LastUsed) > time.Minute {
		// Small chance to improve on use
		improveChance := 5 + (s.Level / 10) // 5-15% chance
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(100) < improveChance {
			// Gain practice points on successful use
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			s.Practice += 5 + rand.Intn(10)
			improved = true
		}
		s.LastUsed = now
	}

	// Calculate success chance
	baseSuccess := s.Level
	modifier := stat - 10 // Stat bonus/penalty

	// Difficulty modifier based on target
	difficultyMod := 0
	if targetLevel > charLevel {
		difficultyMod = -(targetLevel - charLevel) * 5
	} else if targetLevel < charLevel {
		difficultyMod = (charLevel - targetLevel) * 2
	}

	successChance := baseSuccess + modifier + difficultyMod

	// Ensure reasonable bounds
	if successChance < 5 {
		successChance = 5 // Minimum 5% chance
	}
	if successChance > 95 {
		successChance = 95 // Maximum 95% chance
	}

	// Roll for success
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	success := rand.Intn(100) < successChance

	// If successful and we haven't already improved, check for practice
	if success && !improved && now.Sub(s.LastUsed) > time.Minute {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		s.Practice += 2 + rand.Intn(5)
	}

	return success, improved
}

// GetDisplayLevel returns the skill level as a string for display
func (s *Skill) GetDisplayLevel() string {
	if !s.Learned {
		return "unlearned"
	}

	if s.Level < 25 {
		return "novice"
	} else if s.Level < 50 {
		return "apprentice"
	} else if s.Level < 75 {
		return "journeyman"
	} else if s.Level < 90 {
		return "expert"
	} else if s.Level < 100 {
		return "master"
	} else {
		return "grandmaster"
	}
}

// GetProgress returns practice progress as a percentage
func (s *Skill) GetProgress() int {
	return s.Practice
}

// CanTeach checks if this skill can be taught to another character
func (s *Skill) CanTeach(teacherLevel int) bool {
	// Teacher must be at least 20 levels higher than the skill level
	return teacherLevel >= s.Level+20 && s.Level >= 50
}

