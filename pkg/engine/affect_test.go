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
