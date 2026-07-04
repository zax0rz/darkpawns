package optimization

import (
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
