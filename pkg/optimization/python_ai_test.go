package optimization

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAIBatchProcessor_SubmitHonorsContextTimeout verifies that Submit respects
// the caller's context deadline instead of a hardcoded wait (DP-629).
func TestAIBatchProcessor_SubmitHonorsContextTimeout(t *testing.T) {
	processor := NewAIBatchProcessor(10, 5*time.Second, func(items []AIBatchItem) error {
		// Never respond — the context should cancel the wait.
		return nil
	})
	defer processor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := processor.Submit(ctx, AIRequest{ID: "timeout-test"})
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

// TestAIBatchProcessor_WaitThenCloseAnswersEveryRequest submits requests
// against a deliberately slow handler and verifies that Wait() does not
// return until every request has a response, and that the subsequent
// Close() neither drops a response nor panics (DP-808).
func TestAIBatchProcessor_WaitThenCloseAnswersEveryRequest(t *testing.T) {
	const numRequests = 30

	processor := NewAIBatchProcessor(4, 20*time.Millisecond, func(items []AIBatchItem) error {
		// Simulate a slow AI call so in-flight Submits are still pending
		// when Wait/Close are invoked.
		time.Sleep(15 * time.Millisecond)
		for _, item := range items {
			item.Response <- AIResponse{ID: item.Request.ID, Text: "ok"}
		}
		return nil
	})

	var reqWg sync.WaitGroup
	var answered int64
	for i := 0; i < numRequests; i++ {
		reqWg.Add(1)
		go func(id int) {
			defer reqWg.Done()
			req := AIRequest{ID: string(rune('a' + id%26))}
			resp, err := processor.Submit(context.Background(), req)
			if err != nil {
				t.Errorf("Submit() error = %v", err)
				return
			}
			if resp.Text != "ok" {
				t.Errorf("unexpected response %+v", resp)
				return
			}
			atomic.AddInt64(&answered, 1)
		}(i)
	}

	reqWg.Wait()
	processor.Wait()

	if got := atomic.LoadInt64(&answered); got != numRequests {
		t.Fatalf("answered = %d, want %d", got, numRequests)
	}

	done := make(chan struct{})
	go func() {
		if err := processor.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return")
	}

	// Submitting after Close must fail cleanly rather than panic.
	if _, err := processor.Submit(context.Background(), AIRequest{ID: "after-close"}); err != ErrPoolClosed {
		t.Fatalf("Submit() after Close error = %v, want ErrPoolClosed", err)
	}
}

// TestAIBatchProcessor_CloseFlushesPendingBatchBeforeReturning verifies
// that items still sitting in an under-sized pending batch (never hit
// batchSize, timer hasn't fired yet) are answered by Close() rather than
// dropped (DP-808).
func TestAIBatchProcessor_CloseFlushesPendingBatchBeforeReturning(t *testing.T) {
	processor := NewAIBatchProcessor(10, time.Hour, func(items []AIBatchItem) error {
		for _, item := range items {
			item.Response <- AIResponse{ID: item.Request.ID, Text: "flushed"}
		}
		return nil
	})

	var reqWg sync.WaitGroup
	var answered int64
	for i := 0; i < 3; i++ {
		reqWg.Add(1)
		go func(id int) {
			defer reqWg.Done()
			resp, err := processor.Submit(context.Background(), AIRequest{ID: string(rune('a' + id))})
			if err != nil {
				t.Errorf("Submit() error = %v", err)
				return
			}
			if resp.Text != "flushed" {
				t.Errorf("unexpected response %+v", resp)
				return
			}
			atomic.AddInt64(&answered, 1)
		}(i)
	}

	// Give the goroutines a moment to land in the pending batch (batchSize
	// is 10 and the timer is an hour out, so nothing would flush on its
	// own before Close).
	time.Sleep(20 * time.Millisecond)

	if err := processor.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reqWg.Wait()

	if got := atomic.LoadInt64(&answered); got != 3 {
		t.Fatalf("answered = %d, want 3", got)
	}
}

func TestAIBatchProcessor_FullBatchErrorFansOut(t *testing.T) {
	wantErr := errors.New("batch processing failed")
	processor := NewAIBatchProcessor(2, 5*time.Second, func(items []AIBatchItem) error {
		return wantErr
	})
	defer processor.Close()

	var wg sync.WaitGroup
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := processor.Submit(context.Background(), AIRequest{ID: string(rune('a' + id))})
			results <- err
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("callers blocked waiting for batch error")
	}

	close(results)
	count := 0
	for err := range results {
		count++
		if !errors.Is(err, wantErr) {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if count != 2 {
		t.Errorf("expected 2 errors, got %d", count)
	}
}
