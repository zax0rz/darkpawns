package moderation

import (
	"database/sql"
	"testing"
	"time"
)

// newClosedDBManager returns a Manager whose DB is open but closed, so every
// Exec fails with "sql: database is closed". This exercises the persistence
// error path without a live database. The pq driver is registered package-wide.
func newClosedDBManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("postgres", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}
	return &Manager{
		db:              db,
		hasDB:           true,
		activePenalties: make(map[string][]PlayerPenalty),
		wordFilters:     make([]WordFilterEntry, 0),
		messageHistory:  make(map[string][]time.Time),
	}
}

// TestAddPenalty_DBFailureReturnsError verifies AddPenalty surfaces a DB write
// failure (so cmdBan/cmdMute can warn the admin) while still applying the
// penalty in memory. Regression guard for #45 (silently swallowed DB errors).
func TestAddPenalty_DBFailureReturnsError(t *testing.T) {
	m := newClosedDBManager(t)
	err := m.AddPenalty(PlayerPenalty{
		PlayerName:  "villain",
		PenaltyType: ActionBan,
		IssuedAt:    time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when DB write fails, got nil")
	}
	if got := len(m.GetPlayerPenalties("villain")); got != 1 {
		t.Errorf("penalty should still be applied in memory on DB failure, got %d", got)
	}
}

// TestAddWordFilter_DBFailureReturnsError verifies AddWordFilter surfaces a DB
// write failure while keeping the filter active in memory.
func TestAddWordFilter_DBFailureReturnsError(t *testing.T) {
	m := newClosedDBManager(t)
	err := m.AddWordFilter("badword", false, "censor", "admin")
	if err == nil {
		t.Fatal("expected error when DB write fails, got nil")
	}
	if got := len(m.GetWordFilters()); got != 1 {
		t.Errorf("filter should still be active in memory on DB failure, got %d", got)
	}
}

// TestAddReport_DBFailureReturnsError verifies AddReport surfaces a DB write
// failure so the report command can warn admins.
func TestAddReport_DBFailureReturnsError(t *testing.T) {
	m := newClosedDBManager(t)
	err := m.AddReport(AbuseReport{
		Reporter:   "alice",
		Target:     "bob",
		ReportType: ReportTypeHarassment,
		Timestamp:  time.Now(),
		Status:     ReportStatusPending,
	})
	if err == nil {
		t.Fatal("expected error when DB write fails, got nil")
	}
}

// TestRemoveWordFilter_DBFailureKeepsMemoryEntry verifies RemoveWordFilter
// preserves the in-memory entry when the DB delete fails, keeping the slice in
// sync with the database so the filter does not reappear after a restart.
func TestRemoveWordFilter_DBFailureKeepsMemoryEntry(t *testing.T) {
	m := newClosedDBManager(t)
	m.wordFilters = []WordFilterEntry{
		{ID: 7, Pattern: "badword", Action: FilterActionCensor},
	}

	m.RemoveWordFilter(7)

	filters := m.GetWordFilters()
	if len(filters) != 1 {
		t.Fatalf("expected filter to remain in memory on DB failure, got %d filter(s)", len(filters))
	}
	if filters[0].ID != 7 {
		t.Errorf("remaining filter ID = %d, want 7", filters[0].ID)
	}
}

// TestRemoveWordFilter_NoDBRemovesMemoryEntry confirms the no-DB path removes
// the in-memory entry (there is no DB row to stay in sync with).
func TestRemoveWordFilter_NoDBRemovesMemoryEntry(t *testing.T) {
	m := NewManager(nil)
	m.wordFilters = []WordFilterEntry{
		{ID: 7, Pattern: "badword", Action: FilterActionCensor},
	}

	m.RemoveWordFilter(7)

	if got := len(m.GetWordFilters()); got != 0 {
		t.Errorf("expected filter removed from memory, got %d filter(s)", got)
	}
}

// TestAddPenalty_NoDBReturnsNil confirms the no-DB path reports success.
func TestAddPenalty_NoDBReturnsNil(t *testing.T) {
	m := NewManager(nil)
	if err := m.AddPenalty(PlayerPenalty{
		PlayerName:  "villain",
		PenaltyType: ActionMute,
		IssuedAt:    time.Now(),
	}); err != nil {
		t.Errorf("AddPenalty with no DB should return nil, got %v", err)
	}
}
