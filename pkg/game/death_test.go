package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
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
		{2, "a tiny pile of gold coins"},
		{100, "a small pile of gold coins"},
		{1000, "a pile of gold coins"},
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
		{0, "his"},  // male
		{1, "her"},  // female
		{2, "its"},  // neuter
		{99, "his"}, // unknown defaults male
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
		})
	}
}

// ---------------------------------------------------------------------------
// HandleDeath Tests
// ---------------------------------------------------------------------------

func TestHandleDeath_Mob(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1,
				ShortDesc: "a scary dragon",
				LongDesc:  "A scary dragon is here.",
				Keywords:  "dragon scary",
				Level:     5,
				Exp:       1000,
				Gold:      500,
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Spawn mob
	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Create killer player
	killer := NewPlayer(99, "Hero", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Handle death
	w.HandleDeath(mob, killer, 5) // Fireball

	// Verify mob is dead
	if mob.IsAlive() {
		t.Error("mob should not be alive after HandleDeath")
	}

	// Verify XP and Kills were updated on player
	if killer.Kills != 1 {
		t.Errorf("killer Kills = %d, want 1", killer.Kills)
	}

	// Check if corpse was created in the room
	items := w.roomItems[1001]
	if len(items) != 1 {
		t.Errorf("room items count = %d, want 1 (corpse)", len(items))
	} else {
		corpse := items[0]
		if !corpse.IsCorpse {
			t.Error("spawned item should be a corpse")
		}
		if !strings.Contains(corpse.Runtime.Keywords, "corpse") {
			t.Errorf("corpse keywords = %q, want it to contain 'corpse'", corpse.Runtime.Keywords)
		}
	}
}

func TestHandleDeath_Player(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Create victim player
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetExp(10000)
	victim.Stats.Con = 15
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Killer player
	killer := NewPlayer(2, "Killer", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Handle death
	w.HandleDeath(victim, killer, 303) // Slash

	// Verify victim was moved to respawn room (MortalStartRoom)
	if victim.GetRoom() != MortalStartRoom {
		t.Errorf("victim room = %d, want %d", victim.GetRoom(), MortalStartRoom)
	}

	// Verify XP penalty (combat death = exp/37)
	expectedExp := 10000 - (10000 / 37)
	if victim.GetExp() != expectedExp {
		t.Errorf("victim Exp = %d, want %d", victim.GetExp(), expectedExp)
	}

	// Check corpse created in room 1001
	items := w.roomItems[1001]
	if len(items) != 1 {
		t.Errorf("room items count = %d, want 1 (corpse)", len(items))
	}
}
