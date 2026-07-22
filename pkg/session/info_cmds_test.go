package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestClassName(t *testing.T) {
	tests := []struct {
		class int
		want  string
	}{
		{0, "Mage"},
		{1, "Cleric"},
		{2, "Warrior"},
		{3, "Rogue"},
		{4, "Monk"},
		{5, "Paladin"},
		{6, "Ranger"},
		{7, "Thief"},
		{8, "Bard"},
		{9, "Warlock"},
		{10, "Class 10"},
		{-1, "Class -1"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := className(tt.class); got != tt.want {
				t.Errorf("className(%d) = %q, want %q", tt.class, got, tt.want)
			}
		})
	}
}

func TestPositionName(t *testing.T) {
	tests := []struct {
		pos  int
		want string
	}{
		{combat.PosDead, "Dead"},
		{combat.PosMortally, "Mortally Wounded"},
		{combat.PosIncap, "Incapacitated"},
		{combat.PosStunned, "Stunned"},
		{combat.PosSleeping, "Sleeping"},
		{combat.PosResting, "Resting"},
		{combat.PosSitting, "Sitting"},
		{combat.PosFighting, "Fighting"},
		{combat.PosStanding, "Standing"},
		{99, "Pos 99"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := positionName(tt.pos); got != tt.want {
				t.Errorf("positionName(%d) = %q, want %q", tt.pos, got, tt.want)
			}
		})
	}
}

func TestConditionLabel(t *testing.T) {
	tests := []struct {
		name   string
		hunger int
		thirst int
		drunk  int
		want   string
	}{
		{name: "normal", hunger: 12, thirst: 12, drunk: 0, want: "Normal"},
		{name: "hungry", hunger: 4, thirst: 12, drunk: 0, want: "Hungry"},
		{name: "full", hunger: 24, thirst: 12, drunk: 0, want: "Full"},
		{name: "thirsty", hunger: 12, thirst: 4, drunk: 0, want: "Thirsty"},
		{name: "hydrated", hunger: 12, thirst: 24, drunk: 0, want: "Hydrated"},
		{name: "drunk", hunger: 12, thirst: 12, drunk: 5, want: "Drunk"},
		{name: "hungry+thirsty+drunk", hunger: 4, thirst: 4, drunk: 5, want: "Hungry, Thirsty, Drunk"},
		{name: "full+hydrated", hunger: 24, thirst: 24, drunk: 0, want: "Full, Hydrated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := game.NewPlayer(1, "Tester", 1001)
			p.Hunger = tt.hunger
			p.Thirst = tt.thirst
			p.Drunk = tt.drunk
			if got := conditionLabel(p); got != tt.want {
				t.Errorf("conditionLabel(%+v) = %q, want %q", tt, got, tt.want)
			}
		})
	}
}
