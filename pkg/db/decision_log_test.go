package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
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
