package session

import (
	"testing"
)

// TestGrabAliasResolvesToHold verifies that the 'grab' command is registered
// as an alias for 'hold' (DP-639).
func TestGrabAliasResolvesToHold(t *testing.T) {
	holdEntry, ok := cmdRegistry.Lookup("hold")
	if !ok {
		t.Fatal("'hold' command not found in registry")
	}

	grabEntry, ok := cmdRegistry.Lookup("grab")
	if !ok {
		t.Fatal("'grab' alias not found in registry")
	}

	if grabEntry != holdEntry {
		t.Error("'grab' does not resolve to the same command entry as 'hold'")
	}

	if grabEntry.Name != "hold" {
		t.Errorf("grab entry primary name = %q, want 'hold'", grabEntry.Name)
	}
}
