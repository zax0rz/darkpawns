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

// TestPermanentAffect tests that permanent affects (duration -1, GODs only)
// never expire — the C sentinel from src/magic.c affect_update().

// TestDurationZeroExpires guards DP-1013: duration 0 expires on the next update
// (C affect_update() removes anything that is not >= 1 or == -1). The engine
// previously treated duration 0 as permanent, diverging from game.AffectUpdate.

// TestNewAffect_NegativeDuration guards invalid durations < -1: -1 is permanent
// (GODs only), 0 expires immediately, and any value below -1 is invalid. Both
// constructors must clamp duration to -1 so ExpiresAt stays consistent with
// IsExpired instead of landing in the past.

// TestGetType_DeterministicMultipleFlags guards DP-1018: GetType() previously
// ranged over the StatusAffectFlags map, whose iteration order is randomized,
// so an affect with multiple AFF bits set returned a different affType on
// different calls. Resolution must now be deterministic (lowest flag value
// wins): AFFBlind (1<<0, affType 100) < AFFPoison (1<<11, affType 111).

// TestGetType_SingleFlag confirms a single-bit affect resolves to its affType.

// TestNewAffectDirectStackKey guards the non-spell dedup key: NewAffectDirect
// with spellID=0 (item/equipment affects) must not collapse every flags-based
// affect onto spellStackKey(0) == "spell_0", which made distinct statuses such
// as AFFSanctuary and AFFPoison share a StackID and collide on dedup.
func TestNewAffectDirectStackKey(t *testing.T) {
	sanctuary := NewAffectDirect(0, ApplyNone, 10, 5, AFFSanctuary, "item_a")
	poison := NewAffectDirect(0, ApplyNone, 10, 5, AFFPoison, "item_b")

	if sanctuary.StackID == "" {
		t.Error("flags-based non-spell affect should get a StackID")
	}
	if poison.StackID == "" {
		t.Error("flags-based non-spell affect should get a StackID")
	}
	if sanctuary.StackID == poison.StackID {
		t.Errorf("distinct flags must yield distinct StackIDs, got %q for both", sanctuary.StackID)
	}

	// Non-spell affects with identical flags still dedup to one key.
	twin := NewAffectDirect(0, ApplyNone, 10, 5, AFFSanctuary, "item_c")
	if twin.StackID != sanctuary.StackID {
		t.Errorf("same flags must yield the same StackID: %q vs %q", twin.StackID, sanctuary.StackID)
	}

	// Spell-based affects keep the spell stack key.
	spellAffect := NewAffectDirect(42, ApplyNone, 10, 0, AFFBlind, "blindness")
	if got, want := spellAffect.StackID, spellStackKey(42); got != want {
		t.Errorf("spell-based affect StackID = %q, want %q", got, want)
	}
}
