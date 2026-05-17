package engine

import (
	"testing"
	"time"
)

// MockAffectable is a test implementation of the Affectable interface
type MockAffectable struct {
	name         string
	id           int
	strength     int
	dexterity    int
	intelligence int
	wisdom       int
	constitution int
	charisma     int
	hitRoll      int
	damageRoll   int
	armorClass   int
	thac0        int
	hp           int
	maxHP        int
	mana         int
	maxMana      int
	movement     int
	statusFlags  uint64
	messages     []string
}

func NewMockAffectable(name string, id int) *MockAffectable {
	return &MockAffectable{
		name:         name,
		id:           id,
		strength:     10,
		dexterity:    10,
		intelligence: 10,
		wisdom:       10,
		constitution: 10,
		charisma:     10,
		hitRoll:      0,
		damageRoll:   0,
		armorClass:   10,
		thac0:        20,
		hp:           100,
		maxHP:        100,
		mana:         50,
		maxMana:      50,
		movement:     100,
		statusFlags:  0,
		messages:     make([]string, 0),
	}
}

func (m *MockAffectable) GetAffects() []*Affect { return nil }
func (m *MockAffectable) SetAffects([]*Affect)  {}
func (m *MockAffectable) GetName() string       { return m.name }
func (m *MockAffectable) GetID() int            { return m.id }

func (m *MockAffectable) GetStrength() int      { return m.strength }
func (m *MockAffectable) SetStrength(v int)     { m.strength = v }
func (m *MockAffectable) GetDexterity() int     { return m.dexterity }
func (m *MockAffectable) SetDexterity(v int)    { m.dexterity = v }
func (m *MockAffectable) GetIntelligence() int  { return m.intelligence }
func (m *MockAffectable) SetIntelligence(v int) { m.intelligence = v }
func (m *MockAffectable) GetWisdom() int        { return m.wisdom }
func (m *MockAffectable) SetWisdom(v int)       { m.wisdom = v }
func (m *MockAffectable) GetConstitution() int  { return m.constitution }
func (m *MockAffectable) SetConstitution(v int) { m.constitution = v }
func (m *MockAffectable) GetCharisma() int      { return m.charisma }
func (m *MockAffectable) SetCharisma(v int)     { m.charisma = v }

func (m *MockAffectable) GetHitRoll() int     { return m.hitRoll }
func (m *MockAffectable) SetHitRoll(v int)    { m.hitRoll = v }
func (m *MockAffectable) GetDamageRoll() int  { return m.damageRoll }
func (m *MockAffectable) SetDamageRoll(v int) { m.damageRoll = v }
func (m *MockAffectable) IsNPC() bool         { return false }

func (m *MockAffectable) GetArmorClass() int  { return m.armorClass }
func (m *MockAffectable) SetArmorClass(v int) { m.armorClass = v }
func (m *MockAffectable) GetTHAC0() int       { return m.thac0 }
func (m *MockAffectable) SetTHAC0(v int)      { m.thac0 = v }

func (m *MockAffectable) GetHP() int        { return m.hp }
func (m *MockAffectable) SetHP(v int)       { m.hp = v }
func (m *MockAffectable) GetMaxHP() int     { return m.maxHP }
func (m *MockAffectable) SetMaxHP(v int)    { m.maxHP = v }
func (m *MockAffectable) GetMana() int      { return m.mana }
func (m *MockAffectable) SetMana(v int)     { m.mana = v }
func (m *MockAffectable) GetMaxMana() int   { return m.maxMana }
func (m *MockAffectable) SetMaxMana(v int)  { m.maxMana = v }
func (m *MockAffectable) GetMovement() int  { return m.movement }
func (m *MockAffectable) SetMovement(v int) { m.movement = v }

func (m *MockAffectable) HasStatusFlag(flag uint64) bool { return m.statusFlags&flag != 0 }
func (m *MockAffectable) SetStatusFlag(flag uint64)      { m.statusFlags |= flag }
func (m *MockAffectable) ClearStatusFlag(flag uint64)    { m.statusFlags &^= flag }

