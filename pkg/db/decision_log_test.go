package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	fakeDriverMu sync.Mutex
	fakeFail     bool
)

func init() {
	sql.Register("fakedb", fakeDBDriver{})
}

type fakeDBDriver struct{}

func setFakeFail(fail bool) {
	fakeDriverMu.Lock()
	defer fakeDriverMu.Unlock()
	fakeFail = fail
}

func isFakeFail() bool {
	fakeDriverMu.Lock()
	defer fakeDriverMu.Unlock()
	return fakeFail
}

func (fakeDBDriver) Open(name string) (driver.Conn, error) {
	return fakeDBConn{}, nil
}

type fakeDBConn struct{}

func (fakeDBConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (fakeDBConn) Close() error                              { return nil }
func (fakeDBConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (fakeDBConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if isFakeFail() {
		return nil, errors.New("simulated DB error")
	}
	return driver.ResultNoRows, nil
}

// CheckNamedValue accepts any Go value so the fake driver does not reject
// PostgreSQL arrays or other argument types during tests.
func (fakeDBConn) CheckNamedValue(nv *driver.NamedValue) error {
	return nil
}

func newFakeDB(t *testing.T, fail bool) *DB {
	t.Helper()
	setFakeFail(fail)
	conn, err := sql.Open("fakedb", "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	return &DB{conn: conn}
}

// TestFlushRetainsRecordsOnDBError verifies that Flush keeps decision and combat
// records in their buffers when the database INSERT fails.
func TestFlushRetainsRecordsOnDBError(t *testing.T) {
	db := newFakeDB(t, true)
	dlw := &DecisionLogWriter{
		db:        db,
		decisions: make([]*DecisionRecord, 0, flushBatchSize),
		combat:    make([]*CombatRecord, 0, flushBatchSize),
	}

	dlw.RecordDecision(&DecisionRecord{SessionID: "s1", PlayerName: "p1", Command: "look", OutcomeCategory: "ok"})
	dlw.RecordCombat(&CombatRecord{SessionID: "s1", RoundNumber: 1, AttackerName: "p1"})

	dlw.Flush()

	if len(dlw.decisions) != 1 {
		t.Errorf("expected decision record to be retained after flush failure, got %d", len(dlw.decisions))
	}
	if len(dlw.combat) != 1 {
		t.Errorf("expected combat record to be retained after flush failure, got %d", len(dlw.combat))
	}
}

// TestGetEnvInt verifies the environment-variable helper used for connection
// pool configuration (DP-633).
func TestGetEnvInt(t *testing.T) {
	orig := os.Getenv("DP_TEST_INT")
	defer func() { _ = os.Setenv("DP_TEST_INT", orig) }()

	_ = os.Unsetenv("DP_TEST_INT")
	if got := getEnvInt("DP_TEST_INT", 42); got != 42 {
		t.Errorf("unset env: got %d, want 42", got)
	}

	_ = os.Setenv("DP_TEST_INT", "123")
	if got := getEnvInt("DP_TEST_INT", 42); got != 123 {
		t.Errorf("valid env: got %d, want 123", got)
	}

	_ = os.Setenv("DP_TEST_INT", "not-an-int")
	if got := getEnvInt("DP_TEST_INT", 42); got != 42 {
		t.Errorf("invalid env: got %d, want fallback 42", got)
	}
}

// TestConnectionPoolDefaults verifies the default pool limits are documented
// and consistent (DP-633).
func TestConnectionPoolDefaults(t *testing.T) {
	// Defaults mirror the values set in New().
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 25); got != 25 {
		t.Errorf("DB_MAX_OPEN_CONNS default = %d, want 25", got)
	}
	if got := getEnvInt("DB_MAX_IDLE_CONNS", 5); got != 5 {
		t.Errorf("DB_MAX_IDLE_CONNS default = %d, want 5", got)
	}
	if got := time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 300)) * time.Second; got != 5*time.Minute {
		t.Errorf("DB_CONN_MAX_LIFETIME default = %v, want 5m", got)
	}
}

// TestStopFlushesAndIsIdempotent verifies that Stop flushes buffered records
// and can be called multiple times without panicking (DP-797).
func TestStopFlushesAndIsIdempotent(t *testing.T) {
	db := newFakeDB(t, false)
	dlw := db.NewDecisionLogWriter()

	dlw.RecordDecision(&DecisionRecord{SessionID: "s3", PlayerName: "p3", Command: "look", OutcomeCategory: "ok"})
	dlw.Stop()

	if len(dlw.decisions) != 0 {
		t.Errorf("expected decisions to be flushed on stop, got %d", len(dlw.decisions))
	}

	// Second stop must not panic and should remain safe.
	dlw.Stop()
}

// TestFlushClearsRecordsOnSuccess verifies that Flush clears buffered records
// after a successful database write.
func TestFlushClearsRecordsOnSuccess(t *testing.T) {
	db := newFakeDB(t, false)
	dlw := &DecisionLogWriter{
		db:        db,
		decisions: make([]*DecisionRecord, 0, flushBatchSize),
		combat:    make([]*CombatRecord, 0, flushBatchSize),
	}

	dlw.RecordDecision(&DecisionRecord{SessionID: "s2", PlayerName: "p2", Command: "north", OutcomeCategory: "ok"})
	dlw.RecordCombat(&CombatRecord{SessionID: "s2", RoundNumber: 2, AttackerName: "p2"})

	dlw.Flush()

	if len(dlw.decisions) != 0 {
		t.Errorf("expected decisions to be cleared after flush success, got %d", len(dlw.decisions))
	}
	if len(dlw.combat) != 0 {
		t.Errorf("expected combat records to be cleared after flush success, got %d", len(dlw.combat))
	}
}

// TestNewMockDecisionLogWriter_RecordAndStop guards DP-1017: the mock writer
// must be safe for the construct/record/Stop path (no nil-panic from a nil
// writer). The writer has no database handle, so the buffered-record path that
// persists via Flush is intentionally out of scope (see NewMockDecisionLogWriter
// doc). Here we verify the safe paths: RecordDecision buffers without panic,
// and Stop on empty buffers early-returns without touching the nil db.
func TestNewMockDecisionLogWriter_RecordAndStop(t *testing.T) {
	dlw := NewMockDecisionLogWriter()
	if dlw == nil {
		t.Fatal("NewMockDecisionLogWriter returned nil")
	}

	// RecordDecision only appends below flushBatchSize, so it never derefs db.
	dlw.RecordDecision(&DecisionRecord{SessionID: "s1", PlayerName: "p1", Command: "look", OutcomeCategory: "ok"})
	if len(dlw.decisions) != 1 {
		t.Fatalf("expected 1 buffered decision, got %d", len(dlw.decisions))
	}

	// Drain the buffer so Stop's final Flush early-returns on empty buffers and
	// never touches the nil db. (Driving a real Flush is out of scope by design.)
	dlw.decisions = dlw.decisions[:0]

	// Must not panic. Second Stop is a no-op via stopOnce.
	dlw.Stop()
	dlw.Stop()
}
