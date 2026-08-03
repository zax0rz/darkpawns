package engine

import (
	"testing"
)

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

// TestPermanentAffect tests that permanent affects (duration -1, GODs only)
// never expire — the C sentinel from src/magic.c affect_update().
func TestPermanentAffect(t *testing.T) {
	affect := NewAffect(0, ApplyStr, -1, 5, "permanent")

	for i := 0; i < 10; i++ {
		expired := affect.Tick()
		if expired {
			t.Error("Permanent affect (duration -1) should never expire")
		}
		if affect.Duration != -1 {
			t.Errorf("Permanent affect duration should remain -1, got %d", affect.Duration)
		}
	}
	if affect.IsExpired() {
		t.Error("Permanent affect (duration -1) should not report IsExpired")
	}
}

// TestDurationZeroExpires guards DP-1013: duration 0 expires on the next update
// (C affect_update() removes anything that is not >= 1 or == -1). The engine
// previously treated duration 0 as permanent, diverging from game.AffectUpdate.
func TestDurationZeroExpires(t *testing.T) {
	affect := NewAffect(0, ApplyStr, 0, 5, "zero")
	if !affect.Tick() {
		t.Error("duration-0 affect should expire on tick (C else-branch → affect_remove)")
	}
	if !affect.IsExpired() {
		t.Error("duration-0 affect should report IsExpired")
	}
}

// TestNewAffect_NegativeDuration guards invalid durations < -1: -1 is permanent
// (GODs only), 0 expires immediately, and any value below -1 is invalid. Both
// constructors must clamp duration to -1 so ExpiresAt stays consistent with
// IsExpired instead of landing in the past.
func TestNewAffect_NegativeDuration(t *testing.T) {
	affect := NewAffect(0, ApplyStr, -2, 5, "bad")
	if affect.Duration != -1 {
		t.Errorf("NewAffect: duration -2 should be clamped to -1, got %d", affect.Duration)
	}
	if affect.IsExpired() {
		t.Error("clamped permanent affect should not report IsExpired")
	}

	direct := NewAffectDirect(0, ApplyStr, -2, 5, 0, "bad")
	if direct.Duration != -1 {
		t.Errorf("NewAffectDirect: duration -2 should be clamped to -1, got %d", direct.Duration)
	}
	if direct.IsExpired() {
		t.Error("clamped permanent direct affect should not report IsExpired")
	}
}

// TestGetType_DeterministicMultipleFlags guards DP-1018: GetType() previously
// ranged over the StatusAffectFlags map, whose iteration order is randomized,
// so an affect with multiple AFF bits set returned a different affType on
// different calls. Resolution must now be deterministic (lowest flag value
// wins): AFFBlind (1<<0, affType 100) < AFFPoison (1<<11, affType 111).
func TestGetType_DeterministicMultipleFlags(t *testing.T) {
	a := &Affect{Flags: AFFBlind | AFFPoison}

	first := a.GetType()
	if first != 100 {
		t.Fatalf("GetType() = %d, want 100 (AFFBlind, lowest flag)", first)
	}
	// Repeated calls must agree — map-iteration nondeterminism would surface here.
	for i := 0; i < 1000; i++ {
		if got := a.GetType(); got != first {
			t.Fatalf("GetType() nondeterministic: got %d on call %d, want %d", got, i, first)
		}
	}
}

// TestGetType_SingleFlag confirms a single-bit affect resolves to its affType.
func TestGetType_SingleFlag(t *testing.T) {
	a := &Affect{Flags: AFFPoison} // 1<<11 → affType 111
	if got := a.GetType(); got != 111 {
		t.Fatalf("GetType() = %d, want 111 (AFFPoison)", got)
	}
}
