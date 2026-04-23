// Package events implements a timer-based event queue for Dark Pawns MUD.
//
// Based on the original C event system from src/events.c and src/queue.c
// written by Eric Green (ejg3@cornell.edu). The Go implementation uses
// container/heap for the priority queue instead of the original bucketed
// linked-list approach.
//
// In the original C code:
//   - event_init() initializes the global event_q
//   - event_create(func, event_obj, when) creates an event firing at pulse+when
//   - event_cancel(event) removes and frees an event
//   - event_process() fires all events whose time <= current pulse
//   - Events are processed once per heartbeat() call in comm.c
//
// The event func returns a long: if > 0, the event is re-enqueued for
// new_time + pulse (3/6/98 ejg change).
package events

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// EventFunc is the callback signature for event handlers.
// Matches EVENTFUNC(name) long (name)(void *event_obj) from events.h.
// The return value: if > 0, the event is re-scheduled after that many
// game pulses (source: events.c event_process(), 3/6/98 ejg change).
type EventFunc func(ctx context.Context, source, target, obj, argument int, trigger string, eventType int) int64

// Event represents a single scheduled event.
// Based on struct event in src/events.c.
type Event struct {
	ID        uint64
	Source    int    // mob/room/player vnum or instance ID
	Target    int    // target vnum or instance ID
	Obj       int    // object vnum or instance ID
	Argument  int    // numeric argument
	Trigger   string // trigger name (function to call in Lua script)
	EventType int    // event type (LT_MOB, LT_OBJ, LT_ROOM from structs.h)
	When      int64  // absolute pulse number when event fires
	Func      EventFunc
	Cancelled bool
}

// eventHeap implements heap.Interface for priority queue ordering by When.
type eventHeap []*Event

func (h eventHeap) Len() int           { return len(h) }
func (h eventHeap) Less(i, j int) bool { return h[i].When < h[j].When }
func (h eventHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *eventHeap) Push(x interface{}) {
	*h = append(*h, x.(*Event))
}

func (h *eventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	event := old[n-1]
	*h = old[:n-1]
	return event
}

// EventQueue manages scheduled events.
// Based on the global event_q in src/events.c.
type EventQueue struct {
	mu        sync.Mutex
	events    eventHeap
	nextID    uint64
	stopCh    chan struct{}
	pulse     int64 // current game pulse, incremented by Process()
	pulseDur  time.Duration
	started   bool
}

// NewEventQueue creates a new event queue.
// Based on event_init() in src/events.c.
//
// pulseDuration controls how often the game "ticks". In the original C code,
// OPT_USEC = 100000 gives PASSES_PER_SEC = 10 (10 pulses per second).
// PULSE_VIOLENCE = 2 RL_SEC = 20 pulses (2 seconds).
// The delay parameter to create_event is in PULSE_VIOLENCE units (2 sec each).
func NewEventQueue(pulseDuration time.Duration) *EventQueue {
	if pulseDuration <= 0 {
		pulseDuration = 100 * time.Millisecond // 10 pulses/sec, matching original
	}
	return &EventQueue{
		stopCh:   make(chan struct{}),
		pulseDur: pulseDuration,
	}
}

// Create schedules a new event.
// Based on event_create() in src/events.c lines 53-66.
//
// delay is in game pulses. In Lua scripts, delay is typically passed as
// a small integer (e.g., 1 = one PULSE_VIOLENCE, 6 = 6 PULSE_VIOLENCE).
// The caller should convert from game-time units to pulses before calling.
func (eq *EventQueue) Create(delay int64, source, target, obj, argument int, trigger string, eventType int, fn EventFunc) uint64 {
	if delay < 1 {
		delay = 1 // events.c: "make sure its in the future"
	}

	id := atomic.AddUint64(&eq.nextID, 1)
	evt := &Event{
		ID:        id,
		Source:    source,
		Target:    target,
		Obj:       obj,
		Argument:  argument,
		Trigger:   trigger,
		EventType: eventType,
		When:      eq.pulse + delay,
		Func:      fn,
	}

	eq.mu.Lock()
	heap.Push(&eq.events, evt)
	eq.mu.Unlock()

	return id
}

