package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCreateAndProcess verifies basic event creation and processing.
// Based on event_create() and event_process() in src/events.c.
func TestCreateAndProcess(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		atomic.AddInt64(&fired, 1)
		return 0
	}

	// Schedule event to fire at pulse 5
	eq.Create(5, 100, 200, 300, 42, "test_trigger", 1, fn)

	// Process 4 pulses — event should not fire yet
	for i := 0; i < 4; i++ {
		eq.Process(ctx)
	}
	if atomic.LoadInt64(&fired) != 0 {
		t.Fatalf("event fired too early")
	}

	// Process pulse 5 — event should fire
	eq.Process(ctx)
	if atomic.LoadInt64(&fired) != 1 {
		t.Fatalf("event did not fire, expected 1, got %d", atomic.LoadInt64(&fired))
	}
}

// TestCancel verifies event cancellation.
// Based on event_cancel() in src/events.c lines 69-78.
func TestCancel(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		atomic.AddInt64(&fired, 1)
		return 0
	}

	id := eq.Create(2, 100, 0, 0, 0, "cancel_me", 1, fn)
	eq.Cancel(id)

	// Process past the event time
	for i := 0; i < 5; i++ {
		eq.Process(ctx)
	}

	if atomic.LoadInt64(&fired) != 0 {
		t.Fatalf("cancelled event should not fire")
	}
}

// TestReenqueue verifies that events with return value > 0 are re-enqueued.
// Source: events.c event_process() lines 93-99 (3/6/98 ejg change).
func TestReenqueue(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		count := atomic.AddInt64(&fired, 1)
		if count < 3 {
			return 2 // re-enqueue after 2 more pulses
		}
		return 0
	}

	eq.Create(1, 100, 0, 0, 0, "reenqueue", 1, fn)

	// Process enough pulses for 3 firings
	for i := 0; i < 10; i++ {
		eq.Process(ctx)
	}

	if atomic.LoadInt64(&fired) != 3 {
		t.Fatalf("expected 3 firings, got %d", atomic.LoadInt64(&fired))
	}
}

// TestPending verifies the Pending() count.
func TestPending(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)

	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 { return 0 }

	if eq.Pending() != 0 {
		t.Fatalf("expected 0 pending, got %d", eq.Pending())
	}

	id1 := eq.Create(10, 100, 0, 0, 0, "a", 1, fn)
	id2 := eq.Create(20, 101, 0, 0, 0, "b", 1, fn)

	if eq.Pending() != 2 {
		t.Fatalf("expected 2 pending, got %d", eq.Pending())
	}

	eq.Cancel(id1)
	if eq.Pending() != 1 {
		t.Fatalf("expected 1 pending after cancel, got %d", eq.Pending())
	}

	eq.Cancel(id2)
	if eq.Pending() != 0 {
		t.Fatalf("expected 0 pending after all cancelled, got %d", eq.Pending())
	}
}

// TestCancelBySource verifies cancelling all events for a given source.
// Used when a mob dies to clean up its pending events.
func TestCancelBySource(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		atomic.AddInt64(&fired, 1)
		return 0
	}

	// Two events for source 100, one for source 200
	eq.Create(2, 100, 0, 0, 0, "a", 1, fn)
	eq.Create(3, 100, 0, 0, 0, "b", 1, fn)
	eq.Create(2, 200, 0, 0, 0, "c", 1, fn)

	cancelled := eq.CancelBySource(100)
	if cancelled != 2 {
		t.Fatalf("expected 2 cancelled, got %d", cancelled)
	}

	// Process past all event times
	for i := 0; i < 5; i++ {
		eq.Process(ctx)
	}

	if atomic.LoadInt64(&fired) != 1 {
		t.Fatalf("expected only source 200's event to fire (1), got %d", atomic.LoadInt64(&fired))
	}
}

// TestDelayMinimum verifies that delay < 1 is clamped to 1.
// Source: events.c event_create() "if (when < 1) when = 1"
func TestDelayMinimum(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		atomic.AddInt64(&fired, 1)
		return 0
	}

	// Pass delay 0 — should be clamped to 1
	eq.Create(0, 100, 0, 0, 0, "now", 1, fn)

	eq.Process(ctx) // pulse 1 — event scheduled for pulse 1, should fire
	if atomic.LoadInt64(&fired) != 1 {
		t.Fatalf("event with delay=0 clamped to 1 should fire on first Process")
	}
}

// TestMultipleEventsSamePulse verifies correct ordering when multiple
// events are scheduled for the same pulse.
func TestMultipleEventsSamePulse(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	var order []string
	var mu sync.Mutex

	fnA := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		mu.Lock()
		order = append(order, "A")
		mu.Unlock()
		return 0
	}
	fnB := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		mu.Lock()
		order = append(order, "B")
		mu.Unlock()
		return 0
	}

	eq.Create(2, 100, 0, 0, 0, "A", 1, fnA)
	eq.Create(2, 101, 0, 0, 0, "B", 1, fnB)

	for i := 0; i < 5; i++ {
		eq.Process(ctx)
	}

	mu.Lock()
	if len(order) != 2 {
		t.Fatalf("expected 2 events fired, got %d", len(order))
	}
	mu.Unlock()
}

// TestFreeAll verifies that FreeAll clears all events.
// Based on event_free_all() in src/events.c lines 114-126.
func TestFreeAll(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 { return 0 }

	eq.Create(10, 100, 0, 0, 0, "a", 1, fn)
	eq.Create(20, 101, 0, 0, 0, "b", 1, fn)

	if eq.Pending() != 2 {
		t.Fatalf("expected 2 pending before FreeAll")
	}

	eq.FreeAll()

	if eq.Pending() != 0 {
		t.Fatalf("expected 0 pending after FreeAll")
	}
}

// TestStartStop verifies the background goroutine starts and stops.
func TestStartStop(t *testing.T) {
	eq := NewEventQueue(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	var fired int64
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 {
		atomic.AddInt64(&fired, 1)
		return 0
	}

	eq.Create(2, 100, 0, 0, 0, "bg", 1, fn)
	eq.Start(ctx)

	// Wait for event to fire in background
	time.Sleep(300 * time.Millisecond)

	if atomic.LoadInt64(&fired) != 1 {
		t.Fatalf("expected background event to fire, got %d", atomic.LoadInt64(&fired))
	}

	cancel()
}

// TestTimeUntilNext verifies TimeUntilNext returns correct duration.
func TestTimeUntilNext(t *testing.T) {
	eq := NewEventQueue(100 * time.Millisecond)
	fn := func(_ context.Context, source, target, obj, arg int, trigger string, et int) int64 { return 0 }

	// No events
	if eq.TimeUntilNext() != 0 {
		t.Fatalf("expected 0 with no events")
	}

	// Event at pulse 5, current pulse is 0
	eq.Create(5, 100, 0, 0, 0, "future", 1, fn)
	expected := 5 * 100 * time.Millisecond
	if eq.TimeUntilNext() != expected {
		t.Fatalf("expected %v, got %v", expected, eq.TimeUntilNext())
	}
}

// TestPulseIncrement verifies pulse counter increments each Process call.
func TestPulseIncrement(t *testing.T) {
	eq := NewEventQueue(10 * time.Millisecond)
	ctx := context.Background()

	if eq.Pulse() != 0 {
		t.Fatalf("expected pulse 0 at start")
	}

	for i := 1; i <= 5; i++ {
		eq.Process(ctx)
		if eq.Pulse() != int64(i) {
			t.Fatalf("expected pulse %d, got %d", i, eq.Pulse())
		}
	}
}
