// Package mudlog is the observer tap on game.MudLog (mudlog-push design,
// 2026-10-09). It queues each MudLog call without blocking, replays them
// from an in-memory ring, and fans them out to live subscribers and (via
// providers) outbound notifications.
//
// It is a pure observer by construction: Tap performs one non-blocking
// channel send and nothing else on the caller's goroutine — no locks, no
// I/O, no allocation beyond the Event — because MudLog runs under audited
// lock states (#1815, #1825) and its delivery to immortals and the file
// must not change in order, timing or content (R1).
package mudlog

import (
	"sync"
	"sync/atomic"
	"time"
)

// Event is one MudLog call as the tap saw it.
type Event struct {
	Seq    uint64    `json:"seq"`
	Time   time.Time `json:"time"`
	Key    string    `json:"key,omitempty"` // C site id, slice 2; "" until then
	Typ    int       `json:"type"`          // C's OFF/BRF/NRM/CMP (0-3, utils.h:114-117)
	Level  int       `json:"level"`         // min immortal level; negative = file only
	ToFile bool      `json:"to_file"`
	Text   string    `json:"text"`
}

// Queue and ring defaults (design §7 and §12.5).
const (
	DefaultQueue   = 4096
	DefaultRingCap = 1000

	// DefaultImmortalLevel matches LVL_IMMORT (structs.h:620): the default
	// virtual-immortal level for subscriptions.
	DefaultImmortalLevel = 31

	// DefaultBriefType matches BRF (utils.h:114): the default syslog type
	// for subscriptions.
	DefaultBriefType = 1
)

// Tap offers one event to the process feed. It never blocks: when the queue
// is full the event is dropped and counted, and the game continues. Before
// Start the queue still accepts events (the dispatcher has not drained
// them), so boot-time mudlog lines are retained up to the queue depth.
func Tap(key, str string, typ, level int, toFile bool) {
	defaultFeed.Tap(key, str, typ, level, toFile)
}

// Tap offers one event to this feed; see the package-level Tap.
func (f *Feed) Tap(key, str string, typ, level int, toFile bool) {
	f.tap(key, str, typ, level, toFile)
}

// Feed is one tap → dispatcher → ring/subscribers pipeline.
type Feed struct {
	queue   chan Event
	ringCap int

	started atomic.Bool
	stopped atomic.Bool
	dropped atomic.Int64
	served  atomic.Int64
	lastID  atomic.Uint64

	mu   sync.RWMutex // guards ring, subs and seq (dispatcher + readers)
	ring []Event
	seq  uint64
	subs map[*subscriber]struct{}
}

// subscriber receives live events until it is kicked (too slow) or
// cancelled. kick closes done exactly once; events itself is never closed
// by the dispatcher, so a concurrent deliver can only ever fail the
// closed-check, never send on a closed channel.
type subscriber struct {
	events chan Event
	done   chan struct{}
	kick   sync.Once
	gone   atomic.Bool
}

func (s *subscriber) shutdown() {
	if s.gone.CompareAndSwap(false, true) {
		s.kick.Do(func() { close(s.done) })
	}
}

func (s *subscriber) deliver(e Event) {
	if s.gone.Load() {
		return
	}
	select {
	case s.events <- e:
	default:
		// A slow subscriber must never slow the dispatcher: kick it. The
		// stream client re-syncs from the ring via its last seen seq.
		s.shutdown()
	}
}

var defaultFeed = NewFeed(DefaultQueue, DefaultRingCap)

// NewFeed builds a Feed without starting it.
func NewFeed(queue, ringCap int) *Feed {
	if queue <= 0 {
		queue = DefaultQueue
	}
	if ringCap <= 0 {
		ringCap = DefaultRingCap
	}
	return &Feed{queue: make(chan Event, queue), ringCap: ringCap}
}

// Start launches the dispatcher goroutine. Calling Start on an already
// started feed is a no-op.
func (f *Feed) Start() {
	if f.started.CompareAndSwap(false, true) {
		go f.dispatch()
	}
}

// Stop ends dispatch and kicks every subscriber. Tap after Stop drops
// (counted).
func (f *Feed) Stop() {
	if f.stopped.CompareAndSwap(false, true) {
		close(f.queue)
	}
}

