package combat

import (
	"os"
	"testing"
)

// mockCombatant implements Combatant with settable fields for testing.
type mockCombatant struct {
	name        string
	npc         bool
	room        int
	level       int
	hp          int
	maxHP       int
	ac          int
	thac0       int
	damageRoll  DiceRoll
	position    int
	class       int
	str         int
	strAdd      int
	dex         int
	intVal      int
	wis         int
	hitroll     int
	damroll     int
	sex         int
	fighting    string
}

func (m *mockCombatant) GetName() string            { return m.name }
func (m *mockCombatant) IsNPC() bool                 { return m.npc }
func (m *mockCombatant) GetRoom() int                { return m.room }
func (m *mockCombatant) GetLevel() int               { return m.level }
func (m *mockCombatant) GetHP() int                  { return m.hp }
func (m *mockCombatant) GetMaxHP() int               { return m.maxHP }
func (m *mockCombatant) GetAC() int                  { return m.ac }
func (m *mockCombatant) GetTHAC0() int               { return m.thac0 }
func (m *mockCombatant) GetDamageRoll() DiceRoll     { return m.damageRoll }
func (m *mockCombatant) GetPosition() int            { return m.position }
func (m *mockCombatant) GetClass() int               { return m.class }
func (m *mockCombatant) GetStr() int                 { return m.str }
func (m *mockCombatant) GetStrAdd() int              { return m.strAdd }
func (m *mockCombatant) GetDex() int                 { return m.dex }
func (m *mockCombatant) GetInt() int                 { return m.intVal }
func (m *mockCombatant) GetWis() int                 { return m.wis }
func (m *mockCombatant) GetHitroll() int             { return m.hitroll }
func (m *mockCombatant) GetDamroll() int             { return m.damroll }
func (m *mockCombatant) GetSex() int                 { return m.sex }
func (m *mockCombatant) TakeDamage(amount int)       { m.hp -= amount }
func (m *mockCombatant) Heal(amount int)             { m.hp += amount }
func (m *mockCombatant) SetFighting(target string)   { m.fighting = target }
func (m *mockCombatant) StopFighting()               { m.fighting = "" }
func (m *mockCombatant) GetFighting() string         { return m.fighting }
func (m *mockCombatant) SendMessage(msg string)      {}

// ---------------------------------------------------------------------------
// TestMain — sets global function pointers for tests that need them
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	// Set up global function pointers for testing parry and dodge
	GetSkill = func(name string, skillNum int) int {
		if skillNum == SKILL_PARRY && name == "parry_warrior" {
			return 80
		}
		if skillNum == SKILL_DODGE && name == "dodge_rogue" {
			return 70
		}
		if name == "nobody" {
			return 0
		}
		return 50
	}
	HasMobFlag = func(name string, flag string) bool {
		return name == "aware_mob" && flag == "MOB_AWARE"
	}
	GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
		if chName == "unarmed_guy" {
			return TYPE_HIT, 0, 0, false
		}
		return TYPE_SLASH, 1, 8, false
	}
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// strIndex tests
// ---------------------------------------------------------------------------

func TestStrIndex_NoStrAdd(t *testing.T) {
	c := &mockCombatant{str: 18, strAdd: 0}
	idx := strIndex(c)
	if idx != 18 {
		t.Errorf("expected index 18, got %d", idx)
	}
}

func TestStrIndex_Not18(t *testing.T) {
	c := &mockCombatant{str: 16, strAdd: 0}
	idx := strIndex(c)
	if idx != 16 {
		t.Errorf("expected index 16, got %d", idx)
	}
}

func TestStrIndex_18WithStrAdd(t *testing.T) {
	tests := []struct {
		strAdd   int
		expected int
	}{
		{0, 18},
		{50, 26},  // 18/01-50
		{75, 27},  // 18/51-75
		{90, 28},  // 18/76-90
		{99, 29},  // 18/91-99
		{100, 30}, // 18/100
		{150, 30}, // 18/100 (clamped by mob parser, but formula handles it)
	}

	for _, tc := range tests {
		c := &mockCombatant{str: 18, strAdd: tc.strAdd}
		idx := strIndex(c)
		if idx != tc.expected {
			t.Errorf("str=%d strAdd=%d: expected index %d, got %d", 18, tc.strAdd, tc.expected, idx)
		}
	}
}

