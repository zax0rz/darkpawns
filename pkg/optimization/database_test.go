package optimization

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestQueryOptimizerStatsAreCopies ensures GetStats and GetSlowQueries return
// copies of the internal QueryStat objects, so external mutation does not
// corrupt optimizer state.
func TestQueryOptimizerStatsAreCopies(t *testing.T) {
	qo := NewQueryOptimizer(10, 1*time.Millisecond)

	qo.RecordQuery("SELECT 1", 5*time.Millisecond, true)

	stats := qo.GetStats()
	stat := stats["SELECT 1"]
	if stat == nil {
		t.Fatal("expected stat for SELECT 1")
	}
	stat.Count = 999

	slow := qo.GetSlowQueries()
	if len(slow) != 1 {
		t.Fatalf("expected 1 slow query, got %d", len(slow))
	}
	// Mutate the returned slow-query pointer.
	slow[0].AvgDuration = 0

	fresh := qo.GetStats()["SELECT 1"]
	if fresh.Count != 1 {
		t.Errorf("internal Count mutated through returned pointer: got %d, want 1", fresh.Count)
	}
	if fresh.AvgDuration != 5*time.Millisecond {
		t.Errorf("internal AvgDuration mutated through returned pointer: got %v, want %v", fresh.AvgDuration, 5*time.Millisecond)
	}
}

// TestQueryOptimizerStatsConcurrency exercises RecordQuery concurrently with
// the read-only snapshot APIs. Run with -race to detect data races.
func TestQueryOptimizerStatsConcurrency(t *testing.T) {
	qo := NewQueryOptimizer(100, 10*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				qo.RecordQuery("SELECT concurrency", time.Duration(j)*time.Microsecond, true)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = qo.GetStats()
				_ = qo.GetSlowQueries()
			}
		}()
	}
	wg.Wait()
}

// rowsErrDriver is a minimal sql.Driver that returns rows whose Next call
// fails, causing rows.Err() to surface the error after iteration.
type rowsErrDriver struct{}

type rowsErrConn struct{}

type rowsErrStmt struct{}

type rowsErrRows struct{}

func init() {
	sql.Register("rowsErrDriver", rowsErrDriver{})
}

func (rowsErrDriver) Open(name string) (driver.Conn, error) { return rowsErrConn{}, nil }

func (rowsErrConn) Prepare(query string) (driver.Stmt, error) { return rowsErrStmt{}, nil }
func (rowsErrConn) Close() error                              { return nil }
func (rowsErrConn) Begin() (driver.Tx, error)                 { return nil, errors.New("not supported") }

func (rowsErrStmt) Close() error  { return nil }
func (rowsErrStmt) NumInput() int { return 1 }
func (rowsErrStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("not supported")
}
func (rowsErrStmt) Query(args []driver.Value) (driver.Rows, error) { return rowsErrRows{}, nil }

func (rowsErrRows) Columns() []string {
	return []string{"attname", "most_common_vals", "most_common_freqs", "histogram_bounds", "correlation"}
}
func (rowsErrRows) Close() error { return nil }
func (rowsErrRows) Next(dest []driver.Value) error {
	return errors.New("simulated rows iteration error")
}

func TestAnalyzeTable_RowsError(t *testing.T) {
	db, err := sql.Open("rowsErrDriver", "")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	defer db.Close()

	ia := NewIndexAnalyzer(db)
	_, err = ia.AnalyzeTable("players")
	if err == nil {
		t.Fatal("AnalyzeTable expected error, got nil")
	}
	if !strings.Contains(err.Error(), "iterating pg_stats rows") {
		t.Errorf("expected error to mention 'iterating pg_stats rows', got: %v", err)
	}
}

// TestBatchProcessor_FlushLoopRequeuesOnFailure verifies that when a background
// (timer-driven) flush fails, the batch is requeued and retried rather than
// silently dropped — at-least-once semantics for the async path.
func TestBatchProcessor_FlushLoopRequeuesOnFailure(t *testing.T) {
	var mu sync.Mutex
	var calls int
	var delivered []BatchOperation

	// Fail the entire first flush cycle (flushLocked retries 3× internally),
	// then succeed. If the failed batch were dropped, it would never arrive.
	bp := NewBatchProcessor(10, 20*time.Millisecond, func(ops []BatchOperation) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls <= 3 {
			return errors.New("simulated flush failure")
		}
		delivered = append(delivered, ops...)
		return nil
	})
	defer bp.Close()

	bp.Add(BatchOperation{Type: "insert", Table: "test", Data: "data"})

	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		n := len(delivered)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("operation never delivered — requeue dropped the batch")
		case <-time.After(20 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if delivered[0].Data != "data" {
		t.Errorf("wrong operation delivered after requeue: %+v", delivered[0])
	}
}

func TestBatchProcessor_CloseReturnsError(t *testing.T) {
	flushErr := errors.New("simulated flush failure")
	bp := NewBatchProcessor(10, time.Hour, func(ops []BatchOperation) error {
		return flushErr
	})

	bp.Add(BatchOperation{Type: "insert", Table: "test", Data: "data"})

	if err := bp.Close(); err != flushErr {
		t.Errorf("Close() expected %v, got %v", flushErr, err)
	}
}
