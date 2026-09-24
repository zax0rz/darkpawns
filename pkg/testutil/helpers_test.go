package testutil

import (
	"testing"

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
