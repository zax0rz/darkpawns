package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestIsAutoLootEnabled_NilPlayer safely returns false for a nil player.
func TestIsAutoLootEnabled_NilPlayer(t *testing.T) {
	if IsAutoLootEnabled(nil) {
		t.Error("IsAutoLootEnabled(nil) should be false")
	}
}

// TestIsAutoLootEnabled_UsesExistingFlag verifies that the helper reflects the
// same PrfAutoLoot flag used by cmdToggle.
func TestIsAutoLootEnabled_UsesExistingFlag(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Looter", 1001, true)

	// Enable the flag directly.
	s.player.SetPlrFlag(game.PrfAutoLoot, true)
	if !IsAutoLootEnabled(s.player) {
		t.Error("IsAutoLootEnabled should reflect PrfAutoLoot when set directly")
	}

	// Disable the flag directly.
	s.player.SetPlrFlag(game.PrfAutoLoot, false)
	if IsAutoLootEnabled(s.player) {
		t.Error("IsAutoLootEnabled should reflect PrfAutoLoot when cleared directly")
	}
}
