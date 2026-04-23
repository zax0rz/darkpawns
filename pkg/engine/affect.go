package engine

import (
	"strconv"
	"time"
)

// AffectType represents the type of modifier an affect applies
type AffectType int

const (
	// Stat modifiers
	AffectStrength AffectType = iota
	AffectDexterity
	AffectIntelligence
	AffectWisdom
	AffectConstitution
	AffectCharisma
	
	// Combat modifiers
	AffectHitRoll
	AffectDamageRoll
	AffectArmorClass
	AffectTHAC0
	AffectHP
	AffectMaxHP
	AffectMana
	AffectMaxMana
	AffectMovement
	
	// Status effects
	AffectBlind
	AffectInvisible
	AffectDetectInvisible
	AffectDetectMagic
	AffectSanctuary
	AffectFlying
	AffectFloating
	AffectPassDoor
	AffectSneak
	AffectHide
	AffectCharm
	AffectPoison
	AffectSleep
	AffectStunned
	AffectParalyzed
	AffectFlaming
	AffectHaste
	AffectSlow
	AffectProtectionEvil
	AffectProtectionGood
	AffectFear
	AffectCurse
	AffectSilence
	AffectWaterBreathing
	AffectRegeneration
	AffectInfrared
	AffectUltraviolet
)

// Affect represents a temporary effect on a character, mob, or object
type Affect struct {
	// Core properties
	ID          string    // Unique identifier for this affect
	Type        AffectType // What type of affect this is
	Duration    int       // Duration in ticks (0 = permanent until removed)
	Magnitude   int       // Magnitude of the effect (positive for buffs, negative for debuffs)
	
	// Flags
	Flags       uint64    // Bitmask of affect flags
	
	// Metadata
	Source      string    // Source of the affect (spell name, item name, etc.)
	AppliedAt   time.Time // When the affect was applied
	ExpiresAt   time.Time // When the affect expires (calculated from Duration)
	
	// Stacking information
	StackID     string    // ID for stacking purposes (empty = doesn't stack)
	MaxStacks   int       // Maximum number of stacks (0 = infinite, 1 = doesn't stack)
}

// NewAffect creates a new affect with the given parameters
func NewAffect(affectType AffectType, duration int, magnitude int, source string) *Affect {
	now := time.Now()
	affect := &Affect{
		ID:        generateAffectID(),
		Type:      affectType,
		Duration:  duration,
		Magnitude: magnitude,
		Source:    source,
		AppliedAt: now,
		ExpiresAt: now.Add(time.Duration(duration) * time.Second), // Assuming 1 tick = 1 second for now
		Flags:     0,
		StackID:   "", // Default: doesn't stack
		MaxStacks: 1,
	}
	
	// Set default StackID for certain affect types
	switch affectType {
	case AffectPoison, AffectHaste, AffectSlow, AffectRegeneration:
		affect.StackID = strconv.Itoa(int(affectType))
		affect.MaxStacks = 1 // Most effects don't stack with themselves
	}
	
	return affect
}

// IsExpired checks if the affect has expired
func (a *Affect) IsExpired() bool {
	if a.Duration == 0 {
		return false // Permanent
	}
	return time.Now().After(a.ExpiresAt)
}

// Tick reduces the duration by one tick and returns true if expired
func (a *Affect) Tick() bool {
	if a.Duration == 0 {
		return false // Permanent, never expires
	}
	
	// Reduce duration
	a.Duration--
	if a.Duration <= 0 {
		return true
	}
	
	// Update expiration time
	a.ExpiresAt = a.ExpiresAt.Add(-time.Second)
	return false
}

// GetRemainingDuration returns the remaining duration in ticks
func (a *Affect) GetRemainingDuration() int {
	return a.Duration
}

// SetFlag sets a specific flag
func (a *Affect) SetFlag(flag uint64) {
	a.Flags |= flag
}

// ClearFlag clears a specific flag
func (a *Affect) ClearFlag(flag uint64) {
	a.Flags &^= flag
}

// HasFlag checks if a specific flag is set
func (a *Affect) HasFlag(flag uint64) bool {
	return a.Flags&flag != 0
}

// Helper function to generate a unique affect ID
func generateAffectID() string {
	return "aff_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// Helper function to generate a random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		// Simple pseudo-random for demo purposes
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}