func (m *MockAffectable) SendMessage(msg string) {
	m.messages = append(m.messages, msg)
}

func (m *MockAffectable) GetMessages() []string {
	return m.messages
}

func (m *MockAffectable) ClearMessages() {
	m.messages = make([]string, 0)
}

// TestNewAffect tests creating a new affect with the unified API
func TestNewAffect(t *testing.T) {
	affect := NewAffect(0, ApplyStr, 10, 5, "test spell")

	if affect.SpellID != 0 {
		t.Errorf("Expected spell ID 0, got %d", affect.SpellID)
	}
	if affect.Location != ApplyStr {
		t.Errorf("Expected location ApplyStr (%d), got %d", ApplyStr, affect.Location)
	}
	if affect.Duration != 10 {
		t.Errorf("Expected duration 10, got %d", affect.Duration)
	}
	if affect.Magnitude != 5 {
		t.Errorf("Expected magnitude 5, got %d", affect.Magnitude)
	}
	if affect.Source != "test spell" {
		t.Errorf("Expected source 'test spell', got %s", affect.Source)
	}
	if affect.ID == "" {
		t.Error("Expected non-empty affect ID")
	}
}

// TestAffectTick tests ticking an affect
func TestAffectTick(t *testing.T) {
	affect := NewAffect(0, ApplyStr, 3, 5, "test")

	expired := affect.Tick()
	if expired {
		t.Error("Affect should not expire after first tick")
	}
	if affect.Duration != 2 {
		t.Errorf("Expected duration 2 after tick, got %d", affect.Duration)
	}

	affect.Tick()
	expired = affect.Tick()
	if !expired {
		t.Error("Affect should expire after third tick")
	}
	if affect.Duration != 0 {
		t.Errorf("Expected duration 0 after expiration, got %d", affect.Duration)
	}
}

// TestPermanentAffect tests that permanent affects don't expire
func TestPermanentAffect(t *testing.T) {
	affect := NewAffect(0, ApplyStr, 0, 5, "permanent")

	for i := 0; i < 10; i++ {
		expired := affect.Tick()
		if expired {
			t.Error("Permanent affect should never expire")
		}
		if affect.Duration != 0 {
			t.Errorf("Permanent affect duration should remain 0, got %d", affect.Duration)
		}
	}
}

// TestApplyAffect tests applying a stat affect to an entity
func TestApplyAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffect(0, ApplyStr, 10, 5, "strength potion")
	success := am.ApplyAffect(mock, affect)

	if !success {
		t.Error("Failed to apply affect")
	}
	if mock.GetStrength() != 15 {
		t.Errorf("Expected strength 15 after affect, got %d", mock.GetStrength())
	}
	messages := mock.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected affect application message")
	}
}

// TestRemoveAffect tests removing a stat affect from an entity
func TestRemoveAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffect(0, ApplyStr, 10, 5, "strength potion")
	am.ApplyAffect(mock, affect)
	mock.ClearMessages()

	success := am.RemoveAffect(mock, affect.ID)
	if !success {
		t.Error("Failed to remove affect")
	}
	if mock.GetStrength() != 10 {
		t.Errorf("Expected strength 10 after affect removal, got %d", mock.GetStrength())
	}
	messages := mock.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected affect removal message")
	}
}

