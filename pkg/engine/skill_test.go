package engine

import (
	"testing"
	"time"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill("swords", "Swordsmanship", SkillTypeCombat, 3)

	if skill.Name != "swords" {
		t.Errorf("Expected skill name 'swords', got '%s'", skill.Name)
	}

	if skill.DisplayName != "Swordsmanship" {
		t.Errorf("Expected display name 'Swordsmanship', got '%s'", skill.DisplayName)
	}

	if skill.Type != SkillTypeCombat {
		t.Errorf("Expected skill type Combat, got %v", skill.Type)
	}

	if skill.Difficulty != 3 {
		t.Errorf("Expected difficulty 3, got %d", skill.Difficulty)
	}

	if skill.Level != 0 {
		t.Errorf("Expected level 0, got %d", skill.Level)
	}

	if skill.Learned {
		t.Error("Expected skill not learned initially")
	}
}

func TestSkillCanLearn(t *testing.T) {
	tests := []struct {
		name       string
		skillType  SkillType
		difficulty int
		charLevel  int
		stat       int
		expected   bool
	}{
		{
			name:       "Combat skill with sufficient level and stat",
			skillType:  SkillTypeCombat,
			difficulty: 3,
			charLevel:  5,
			stat:       12,
			expected:   true,
		},
		{
			name:       "Combat skill with insufficient level",
			skillType:  SkillTypeCombat,
			difficulty: 5,
			charLevel:  3,
			stat:       15,
			expected:   false,
		},
		{
			name:       "Combat skill with insufficient stat",
			skillType:  SkillTypeCombat,
			difficulty: 3,
			charLevel:  5,
			stat:       8,
			expected:   false,
		},
		{
			name:       "Magic skill with sufficient stats",
			skillType:  SkillTypeMagic,
			difficulty: 4,
			charLevel:  6,
			stat:       14,
			expected:   true,
		},
		{
			name:       "Utility skill with low requirements",
			skillType:  SkillTypeUtility,
			difficulty: 2,
			charLevel:  3,
			stat:       9,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := NewSkill("test", "Test Skill", tt.skillType, tt.difficulty)
			result := skill.CanLearn(tt.charLevel, tt.stat)

			if result != tt.expected {
				t.Errorf("CanLearn() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSkillLearn(t *testing.T) {
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)

	skill.Learn()

	if !skill.Learned {
		t.Error("Expected skill to be learned after Learn()")
	}

	if skill.Level != 1 {
		t.Errorf("Expected level 1 after learning, got %d", skill.Level)
	}

	if skill.Practice != 0 {
		t.Errorf("Expected practice 0 after learning, got %d", skill.Practice)
	}

	// Check that LastUsed is set (should be very recent)
	if time.Since(skill.LastUsed) > time.Second {
		t.Error("Expected LastUsed to be set to recent time")
	}
}

func TestSkillPractice(t *testing.T) {
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	skill.Learn()

	// Can't practice unlearned skill
	unlearnedSkill := NewSkill("unlearned", "Unlearned", SkillTypeCombat, 3)
	if unlearnedSkill.PracticeSkill(5, 12) {
		t.Error("Should not be able to practice unlearned skill")
	}

	// Practice the skill multiple times
	leveledUp := false
	for i := 0; i < 20; i++ {
		if skill.PracticeSkill(5, 15) {
			leveledUp = true
			break
		}
	}

	// Should eventually level up with good stats
	if !leveledUp {
		t.Error("Expected skill to level up with practice")
	}

	if skill.Level < 2 {
		t.Errorf("Expected level >= 2 after leveling up, got %d", skill.Level)
	}
}

func TestSkillUse(t *testing.T) {
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	skill.Learn()
	skill.Level = 50 // Set to a reasonable level for testing

	// Test using the skill
	success, improved := skill.UseSkill(10, 15, 10)

	// Should get some result
	if success && improved {
		t.Log("Skill use succeeded and improved")
	} else if success {
		t.Log("Skill use succeeded")
	} else {
		t.Log("Skill use failed")
	}

	// Can't use unlearned skill
	unlearnedSkill := NewSkill("unlearned", "Unlearned", SkillTypeCombat, 3)
	success, improved = unlearnedSkill.UseSkill(10, 15, 10)
	if success || improved {
		t.Error("Unlearned skill should not be usable")
	}
}

func TestUseSkill_FallbackPracticeWhenEligible(t *testing.T) {
	oldRand := skillRand
	defer func() { skillRand = oldRand }()

	call := 0
	skillRand = func(n int) int {
		call++
		switch call {
		case 1: // improve chance (level 50 => 10%): no improvement
			return 20
		case 2: // success chance (level 50, stat 15 => 55%): success
			return 20
		case 3: // fallback practice points: 2 + 0
			return 0
		default:
			return 0
		}
	}

	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	skill.Learn()
	skill.Level = 50
	skill.LastUsed = time.Now().Add(-2 * time.Minute)
	skill.Practice = 0

	success, improved := skill.UseSkill(10, 15, 10)

	if !success {
		t.Error("expected successful skill use")
	}
	if improved {
		t.Error("expected no improvement")
	}
	if skill.Practice != 2 {
		t.Errorf("expected Practice 2 from fallback practice, got %d", skill.Practice)
	}
}

func TestSkillGetDisplayLevel(t *testing.T) {
	tests := []struct {
		level    int
		learned  bool
		expected string
	}{
		{0, false, "unlearned"},
		{0, true, "novice"},
		{10, true, "novice"},
		{30, true, "apprentice"},
		{55, true, "journeyman"},
		{80, true, "expert"},
		{95, true, "master"},
		{100, true, "grandmaster"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
			skill.Learned = tt.learned
			skill.Level = tt.level

			result := skill.GetDisplayLevel()
			if result != tt.expected {
				t.Errorf("GetDisplayLevel() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestSkillCanTeach(t *testing.T) {
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	skill.Level = 60 // High enough to teach

	// Teacher level too low
	if skill.CanTeach(70) {
		t.Error("Teacher level 70 should not be able to teach level 60 skill")
	}

	// Teacher level sufficient
	if !skill.CanTeach(85) {
		t.Error("Teacher level 85 should be able to teach level 60 skill")
	}

	// Skill level too low to teach
	lowSkill := NewSkill("low", "Low Skill", SkillTypeCombat, 3)
	lowSkill.Level = 40
	lowSkill.Learn()

	if lowSkill.CanTeach(100) {
		t.Error("Level 40 skill should not be teachable regardless of teacher level")
	}
}

func TestNewSkillManager(t *testing.T) {
	sm := NewSkillManager()

	if sm == nil {
		t.Fatal("NewSkillManager() returned nil")
	}

	if sm.GetSlots() != 10 {
		t.Errorf("Expected default slots 10, got %d", sm.GetSlots())
	}

	if sm.GetSkillPoints() != 0 {
		t.Errorf("Expected initial skill points 0, got %d", sm.GetSkillPoints())
	}
}

func TestSkillManagerGetAllSkills(t *testing.T) {
	sm := NewSkillManager()
	sm.InitializeDefaultSkills()

	allSkills := sm.GetAllSkills()

	if len(allSkills) == 0 {
		t.Error("Expected some default skills")
	}

	// Check that skills are sorted by name
	for i := 1; i < len(allSkills); i++ {
		if allSkills[i-1].Name > allSkills[i].Name {
			t.Error("Skills should be sorted by name")
		}
	}
}
