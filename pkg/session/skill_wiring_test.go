package session

import "testing"

// TestBucketASkillsRegistered guards the port-completion wiring: skill handlers
// that exist in pkg/command must stay reachable through the command registry.
// These were all implemented but never registered (see
// docs/port-reachability-map.md, Bucket A) — this test fails if any falls back
// out of the registry.
func TestBucketASkillsRegistered(t *testing.T) {
	wired := []string{
		"bearhug", "behead", "bite", "carve", "compare", "cutthroat",
		"disarm", "groinrip", "mindlink", "palm", "point", "scrounge",
		"sharpen", "slug", "smackheads", "strike", "tag", "turn",
		"aid", "alter", "flesh", "serpent", "scan",
		// martial-arts aliases mapped to their C command names
		"dragon", "tiger",
	}
	for _, name := range wired {
		if _, ok := cmdRegistry.Lookup(name); !ok {
			t.Errorf("command %q is not registered (port-completion regression)", name)
		}
	}
}