// TestAffectStacking tests affect stacking rules
func TestAffectStacking(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Create two affects with same StackID (should replace)
	affect1 := NewAffectDirect(0, ApplyNone, 10, 5, AFFPoison, "poison")
	affect1.StackID = "poison"
	affect1.MaxStacks = 1

	affect2 := NewAffectDirect(0, ApplyNone, 5, 3, AFFPoison, "stronger poison")
	affect2.StackID = "poison"
	affect2.MaxStacks = 1

	am.ApplyAffect(mock, affect1)
	if !mock.HasStatusFlag(AFFPoison) {
		t.Error("First poison affect should set poison flag")
	}

	am.ApplyAffect(mock, affect2)

	affects := am.GetAffects(mock)
	poisonCount := 0
	for _, aff := range affects {
		if aff.Flags&AFFPoison != 0 {
			poisonCount++
		}
	}
	if poisonCount != 1 {
		t.Errorf("Expected 1 poison affect after stacking, got %d", poisonCount)
	}
}

// TestStatusAffect tests status flag affect application and removal
func TestStatusAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Apply invisible affect via flag
	affect := NewAffectDirect(0, ApplyNone, 10, 0, AFFInvisible, "invisibility spell")
	am.ApplyAffect(mock, affect)

	if !mock.HasStatusFlag(AFFInvisible) {
		t.Error("Invisible affect should set invisible flag")
	}

	am.RemoveAffect(mock, affect.ID)

	if mock.HasStatusFlag(AFFInvisible) {
		t.Error("Invisible flag should be cleared after affect removal")
	}
}

// TestPeriodicEffect tests periodic effects like poison
func TestPeriodicEffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffectDirect(0, ApplyNone, 3, 5, AFFPoison, "poison")
	am.ApplyAffect(mock, affect)

	initialHP := mock.GetHP()

	am.Tick()
	am.Tick()
	am.Tick()

	expectedHP := initialHP - (5 * 3)
	if mock.GetHP() != expectedHP {
		t.Errorf("Expected HP %d after poison, got %d", expectedHP, mock.GetHP())
	}

	affects := am.GetAffects(mock)
	if len(affects) != 0 {
		t.Errorf("Expected no affects after expiration, got %d", len(affects))
	}
}

// TestRegenerationAffect tests regeneration periodic effect
func TestRegenerationAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	mock.SetHP(50)

	affect := NewAffectDirect(0, ApplyNone, 3, 5, AFFRegeneration, "regeneration")
	am.ApplyAffect(mock, affect)

	am.Tick()
	am.Tick()
	am.Tick()

	expectedHP := 50 + (5 * 3)
	if mock.GetHP() != expectedHP {
		t.Errorf("Expected HP %d after regeneration, got %d", expectedHP, mock.GetHP())
	}
}

// TestTickManager tests the tick manager
func TestTickManager(t *testing.T) {
	am := NewAffectManager()
	tm := NewTickManager(am)

	tm.SetTickInterval(10 * time.Millisecond)
	tm.Start()
	time.Sleep(50 * time.Millisecond)

	if !tm.IsRunning() {
		t.Error("Tick manager should report running while active")
	}

	tm.Stop()
	tm.ManualTick()

	if tm.IsRunning() {
		t.Error("Tick manager should report stopped after Stop()")
	}
}

// TestAffectTickSystem tests the combined affect tick system
func TestAffectTickSystem(t *testing.T) {
	ats := NewAffectTickSystem()
	mock := NewMockAffectable("test", 1)

	affect := NewAffect(42, ApplyStr, 5, 3, "test")
	ats.ApplyAffect(mock, affect)

	if mock.GetStrength() != 13 {
		t.Errorf("Expected strength 13, got %d", mock.GetStrength())
	}

	affects := ats.GetAffects(mock)
	if len(affects) != 1 {
		t.Errorf("Expected 1 affect, got %d", len(affects))
	}

	if !ats.HasAffectBySpell(mock, 42) {
		t.Error("Should have strength affect")
	}

	ats.RemoveAffect(mock, affect.ID)
	if mock.GetStrength() != 10 {
		t.Errorf("Expected strength 10 after removal, got %d", mock.GetStrength())
	}

	ats.ManualTick()
}

