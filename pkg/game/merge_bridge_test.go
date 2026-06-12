package game

import (
	"strings"
	"testing"
)

func TestValidNameOnlineDuplicate(t *testing.T) {
	// Backup original callback
	orig := HasActiveCharacter
	defer func() { HasActiveCharacter = orig }()

	// Wire up callback that returns true for "Aidan" (case-insensitive)
	HasActiveCharacter = func(name string) bool {
		return strings.EqualFold(name, "Aidan")
	}

	// Test case-sensitive match
	if got := ValidName("Aidan"); got != false {
		t.Errorf("ValidName(Aidan) = %v, want false", got)
	}

	// Test case-insensitive match
	if got := ValidName("aidan"); got != false {
		t.Errorf("ValidName(aidan) = %v, want false", got)
	}

	// Test different name
	if got := ValidName("Other"); got != true {
		t.Errorf("ValidName(Other) = %v, want true", got)
	}
}

func TestValidNameNilCallback(t *testing.T) {
	// Backup original callback
	orig := HasActiveCharacter
	defer func() { HasActiveCharacter = orig }()

	// Set to nil
	HasActiveCharacter = nil

	// Test validation does not panic and passes
	if got := ValidName("Test"); got != true {
		t.Errorf("ValidName(Test) = %v, want true", got)
	}
}