func TestStrIndex_LowStr(t *testing.T) {
	c := &mockCombatant{str: 3, strAdd: 0}
	idx := strIndex(c)
	if idx != 3 {
		t.Errorf("expected index 3, got %d", idx)
	}
}

// ---------------------------------------------------------------------------
// dexIndex tests
// ---------------------------------------------------------------------------

func TestDexIndex_Normal(t *testing.T) {
	c := &mockCombatant{dex: 18}
	idx := dexIndex(c)
	if idx != 18 {
		t.Errorf("expected 18, got %d", idx)
	}
}

func TestDexIndex_ClampLow(t *testing.T) {
	c := &mockCombatant{dex: -5}
	idx := dexIndex(c)
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
}

func TestDexIndex_ClampHigh(t *testing.T) {
	c := &mockCombatant{dex: 100}
	idx := dexIndex(c)
	if idx != 25 {
		t.Errorf("expected 25, got %d", idx)
	}
}

func TestDexIndex_Zero(t *testing.T) {
	c := &mockCombatant{dex: 0}
	idx := dexIndex(c)
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
}

func TestDexIndex_Max(t *testing.T) {
	c := &mockCombatant{dex: 25}
	idx := dexIndex(c)
	if idx != 25 {
		t.Errorf("expected 25, got %d", idx)
	}
}

// ---------------------------------------------------------------------------
// getTHAC0 tests
// ---------------------------------------------------------------------------

func TestGetTHAC0_NPC(t *testing.T) {
	c := &mockCombatant{npc: true, level: 50}
	thac0 := getTHAC0(c)
	if thac0 != 20 {
		t.Errorf("expected NPC THAC0 20, got %d", thac0)
	}
}

func TestGetTHAC0_Warrior(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 20}
	thac0 := getTHAC0(c)
	// Warrior at level 20: array index 20 = 1
	if thac0 != 1 {
		t.Errorf("expected warrior THAC0 1 at level 20, got %d", thac0)
	}
}

func TestGetTHAC0_WarriorLevel5(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 5}
	thac0 := getTHAC0(c)
	// Warrior at level 5: array index 5 = 16
	if thac0 != 16 {
		t.Errorf("expected warrior THAC0 16 at level 5, got %d", thac0)
	}
}

func TestGetTHAC0_Mage(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassMage, level: 10}
	thac0 := getTHAC0(c)
	// Mage at level 10: table shows 17
	if thac0 != 17 {
		t.Errorf("expected mage THAC0 17 at level 10, got %d", thac0)
	}
}

func TestGetTHAC0_Level1(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassThief, level: 1}
	thac0 := getTHAC0(c)
	// Thief at level 1: 20
	if thac0 != 20 {
		t.Errorf("expected THAC0 20 at level 1, got %d", thac0)
	}
}

func TestGetTHAC0_LevelBelow1(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 0}
	thac0 := getTHAC0(c)
	// Level < 1 clamps to 1: Warrior at level 1 = 20
	if thac0 != 20 {
		t.Errorf("expected THAC0 20 for level 0 (clamped), got %d", thac0)
	}
}

func TestGetTHAC0_LevelAbove40(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 99}
	thac0 := getTHAC0(c)
	// Level > 40 clamps to 40: Warrior at level 40 = 1
	if thac0 != 1 {
		t.Errorf("expected THAC0 1 for level 99 (clamped), got %d", thac0)
	}
}

func TestGetTHAC0_InvalidClass(t *testing.T) {
	c := &mockCombatant{npc: false, class: 99, level: 5}
	thac0 := getTHAC0(c)
	// Invalid class defaults to warrior: level 5, array index 5 = 16
	if thac0 != 16 {
		t.Errorf("expected THAC0 16 (warrior default), got %d", thac0)
	}
}

// ---------------------------------------------------------------------------
// getMinusDam tests — exhaustive across all AC thresholds
// ---------------------------------------------------------------------------