func (f *Feed) tap(key, str string, typ, level int, toFile bool) {
	if f.stopped.Load() {
		f.dropped.Add(1)
		return
	}
	select {
	case f.queue <- Event{Time: time.Now(), Key: key, Typ: typ, Level: level, ToFile: toFile, Text: str}:
	default:
		f.dropped.Add(1)
	}
}

func (f *Feed) dispatch() {
	for e := range f.queue {
		f.mu.Lock()
		f.seq++
		e.Seq = f.seq
		f.ring = append(f.ring, e)
		if len(f.ring) > f.ringCap {
			f.ring = f.ring[len(f.ring)-f.ringCap:]
		}
		f.lastID.Store(e.Seq)
		subs := make([]*subscriber, 0, len(f.subs))
		for s := range f.subs {
			subs = append(subs, s)
		}
		f.mu.Unlock()
		f.served.Add(1)
		for _, s := range subs {
			s.deliver(e)
		}
	}
	f.mu.Lock()
	for s := range f.subs {
		s.shutdown()
	}
	f.subs = nil
	f.mu.Unlock()
}

// Subscribe returns a live event channel from after the given seq (0 =
// from the next event; replay history with Since). The second return
// value is the kick/cancel channel: it closes when the subscriber is too
// slow or the feed stops. The final func unsubscribes.
func (f *Feed) Subscribe(afterSeq uint64) (<-chan Event, <-chan struct{}, func()) {
	s := &subscriber{events: make(chan Event, 256), done: make(chan struct{})}
	f.mu.Lock()
	if f.subs == nil {
		f.subs = map[*subscriber]struct{}{}
	}
	f.subs[s] = struct{}{}
	f.mu.Unlock()
	cancel := func() {
		f.mu.Lock()
		delete(f.subs, s)
		f.mu.Unlock()
		s.shutdown()
	}
	return s.events, s.done, cancel
}

// Since returns up to limit events with Seq > afterSeq in order. When more
// events than the ring holds have passed, everything still held is
// returned; the client sees the gap in seq numbers and re-syncs.
func (f *Feed) Since(afterSeq uint64, limit int) []Event {
	if limit <= 0 || limit > f.ringCap {
		limit = f.ringCap
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]Event, 0, len(f.ring))
	for _, e := range f.ring {
		if e.Seq > afterSeq {
			out = append(out, e)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}

// LastSeq is the highest assigned sequence number.
func (f *Feed) LastSeq() uint64 { return f.lastID.Load() }

// Stats are the observer's own counters (design §7: drops are visible,
// not silent).
type Stats struct {
	Queued  int64  `json:"queued"`
	Served  int64  `json:"served"`
	Dropped int64  `json:"dropped"`
	LastSeq uint64 `json:"last_seq"`
	Started bool   `json:"started"`
	Stopped bool   `json:"stopped"`
}

// Stats snapshots the counters.
func (f *Feed) Stats() Stats {
	return Stats{
		Queued:  int64(len(f.queue)),
		Served:  f.served.Load(),
		Dropped: f.dropped.Load(),
		LastSeq: f.lastID.Load(),
		Started: f.started.Load(),
		Stopped: f.stopped.Load(),
	}
}

// KeyStat summarizes one event key currently held in the ring. Until slice
// 2 stamps call sites, most keys are "" (shown as "unkeyed").
type KeyStat struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
	Level int    `json:"last_level"`
	Type  int    `json:"last_type"`
}

// Catalog summarizes live keys. The static C-site catalog lands with the
// slice-2 key sweep.
func (f *Feed) Catalog() []KeyStat {
	f.mu.RLock()
	defer f.mu.RUnlock()
	order := make([]string, 0, 8)
	byKey := map[string]*KeyStat{}
	for _, e := range f.ring {
		k := e.Key
		if k == "" {
			k = "unkeyed"
		}
		st, ok := byKey[k]
		if !ok {
			st = &KeyStat{Key: k}
			byKey[k] = st
			order = append(order, k)
		}
		st.Count++
		st.Level = e.Level
		st.Type = e.Typ
	}
	out := make([]KeyStat, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	return out
}
