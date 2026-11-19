package combat

import (
	"fmt"
	"strings"
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

func TestTakeDamage_JailGuardDoesNotSubdueInvisibleVictim(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	guard := &mockCombatant{
		name: "jail guard", npc: true, room: 8117, level: 30,
		hp: 80, maxHP: 100, position: PosFighting,
	}
	victim := &mockCombatant{
		name: "Thief", room: 8117, level: 20,
		hp: 100, maxHP: 100, position: PosFighting,
	}
	SetCallbacks(&GameCallbacks{
		MobHasJailGuardSpec: func(Combatant) bool { return true },
		CanSee:              func(observer, subject Combatant) bool { return false },
		HasAffect:           func(Combatant, int) bool { return false },
		HasAffectStr:        func(Combatant, string) bool { return false },
	})

	takeDamageFrom(guard, victim, 1, TYPE_UNDEFINED, nil, false)
	if victim.GetHP() != 99 {
		t.Fatalf("invisible victim HP = %d, want normal damage to 99", victim.GetHP())
	}
}

// ---------------------------------------------------------------------------
// BackstabMult — formula: float64(level)*0.2 + 1.0, truncated to int (C fidelity)
// Source: src/class.c — backstab_mult() returns int, so the float result truncates.
// ---------------------------------------------------------------------------

func TestBackstabMult(t *testing.T) {
	tests := []struct {
		level int
		want  float64
	}{
		{0, 1.0},   // level 0: guard → 1.0
		{1, 1.0},   // 1.2 → trunc 1
		{5, 2.0},   // 5*0.2+1.0
		{10, 3.0},  // 10*0.2+1.0
		{14, 3.0},  // 3.8 → trunc 3 (DP-1033)
		{19, 4.0},  // 4.8 → trunc 4 (DP-1033)
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
// CalcXPShare
// ---------------------------------------------------------------------------

func TestCalcXPShare(t *testing.T) {
	// ch < victim: victim is higher → share unchanged (levelDiff negative, no reduction)
	if diff := CalcXPShare(10, 20, 100, false, maxExpGain); diff <= 0 {
		t.Errorf("CalcXPShare(ch=10, victim=20) = %d, want positive", diff)
	}

	// ch > victim: ch is higher → share decreases
	if diff := CalcXPShare(30, 5, 100, false, maxExpGain); diff >= 100 {
		t.Errorf("CalcXPShare(ch=30, victim=5) = %d, want < 100", diff)
	}

	// Same level 15: no level adjustment, no over-20 penalty → 100
	if diff := CalcXPShare(15, 15, 100, false, maxExpGain); diff != 100 {
		t.Errorf("CalcXPShare(same level 15) = %d, want 100", diff)
	}

	if diff := CalcXPShare(30, 10, 99, false, maxExpGain); diff != 23 {
		t.Errorf("CalcXPShare(ch=30, victim=10, base=99) = %d, want 23 per C truncation", diff)
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
		"Alice", "Bob", "slash", "slashes", 0, 0,
	)
	if got != "Alice hits Bob with slash!" {
		t.Errorf("replaceMessageTokens() = %q, want %q", got, "Alice hits Bob with slash!")
	}

	// Pronoun tokens
	got = replaceMessageTokens("$e hits $N with $s blade.", "Warrior", "Enemy", "hit", "hits", 0, 0)
	if got != "He hits Enemy with his blade." {
		t.Errorf("pronoun tokens: got %q", got)
	}

	got = replaceMessageTokens("$e hits $N.", "Alice", "Enemy", "hit", "hits", 1, 0)
	if got != "She hits Enemy." {
		t.Errorf("female pronoun: got %q", got)
	}
}

// ---------------------------------------------------------------------------
// ChangeAlignment
// ---------------------------------------------------------------------------

func TestChangeAlignment(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	var setTo, setCalls int
	SetCallbacks(&GameCallbacks{
		GetAlignment: func(bodyname Combatant) int {
			if bodyname.GetName() == "demon" {
				return -1000
			}
			return 0
		},
		SetAlignment: func(bodyname Combatant, val int) {
			setCalls++
			setTo = val
		},
	})

	paladin := &mockCombatant{name: "paladin", npc: false}
	evil := &mockCombatant{name: "demon", npc: true, sex: 0}

	// src/fight.c:496-501 — GET_ALIGNMENT(ch) += (-GET_ALIGNMENT(victim) -
	// GET_ALIGNMENT(ch)) >> 4, clamped to +/-1000. Killing a -1000 victim at
	// 0 alignment gives 0 + ((1000 - 0) >> 4) = 62. The old test installed a
	// no-op SetAlignment, so it could not tell a real update from none.
	ChangeAlignment(paladin, evil)
	if setCalls != 1 {
		t.Fatalf("SetAlignment called %d times, want 1 (fight.c:496)", setCalls)
	}
	if setTo != 62 {
		t.Errorf("alignment after killing a -1000 victim = %d, want 62 (fight.c:496)", setTo)
	}
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
		GetRaceHate:  func(bodyname Combatant, index int) int { return 1 },
		HasAffect:    func(bodyname Combatant, aff int) bool { return false },
		IsShopkeeper: func(bodyname Combatant) bool { return false },
		HasPlrFlag:   func(bodyname Combatant, flag string) bool { return false },
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
		MakeCorpse: func(bodyvictim Combatant, attackType int) {
			victim := bodyvictim.GetName()

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

// TestGetExpNilGuard_DieWithKiller verifies DieWithKiller does not panic when
// GetExp is nil but GainExp is wired.
func TestGetExpNilGuard_DieWithKiller(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	SetCallbacks(&GameCallbacks{
		GainExp:          func(bodyname Combatant, amount int) {},
		RemoveAllAffects: func(bodyname Combatant) {},
		MakeCorpse:       func(bodyname Combatant, attackType int) {},
		ExtractChar:      func(bodyname Combatant) {},
	})

	victim := &mockCombatant{name: "Victim", room: 100, level: 10}
	killer := &mockCombatant{name: "Killer", room: 100, level: 10}

	// Should not panic even though GetExp is nil.
	DieWithKiller(victim, killer, TYPE_UNDEFINED)
}

// TestTakeDamageDeathBroadcastExcludesVictim: C's act("$n is dead!  R.I.P.",
// ..., TO_ROOM) never reaches the victim, who gets only "You are dead!"
// (fight.c:1581-1584).
func TestTakeDamageDeathBroadcastExcludesVictim(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	var excluded []string
	SetCallbacks(&GameCallbacks{
		Broadcast: func(roomVNum int, msg string, excludedBodies []Combatant) {
			exclude := testBodyNames(excludedBodies)

			if strings.Contains(msg, "R.I.P.") {
				excluded = append(excluded, exclude)
			}
		},
	})

	ch := &mockCombatant{name: "Killer", level: 40, room: 100, position: PosStanding, hp: 100, maxHP: 100}
	victim := &mockCombatant{name: "Victim", npc: true, level: 11, room: 100, position: PosSleeping, hp: 5, maxHP: 100}
	TakeDamageWithDeath(ch, victim, 40, TYPE_HIT, func() {})

	if len(excluded) != 1 || excluded[0] != "Victim" {
		t.Fatalf("R.I.P. broadcast exclusions = %q, want [\"Victim\"]", excluded)
	}
}