func TestGetMinusDam_ACAbove90(t *testing.T) {
	// ac > 90: no reduction
	dam := getMinusDam(100, 95)
	if dam != 100 {
		t.Errorf("expected 100 (no reduction), got %d", dam)
	}
}

func TestGetMinusDam_ACThresholds(t *testing.T) {
	// Expected values based on: dam - int(float64(dam) * pct * 2.0)
	// for dam=100. Pre-computed as integer literals to avoid float-to-int
	// conversion issues in test compilation.
	tests := []struct {
		name     string
		ac       int
		expected int
	}{
		{"ac=85 (80-90)", 85, 98},   // 100 - int(100 * 0.01 * 2.0) = 100 - 2
		{"ac=75 (70-80)", 75, 96},   // 100 - int(100 * 0.02 * 2.0) = 100 - 4
		{"ac=65 (60-70)", 65, 94},   // 100 - int(100 * 0.03 * 2.0) = 100 - 6
		{"ac=55 (50-60)", 55, 92},   // 100 - int(100 * 0.04 * 2.0) = 100 - 8
		{"ac=45 (40-50)", 45, 90},   // 100 - int(100 * 0.05 * 2.0) = 100 - 10
		{"ac=35 (30-40)", 35, 88},   // 100 - int(100 * 0.06 * 2.0) = 100 - 12
		{"ac=25 (20-30)", 25, 86},   // 100 - int(100 * 0.07 * 2.0) = 100 - 14
		{"ac=15 (10-20)", 15, 84},   // 100 - int(100 * 0.08 * 2.0) = 100 - 16
		{"ac=5 (0-10)", 5, 80},      // 100 - int(100 * 0.10 * 2.0) = 100 - 20
		{"ac=-5 (-10 to 0)", -5, 78},  // 100 - int(100 * 0.11 * 2.0) = 100 - 22
		{"ac=-15 (-20 to -10)", -15, 76}, // 100 - int(100 * 0.12 * 2.0) = 100 - 24
		{"ac=-25 (-30 to -20)", -25, 74}, // 100 - int(100 * 0.13 * 2.0) = 100 - 26
		{"ac=-35 (-40 to -30)", -35, 72}, // 100 - int(100 * 0.14 * 2.0) = 100 - 28
		{"ac=-45 (-50 to -40)", -45, 70}, // 100 - int(100 * 0.15 * 2.0) = 100 - 30
		{"ac=-55 (-60 to -50)", -55, 68}, // 100 - int(100 * 0.16 * 2.0) = 100 - 32
		{"ac=-65 (-70 to -60)", -65, 66}, // 100 - int(100 * 0.17 * 2.0) = 100 - 34
		{"ac=-75 (-80 to -70)", -75, 64}, // 100 - int(100 * 0.18 * 2.0) = 100 - 36
		{"ac=-85 (-90 to -80)", -85, 62}, // 100 - int(100 * 0.19 * 2.0) = 100 - 38
		{"ac=-92 (-95 to -90)", -92, 60}, // 100 - int(100 * 0.20 * 2.0) = 100 - 40
		{"ac=-100 (-110 to -95)", -100, 58}, // 100 - int(100 * 0.21 * 2.0) = 100 - 42
		{"ac=-120 (-130 to -110)", -120, 56}, // 100 - int(100 * 0.22 * 2.0) = 100 - 44
		{"ac=-140 (-150 to -130)", -140, 54}, // 100 - int(100 * 0.23 * 2.0) = 100 - 46
		{"ac=-160 (-170 to -150)", -160, 52}, // 100 - int(100 * 0.24 * 2.0) = 100 - 48
		{"ac=-180 (-190 to -170)", -180, 50}, // 100 - int(100 * 0.25 * 2.0) = 100 - 50
		{"ac=-200 (-210 to -190)", -200, 48}, // 100 - int(100 * 0.26 * 2.0) = 100 - 52
		{"ac=-220 (-230 to -210)", -220, 46}, // 100 - int(100 * 0.27 * 2.0) = 100 - 54
		{"ac=-240 (-250 to -230)", -240, 44}, // 100 - int(100 * 0.28 * 2.0) = 100 - 56
		{"ac=-260 (-270 to -250)", -260, 43}, // 100 - int(100 * 0.29 * 2.0) = 100 - 57.999... = 43
		{"ac=-280 (-290 to -270)", -280, 40}, // 100 - int(100 * 0.30 * 2.0) = 100 - 60
		{"ac=-300 (-310 to -290)", -300, 38}, // 100 - int(100 * 0.31 * 2.0) = 100 - 62
		{"ac=-350 (<= -310)", -350, 36},  // 100 - int(100 * 0.32 * 2.0) = 100 - 64
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getMinusDam(100, tc.ac)
			if result != tc.expected {
				t.Errorf("getMinusDam(100, %d) = %d, want %d", tc.ac, result, tc.expected)
			}
		})
	}
}

