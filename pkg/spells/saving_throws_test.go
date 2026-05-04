package spells

import (
	"testing"
)

// mockChar implements baseClass for testing CheckSavingThrow.
type mockChar struct {
	class int
	level int
}

func (m mockChar) GetClass() int { return m.class }
func (m mockChar) GetLevel() int { return m.level }

func TestGetSavingThrow_WarriorLevel1Para(t *testing.T) {
	got := GetSavingThrow(3, 1, SaveParalysis)
	expected := 70
	if got != expected {
		t.Errorf("Warrior(class=3) level=1 PARA = %d, want %d", got, expected)
	}
}

func TestGetSavingThrow_WarriorLevel30Para(t *testing.T) {
	got := GetSavingThrow(3, 30, SaveParalysis)
	expected := 12
	if got != expected {
		t.Errorf("Warrior(class=3) level=30 PARA = %d, want %d", got, expected)
	}
}

func TestGetSavingThrow_MageLevel1Spell(t *testing.T) {
	got := GetSavingThrow(0, 1, SaveSpell)
	expected := 60
	if got != expected {
		t.Errorf("Mage(class=0) level=1 SPELL = %d, want %d", got, expected)
	}
}

func TestGetSavingThrow_MageLevel30Spell(t *testing.T) {
	got := GetSavingThrow(0, 30, SaveSpell)
	expected := 8
	if got != expected {
		t.Errorf("Mage(class=0) level=30 SPELL = %d, want %d", got, expected)
	}
}

func TestGetSavingThrow_Level0Returns90(t *testing.T) {
	got := GetSavingThrow(3, 0, SaveParalysis)
	expected := 90
	if got != expected {
		t.Errorf("Any class level=0 PARA = %d, want 90", got)
	}
}

func TestGetSavingThrow_Level30PlusWarriorPara(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{31, 11},
		{35, 7},
		{40, 2},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetSavingThrow(3, tt.level, SaveParalysis)
			if got != tt.want {
				t.Errorf("Warrior(3) level=%d PARA = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestGetSavingThrow_OutOfBoundsClass(t *testing.T) {
	// Negative class should not panic, defaults to class 0
	got := GetSavingThrow(-1, 1, SaveSpell)
	expected := 60 // Same as mage class 0, level 1, spell
	if got != expected {
		t.Errorf("class=-1 level=1 SPELL = %d, want %d (defaults to class 0)", got, expected)
	}
}

func TestGetSavingThrow_OutOfBoundsClassHigh(t *testing.T) {
	// Class > 11 should not panic
	got := GetSavingThrow(99, 1, SaveParalysis)
	expected := 70 // defaults to class 0, level 1 PARA
	if got != expected {
		t.Errorf("class=99 level=1 PARA = %d, want %d", got, expected)
	}
}

func TestGetSavingThrow_NegativeLevel(t *testing.T) {
	// Negative level should not panic, clamped to 0
	got := GetSavingThrow(3, -5, SaveParalysis)
	if got != 90 {
		t.Errorf("level=-5 PARA = %d, want 90 (clamped to 0)", got)
	}
}

func TestGetSavingThrow_HighLevel(t *testing.T) {
	// Level > 40 should not panic, clamped to maxLevelIndex-1 (40)
	got := GetSavingThrow(3, 99, SaveParalysis)
	if got != 2 {
		t.Errorf("level=99 PARA = %d, want 2 (clamped to index 40)", got)
	}
}

func TestGetSavingThrow_OutOfBoundsSaveType(t *testing.T) {
	// Negative saveType should not panic, defaults to SaveSpell(4)
	got := GetSavingThrow(0, 1, SavingThrowType(-1))
	expected := 60
	if got != expected {
		t.Errorf("saveType=-1 = %d, want %d (defaults to SPELL)", got, expected)
	}
}

func TestGetSavingThrow_SaveTypeOutOfRange(t *testing.T) {
	// SaveType >= SaveCount should default to SaveSpell
	got := GetSavingThrow(0, 1, SaveCount) // SaveCount = 5
	expected := 60
	if got != expected {
		t.Errorf("saveType=SaveCount = %d, want %d (defaults to SPELL)", got, expected)
	}
}

func TestDice_Basic(t *testing.T) {
	for i := 0; i < 50; i++ {
		got := Dice(1, 6)
		if got < 1 || got > 6 {
			t.Errorf("Dice(1,6) = %d, want 1-6", got)
		}
	}
}

func TestDice_MultipleDice(t *testing.T) {
	for i := 0; i < 50; i++ {
		got := Dice(3, 6)
		if got < 3 || got > 18 {
			t.Errorf("Dice(3,6) = %d, want 3-18", got)
		}
	}
}

func TestDice_ZeroOrNegative(t *testing.T) {
	tests := []struct {
		num, sides int
	}{
		{0, 6},
		{2, 0},
		{-1, 6},
		{2, -1},
		{0, 0},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := Dice(tt.num, tt.sides); got != 0 {
				t.Errorf("Dice(%d,%d) = %d, want 0", tt.num, tt.sides, got)
			}
		})
	}
}

func TestCheckSavingThrow_DoesNotPanic(t *testing.T) {
	tests := []struct {
		name     string
		ch       interface{}
		saveType SavingThrowType
	}{
		{"mock warrior", mockChar{class: 3, level: 10}, SaveParalysis},
		{"mock mage", mockChar{class: 0, level: 5}, SaveSpell},
		{"mock high level", mockChar{class: 3, level: 40}, SaveParalysis},
		{"mock level 0", mockChar{class: 3, level: 0}, SaveParalysis},
		{"mock negative class", mockChar{class: -1, level: 1}, SaveParalysis},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckSavingThrow(tt.ch, tt.saveType)
			// Should return a bool without panicking
			if result != true && result != false {
				t.Errorf("CheckSavingThrow returned non-bool: %v", result)
			}
		})
	}
}

func TestCheckSavingThrow_UnsupportedTypeReturnsFalse(t *testing.T) {
	// Pass an int (not implementing baseClass) — should return false
	result := CheckSavingThrow(42, SaveParalysis)
	if result != false {
		t.Errorf("CheckSavingThrow(int) = %v, want false", result)
	}
}

func TestCheckSavingThrow_HighLevelSavesOften(t *testing.T) {
	// A level 40 warrior has PARA=2, so save<roll means 2<roll, saving ~97% of the time.
	// Run 100 iterations — expect at least 90 saves.
	c := mockChar{class: 3, level: 40}
	saves := 0
	iterations := 100
	for i := 0; i < iterations; i++ {
		if CheckSavingThrow(c, SaveParalysis) {
			saves++
		}
	}
	if saves < 80 {
		t.Errorf("Warrior(3) level=40 PARA saved %d/%d times, expected >= 80", saves, iterations)
	}
}

func TestCheckSavingThrow_Level0SavesRarely(t *testing.T) {
	// A level 0 warrior has PARA=90, so save<roll means 90<roll, saving ~9% of the time.
	// Run 100 iterations — expect fewer than 30 saves.
	c := mockChar{class: 3, level: 0}
	saves := 0
	iterations := 100
	for i := 0; i < iterations; i++ {
		if CheckSavingThrow(c, SaveParalysis) {
			saves++
		}
	}
	if saves > 30 {
		t.Errorf("Warrior(3) level=0 PARA saved %d/%d times, expected <= 30", saves, iterations)
	}
}
