package optimization

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAdvancedWorkerPoolCloseWhileTaskRunningDoesNotDeadlock(t *testing.T) {
	pool := NewAdvancedWorkerPool(1, 1)

	started := make(chan struct{})
	release := make(chan struct{})

	if err := pool.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	<-started

	closed := make(chan struct{})
	go func() {
		pool.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned before the running task finished")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after the running task finished")
	}
}

func TestAdvancedWorkerPoolCloseWithPriorityBackoffDoesNotPanicOrHang(t *testing.T) {
	pool := NewAdvancedWorkerPool(1, 2)

	started := make(chan struct{})
	release := make(chan struct{})

	if err := pool.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Submit() blocking task error = %v", err)
	}
	<-started

	for i := 0; i < 2; i++ {
		if err := pool.Submit(func() {}); err != nil {
			t.Fatalf("Submit() queued task %d error = %v", i, err)
		}
	}

	if err := pool.SubmitWithPriority(func() {}, 2); err != nil {
		t.Fatalf("SubmitWithPriority() error = %v", err)
	}

	deadline := time.After(100 * time.Millisecond)
	for atomic.LoadInt64(&pool.metrics.PriorityLength) > 0 {
		select {
		case <-deadline:
			t.Fatal("priority task was not dispatched")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	closed := make(chan struct{})
	go func() {
		pool.Close()
		close(closed)
	}()

	close(release)

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return with priority backoff work pending")
	}
}

func TestAdvancedWorkerPoolCloseWithFullQueueDoesNotPanic(t *testing.T) {
	for range 50 {
		pool := NewAdvancedWorkerPool(1, 2)

		started := make(chan struct{})
		release := make(chan struct{})

		if err := pool.Submit(func() {
			close(started)
			<-release
		}); err != nil {
			t.Fatalf("Submit() blocking task error = %v", err)
		}
		<-started

		// Fill the task queue so the next high-priority task spawns a backoff goroutine.
		for i := 0; i < 2; i++ {
			if err := pool.Submit(func() {}); err != nil {
				t.Fatalf("Submit() queued task %d error = %v", i, err)
			}
		}

		// Submit a high-priority task; when the queue is full the dispatcher spawns
		// a backoff goroutine that will try to send on taskQueue after a short sleep.
		if err := pool.SubmitWithPriority(func() {}, 2); err != nil {
			t.Fatalf("SubmitWithPriority() error = %v", err)
		}

		// Close immediately while the backoff goroutine may still be sleeping.
		// Before the fix this could panic with "send on closed channel".
		closed := make(chan struct{})
		go func() {
			pool.Close()
			close(closed)
		}()

		close(release)

		select {
		case <-closed:
		case <-time.After(time.Second):
			t.Fatal("Close did not return")
		}
	}
}

// TestAdvancedWorkerPoolCloseDrainsPriorityQueue verifies that high-priority
// tasks sitting in the priority queue (not yet forwarded to taskQueue) are
// executed during Close rather than silently dropped. Before the drain fix,
// the priorityDispatcher exited on the stop signal and abandoned any buffered
// priority tasks.
func TestAdvancedWorkerPoolCloseDrainsPriorityQueue(t *testing.T) {
	const prioTasks = 5
	// queueSize 20 -> priority queue size 10, enough to buffer all prio tasks.
	pool := NewAdvancedWorkerPool(1, 20)

	// Occupy the single worker with a task that blocks until we release it, so
	// the priority queue is not drained by the dispatcher before Close.
	started := make(chan struct{})
	release := make(chan struct{})
	if err := pool.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Submit blocking task: %v", err)
	}
	<-started

	// Also fill the task queue so the dispatcher cannot immediately forward
	// priority tasks into it (it would spawn backoff goroutines instead).
	for i := 0; i < 4; i++ {
		if err := pool.Submit(func() {}); err != nil {
			t.Fatalf("Submit filler %d: %v", i, err)
		}
	}

	// Submit high-priority tasks. These land in the priority queue and, with
	// the task queue full, are not yet forwarded.
	var ran atomic.Int32
	for i := 0; i < prioTasks; i++ {
		if err := pool.SubmitWithPriority(func() {
			ran.Add(1)
		}, 2); err != nil {
			t.Fatalf("SubmitWithPriority %d: %v", i, err)
		}
	}

	// Close while priority tasks are still pending. The drain must run them.
	closed := make(chan struct{})
	go func() {
		pool.Close()
		close(closed)
	}()

	// Release the blocking worker so Close can make progress.
	close(release)

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return while draining priority queue")
	}

	// Every submitted priority task must have executed; none should be dropped.
	if got := ran.Load(); got != int32(prioTasks) {
		t.Fatalf("priority tasks executed during Close = %d, want %d (items were dropped)", got, prioTasks)
	}
}

