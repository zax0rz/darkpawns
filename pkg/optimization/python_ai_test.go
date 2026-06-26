package optimization

import (
	"errors"
	"sync"
	"testing"
	"time"
)

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
			_, err := processor.Submit(AIRequest{ID: string(rune('a' + id))})
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
