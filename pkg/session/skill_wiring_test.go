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

// TestBucketEToggleCommandsRegistered guards the port-completion wiring for
// preference toggles (see docs/port-reachability-map.md, Bucket E):
// src/interpreter.c registers each of these as its own top-level command
// routed through do_gen_tog — not through a unified "toggle <name>"
// dispatcher. Before this fix only the literal, never-typed "gentog" command
// name reached the (correct) game.doGenTog logic.
func TestBucketEToggleCommandsRegistered(t *testing.T) {
	wired := []string{
		"nosummon", "nohassle", "brief", "compact", "notell", "noauction",
		"noshout", "nogossip", "nograts", "nowiz", "quest", "roomflags",
		"norepeat", "holylight", "nonewbie", "noctell", "nobroadcast",
	}
	for _, name := range wired {
		if _, ok := cmdRegistry.Lookup(name); !ok {
			t.Errorf("toggle command %q is not registered (port-completion regression)", name)
		}
	}
	if _, ok := cmdRegistry.Lookup("gentog"); ok {
		t.Errorf("dead %q alias should have been removed once real toggle names were wired", "gentog")
	}
}