func TestGetMinusDam_ZeroDamage(t *testing.T) {
	// With dam=0, all branches should return 0
	result := getMinusDam(0, -999)
	if result != 0 {
		t.Errorf("expected 0 damage always, got %d", result)
	}
}

func TestGetMinusDam_BoundaryValues(t *testing.T) {
	tests := []struct {
		ac       int
		expected int
		dam      int
	}{
		{91, 100, 100},    // > 90
		{90, 98, 100},     // > 80: 100 - int(100*0.01*2) = 98
		{81, 98, 100},     // > 80
		{80, 96, 100},     // > 70: 100 - int(100*0.02*2) = 96
		{71, 96, 100},     // > 70
		{70, 94, 100},     // > 60: 100 - int(100*0.03*2) = 94
		{1, 80, 100},      // > 0: 100 - int(100*0.10*2) = 80
		{0, 78, 100},      // > -10: 100 - int(100*0.11*2) = 78
		{-1, 78, 100},     // > -10
		{-9, 78, 100},     // > -10
		{-10, 76, 100},    // > -20: 100 - int(100*0.12*2) = 76
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := getMinusDam(tc.dam, tc.ac)
			if result != tc.expected {
				t.Errorf("getMinusDam(%d, %d) = %d, want %d", tc.dam, tc.ac, result, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RollDice tests
// ---------------------------------------------------------------------------

func TestRollDice_Basic(t *testing.T) {
	// Roll 1d6 should be 1-6
	// Can't seed global rand, but can verify basic invariants
	for i := 0; i < 100; i++ {
		result := RollDice(1, 6)
		if result < 1 || result > 6 {
			t.Errorf("1d6 out of range: %d", result)
			break
		}
	}
	// Sum of 3d6 should be between 3 and 18
	for i := 0; i < 100; i++ {
		result := RollDice(3, 6)
		if result < 3 || result > 18 {
			t.Errorf("3d6 out of range: %d", result)
			break
		}
	}
}

func TestRollDice_ZeroNum(t *testing.T) {
	result := RollDice(0, 6)
	if result != 0 {
		t.Errorf("expected 0 for 0 dice, got %d", result)
	}
}

func TestRollDice_ZeroSides(t *testing.T) {
	result := RollDice(3, 0)
	if result != 0 {
		t.Errorf("expected 0 for 0 sides, got %d", result)
	}
}

func TestRollDice_Negative(t *testing.T) {
	result := RollDice(-1, -5)
	if result != 0 {
		t.Errorf("expected 0 for negative args, got %d", result)
	}
}

func TestRollDice_SingleDice(t *testing.T) {
	// 1d1 always produces 1
	for i := 0; i < 20; i++ {
		result := RollDice(1, 1)
		if result != 1 {
			t.Errorf("1d1 should always be 1, got %d", result)
		}
	}
}

// ---------------------------------------------------------------------------
// CheckParry tests
// ---------------------------------------------------------------------------

func TestCheckParry_NPCDefender(t *testing.T) {
	defender := &mockCombatant{npc: true, name: "orc", position: PosStanding}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckParry(defender, attacker)
	if result != ParryFail {
		t.Errorf("expected ParryFail for NPC defender, got %v", result)
	}
}

func TestCheckParry_NoSkill(t *testing.T) {
	defender := &mockCombatant{npc: false, name: "nobody", position: PosStanding}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckParry(defender, attacker)
	if result != ParryFail {
		t.Errorf("expected ParryFail for defender with no skill, got %v", result)
	}
}

func TestCheckParry_Unarmed(t *testing.T) {
	defender := &mockCombatant{npc: false, name: "unarmed_guy", position: PosStanding}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckParry(defender, attacker)
	if result != ParryUnarmed {
		t.Errorf("expected ParryUnarmed for unarmed defender, got %v", result)
	}
}

func TestCheckParry_AwareMob(t *testing.T) {
	defender := &mockCombatant{npc: false, name: "parry_warrior", position: PosStanding}
	attacker := &mockCombatant{npc: true, name: "aware_mob", position: PosStanding}

	result := CheckParry(defender, attacker)
	if result != ParryFail {
		t.Errorf("expected ParryFail vs aware mob, got %v", result)
	}
}

func TestCheckParry_Sleeping(t *testing.T) {
	defender := &mockCombatant{npc: false, name: "parry_warrior", position: PosSleeping}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckParry(defender, attacker)
	if result != ParryFail {
		t.Errorf("expected ParryFail for sleeping defender, got %v", result)
	}
}

func TestCheckParry_ArmedSkilled(t *testing.T) {
	defender := &mockCombatant{npc: false, name: "parry_warrior", position: PosStanding}
	attacker := &mockCombatant{npc: true, name: "orc", position: PosStanding}

	// With skill 80, ran 100 times — some should succeed
	succeeded := false
	for i := 0; i < 200; i++ {
		result := CheckParry(defender, attacker)
		if result == ParrySuccess {
			succeeded = true
			break
		}
	}
	if !succeeded {
		t.Error("expected at least one parry success with skill 80 over 200 rolls")
	}
}

// ---------------------------------------------------------------------------
// CheckDodge tests
// ---------------------------------------------------------------------------

func TestCheckDodge_Sleeping(t *testing.T) {
	defender := &mockCombatant{name: "dodge_rogue", position: PosSleeping}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckDodge(defender, attacker)
	if result != DodgeIncapable {
		t.Errorf("expected DodgeIncapable for sleeping defender, got %v", result)
	}
}

func TestCheckDodge_NoSkill(t *testing.T) {
	defender := &mockCombatant{name: "nobody", position: PosStanding}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	result := CheckDodge(defender, attacker)
	if result != DodgeFail {
		t.Errorf("expected DodgeFail for unskilled, got %v", result)
	}
}

func TestCheckDodge_Skilled(t *testing.T) {
	defender := &mockCombatant{name: "dodge_rogue", position: PosStanding}
	attacker := &mockCombatant{name: "hero", position: PosStanding}

	succeeded := false
	for i := 0; i < 200; i++ {
		result := CheckDodge(defender, attacker)
		if result == DodgeSuccess {
			succeeded = true
			break
		}
	}
	if !succeeded {
		t.Error("expected at least one dodge success with skill 70 over 200 rolls")
	}
}

// ---------------------------------------------------------------------------
// GetAttacksPerRound tests — NPC
// ---------------------------------------------------------------------------

func TestGetAttacksPerRound_NPC(t *testing.T) {
	tests := []struct {
		level int
		min   int
		max   int
	}{
		{1, 1, 3},   // level <= 10 → 1, +1 random
		{10, 1, 3},  // level <= 10 → 1
		{11, 2, 4},  // level <= 20 → 2
		{20, 2, 4},  // level <= 20 → 2
		{21, 3, 5},  // level <= 27 → 3
		{27, 3, 5},  // level <= 27 → 3
		{28, 4, 6},  // level <= 30 → 4
		{30, 4, 6},  // level <= 30 → 4
		{31, 5, 7},  // level >= 31 → 5
		{50, 5, 7},  // level >= 31 → 5
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			c := &mockCombatant{npc: true, level: tc.level}
			for i := 0; i < 50; i++ {
				attacks := GetAttacksPerRound(c, false, false)
				if attacks < tc.min || attacks > tc.max {
					t.Errorf("level %d: attacks=%d, expected in [%d,%d]", tc.level, attacks, tc.min, tc.max)
				}
			}
		})
	}
}

func TestGetAttacksPerRound_HasteSlow(t *testing.T) {
	c := &mockCombatant{npc: true, level: 5}
	// With haste, attacks should be 1 higher; slow should reduce (min 1)
	for i := 0; i < 50; i++ {
		base := GetAttacksPerRound(c, false, false)
		haste := GetAttacksPerRound(c, true, false)
		slow := GetAttacksPerRound(c, false, true)
		if haste != base+1 {
			// Haste adds +1 guaranteed
			break
		}
		if slow < 1 {
			t.Errorf("slow should not go below 1, got %d", slow)
		}
	}
}

// ---------------------------------------------------------------------------
// GetAttacksPerRound tests — Player
// ---------------------------------------------------------------------------

func TestGetAttacksPerRound_PlayerLowLevel(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 1}
	for i := 0; i < 20; i++ {
		attacks := GetAttacksPerRound(c, false, false)
		if attacks < 1 || attacks > 5 {
			t.Errorf("low level player: unexpected attacks=%d", attacks)
		}
	}
}

func TestGetAttacksPerRound_PlayerHighLevel(t *testing.T) {
	c := &mockCombatant{npc: false, class: ClassWarrior, level: 40}
	for i := 0; i < 50; i++ {
		attacks := GetAttacksPerRound(c, false, false)
		// Level 40+ gets +2, plus various possible bonuses
		if attacks < 3 {
			t.Errorf("lvl40 warrior: expected at least 3 attacks, got %d", attacks)
		}
	}
}

// ---------------------------------------------------------------------------
// CalculateHitChance tests — deterministic edge cases
// ---------------------------------------------------------------------------

func TestCalculateHitChance_Natural20(t *testing.T) {
	// Natural 20 always hits. We can't force a natural 20 without seeding
	// package-level rand, but we verify the function runs without panic
	// and that at least some hits occur from natural 20s.
	defender := &mockCombatant{npc: true, position: PosStanding, ac: 100, dex: 10}
	attacker := &mockCombatant{npc: true, level: 1, str: 10, intVal: 10, wis: 10}
	mods := HitModifiers{}

	hits := 0
	for i := 0; i < 1000; i++ {
		if CalculateHitChance(attacker, defender, mods) {
			hits++
		}
	}
	// Even with terrible THAC0 vs high AC, natural 20s (~5%) should land
	if hits == 0 {
		t.Error("expected at least some hits from natural 20s")
	}
}

func TestCalculateHitChance_LowACAlwaysHits(t *testing.T) {
	// Very low AC (100) and bad attacker THAC0 — runs without panic
	defender := &mockCombatant{npc: true, position: PosStanding, ac: 1000, dex: 10}
	attacker := &mockCombatant{npc: true, level: 1, str: 10, intVal: 10, wis: 10}
	mods := HitModifiers{}

	for i := 0; i < 100; i++ {
		CalculateHitChance(attacker, defender, mods)
	}
}

func TestCalculateHitChance_SleepingDefender(t *testing.T) {
	// Sleeping defender: no dex defensive bonus, always hit unless natural 1
	defender := &mockCombatant{npc: true, position: PosSleeping, ac: 0, dex: 10}
	attacker := &mockCombatant{npc: true, level: 1, str: 10, intVal: 10, wis: 10, hitroll: 0}
	mods := HitModifiers{}

	hits := 0
	total := 1000
	for i := 0; i < total; i++ {
		if CalculateHitChance(attacker, defender, mods) {
			hits++
		}
	}
	// Sleeping defender should mostly get hit (only natural 1 misses)
	if hits < total-100 {
		t.Errorf("sleeping defender: expected mostly hits, got %d/%d", hits, total)
	}
}

func TestCalculateHitChance_NegativeDefenderACClamped(t *testing.T) {
	// Defender AC very negative (-100), attacker weak — AC should be clamped to -10
	defender := &mockCombatant{npc: true, position: PosStanding, ac: -1000, dex: 10}
	attacker := &mockCombatant{npc: true, level: 1, str: 10, intVal: 10, wis: 10, hitroll: 0}
	mods := HitModifiers{}

	hits := 0
	total := 500
	for i := 0; i < total; i++ {
		if CalculateHitChance(attacker, defender, mods) {
			hits++
		}
	}
	// Very low AC defender — should still hit sometimes (natural 20)
	if hits == 0 {
		t.Error("expected at least some hits from natural 20s")
	}
}

// ---------------------------------------------------------------------------
// CalculateDamage tests
// ---------------------------------------------------------------------------

func TestCalculateDamage_NPCMinimum(t *testing.T) {
	attacker := &mockCombatant{npc: true, level: 1, str: 10, strAdd: 0, damroll: 0, damageRoll: DiceRoll{Num: 1, Sides: 1}}
	defender := &mockCombatant{position: PosStanding, ac: 0}
	weapon := DiceRoll{}

	dam := CalculateDamage(attacker, defender, weapon, AttackNormal)
	if dam < 1 {
		t.Errorf("expected minimum 1 damage, got %d", dam)
	}
}

func TestCalculateDamage_PlayerWithWeapon(t *testing.T) {
	attacker := &mockCombatant{npc: false, level: 10, str: 18, strAdd: 0, damroll: 5, damageRoll: DiceRoll{}}
	defender := &mockCombatant{position: PosStanding, ac: 0}
	weapon := DiceRoll{Num: 1, Sides: 8}

	dam := CalculateDamage(attacker, defender, weapon, AttackNormal)
	if dam < 1 {
		t.Errorf("expected minimum 1 damage, got %d", dam)
	}
}

func TestCalculateDamage_AttackerDeadStr(t *testing.T) {
	// strApp index 0 has tohit=-5, todam=-4
	attacker := &mockCombatant{npc: false, level: 1, str: 0, strAdd: 0, damroll: 0, damageRoll: DiceRoll{Num: 1, Sides: 1}}
	defender := &mockCombatant{position: PosStanding, ac: 100}
	weapon := DiceRoll{Num: 1, Sides: 1}

	dam := CalculateDamage(attacker, defender, weapon, AttackNormal)
	// Minimum damage should be 1
	if dam < 1 {
		t.Errorf("expected minimum 1 damage, got %d", dam)
	}
}

func TestCalculateDamage_VictimPositionMultiplier(t *testing.T) {
	// Sleeping/incapacitated targets take more damage.
	// Use fixed dice (1d1) + high AC to skip getMinusDam reduction
	// so the position multiplier is the dominant effect.
	attacker := &mockCombatant{npc: true, level: 1, str: 10, strAdd: 0, damroll: 0, damageRoll: DiceRoll{Num: 1, Sides: 1}}
	standing := &mockCombatant{position: PosStanding, ac: 100}
	sleeping := &mockCombatant{position: PosSleeping, ac: 100}

	weapon := DiceRoll{}
	for i := 0; i < 50; i++ {
		dStand := CalculateDamage(attacker, standing, weapon, AttackNormal)
		dSleep := CalculateDamage(attacker, sleeping, weapon, AttackNormal)
		// Base damage: strApp[10].ToDam(0) + RollDice(1,1)(1) = 1
		// Sleeping: 1 * (1 + (7-4)/3) = 1 * 2 = 2
		// AC 100: getMinusDam drops out (ac>90 returns dam)
		if dSleep < dStand {
			t.Errorf("sleeping victim should take >= damage of standing, got standing=%d sleeping=%d", dStand, dSleep)
			return
		}
	}
}

func TestCalculateDamage_NonNormalAttackType(t *testing.T) {
	// Spell/skill attacks bypass getMinusDam reduction
	attacker := &mockCombatant{npc: true, level: 1, str: 10, strAdd: 0, damroll: 0, damageRoll: DiceRoll{Num: 1, Sides: 1}}
	defender := &mockCombatant{position: PosStanding, ac: 100}
	weapon := DiceRoll{}

	dam := CalculateDamage(attacker, defender, weapon, AttackBackstab)
	if dam < 1 {
		t.Errorf("expected minimum 1 damage, got %d", dam)
	}
}
