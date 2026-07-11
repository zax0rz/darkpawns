package testutil

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

func TestMockDatabase_PlayerOperations(t *testing.T) {
	m := NewMockDatabase()

	// 1. Get non-existent
	p, err := m.GetPlayer("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil player, got %+v", p)
	}

	// 2. Create player
	rec := &db.PlayerRecord{
		Name:      "Zach",
		Level:     1,
		RoomVNum:  8004,
		Health:    100,
		MaxHealth: 100,
	}
	err = m.CreatePlayer(rec)
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if rec.ID != 1 {
		t.Errorf("expected ID 1, got %d", rec.ID)
	}

	// 3. Get existing
	p, err = m.GetPlayer("Zach")
	if err != nil {
		t.Fatalf("failed to get player: %v", err)
	}
	if p == nil || p.Name != "Zach" {
		t.Fatalf("invalid player record returned: %+v", p)
	}

	// 4. Duplicate name constraint check
	err = m.CreatePlayer(rec)
	if err == nil {
		t.Error("expected constraint error on duplicate player creation, got nil")
	}

	// 5. Update Password
	err = m.UpdatePassword(1, "hashed_password")
	if err != nil {
		t.Fatalf("failed to update password: %v", err)
	}
	p, _ = m.GetPlayer("Zach")
	if p.Password != "hashed_password" {
		t.Errorf("expected password 'hashed_password', got %q", p.Password)
	}
}

func TestMockDatabase_NarrativeMemory(t *testing.T) {
	m := NewMockDatabase()

	mem := &db.NarrativeMemory{
		AgentName: "Brenda",
		EventType: "mob_kill",
		Summary:   "Killed a rat.",
		Salience:  0.8,
		SessionID: "session_123",
	}

	id, err := m.WriteNarrativeMemory(mem)
	if err != nil {
		t.Fatalf("failed to write memory: %v", err)
	}
	if id != 1 {
		t.Errorf("expected memory ID 1, got %d", id)
	}

	// Bootstrap
	bootstrap, err := m.BootstrapMemories("Brenda", 5)
	if err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}
	if len(bootstrap) != 1 || bootstrap[0].Summary != "Killed a rat." {
		t.Errorf("invalid bootstrap content: %+v", bootstrap)
	}

	// Session summary
	err = m.WriteSessionSummary("Brenda", "session_123", "Consolidated play session.", 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to write summary: %v", err)
	}

	summaries, err := m.GetSessionSummaries("Brenda", 5)
	if err != nil {
		t.Fatalf("failed to get summaries: %v", err)
	}
	if len(summaries) != 1 || summaries[0] != "Consolidated play session." {
		t.Errorf("invalid summaries content: %v", summaries)
	}
}

func TestAssertBehaviorMatchesC(t *testing.T) {
	fakeT := &testing.T{}

	// Exact match does not fail
	AssertBehaviorMatchesC(fakeT, "test exact match", func() string { return "expected text" }, "expected text")
	if fakeT.Failed() {
		t.Error("expected assertion to succeed, but fakeT failed")
	}

	// Mismatch fails
	AssertBehaviorMatchesC(fakeT, "test mismatch", func() string { return "actual text" }, "expected text")
	if !fakeT.Failed() {
		t.Error("expected assertion to fail, but fakeT did not fail")
	}
}

// TestMockDatabase_NewDecisionLogWriter guards DP-1017: the mock previously
// returned nil, so any test that called RecordDecision/Stop on the result
// nil-panicked. The writer must be non-nil and safe for Stop.
func TestMockDatabase_NewDecisionLogWriter(t *testing.T) {
	m := NewMockDatabase()
	w := m.NewDecisionLogWriter()
	if w == nil {
		t.Fatal("NewDecisionLogWriter returned nil")
	}
	// Stop on empty buffers must not panic.
	w.Stop()
}
