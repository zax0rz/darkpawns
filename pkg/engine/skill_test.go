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

func TestSkillManagerLearnSkill(t *testing.T) {
	sm := NewSkillManager()
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	
	// Can't learn without skill points
	if sm.LearnSkill(skill, 5, 12) {
		t.Error("Should not be able to learn skill without points")
	}
	
	// Add skill points
	sm.AddSkillPoints(5)
	
	// Now should be able to learn
	if !sm.LearnSkill(skill, 5, 12) {
		t.Error("Should be able to learn skill with points and requirements")
	}
	
	// Can't learn same skill twice
	if sm.LearnSkill(skill, 5, 12) {
		t.Error("Should not be able to learn same skill twice")
	}
	
	// Check that skill points were deducted
	if sm.GetSkillPoints() != 2 { // 5 - 3 difficulty
		t.Errorf("Expected 2 skill points remaining, got %d", sm.GetSkillPoints())
	}
}

func TestSkillManagerPracticeSkill(t *testing.T) {
	sm := NewSkillManager()
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	
	sm.AddSkillPoints(5)
	sm.LearnSkill(skill, 5, 12)
	
	// Practice the skill
	leveledUp := sm.PracticeSkill("test", 5, 15)
	
	// Might or might not level up depending on RNG
	if leveledUp {
		t.Log("Skill leveled up with practice")
	} else {
		t.Log("Skill practiced but didn't level up")
	}
	
	// Can't practice non-existent skill
	if sm.PracticeSkill("nonexistent", 5, 15) {
		t.Error("Should not be able to practice non-existent skill")
	}
}

func TestSkillManagerUseSkill(t *testing.T) {
	sm := NewSkillManager()
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	
	sm.AddSkillPoints(5)
	sm.LearnSkill(skill, 5, 12)
	
	// Use the skill
	success, improved := sm.UseSkill("test", 5, 15, 5)
	
	// Should get some result
	if success || improved {
		t.Log("Skill use returned:", success, improved)
	}
	
	// Can't use non-existent skill
	success, improved = sm.UseSkill("nonexistent", 5, 15, 5)
	if success || improved {
		t.Error("Should not be able to use non-existent skill")
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

func TestSkillManagerGetLearnedSkills(t *testing.T) {
	sm := NewSkillManager()
	sm.InitializeDefaultSkills()
	
	// Initially no learned skills
	learned := sm.GetLearnedSkills()
	if len(learned) != 0 {
		t.Errorf("Expected 0 learned skills initially, got %d", len(learned))
	}
	
	// Learn a skill
	sm.AddSkillPoints(10)
	swords := sm.GetSkill("swords")
	if swords == nil {
		t.Fatal("Expected to find swords skill")
	}
	
	if !sm.LearnSkill(swords, 5, 12) {
		t.Error("Failed to learn swords skill")
	}
	
	// Now should have one learned skill
	learned = sm.GetLearnedSkills()
	if len(learned) != 1 {
		t.Errorf("Expected 1 learned skill, got %d", len(learned))
	}
	
	if learned[0].Name != "swords" {
		t.Errorf("Expected learned skill to be 'swords', got '%s'", learned[0].Name)
	}
}

func TestSkillManagerForgetSkill(t *testing.T) {
	sm := NewSkillManager()
	skill := NewSkill("test", "Test Skill", SkillTypeCombat, 3)
	
	sm.AddSkillPoints(5)
	sm.LearnSkill(skill, 5, 12)
	
	initialPoints := sm.GetSkillPoints()
	
	// Forget the skill
	if !sm.ForgetSkill("test") {
		t.Error("Failed to forget skill")
	}
	
	// Should have refunded some points
	finalPoints := sm.GetSkillPoints()
	expectedRefund := (3 + 1) / 2 // Half of difficulty, rounded up = 2
	expectedPoints := initialPoints + expectedRefund
	
	if finalPoints != expectedPoints {
		t.Errorf("Expected %d skill points after forgetting, got %d", expectedPoints, finalPoints)
	}
	
	// Skill should no longer be learned
	if sm.HasSkill("test") {
		t.Error("Skill should not be learned after forgetting")
	}
	
	// Can't forget non-existent skill
	if sm.ForgetSkill("nonexistent") {
		t.Error("Should not be able to forget non-existent skill")
	}
}

func TestSkillManagerTeachSkill(t *testing.T) {
	teacher := NewSkillManager()
	student := NewSkillManager()
	
	// Create a teachable skill
	skill := NewSkill("teachable", "Teachable Skill", SkillTypeCombat, 3)
	skill.Level = 60 // High enough to teach
	
	teacher.AddSkillPoints(10)
	teacher.LearnSkill(skill, 20, 15) // Teacher learns at high level
	
	student.AddSkillPoints(10)
	
	// Teacher should be able to teach student
	if !teacher.TeachSkill("teachable", student, 85, 5, 12) {
		t.Error("Teacher should be able to teach skill")
	}
	
	// Student should now have the skill
	if !student.HasSkill("teachable") {
		t.Error("Student should have learned the skill")
	}
	
	// Student's skill should start at level 1
	studentSkill := student.GetSkill("teachable")
	if studentSkill == nil || studentSkill.Level != 1 {
		t.Error("Student's skill should start at level 1")
	}
}

func TestSkillManagerSlots(t *testing.T) {
	sm := NewSkillManager()
	
	// Default slots
	if sm.GetSlots() != 10 {
		t.Errorf("Expected 10 slots, got %d", sm.GetSlots())
	}
	
	if sm.GetUsedSlots() != 0 {
		t.Errorf("Expected 0 used slots, got %d", sm.GetUsedSlots())
	}
	
	if sm.GetAvailableSlots() != 10 {
		t.Errorf("Expected 10 available slots, got %d", sm.GetAvailableSlots())
	}
	
	// Learn some skills
	sm.AddSkillPoints(20)
	
	skill1 := NewSkill("skill1", "Skill 1", SkillTypeCombat, 3)
	skill2 := NewSkill("skill2", "Skill 2", SkillTypeCombat, 3)
	
	sm.LearnSkill(skill1, 5, 12)
	sm.LearnSkill(skill2, 5, 12)
	
	if sm.GetUsedSlots() != 2 {
		t.Errorf("Expected 2 used slots, got %d", sm.GetUsedSlots())
	}
	
	if sm.GetAvailableSlots() != 8 {
		t.Errorf("Expected 8 available slots, got %d", sm.GetAvailableSlots())
	}
	
	// Increase slots
	sm.IncreaseSlots(5)
	
	if sm.GetSlots() != 15 {
		t.Errorf("Expected 15 slots after increase, got %d", sm.GetSlots())
	}
	
	if sm.GetAvailableSlots() != 13 {
		t.Errorf("Expected 13 available slots after increase, got %d", sm.GetAvailableSlots())
	}
}