// TestHasAffectBySpell tests spell-based affect lookup
func TestHasAffectBySpell(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffect(42, ApplyStr, 10, 5, "test spell")
	am.ApplyAffect(mock, affect)

	if !am.HasAffectBySpell(mock, 42) {
		t.Error("Should have affect from spell 42")
	}
	if am.HasAffectBySpell(mock, 99) {
		t.Error("Should not have affect from spell 99")
	}
}

// TestRemoveAffectsBySpell tests removing all affects from one spell
func TestRemoveAffectsBySpell(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Bless creates two affects from spell 42
	aff1 := NewAffect(42, ApplyStr, 10, 2, "bless str")
	aff2 := NewAffect(42, ApplyHitroll, 10, 2, "bless hitroll")
	am.ApplyAffect(mock, aff1)
	am.ApplyAffect(mock, aff2)

	if mock.GetStrength() != 12 {
		t.Errorf("Expected strength 12, got %d", mock.GetStrength())
	}
	if mock.GetHitRoll() != 2 {
		t.Errorf("Expected hitroll 2, got %d", mock.GetHitRoll())
	}

	removed := am.RemoveAffectsBySpell(mock, 42)
	if removed != 2 {
		t.Errorf("Expected 2 affects removed, got %d", removed)
	}
	if mock.GetStrength() != 10 {
		t.Errorf("Expected strength 10 after removal, got %d", mock.GetStrength())
	}
	if mock.GetHitRoll() != 0 {
		t.Errorf("Expected hitroll 0 after removal, got %d", mock.GetHitRoll())
	}
}

// TestMultiAffectFromOneSpell tests multiple affects from a single spell
func TestMultiAffectFromOneSpell(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Bless: +2 hitroll, +1 STR, + saving throw
	aff1 := NewAffect(42, ApplyHitroll, 6, 2, "bless")
	aff2 := NewAffect(42, ApplyStr, 6, 1, "bless")
	aff3 := NewAffect(42, ApplySavingSpell, 6, -2, "bless")
	am.ApplyAffect(mock, aff1)
	am.ApplyAffect(mock, aff2)
	am.ApplyAffect(mock, aff3)

	if mock.GetHitRoll() != 2 {
		t.Errorf("Expected hitroll 2, got %d", mock.GetHitRoll())
	}
	if mock.GetStrength() != 11 {
		t.Errorf("Expected strength 11, got %d", mock.GetStrength())
	}

	// All 3 affects should exist
	affects := am.GetAffects(mock)
	if len(affects) != 3 {
		t.Errorf("Expected 3 affects, got %d", len(affects))
	}

	// Remove all from spell 42 at once
	removed := am.RemoveAffectsBySpell(mock, 42)
	if removed != 3 {
		t.Errorf("Expected 3 affects removed, got %d", removed)
	}
}

// TestFlagReferenceCounting tests that removing one of multiple flag-setting affects
// preserves the flag for the remaining affect
func TestFlagReferenceCounting(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Two different spells both set AFFDetectInvisible
	aff1 := NewAffectDirect(10, ApplyNone, 10, 0, AFFDetectInvisible, "detect invis 1")
	aff1.StackID = "" // No dedup — both can exist
	aff2 := NewAffectDirect(20, ApplyNone, 5, 0, AFFDetectInvisible, "detect invis 2")
	aff2.StackID = ""

	am.ApplyAffect(mock, aff1)
	am.ApplyAffect(mock, aff2)

	if !mock.HasStatusFlag(AFFDetectInvisible) {
		t.Error("Should have detect invisible flag")
	}

	// Remove first — flag should remain (second still needs it)
	am.RemoveAffect(mock, aff1.ID)
	if !mock.HasStatusFlag(AFFDetectInvisible) {
		t.Error("Flag should still be set after removing first affect")
	}

	// Remove second — flag should clear
	am.RemoveAffect(mock, aff2.ID)
	if mock.HasStatusFlag(AFFDetectInvisible) {
		t.Error("Flag should be cleared after removing all affects")
	}
}
