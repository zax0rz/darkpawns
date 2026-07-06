package combat

import "testing"

// TestCbGetRace_PrefersCallbacks verifies that a PR2 state hook reads from
// GameCallbacks when wired and returns the zero value when the callback is nil.
func TestCbGetRace_PrefersCallbacks(t *testing.T) {
	origCallbacks := callbacks
	defer func() { callbacks = origCallbacks }()

	// Callback path.
	callbacks = &GameCallbacks{
		GetRace: func(name string) int {
			if name == "Alice" {
				return 7
			}
			return 0
		},
	}
	if got := cbGetRace("Alice"); got != 7 {
		t.Errorf("cbGetRace(callbacks) = %d, want 7", got)
	}

	// Default path: callback nil.
	callbacks = &GameCallbacks{}
	if got := cbGetRace("Bob"); got != 0 {
		t.Errorf("cbGetRace(default) = %d, want 0", got)
	}

	// Default path: no callbacks instance.
	callbacks = nil
	if got := cbGetRace("Carol"); got != 0 {
		t.Errorf("cbGetRace(default) = %d, want 0", got)
	}
}
