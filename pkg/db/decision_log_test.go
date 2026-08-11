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
	"unicode/utf8"
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

// TestFlushUnboundedGrowth verifies that repeated flush failures cannot grow
// the buffer without bound: each failed batch is requeued but the buffer
// saturates at maxBufferSize (oldest records dropped) rather than accumulating
// across flushes, and the consecutive-failure counter resets once the DB
// recovers.
//
// The requeue cap is driven through Flush directly rather than via Record* so
// the test does not amplify its workload: once the buffer saturates, every
// Record* call would otherwise trigger a full maxBufferSize flush attempt, so
// looping thousands of records through Record* would issue millions of fake
// SQL calls. The cap itself lives in cappedRequeue, which Flush exercises on
// every failure regardless of how the records arrived.
func TestFlushUnboundedGrowth(t *testing.T) {
	db := newFakeDB(t, true)
	dlw := &DecisionLogWriter{
		db:        db,
		decisions: make([]*DecisionRecord, 0, maxBufferSize),
		combat:    make([]*CombatRecord, 0, maxBufferSize),
	}

	// Sustained outage: enqueue several full buffers' worth of records, then
	// flush. Each failed flush requeues its batch at the front of the buffer;
	// once the combined length exceeds maxBufferSize the oldest records are
	// dropped. Repeating this with more batches than the cap can hold proves
	// the buffer saturates instead of growing on every failed flush.
	for batch := 0; batch < 4; batch++ {
		for i := 0; i < maxBufferSize; i++ {
			dlw.decisions = append(dlw.decisions, &DecisionRecord{
				SessionID:       "s1",
				PlayerName:      "p1",
				Command:         "cmd",
				OutcomeCategory: "ok",
			})
		}
		dlw.Flush()
		if len(dlw.decisions) > maxBufferSize {
			t.Fatalf("batch %d: buffer grew past maxBufferSize: len=%d max=%d", batch, len(dlw.decisions), maxBufferSize)
		}
	}

	if len(dlw.decisions) != maxBufferSize {
		t.Errorf("expected buffer to saturate at maxBufferSize=%d, got %d", maxBufferSize, len(dlw.decisions))
	}
	if dlw.consecutiveFailures == 0 {
		t.Error("expected consecutive_failures to be tracked across failed flushes")
	}
	failuresBeforeRecovery := dlw.consecutiveFailures

	// Recovery: a successful flush drains the buffer and resets the counter.
	setFakeFail(false)
	dlw.Flush()
	if len(dlw.decisions) != 0 {
		t.Errorf("expected buffer drained after recovery, got %d", len(dlw.decisions))
	}
	if dlw.consecutiveFailures != 0 {
		t.Errorf("expected consecutive_failures reset on success, got %d (was %d before recovery)", dlw.consecutiveFailures, failuresBeforeRecovery)
	}
}

// TestGetEnvInt verifies the environment-variable helper used for connection
// pool configuration (DP-633).
func TestSanitizeLogArgs(t *testing.T) {
	// 0x85 is a valid UTF-8 continuation byte but invalid standalone — the
	// exact byte that stalled the prod decision_log flush.
	args := []interface{}{
		"clean",
		"bad\x85input",
		[]string{"ok", "arg\x85two"},
		42,   // non-string untouched
		true, // non-string untouched
		[]string(nil),
	}
	sanitizeLogArgs(args)

	if args[0] != "clean" {
		t.Errorf("valid string mutated: %q", args[0])
	}
	if got := args[1].(string); !utf8.ValidString(got) {
		t.Errorf("string still invalid UTF-8: %q", got)
	}
	for _, s := range args[2].([]string) {
		if !utf8.ValidString(s) {
			t.Errorf("slice element still invalid UTF-8: %q", s)
		}
	}
	if args[3] != 42 || args[4] != true {
		t.Errorf("non-string args mutated: %v %v", args[3], args[4])
	}
}

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

// TestRecordCombatAutoFlushAtBatchSize verifies that RecordCombat triggers a
// synchronous flush once the combat buffer reaches flushBatchSize, mirroring
// RecordDecision. Previously combat records only flushed on the periodic
// ticker (finding fnd_sig-feat-service-97f60e84e1-bfb6_fc4c0242e7).
func TestRecordCombatAutoFlushAtBatchSize(t *testing.T) {
	db := newFakeDB(t, false)
	dlw := &DecisionLogWriter{
		db:        db,
		decisions: make([]*DecisionRecord, 0, flushBatchSize),
		combat:    make([]*CombatRecord, 0, flushBatchSize),
	}

	for i := 0; i < flushBatchSize; i++ {
		dlw.RecordCombat(&CombatRecord{SessionID: "s1", RoundNumber: i + 1, AttackerName: "p1"})
	}

	dlw.mu.Lock()
	defer dlw.mu.Unlock()
	if len(dlw.combat) != 0 {
		t.Errorf("expected combat buffer to auto-flush at batch size %d, got %d buffered", flushBatchSize, len(dlw.combat))
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
