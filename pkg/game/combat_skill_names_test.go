package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestCombatSkillNameMapsCNumbers(t *testing.T) {
	tests := []struct {
		name     string
		skillNum int
		want     string
	}{
		{"retreat", combat.SKILL_RETREAT, SkillRetreat},
		{"escape", combat.SKILL_ESCAPE, SkillEscape},
		{"parry", combat.SKILL_PARRY, SkillParry},
		{"unknown", 9999, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := combatSkillName(tt.skillNum); got != tt.want {
				t.Fatalf("combatSkillName(%d) = %q, want %q", tt.skillNum, got, tt.want)
			}
		})
	}
}
