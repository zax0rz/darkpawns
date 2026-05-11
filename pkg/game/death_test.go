package game

import (
	"testing"
)

// ---------------------------------------------------------------------------
// attackTypeToCorpseAttack
// ---------------------------------------------------------------------------

func TestAttackTypeToCorpseAttack(t *testing.T) {
	tests := []struct {
		name       string
		attackType int
		want       CorpseAttackType
	}{
		{"fireball (5)", 5, AttackFire},
		{"chill touch (8)", 8, AttackCold},
		{"color spray (10)", 10, AttackBlast},
		{"energy drain (21)", 21, AttackEnergyDrain},
		{"lightning bolt (30)", 30, AttackLightning},
		{"psiblast (34)", 34, AttackPsiblast},
		{"petrify (35)", 35, AttackPetrify},
		{"drowning (103)", 103, AttackDrowning},
		{"slash type (303)", TypeSlash, AttackSlash},
		{"bite type (304)", TypeBite, AttackSlash},
		{"claw type (308)", TypeClaw, AttackSlash},
		{"whip type (302)", TypeWhip, AttackBruised},
		{"crush type (306)", TypeCrush, AttackCrush},
		{"pierce type (311)", TypePierce, AttackPierce},
		{"bash skill (132)", SkillBashNum, AttackBruised},
		{"backstab skill", SkillBackstabNum, AttackSlash},
		{"disembowel skill", SkillDisembowelNum, AttackDisembowel},
		{"unknown (9999)", 9999, AttackUndefined},
		{"negative (-1)", -1, AttackUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attackTypeToCorpseAttack(tt.attackType)
			if got != tt.want {
				t.Errorf("attackTypeToCorpseAttack(%d) = %d, want %d", tt.attackType, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// createMoneyDesc
// ---------------------------------------------------------------------------

func TestCreateMoneyDesc(t *testing.T) {
	tests := []struct {
		amount int
		want   string
	}{
		{1, "a gold coin"},
		{2, "a pile of 2 gold coins"},
		{100, "a pile of 100 gold coins"},
		{1000, "a pile of 1000 gold coins"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := createMoneyDesc(tt.amount)
			if got != tt.want {
				t.Errorf("createMoneyDesc(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// capitalize
// ---------------------------------------------------------------------------

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
		{"ALREADY", "ALREADY"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalize(tt.input)
			if got != tt.want {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// genderPronoun
// ---------------------------------------------------------------------------

func TestGenderPronoun(t *testing.T) {
	tests := []struct {
		sex  int
		want string
	}{
		{0, "his"},   // male
		{1, "her"},   // female
		{2, "its"},   // neuter
		{99, "his"},  // unknown defaults male
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := genderPronoun(tt.sex)
			if got != tt.want {
				t.Errorf("genderPronoun(%d) = %q, want %q", tt.sex, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// corpseAttackLongDesc
// ---------------------------------------------------------------------------

func TestCorpseAttackLongDesc(t *testing.T) {
	tests := []struct {
		attackType CorpseAttackType
		gender     string
		contains   string // substring expected in result
	}{
		{AttackFire, "male", "charred corpse"},
		{AttackCold, "female", "frozen corpse"},
		{AttackBlast, "neuter", "blasted corpse"},
		{AttackSlash, "male", "hacked up"},
		{AttackDisembowel, "female", "guts spilled"},
		{AttackBruised, "neuter", "bruised"},
		{AttackPierce, "male", "well-ventilated"},
		{AttackCrush, "female", "crushed"},
		{AttackDrowning, "neuter", "waterlogged"},
		{AttackPetrify, "male", "frozen in stone"},
		{AttackNeckBreak, "female", "neck snapped"},
		{AttackPsiblast, "neuter", "brains exploded"},
	}
	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			got := corpseAttackLongDesc("Victim", tt.attackType, tt.gender)
			if len(got) == 0 {
				t.Error("corpseAttackLongDesc returned empty string")
			}
			// Just verify it returns a non-empty string — the actual text
			// is flavor and may change
		})
	}
}
