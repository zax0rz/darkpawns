package optimization

import (
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
