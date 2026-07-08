package optimization

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestWorkerPool_WaitBlocksUntilAllTasksRun verifies that Wait() only
// returns once every task submitted before the call has actually executed,
// so callers no longer need to guess at a sleep duration before Close()
// (DP-746).
func TestWorkerPool_WaitBlocksUntilAllTasksRun(t *testing.T) {
	const numTasks = 200

	pool := NewWorkerPool(5)

	var completed int64
	for i := 0; i < numTasks; i++ {
		task := func() {
			atomic.AddInt64(&completed, 1)
		}
		// The queue is bounded (workers*10); retry on ErrPoolFull rather
		// than requiring every one of numTasks to land in one shot, since
		// nothing prevents workers from draining concurrently with Submit.
		for {
			err := pool.Submit(task)
			if err == nil {
				break
			}
			if err != ErrPoolFull {
				t.Fatalf("Submit() error = %v", err)
			}
			time.Sleep(time.Millisecond)
		}
	}

	pool.Wait()

	if got := atomic.LoadInt64(&completed); got != numTasks {
		t.Fatalf("after Wait(): completed = %d, want %d", got, numTasks)
	}

	// Close should be a safe, deterministic no-op at this point: no queued
	// work remains, so it should return promptly without losing anything.
	done := make(chan struct{})
	go func() {
		pool.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after Wait had already drained the pool")
	}

	if got := atomic.LoadInt64(&completed); got != numTasks {
		t.Fatalf("after Close(): completed = %d, want %d", got, numTasks)
	}
}

// TestWorkerPool_CloseDrainsQueuedTasks verifies that Close() alone (without
// a prior Wait) still runs every task that was successfully queued, and
// does not panic (DP-746).
func TestWorkerPool_CloseDrainsQueuedTasks(t *testing.T) {
	// Stay within the pool's buffered queue capacity (workers*10) so every
	// Submit below succeeds even though nothing is draining the queue yet;
	// the point of this test is that Close() itself drains what's queued.
	const numTasks = 30

	pool := NewWorkerPool(4)

	var completed int64
	for i := 0; i < numTasks; i++ {
		if err := pool.Submit(func() {
			time.Sleep(time.Millisecond)
			atomic.AddInt64(&completed, 1)
		}); err != nil {
			t.Fatalf("Submit() error = %v", err)
		}
	}

	pool.Close()

	if got := atomic.LoadInt64(&completed); got != numTasks {
		t.Fatalf("completed = %d, want %d", got, numTasks)
	}

	// Submitting after Close must fail cleanly rather than panic.
	if err := pool.Submit(func() {}); err != ErrPoolClosed {
		t.Fatalf("Submit() after Close error = %v, want ErrPoolClosed", err)
	}

	// Close must also be idempotent.
	pool.Close()
}
