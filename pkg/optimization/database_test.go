package optimization

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
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

func (rowsErrStmt) Close() error { return nil }
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

func TestBatchProcessor_FlushLoopError(t *testing.T) {
	var buf bytes.Buffer
	textHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(textHandler))
	defer slog.SetDefault(oldLogger)

	flushErr := errors.New("simulated flush failure")
	bp := NewBatchProcessor(10, 20*time.Millisecond, func(ops []BatchOperation) error {
		return flushErr
	})

	bp.Add(BatchOperation{Type: "insert", Table: "test", Data: "data"})

	// Wait long enough for the ticker to fire and flushLocked to exhaust
	// its three retries (0ms + 100ms + 400ms backoff).
	time.Sleep(700 * time.Millisecond)

	// Close before inspecting logs so the background goroutine stops writing
	// to the shared buffer.
	_ = bp.Close()

	logs := buf.String()
	if !strings.Contains(logs, "background flush failed, data may be lost") {
		t.Errorf("expected error log for background flush failure, got:\n%s", logs)
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
