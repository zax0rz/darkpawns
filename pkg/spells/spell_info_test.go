package spells

import (
	"testing"
)

func TestSetAndGetSpellInfo(t *testing.T) {
	// Fresh test — set then get
	info := &SpellInfo{
		MinPosition: PosStanding,
		ManaMin:     10,
		ManaMax:     100,
		ManaChange:  5,
		Routines: SpellRoutines{
			Routines: RoutineDamage | RoutineAffects,
			Violent:  true,
			Targets:  TarCharRoom | TarFightVict,
		},
	}
	SetSpellInfo(42, info)

	got := GetSpellInfo(42)
	if got == nil {
		t.Fatal("GetSpellInfo(42) returned nil after SetSpellInfo")
	}
	if got.MinPosition != PosStanding {
		t.Errorf("MinPosition = %v, want %v", got.MinPosition, PosStanding)
	}
	if got.ManaMin != 10 {
		t.Errorf("ManaMin = %d, want 10", got.ManaMin)
	}
	if got.ManaMax != 100 {
		t.Errorf("ManaMax = %d, want 100", got.ManaMax)
	}
	if got.ManaChange != 5 {
		t.Errorf("ManaChange = %d, want 5", got.ManaChange)
	}
	if got.Routines.Routines != RoutineDamage|RoutineAffects {
		t.Errorf("Routines = %d, want %d", got.Routines.Routines, RoutineDamage|RoutineAffects)
	}
	if !got.Routines.Violent {
		t.Error("Routines.Violent = false, want true")
	}
	if got.Routines.Targets != TarCharRoom|TarFightVict {
		t.Errorf("Targets = %d, want %d", got.Routines.Targets, TarCharRoom|TarFightVict)
	}
}

func TestGetSpellInfo_Nonexistent(t *testing.T) {
	got := GetSpellInfo(99999)
	if got != nil {
		t.Errorf("GetSpellInfo(99999) = %v, want nil", got)
	}
}

func TestSetSpellInfo_NegativeIndex(t *testing.T) {
	// SetSpellInfo with negative index should be a no-op
	SetSpellInfo(-1, &SpellInfo{ManaMin: 99})
	if got := GetSpellInfo(-1); got != nil {
		t.Errorf("GetSpellInfo(-1) should not be stored, got %v", got)
	}
}

func TestSetupSpellInfo(t *testing.T) {
	// setupSpellInfo calls SetSpellInfo internally with correct fields
	setupSpellInfo(7, PosFighting, 5, 50, 3,
		RoutineDamage|RoutineGroups, false, TarFightVict|TarSelfOnly)

	got := GetSpellInfo(7)
	if got == nil {
		t.Fatal("GetSpellInfo(7) returned nil after setupSpellInfo")
	}
	if got.MinPosition != PosFighting {
		t.Errorf("MinPosition = %v, want %v", got.MinPosition, PosFighting)
	}
	if got.ManaMin != 5 {
		t.Errorf("ManaMin = %d, want 5", got.ManaMin)
	}
	if got.ManaMax != 50 {
		t.Errorf("ManaMax = %d, want 50", got.ManaMax)
	}
	if got.ManaChange != 3 {
		t.Errorf("ManaChange = %d, want 3", got.ManaChange)
	}
	if got.Routines.Routines != RoutineDamage|RoutineGroups {
		t.Errorf("Routines = %d, want %d", got.Routines.Routines, RoutineDamage|RoutineGroups)
	}
	if got.Routines.Violent != false {
		t.Error("Routines.Violent should be false")
	}
	if got.Routines.Targets != TarFightVict|TarSelfOnly {
		t.Errorf("Targets = %d, want %d", got.Routines.Targets, TarFightVict|TarSelfOnly)
	}
}

func TestHasRoutine(t *testing.T) {
	si := &SpellInfo{
		Routines: SpellRoutines{
			Routines: RoutineDamage | RoutineAffects | RoutineAreas,
		},
	}

	tests := []struct {
		routine  MagRoutine
		expected bool
	}{
		{RoutineDamage, true},
		{RoutineAffects, true},
		{RoutineAreas, true},
		{RoutinePoints, false},
		{RoutineUnaffects, false},
		{RoutineSummons, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := si.HasRoutine(tt.routine); got != tt.expected {
				t.Errorf("HasRoutine(%d) = %v, want %v", tt.routine, got, tt.expected)
			}
		})
	}
}

func TestHasRoutine_NilReceiver(t *testing.T) {
	var si *SpellInfo = nil
	if got := si.HasRoutine(RoutineDamage); got != false {
		t.Errorf("nil SpellInfo.HasRoutine = %v, want false", got)
	}
}

