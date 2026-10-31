package combat

import (
	"testing"
)

// sexPronouns is the pronoun table replaceMessageTokens uses. The token and
// #w/#W substitutions themselves are covered by TestReplaceMessageTokens in
// fight_core_test.go, which exercises the same production function.
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
