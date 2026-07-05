package combat

import (
	"testing"
)

// TestNPCGetAttacksPerRound_Deterministic verifies NPC attack scheduler deterministic base counts and threshold transitions.
func TestNPCGetAttacksPerRound_Deterministic(t *testing.T) {
	// Scripted roller returning 901 to ensure the random bonus (number(0, 900) < level) always fails.
	roller := NewScriptedRoller([]int{901})

	tests := []struct {
		level int
		want  int
	}{
		{level: 5, want: 1},
		{level: 10, want: 1},
		{level: 11, want: 2},
		{level: 15, want: 2},
		{level: 20, want: 2},
		{level: 21, want: 3},
		{level: 27, want: 3},
		{level: 28, want: 4},
		{level: 30, want: 4},
		{level: 31, want: 5},
		{level: 50, want: 5},
	}

	WithRoller(roller, func() {
		for _, tt := range tests {
			c := &mockCombatant{npc: true, level: tt.level}
			got := GetAttacksPerRound(c, false, false)
			if got != tt.want {
				t.Errorf("NPC Level %d attacks = %d; want %d", tt.level, got, tt.want)
			}
		}
	})
}

// TestNPCGetAttacksPerRound_RandomBonus verifies NPC attack count gets +1 when the random check succeeds.
func TestNPCGetAttacksPerRound_RandomBonus(t *testing.T) {
	// Scripted roller returning 0 (which is < level) to ensure the random bonus always succeeds.
	roller := NewScriptedRoller([]int{0})

	WithRoller(roller, func() {
		c := &mockCombatant{npc: true, level: 10}
		got := GetAttacksPerRound(c, false, false)
		if got != 2 { // 1 base + 1 bonus
			t.Errorf("NPC Level 10 with successful bonus got %d attacks; want 2", got)
		}
	})
}

// TestPCGetAttacksPerRound_Deterministic verifies PC attack count logic using a scripted roller.
func TestPCGetAttacksPerRound_Deterministic(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		class    int
		hasHaste bool
		hasSlow  bool
		rolls    []int
		want     int
	}{
		{
			name: "Warrior Level 11, passes class check",
			level: 11, class: ClassWarrior,
			rolls: []int{70, 5}, // 70 < 60+11 (passes warrior check), 5 != 0 (fails level > 30 check)
			want: 2,
		},
		{
			name: "Warrior Level 11, fails class check",
			level: 11, class: ClassWarrior,
			rolls: []int{72, 5}, // 72 >= 71 (fails warrior check), 5 != 0
			want: 1,
		},
		{
			name: "Avatar Level 13, passes class check",
			level: 13, class: ClassAvatar,
			rolls: []int{70, 5}, // 70 < 60+13 (passes class check), 5 != 0
			want: 2,
		},
		{
			name: "Thief Level 16, passes class check",
			level: 16, class: ClassThief,
			rolls: []int{45, 5}, // 45 < 30+16 (passes class check), 5 != 0
			want: 2,
		},
		{
			name: "Warrior Level 40 (high level), haste active",
			level: 40, class: ClassWarrior, hasHaste: true,
			rolls: []int{101, 101}, // fails level 10 and 25 check (level 30 short-circuited)
			want: 5, // 1 base + 1 (lvl > 30) + 2 (lvl > 39) + 1 (haste) = 5
		},
		{
			name: "Mage Level 30, slow active, passes level > 25 and level 30 checks",
			level: 30, class: ClassMage, hasSlow: true,
			rolls: []int{50, 0}, // 50 < 75 (passes >25), 0 == 0 (passes 30 check)
			want: 2, // 1 base + 1 (>25) + 1 (0==0 check) - 1 (slow) = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roller := NewScriptedRoller(tt.rolls)
			WithRoller(roller, func() {
				c := &mockCombatant{npc: false, level: tt.level, class: tt.class}
				got := GetAttacksPerRound(c, tt.hasHaste, tt.hasSlow)
				if got != tt.want {
					t.Errorf("got %d attacks; want %d", got, tt.want)
				}
			})
		})
	}
}

// TestPCGetAttacksPerRound_Statistical verifies PC attack round probability distribution over 10,000 runs.
func TestPCGetAttacksPerRound_Statistical(t *testing.T) {
	// Seeded roller for deterministic statistical tests
	roller := NewSeededRoller(12345, 67890)

	// Test: Warrior level 20.
	// Expected:
	// - Base: 1
	// - Warrior bonus level > 10: (60 + 20) = 80% chance (+0.8 expected)
	// - Level > 25: 0%
	// - Level > 30: 0%
	// - number(0, 500) == 0: 1/501 chance (+0.002 expected)
	// Total expected: 1.802
	var totalAttacks int
	const iterations = 10000

	c := &mockCombatant{npc: false, level: 20, class: ClassWarrior}
	WithRoller(roller, func() {
		for i := 0; i < iterations; i++ {
			totalAttacks += GetAttacksPerRound(c, false, false)
		}
	})

	mean := float64(totalAttacks) / float64(iterations)
	expected := 1.802

	if mean < expected-0.02 || mean > expected+0.02 {
		t.Errorf("Warrior Level 20 mean attacks = %f; expected ~%f (+/- 0.02)", mean, expected)
	}
}