func TestHasTarget(t *testing.T) {
	si := &SpellInfo{
		Routines: SpellRoutines{
			Targets: TarCharRoom | TarFightVict | TarObjInv,
		},
	}

	tests := []struct {
		target   TargetFlags
		expected bool
	}{
		{TarCharRoom, true},
		{TarFightVict, true},
		{TarObjInv, true},
		{TarCharWorld, false},
		{TarSelfOnly, false},
		{TarNotSelf, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := si.HasTarget(tt.target); got != tt.expected {
				t.Errorf("HasTarget(%d) = %v, want %v", tt.target, got, tt.expected)
			}
		})
	}
}

func TestHasTarget_NilReceiver(t *testing.T) {
	var si *SpellInfo = nil
	if got := si.HasTarget(TarIgnore); got != false {
		t.Errorf("nil SpellInfo.HasTarget = %v, want false", got)
	}
}

func TestGetManaCost_Clamping(t *testing.T) {
	// ManaMax=100, ManaMin=10, ManaChange=5
	// cost = 100 - (5 * level), clamped to [10, Inf) then bottom-clamped to >=0
	si := &SpellInfo{ManaMax: 100, ManaMin: 10, ManaChange: 5}

	tests := []struct {
		level    int
		expected int
		note     string
	}{
		{0, 100, "level 0 → mana max"},
		{1, 95, "level 1"},
		{10, 50, "level 10"},
		{18, 10, "hits ManaMin at level 18 (100-90=10)"},
		{19, 10, "clamped to ManaMin (100-95=5 → 10)"},
		{50, 10, "clamped to ManaMin at high level"},
	}

	for _, tt := range tests {
		t.Run(tt.note, func(t *testing.T) {
			if got := si.GetManaCost(tt.level); got != tt.expected {
				t.Errorf("GetManaCost(%d) = %d, want %d", tt.level, got, tt.expected)
			}
		})
	}
}

func TestGetManaCost_NeverBelowZero(t *testing.T) {
	// ManaMin=0, ManaMax=10, ManaChange=20 — high change will push negative
	// cost = 10 - (20 * level). For level 1: 10-20 = -10 → clipped to 0
	si := &SpellInfo{ManaMax: 10, ManaMin: 0, ManaChange: 20}

	if got := si.GetManaCost(1); got < 0 {
		t.Errorf("GetManaCost(1) = %d, want >= 0", got)
	}
	if got := si.GetManaCost(10); got != 0 {
		t.Errorf("GetManaCost(10) = %d, want 0", got)
	}
}

func TestGetManaCost_NilReceiver(t *testing.T) {
	var si *SpellInfo = nil
	if got := si.GetManaCost(5); got != 0 {
		t.Errorf("nil SpellInfo.GetManaCost = %d, want 0", got)
	}
}

func TestIsViolent(t *testing.T) {
	violent := &SpellInfo{
		Routines: SpellRoutines{Violent: true},
	}
	if !violent.IsViolent() {
		t.Error("violent.IsViolent() = false, want true")
	}

	nonViolent := &SpellInfo{
		Routines: SpellRoutines{Violent: false},
	}
	if nonViolent.IsViolent() {
		t.Error("nonViolent.IsViolent() = true, want false")
	}
}

func TestIsViolent_NilReceiver(t *testing.T) {
	var si *SpellInfo = nil
	if si.IsViolent() {
		t.Error("nil SpellInfo.IsViolent() = true, want false")
	}
}

func TestAttackTypes_TableSize(t *testing.T) {
	if len(AttackTypes) < 20 {
		t.Errorf("AttackTypes has %d entries, want at least 20", len(AttackTypes))
	}
}

func TestAttackTypes_IndexZeroIsEmpty(t *testing.T) {
	if AttackTypes[0].Singular != "" {
		t.Errorf("AttackTypes[0].Singular = %q, want empty", AttackTypes[0].Singular)
	}
	if AttackTypes[0].Plural != "" {
		t.Errorf("AttackTypes[0].Plural = %q, want empty", AttackTypes[0].Plural)
	}
}

func TestAttackTypes_EntriesHaveValues(t *testing.T) {
	for i := 1; i < len(AttackTypes); i++ {
		if AttackTypes[i].Singular == "" {
			t.Errorf("AttackTypes[%d].Singular is empty", i)
		}
		if AttackTypes[i].Plural == "" {
			t.Errorf("AttackTypes[%d].Plural is empty", i)
		}
		if AttackTypes[i].Singular == AttackTypes[i].Plural {
			t.Errorf("AttackTypes[%d].Singular == Plural == %q, expected different forms", i, AttackTypes[i].Singular)
		}
	}
}
