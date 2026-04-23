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
func (m *MockAffectable) IsNPC() bool { return false }

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

// TestNewAffect tests creating a new affect
func TestNewAffect(t *testing.T) {
	affect := NewAffect(AffectStrength, 10, 5, "test spell")

	if affect.Type != AffectStrength {
		t.Errorf("Expected affect type AffectStrength, got %v", affect.Type)
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
	affect := NewAffect(AffectStrength, 3, 5, "test")

	// Tick once
	expired := affect.Tick()
	if expired {
		t.Error("Affect should not expire after first tick")
	}
	if affect.Duration != 2 {
		t.Errorf("Expected duration 2 after tick, got %d", affect.Duration)
	}

	// Tick twice more
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
	affect := NewAffect(AffectStrength, 0, 5, "permanent")

	// Permanent affect should not expire
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

// TestApplyAffect tests applying an affect to an entity
func TestApplyAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffect(AffectStrength, 10, 5, "strength potion")
	success := am.ApplyAffect(mock, affect)

	if !success {
		t.Error("Failed to apply affect")
	}

	// Check that strength was modified
	if mock.GetStrength() != 15 {
		t.Errorf("Expected strength 15 after affect, got %d", mock.GetStrength())
	}

	// Check that message was sent
	messages := mock.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected affect application message")
	}
}

// TestRemoveAffect tests removing an affect from an entity
func TestRemoveAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	affect := NewAffect(AffectStrength, 10, 5, "strength potion")
	am.ApplyAffect(mock, affect)

	// Clear messages
	mock.ClearMessages()

	// Remove the affect
	success := am.RemoveAffect(mock, affect.ID)
	if !success {
		t.Error("Failed to remove affect")
	}

	// Check that strength was restored
	if mock.GetStrength() != 10 {
		t.Errorf("Expected strength 10 after affect removal, got %d", mock.GetStrength())
	}

	// Check that removal message was sent
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
	affect1 := NewAffect(AffectPoison, 10, 5, "poison")
	affect1.StackID = "poison"
	affect1.MaxStacks = 1

	affect2 := NewAffect(AffectPoison, 5, 3, "stronger poison")
	affect2.StackID = "poison"
	affect2.MaxStacks = 1

	// Apply first affect
	am.ApplyAffect(mock, affect1)
	if !mock.HasStatusFlag(1 << 11) { // Poison flag
		t.Error("First poison affect should set poison flag")
	}

	// Apply second affect (should replace first)
	am.ApplyAffect(mock, affect2)

	// Check that only one poison affect exists
	affects := am.GetAffects(mock)
	poisonCount := 0
	for _, aff := range affects {
		if aff.Type == AffectPoison {
			poisonCount++
		}
	}

	if poisonCount != 1 {
		t.Errorf("Expected 1 poison affect after stacking, got %d", poisonCount)
	}
}

// TestStatusAffect tests status affect application and removal
func TestStatusAffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Apply invisible affect
	affect := NewAffect(AffectInvisible, 10, 0, "invisibility spell")
	am.ApplyAffect(mock, affect)

	// Check that invisible flag is set
	if !mock.HasStatusFlag(1 << 1) { // Invisible flag
		t.Error("Invisible affect should set invisible flag")
	}

	// Remove affect
	am.RemoveAffect(mock, affect.ID)

	// Check that invisible flag is cleared
	if mock.HasStatusFlag(1 << 1) {
		t.Error("Invisible flag should be cleared after affect removal")
	}
}

// TestPeriodicEffect tests periodic effects like poison
func TestPeriodicEffect(t *testing.T) {
	am := NewAffectManager()
	mock := NewMockAffectable("test", 1)
	am.RegisterEntity(mock)

	// Apply poison affect
	affect := NewAffect(AffectPoison, 3, 5, "poison")
	am.ApplyAffect(mock, affect)

	initialHP := mock.GetHP()

	// Process ticks (poison should damage each tick)
	am.Tick() // Tick 1
	am.Tick() // Tick 2
	am.Tick() // Tick 3 (should expire)

	// Poison should have done damage each tick
	// Default poison damage is 1 per tick if magnitude is 0
	expectedHP := initialHP - 3 // 3 ticks of poison damage
	if mock.GetHP() != expectedHP {
		t.Errorf("Expected HP %d after poison, got %d", expectedHP, mock.GetHP())
	}

	// Check that poison affect was removed after expiration
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

	// Set HP to less than max
	mock.SetHP(50)

	// Apply regeneration affect
	affect := NewAffect(AffectRegeneration, 3, 5, "regeneration")
	am.ApplyAffect(mock, affect)

	// Process ticks (regeneration should heal each tick)
	am.Tick() // Tick 1
	am.Tick() // Tick 2
	am.Tick() // Tick 3 (should expire)

	// Regeneration should have healed each tick
	// Default regeneration is 1 per tick if magnitude is 0
	expectedHP := 50 + 3 // 3 ticks of regeneration
	if mock.GetHP() != expectedHP {
		t.Errorf("Expected HP %d after regeneration, got %d", expectedHP, mock.GetHP())
	}
}

// TestTickManager tests the tick manager
func TestTickManager(t *testing.T) {
	am := NewAffectManager()
	tm := NewTickManager(am)

	// Set a very short tick interval for testing
	tm.SetTickInterval(10 * time.Millisecond)

	// Start the tick manager
	tm.Start()

	// Give it time to process a few ticks
	time.Sleep(50 * time.Millisecond)

	// Stop the tick manager
	tm.Stop()

	// Manual tick should still work
	tm.ManualTick()

	if !tm.IsRunning() {
		t.Error("Tick manager should report running while active")
	}
}

// TestAffectTickSystem tests the combined affect tick system
func TestAffectTickSystem(t *testing.T) {
	ats := NewAffectTickSystem()
	mock := NewMockAffectable("test", 1)

	// Apply an affect
	affect := NewAffect(AffectStrength, 5, 3, "test")
	ats.ApplyAffect(mock, affect)

	// Check that affect was applied
	if mock.GetStrength() != 13 {
		t.Errorf("Expected strength 13, got %d", mock.GetStrength())
	}

	// Get affects
	affects := ats.GetAffects(mock)
	if len(affects) != 1 {
		t.Errorf("Expected 1 affect, got %d", len(affects))
	}

	// Check has affect
	if !ats.HasAffect(mock, AffectStrength) {
		t.Error("Should have strength affect")
	}

	// Remove affect
	ats.RemoveAffect(mock, affect.ID)
	if mock.GetStrength() != 10 {
		t.Errorf("Expected strength 10 after removal, got %d", mock.GetStrength())
	}

	// Manual tick
	ats.ManualTick()
}
