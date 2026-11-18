package session

import (
	"testing"
	"time"
)

// A login that takes over a registered session closes the old one inside
// register() while holding m.mu. Closing a descriptor untracks it from the
// users descriptor set; if that took m.mu too, the manager would deadlock and
// every later login would hang. Found by entry-deleted-menu-reuse, which
// re-registers a name whose old session is still in m.sessions.
func TestRegisterTakeoverDoesNotDeadlockOnDescriptorUntrack(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)

	old := m.NewSession()
	if err := m.register("Takeover", old); err != nil {
		t.Fatalf("register old: %v", err)
	}
	fresh := m.NewSession()

	done := make(chan error, 1)
	go func() { done <- m.register("Takeover", fresh) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("register fresh: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("register deadlocked while closing the taken-over session")
	}

	for _, s := range m.activeDescriptors() {
		if s == old {
			t.Fatal("taken-over session is still listed as a live descriptor")
		}
	}
}
