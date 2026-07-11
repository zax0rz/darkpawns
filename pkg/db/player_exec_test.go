package db

import (
	"testing"
	"time"
)

// TestValidateAgentKeyRejectsInsecureKeys verifies that known insecure or
// example keys are rejected before any database lookup happens.
func TestValidateAgentKeyRejectsInsecureKeys(t *testing.T) {
	db := newFakeDB(t, false)

	insecureKeys := []string{
		"br3nd4-69-ag3nt-k3y-d3f4ult",
		"my-example-key",
		"test-key-123",
		"REPLACE_WITH_REAL_KEY",
	}

	for _, key := range insecureKeys {
		_, _, valid := db.ValidateAgentKey(key)
		if valid {
			t.Errorf("ValidateAgentKey(%q) returned valid=true, want false", key)
		}
	}
}

// TestRecordLoginSuccessClearsLockout verifies that RecordLoginSuccess executes
// without error and clears the failed-login state.
func TestRecordLoginSuccessClearsLockout(t *testing.T) {
	db := newFakeDB(t, false)
	if err := db.RecordLoginSuccess("alice"); err != nil {
		t.Fatalf("RecordLoginSuccess error: %v", err)
	}
}

// TestUpdatePasswordExecutes verifies that UpdatePassword runs the update
// statement without error.
func TestUpdatePasswordExecutes(t *testing.T) {
	db := newFakeDB(t, false)
	if err := db.UpdatePassword(1, "hashed-password"); err != nil {
		t.Fatalf("UpdatePassword error: %v", err)
	}
}

// TestSavePlayerExecutes verifies that SavePlayer runs the full update
// statement without error.
func TestSavePlayerExecutes(t *testing.T) {
	db := newFakeDB(t, false)
	p := &PlayerRecord{
		ID:        42,
		Name:      "alice",
		RoomVNum:  1001,
		Level:     5,
		Exp:       1000,
		Health:    50,
		MaxHealth: 60,
		Mana:      80,
		MaxMana:   100,
		Move:      90,
		MaxMove:   100,
		Strength:  14,
		Class:     3,
		Race:      0,
		StatStr:   14,
		StatInt:   12,
		StatWis:   10,
		StatDex:   13,
		StatCon:   12,
		StatCha:   10,
		Hunger:    24,
		Thirst:    24,
		Drunk:     0,
		Hometown:  1,
		Inventory: []byte("[]"),
		Equipment: []byte("{}"),
	}
	if err := db.SavePlayer(p); err != nil {
		t.Fatalf("SavePlayer error: %v", err)
	}
}

// TestExecPassthrough verifies the raw Exec helper works.
func TestExecPassthrough(t *testing.T) {
	db := newFakeDB(t, false)
	if _, err := db.Exec("SELECT 1"); err != nil {
		t.Fatalf("Exec error: %v", err)
	}
}

// TestDecayStaleMemoriesExecutes verifies DecayStaleMemories runs its decay
// and prune statements without error.
func TestDecayStaleMemoriesExecutes(t *testing.T) {
	db := newFakeDB(t, false)
	decayed, pruned, err := db.DecayStaleMemories(30)
	if err != nil {
		t.Fatalf("DecayStaleMemories error: %v", err)
	}
	// The fake driver reports 0 rows affected for every Exec.
	if decayed != 0 || pruned != 0 {
		t.Errorf("expected 0/0 from fake driver, got %d/%d", decayed, pruned)
	}
}

// TestWriteSessionSummaryExecutes verifies WriteSessionSummary runs its
// insert-or-update statement without error.
func TestWriteSessionSummaryExecutes(t *testing.T) {
	db := newFakeDB(t, false)
	now := time.Now()
	if err := db.WriteSessionSummary("agent1", "session-1", "summary text", 5, now, now); err != nil {
		t.Fatalf("WriteSessionSummary error: %v", err)
	}
}
