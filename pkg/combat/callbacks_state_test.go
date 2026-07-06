package combat

import "testing"

// TestCbGetRace_PrefersCallbacks verifies that a PR2 state hook reads from
// GameCallbacks when wired and falls back to the legacy package variable when
// the callback is nil.
func TestCbGetRace_PrefersCallbacks(t *testing.T) {
	origCallbacks := callbacks
	origGetRace := GetRace
	defer func() {
		callbacks = origCallbacks
		GetRace = origGetRace
	}()

	// Callback path.
	callbacks = &GameCallbacks{
		GetRace: func(name string) int {
			if name == "Alice" {
				return 7
			}
			return 0
		},
	}
	GetRace = nil
	if got := cbGetRace("Alice"); got != 7 {
		t.Errorf("cbGetRace(callbacks) = %d, want 7", got)
	}

	// Fallback path: callback nil, legacy var set.
	callbacks = &GameCallbacks{}
	GetRace = func(name string) int {
		if name == "Bob" {
			return 3
		}
		return 0
	}
	if got := cbGetRace("Bob"); got != 3 {
		t.Errorf("cbGetRace(legacy fallback) = %d, want 3", got)
	}

	// Default path: nothing wired.
	callbacks = nil
	GetRace = nil
	if got := cbGetRace("Carol"); got != 0 {
		t.Errorf("cbGetRace(default) = %d, want 0", got)
	}
}
