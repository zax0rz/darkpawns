package combat

import (
	"testing"
)

func TestBasicTokenReplace(t *testing.T) {
	got := basicTokenReplace("$n hits $N.", "Alice", "Bob")
	if got != "Alice hits Bob." {
		t.Errorf("basicTokenReplace() = %q, want %q", got, "Alice hits Bob.")
	}

	got = basicTokenReplace("no tokens here", "Alice", "Bob")
	if got != "no tokens here" {
		t.Errorf("basicTokenReplace() = %q, want %q", got, "no tokens here")
	}

	got = basicTokenReplace("$n hits $N, then $n hits $N again.", "A", "B")
	if got != "A hits B, then A hits B again." {
		t.Errorf("basicTokenReplace() = %q, want %q", got, "A hits B, then A hits B again.")
	}
}

func TestBasicTokenReplace_Pronouns(t *testing.T) {
	// Default (GetCharacterSex nil) → male pronouns
	got := basicTokenReplace("$n raises $s blade.", "Warrior", "Enemy")
	if got != "Warrior raises his blade." {
		t.Errorf("male pronouns: got %q, want %q", got, "Warrior raises his blade.")
	}

	// Wire GetCharacterSex
	orig := GetCharacterSex
	defer func() { GetCharacterSex = orig }()
	GetCharacterSex = func(name string) int {
		if name == "Alice" {
			return 1 // female
		}
		if name == "Golem" {
			return 2 // neuter
		}
		return 0 // male
	}

	got = basicTokenReplace("$n raises $s blade.", "Alice", "Enemy")
	if got != "Alice raises her blade." {
		t.Errorf("female pronouns: got %q, want %q", got, "Alice raises her blade.")
	}

	// Neuter (sex=2)
	got = basicTokenReplace("$e attacks $N.", "Golem", "Enemy")
	if got != "it attacks Enemy." {
		t.Errorf("neuter pronouns: got %q, want %q", got, "it attacks Enemy.")
	}
}

func TestSexPronouns(t *testing.T) {
	tests := []struct {
		name string
		sex  int
		want pronounSet
	}{
		{"male", 0, pronounSet{subjective: "he", objective: "him", possessive: "his"}},
		{"female", 1, pronounSet{subjective: "she", objective: "her", possessive: "her"}},
		{"neuter", 2, pronounSet{subjective: "it", objective: "it", possessive: "its"}},
		{"unknown defaults male", 99, pronounSet{subjective: "he", objective: "him", possessive: "his"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sexPronouns(tt.sex)
			if got != tt.want {
				t.Errorf("sexPronouns(%d) = %v, want %v", tt.sex, got, tt.want)
			}
		})
	}
}

func TestInitSkillMessages(t *testing.T) {
	orig := SkillMessageFunc
	defer func() { SkillMessageFunc = orig }()
	SkillMessageFunc = nil

	InitSkillMessages()

	if SkillMessageFunc == nil {
		t.Fatal("InitSkillMessages() did not set SkillMessageFunc")
	}

	// Unknown attack type → no match
	result := SkillMessageFunc(10, "Alice", "Bob", 9999, 100)
	if result != false {
		t.Errorf("SkillMessageFunc for unknown type = %v, want false", result)
	}
}
