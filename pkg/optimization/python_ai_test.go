package optimization

import (
	"context"
	"errors"
	"sync"
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
