package session

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdAutoLoot_TogglesPrfAutoLoot verifies that the autoloot command
// flips the player's PrfAutoLoot preference flag and reports the new state.
func TestCmdAutoLoot_TogglesPrfAutoLoot(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Looter", 1001, true)

	if IsAutoLootEnabled(s.player) {
		t.Fatal("autoloot should be disabled by default")
	}

	if err := cmdAutoLoot(s, nil); err != nil {
		t.Fatalf("cmdAutoLoot failed: %v", err)
	}
	if !IsAutoLootEnabled(s.player) {
		t.Error("autoloot should be enabled after toggle")
	}

	if err := cmdAutoLoot(s, nil); err != nil {
		t.Fatalf("cmdAutoLoot failed: %v", err)
	}
	if IsAutoLootEnabled(s.player) {
		t.Error("autoloot should be disabled after second toggle")
	}
}

// TestIsAutoLootEnabled_NilPlayer safely returns false for a nil player.
func TestIsAutoLootEnabled_NilPlayer(t *testing.T) {
	if IsAutoLootEnabled(nil) {
		t.Error("IsAutoLootEnabled(nil) should be false")
	}
}

// TestCmdAutoLoot_NilPlayer does not panic when the session has no player.
func TestCmdAutoLoot_NilPlayer(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Ghost", 1001, true)
	s.player = nil

	if err := cmdAutoLoot(s, nil); err != nil {
		t.Fatalf("cmdAutoLoot with nil player failed: %v", err)
	}
}

// TestCmdAutoLoot_ConcurrentToggleRead verifies that concurrent toggles and
// reads do not race. Run with -race to validate.
func TestCmdAutoLoot_ConcurrentToggleRead(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Looter", 1001, true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = cmdAutoLoot(s, nil)
		}()
		go func() {
			defer wg.Done()
			_ = IsAutoLootEnabled(s.player)
		}()
	}
	wg.Wait()
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
