package combat

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// NewNamedCombatant
// ---------------------------------------------------------------------------

func TestNewNamedCombatant(t *testing.T) {
	c := NewNamedCombatant("Alice", 42)
	if c.GetName() != "Alice" {
		t.Errorf("GetName() = %q, want %q", c.GetName(), "Alice")
	}
	if c.GetRoom() != 42 {
		t.Errorf("GetRoom() = %d, want %d", c.GetRoom(), 42)
	}
	if c.IsNPC() {
		t.Error("IsNPC() = true, want false (group members are PCs)")
	}
	if c.GetPosition() != PosStanding {
		t.Errorf("GetPosition() = %d, want %d", c.GetPosition(), PosStanding)
	}
}

// ---------------------------------------------------------------------------
// BackstabMult — formula: float64(level)*0.2 + 1.0
// Source: src/class.c — backstab_mult()
// ---------------------------------------------------------------------------

func TestBackstabMult(t *testing.T) {
	tests := []struct {
		level int
		want  float64
	}{
		{0, 1.0},   // level 0: guard → 1.0
		{1, 1.2},   // 1*0.2+1.0
		{5, 2.0},   // 5*0.2+1.0
		{10, 3.0},  // 10*0.2+1.0
		{25, 6.0},  // 25*0.2+1.0
		{30, 7.0},  // 30*0.2+1.0
		{31, 20.0}, // LVL_IMMORT=31 → cap at 20.0
		{50, 20.0}, // above immort → 20.0
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("level_%d", tt.level), func(t *testing.T) {
			got := BackstabMult(tt.level)
			if got != tt.want {
				t.Errorf("BackstabMult(%d) = %f, want %f", tt.level, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsInGroup — uses HasAffectStr, GetMasterInRoom, GetFellowFollowersInRoom
// ---------------------------------------------------------------------------

func TestIsInGroup(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	ch := &mockCombatant{name: "Alice", npc: false, room: 100}

	// No hooks wired → false
	SetCallbacks(nil)
	if IsInGroup(ch) {
		t.Error("IsInGroup with nil hooks should return false")
	}

	// HasAffectStr true + GetMasterInRoom true → true
	SetCallbacks(&GameCallbacks{
		HasAffectStr: func(name string, aff string) bool {
			return name == "Alice" && aff == AFF_STR_GROUP
		},
		GetMasterInRoom: func(name string, room int) bool {
			return name == "Alice" && room == 100
		},
	})
	if !IsInGroup(ch) {
		t.Error("IsInGroup(Alice) should return true with master in room")
	}

	// HasAffectStr true + GetFellowFollowers true → true
	SetCallbacks(&GameCallbacks{
		HasAffectStr: func(name string, aff string) bool {
			return name == "Alice" && aff == AFF_STR_GROUP
		},
		GetMasterInRoom:          func(name string, room int) bool { return false },
		GetFellowFollowersInRoom: func(name string, room int) bool { return name == "Alice" && room == 100 },
	})
	if !IsInGroup(ch) {
		t.Error("IsInGroup(Alice) should return true with fellow followers")
	}

	// HasAffectStr false → false
	SetCallbacks(&GameCallbacks{
		HasAffectStr: func(name string, aff string) bool { return false },
	})
	if IsInGroup(ch) {
		t.Error("IsInGroup(Alice) should return false without group affect")
	}
}

// ---------------------------------------------------------------------------
// CalcLevelDiff
// ---------------------------------------------------------------------------

func TestCalcLevelDiff(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	// Wire IsInGroup to return false for solo
	SetCallbacks(&GameCallbacks{
		HasAffectStr: func(name string, aff string) bool { return false },
	})

	ch := &mockCombatant{name: "ch", level: 10}
	victim := &mockCombatant{name: "vic", level: 20}

	// ch < victim: victim is higher → share increases (levelDiff negative, no reduction)
	diff := CalcLevelDiff(ch, victim, 100)
	if diff <= 0 {
		t.Errorf("CalcLevelDiff(ch=10, victim=20) = %d, want positive", diff)
	}

	// ch > victim: ch is higher → share decreases
	ch2 := &mockCombatant{name: "ch2", level: 30}
	victim2 := &mockCombatant{name: "vic2", level: 5}
	diff2 := CalcLevelDiff(ch2, victim2, 100)
	if diff2 >= 100 {
		t.Errorf("CalcLevelDiff(ch=30, victim=5) = %d, want < 100", diff2)
	}

	// Same level: returns base (no adjustments for level diff)
	ch3 := &mockCombatant{name: "ch3", level: 15}
	victim3 := &mockCombatant{name: "vic3", level: 15}
	diff3 := CalcLevelDiff(ch3, victim3, 100)
	// At level 15, no level adjustment, no over-20 penalty → should be 100
	if diff3 != 100 {
		t.Errorf("CalcLevelDiff(same level 15) = %d, want 100", diff3)
	}
}

// ---------------------------------------------------------------------------
// randPick
// ---------------------------------------------------------------------------

func TestRandPick(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		v := randPick(s)
		seen[v] = true
		if v < 1 || v > 5 {
			t.Errorf("randPick returned %d, out of range", v)
		}
	}
	if len(seen) < 2 {
		t.Errorf("randPick only returned %d distinct values in 100 iterations", len(seen))
	}
}

func TestRandPick_SingleElement(t *testing.T) {
	got := randPick([]int{42})
	if got != 42 {
		t.Errorf("randPick([42]) = %d, want 42", got)
	}
}

// ---------------------------------------------------------------------------
// GetPositionFromHP
// ---------------------------------------------------------------------------

func TestGetPositionFromHP(t *testing.T) {
	// Positive HP + standing → stays standing
	got := GetPositionFromHP(100, PosStanding)
	if got != PosStanding {
		t.Errorf("GetPositionFromHP(100, standing) = %d, want %d", got, PosStanding)
	}

	// Positive HP + stunned → goes to standing
	got = GetPositionFromHP(5, PosStunned)
	if got != PosStanding {
		t.Errorf("GetPositionFromHP(5, stunned) = %d, want %d", got, PosStanding)
	}

	// Negative HP thresholds: <=-11 dead, <=-6 mortally, <=-3 incap, else stunned
	got = GetPositionFromHP(-2, PosStanding)
	if got != PosStunned {
		t.Errorf("GetPositionFromHP(-2) = %d, want %d (stunned)", got, PosStunned)
	}
	got = GetPositionFromHP(-5, PosStanding)
	if got != PosIncap {
		t.Errorf("GetPositionFromHP(-5) = %d, want %d (incap)", got, PosIncap)
	}
	got = GetPositionFromHP(-8, PosStanding)
	if got != PosMortally {
		t.Errorf("GetPositionFromHP(-8) = %d, want %d (mortally)", got, PosMortally)
	}
	got = GetPositionFromHP(-12, PosStanding)
	if got != PosDead {
		t.Errorf("GetPositionFromHP(-12) = %d, want %d (dead)", got, PosDead)
	}
}

// ---------------------------------------------------------------------------
// replaceMessageTokens
// ---------------------------------------------------------------------------

func TestReplaceMessageTokens(t *testing.T) {
	got := replaceMessageTokens(
		"$n hits $N with #w!",
		"Alice", "Bob", "slash", "slashes", 0,
	)
	if got != "Alice hits Bob with slash!" {
		t.Errorf("replaceMessageTokens() = %q, want %q", got, "Alice hits Bob with slash!")
	}

	// Pronoun tokens
	got = replaceMessageTokens("$e hits $N with $s blade.", "Warrior", "Enemy", "hit", "hits", 0)
	if got != "he hits Enemy with his blade." {
		t.Errorf("pronoun tokens: got %q", got)
	}

	got = replaceMessageTokens("$e hits $N.", "Alice", "Enemy", "hit", "hits", 1)
	if got != "she hits Enemy." {
		t.Errorf("female pronoun: got %q", got)
	}
}

// ---------------------------------------------------------------------------
// GroupGain — smoke test (no hooks wired)
// ---------------------------------------------------------------------------

func TestGroupGain_NoHooks(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	ch := &mockCombatant{name: "Alice", npc: false, level: 10, room: 100}
	victim := &mockCombatant{name: "Orc", npc: true, level: 8}

	SetCallbacks(&GameCallbacks{
		CountGroupMembers:   func(leaderName string, roomVNum int) int { return 1 },
		ApplyToGroupMembers: func(leaderName string, roomVNum int, fn func(string)) { fn(leaderName) },
		GainExp:             func(name string, amount int) {},
		GetExp: func(name string) int {
			if name == "Orc" {
				return 200
			}
			return 0
		},
		GetAlignment: func(name string) int { return 0 },
		SetAlignment: func(name string, val int) {},
	})

	GroupGain(ch, victim) // should not panic
}

// ---------------------------------------------------------------------------
// ChangeAlignment
// ---------------------------------------------------------------------------

func TestChangeAlignment(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{
		GetAlignment: func(name string) int {
			if name == "paladin" {
				return 1000
			}
			return 0
		},
		SetAlignment: func(name string, val int) {},
	})

	paladin := &mockCombatant{name: "paladin", npc: false}
	evil := &mockCombatant{name: "demon", npc: true, sex: 0}

	// Killing evil should make paladin more good
	ChangeAlignment(paladin, evil)
	// Just verify it doesn't panic — actual value depends on implementation
}

func TestChangeAlignment_NilGetAlignment(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{})

	paladin := &mockCombatant{name: "paladin", npc: false}
	evil := &mockCombatant{name: "demon", npc: true}

	// Should not panic when GetAlignment is nil
	ChangeAlignment(paladin, evil)
}

func TestTakeDamage_NilGetRace(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{
		GetRaceHate:  func(name string, index int) int { return 1 },
		HasAffect:    func(name string, aff int) bool { return false },
		IsShopkeeper: func(name string) bool { return false },
		HasPlrFlag:   func(name string, flag string) bool { return false },
		HasRoomFlag:  func(room int, flag string) bool { return false },
	})

	ch := &mockCombatant{name: "ch", npc: false, level: 10, room: 100}
	victim := &mockCombatant{name: "vic", npc: false, level: 10, room: 100}

	// Should not panic when GetRace is nil even though GetRaceHate is non-nil
	TakeDamage(ch, victim, 10, TYPE_HIT)
}

// TestRawKill_NilGetRace verifies that RawKill does not panic when the GetRace
// function pointer is nil and still produces a corpse (DP-620).
func TestRawKill_NilGetRace(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	madeCorpse := false
	SetCallbacks(&GameCallbacks{
		MakeCorpse: func(victim string, attackType int) {
			if victim == "Victim" {
				madeCorpse = true
			}
		},
	})

	ch := &mockCombatant{name: "Victim", room: 100}
	RawKill(ch, TYPE_UNDEFINED)

	if !madeCorpse {
		t.Error("expected MakeCorpse to be called when GetRace is nil")
	}
}

// TestGetExpNilGuard_GroupGain verifies GroupGain does not panic when GetExp is nil.
func TestGetExpNilGuard_GroupGain(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{
		CountGroupMembers: func(leaderName string, roomVNum int) int { return 1 },
		ApplyToGroupMembers: func(leaderName string, roomVNum int, fn func(string)) {
			fn(leaderName)
		},
	})

	leader := &mockCombatant{name: "Leader", room: 100, level: 10}
	victim := &mockCombatant{name: "Victim", room: 100, level: 5}

	// Should not panic; with GetExp nil the base share becomes 0, then clamped to 1.
	GroupGain(leader, victim)
}

// TestGetExpNilGuard_DieWithKiller verifies DieWithKiller does not panic when
// GetExp is nil but GainExp is wired.
func TestGetExpNilGuard_DieWithKiller(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{
		GainExp:          func(name string, amount int) {},
		RemoveAllAffects: func(name string) {},
		MakeCorpse:       func(name string, attackType int) {},
		ExtractChar:      func(name string) {},
	})

	victim := &mockCombatant{name: "Victim", room: 100, level: 10}
	killer := &mockCombatant{name: "Killer", room: 100, level: 10}

	// Should not panic even though GetExp is nil.
	DieWithKiller(victim, killer, TYPE_UNDEFINED)
}
