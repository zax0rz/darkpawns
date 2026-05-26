package game

import (
	"testing"
)

// TestUnregisteredSpecProcs verifies that all special procedures assigned to
// mobiles, objects, and rooms are actually registered in SpecRegistry.
// Any assigned but unregistered spec proc is a silent stub that does nothing at runtime.
func TestUnregisteredSpecProcs(t *testing.T) {
	assignedNames := AllSpecNames()

	// Dedup and count missing specs
	var missing []string
	for name := range assignedNames {
		if _, registered := SpecRegistry[name]; !registered {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		t.Logf("Found %d assigned but unregistered special procedures:", len(missing))
		for _, m := range missing {
			t.Logf("  - %q", m)
		}
		// Asserting this is a failure because unregistered specs represent broken MUD mechanics.
		t.Errorf("FAIL: The following spec procs are assigned but NOT registered: %v", missing)
	} else {
		t.Log("All assigned special procedures are correctly registered!")
	}
}