// Cancel marks an event as cancelled.
// Based on event_cancel() in src/events.c lines 69-78.
// The event is not removed from the heap immediately; it is skipped when
// its turn comes up in Process(). This matches the original C behavior
// where event_cancel calls queue_deq to remove the q_element.
func (eq *EventQueue) Cancel(id uint64) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	for _, evt := range eq.events {
		if evt.ID == id {
			evt.Cancelled = true
			return
		}
	}
}

// CancelBySource cancels all events with the given source ID.
// Used when a mob dies to clean up its pending events.
func (eq *EventQueue) CancelBySource(source int) int {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	cancelled := 0
	for _, evt := range eq.events {
		if evt.Source == source && !evt.Cancelled {
			evt.Cancelled = true
			cancelled++
		}
	}
	return cancelled
}

// Process fires all events whose time has arrived (When <= current pulse).
// Based on event_process() in src/events.c lines 81-101.
//
// This should be called once per game tick (e.g., from the game loop or
// heartbeat). In the original C code, heartbeat() increments pulse then
// calls event_process():
//   heartbeat(++pulse) { event_process(); ... }
//
// Returns the number of events processed.
func (eq *EventQueue) Process(ctx context.Context) int {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	eq.pulse++

	processed := 0
	for eq.events.Len() > 0 {
		evt := eq.events[0]
		if evt.When > eq.pulse {
			break
		}

		// Remove from heap
		heap.Pop(&eq.events)

		if evt.Cancelled {
			continue
		}

		// Call event function. If return > 0, re-enqueue.
		// Source: events.c event_process() lines 93-99 (3/6/98 ejg change)
		if evt.Func != nil {
			newDelay := evt.Func(ctx, evt.Source, evt.Target, evt.Obj, evt.Argument, evt.Trigger, evt.EventType)
			if newDelay > 0 {
				evt.When = eq.pulse + newDelay
				heap.Push(&eq.events, evt)
			}
		}
		processed++
	}

	return processed
}

// Pending returns the count of non-cancelled pending events.
func (eq *EventQueue) Pending() int {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	count := 0
	for _, evt := range eq.events {
		if !evt.Cancelled {
			count++
		}
	}
	return count
}

// Pulse returns the current game pulse.
func (eq *EventQueue) Pulse() int64 {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return eq.pulse
}

// Start begins a background goroutine that calls Process() at each pulse.
// This is optional; callers can also call Process() manually from their
// own game loop (as the original C code does from heartbeat()).
func (eq *EventQueue) Start(ctx context.Context) {
	eq.mu.Lock()
	if eq.started {
		eq.mu.Unlock()
		return
	}
	eq.started = true
	eq.mu.Unlock()

	ticker := time.NewTicker(eq.pulseDur)
	go func() {
		for {
			select {
			case <-ticker.C:
				eq.Process(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-eq.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the background goroutine started by Start().
func (eq *EventQueue) Stop() {
	close(eq.stopCh)
}

// TimeUntilNext returns the duration until the next event fires.
// Returns 0 if no events are pending.
func (eq *EventQueue) TimeUntilNext() time.Duration {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if eq.events.Len() == 0 {
		return 0
	}

	evt := eq.events[0]
	pulsesRemaining := evt.When - eq.pulse
	if pulsesRemaining < 0 {
		pulsesRemaining = 0
	}
	return time.Duration(pulsesRemaining) * eq.pulseDur
}

// FreeAll cancels and clears all events.
// Based on event_free_all() in src/events.c lines 114-126.
func (eq *EventQueue) FreeAll() {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	for eq.events.Len() > 0 {
		heap.Pop(&eq.events)
	}
}
