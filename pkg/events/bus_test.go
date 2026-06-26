package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

type testBusEvent struct{ name string }

func (e testBusEvent) Type() string { return "test.event" }

// TestPublishSnapshotSkipsUnsubscribedHandlers verifies that a handler which
// unsubscribes itself during Publish does not affect the snapshot of handlers
// being iterated.
func TestPublishSnapshotSkipsUnsubscribedHandlers(t *testing.T) {
	bus := NewInProcessBus()
	ctx := context.Background()

	var called1, called2, called3 int64

	var unsub1 func()
	_, unsub1 = bus.Subscribe("test.event", func(_ context.Context, event BusEvent) error {
		atomic.AddInt64(&called1, 1)
		unsub1()
		return nil
	})
	bus.Subscribe("test.event", func(_ context.Context, event BusEvent) error {
		atomic.AddInt64(&called2, 1)
		return nil
	})
	bus.Subscribe("test.event", func(_ context.Context, event BusEvent) error {
		atomic.AddInt64(&called3, 1)
		return nil
	})

	if err := bus.Publish(ctx, testBusEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if got := atomic.LoadInt64(&called1); got != 1 {
		t.Errorf("expected handler 1 to be called once, got %d", got)
	}
	if got := atomic.LoadInt64(&called2); got != 1 {
		t.Errorf("expected handler 2 to be called once, got %d", got)
	}
	if got := atomic.LoadInt64(&called3); got != 1 {
		t.Errorf("expected handler 3 to be called once, got %d", got)
	}
}

// TestConcurrentPublishUnsubscribe exercises the race-prone path of publishing
// while another goroutine unsubscribes.
func TestConcurrentPublishUnsubscribe(t *testing.T) {
	bus := NewInProcessBus()
	ctx := context.Background()

	var count int64
	handler := func(_ context.Context, event BusEvent) error {
		atomic.AddInt64(&count, 1)
		return nil
	}

	const numSubs = 50
	unsubs := make([]func(), numSubs)
	for i := 0; i < numSubs; i++ {
		_, unsubs[i] = bus.Subscribe("test.event", handler)
	}

	var wg sync.WaitGroup
	for i := 0; i < numSubs; i++ {
		wg.Add(1)
		go func(unsub func()) {
			defer wg.Done()
			unsub()
		}(unsubs[i])
	}

	for i := 0; i < 100; i++ {
		_ = bus.Publish(ctx, testBusEvent{})
	}

	wg.Wait()
}