// TestGetMetrics_AtomicRead exercises GetMetrics while concurrent submissions
// mutate the metrics. Run with -race to detect non-atomic reads.
func TestGetMetrics_AtomicRead(t *testing.T) {
	pool := NewAdvancedWorkerPool(4, 16)
	defer pool.Close()

	var submitWg sync.WaitGroup
	var readerWg sync.WaitGroup
	done := make(chan struct{})

	// Goroutines that continuously submit tiny tasks. Submit first, THEN check
	// done — otherwise a fast reader can close(done) before any submitter is
	// scheduled, leaving TasksSubmitted == 0 (this flaked ~77% under plain
	// `go test`; -race scheduling happened to hide it). Submit-first guarantees
	// each of the 4 goroutines lands at least one submission.
	for i := 0; i < 4; i++ {
		submitWg.Add(1)
		go func() {
			defer submitWg.Done()
			for {
				_ = pool.Submit(func() {})
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}

	// Goroutine that continuously reads metrics.
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for i := 0; i < 1000; i++ {
			_ = pool.GetMetrics()
		}
	}()

	readerWg.Wait()
	close(done)
	submitWg.Wait()

	// Sanity check: at least some tasks were submitted.
	m := pool.GetMetrics()
	if m.TasksSubmitted == 0 {
		t.Error("expected non-zero TasksSubmitted")
	}
}

func TestGetQueueStats_NoDivideByZero(t *testing.T) {
	pool := NewAdvancedWorkerPool(2, 8)
	defer pool.Close()

	stats := pool.GetQueueStats()
	successRate, ok := stats["success_rate"].(float64)
	if !ok {
		t.Fatalf("success_rate not float64, got %T", stats["success_rate"])
	}
	if successRate != 0.0 {
		t.Errorf("expected success_rate 0.0 for fresh pool, got %v", successRate)
	}
}

// TestAdvancedWorkerPoolResizeShrinkActuallyReducesWorkers is a regression test
// for DP-811: Resize to a smaller worker count must actually retire workers,
// not silently lie about success. Before the fix, Resize updated p.workers but
// left all goroutines running, so a "shrunk" pool still ran N concurrent tasks.
func TestAdvancedWorkerPoolResizeShrinkActuallyReducesWorkers(t *testing.T) {
	const initial = 8
	pool := NewAdvancedWorkerPool(initial, 64)
	defer pool.Close()

	// Block all initial workers so the concurrency of the next batch is
	// determined entirely by how many workers exist after the shrink.
	block := make(chan struct{})
	var inFlight atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < initial; i++ {
		wg.Add(1)
		if err := pool.Submit(func() {
			defer wg.Done()
			inFlight.Add(1)
			<-block
		}); err != nil {
			t.Fatalf("Submit() error = %v", err)
		}
	}

	// Wait until every initial worker is parked on the block.
	if err := waitForCount(&inFlight, initial); err != nil {
		t.Fatalf("waiting for initial workers to park: %v", err)
	}

	// Shrink to 2. After the block is released, only 2 of the next batch of
	// tasks should be able to run concurrently.
	const resized = 2
	if err := pool.Resize(resized); err != nil {
		t.Fatalf("Resize(%d) error = %v", resized, err)
	}

	close(block)
	wg.Wait()

	// Now measure live concurrency of a fresh batch with the shrunk pool.
	var live atomic.Int32
	var done sync.WaitGroup
	probe := make(chan struct{})
	for i := 0; i < resized; i++ {
		done.Add(1)
		if err := pool.Submit(func() {
			defer done.Done()
			live.Add(1)
			<-probe
		}); err != nil {
			t.Fatalf("Submit() probe error = %v", err)
		}
	}
	if err := waitForCount(&live, resized); err != nil {
		t.Fatalf("waiting for probe tasks to start: %v", err)
	}

	// A third task should not start until one of the probes returns, proving
	// the pool honors the smaller worker count.
	thirdStarted := make(chan error, 1)
	done.Add(1)
	if err := pool.Submit(func() {
		defer done.Done()
		select {
		case thirdStarted <- nil:
		default:
		}
		<-probe
	}); err != nil {
		t.Fatalf("Submit() third error = %v", err)
	}
	select {
	case <-thirdStarted:
		t.Fatal("third task started before a probe finished — Resize did not shrink the pool")
	case <-time.After(25 * time.Millisecond):
	}

	close(probe)
	done.Wait()
}

// waitForCount blocks until the atomic counter reaches want or times out.
func waitForCount(c *atomic.Int32, want int32) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c.Load() >= want {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return errors.New("timed out waiting for counter")
}
