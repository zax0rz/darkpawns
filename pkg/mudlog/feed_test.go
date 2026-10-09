package mudlog

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newStartedFeed(t *testing.T, queue, ring int) *Feed {
	t.Helper()
	f := NewFeed(queue, ring)
	f.Start()
	t.Cleanup(f.Stop)
	return f
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// The tap must return at once with a full queue and count the drop: MudLog
// runs under audited locks and can never wait on the observer (design §3.2).
func TestTapNeverBlocksOnFullQueue(t *testing.T) {
	f := newStartedFeed(t, 2, 100)
	f.tap("", "kept", 1, 31, true)
	waitFor(t, "first event dispatched", func() bool { return f.Stats().Served == 1 })
	for i := 0; i < 1000; i++ {
		done := make(chan struct{})
		go func() {
			f.tap("", "overflow", 1, 31, true)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("tap blocked on a full queue")
		}
	}
	if got := f.Stats().Dropped; got == 0 {
		t.Fatal("overflow was not counted as dropped")
	}
	waitFor(t, "every tap accounted", func() bool {
		st := f.Stats()
		return st.Queued == 0 && st.Served+st.Dropped == 1001
	})
	st := f.Stats()
	if int64(st.LastSeq) != st.Served || st.Served+st.Dropped != 1001 {
		t.Fatalf("served %d + dropped %d != 1001 taps (seq %d)", st.Served, st.Dropped, st.LastSeq)
	}
}

// The ring replays in order from a cursor and caps at its capacity.
func TestSinceReplaysInOrderAndCaps(t *testing.T) {
	f := newStartedFeed(t, 64, 8)
	for i := 0; i < 20; i++ {
		f.tap("", "line", 1, 31, false)
	}
	waitFor(t, "all dispatched", func() bool { return f.LastSeq() == 20 })
	all := f.Since(0, 0)
	if len(all) != 8 {
		t.Fatalf("ring held %d events, want cap 8", len(all))
	}
	if all[0].Seq != 13 || all[7].Seq != 20 {
		t.Fatalf("replay window = %d..%d, want 13..20", all[0].Seq, all[7].Seq)
	}
	tail := f.Since(15, 3)
	if len(tail) != 3 || tail[0].Seq != 16 || tail[2].Seq != 18 {
		t.Fatalf("Since(15,3) = %+v", tail)
	}
}

// A subscriber that stops reading is kicked — never buffered without bound
// and never delaying the dispatcher — while a reading subscriber keeps
// receiving everything.
func TestSlowSubscriberIsKickedNotDispatcher(t *testing.T) {
	f := newStartedFeed(t, 1024, 1024)
	fast, fastKick, fastCancel := f.Subscribe(0)
	defer fastCancel()
	_, slowKick, slowCancel := f.Subscribe(0)
	defer slowCancel()

	var got atomic.Int32
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			select {
			case _, ok := <-fast:
				if !ok {
					return
				}
				got.Add(1)
			case <-fastKick:
				return
			}
		}
	}()

	for i := 0; i < 400; i++ { // slow's buffer is 256
		f.tap("", "burst", 1, 31, false)
	}
	waitFor(t, "all 400 dispatched", func() bool { return f.LastSeq() == 400 })
	waitFor(t, "slow subscriber kicked", func() bool {
		select {
		case <-slowKick:
			return true
		default:
			return false
		}
	})
	waitFor(t, "fast subscriber got everything", func() bool { return got.Load() == 400 })
	select {
	case <-fastKick:
		t.Fatal("reading subscriber was kicked")
	default:
	}
}

// Tap is safe under concurrent hammering (run with -race in CI).
func TestTapRace(t *testing.T) {
	f := newStartedFeed(t, 256, 1024)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				f.tap("k", "x", 1, 31, false)
			}
		}()
	}
	wg.Wait()
	waitFor(t, "drain", func() bool { return f.Stats().Queued == 0 })
	if got := f.LastSeq(); int64(got) != 4000-f.Stats().Dropped {
		t.Fatalf("served %d, dropped %d, want sum 4000", got, f.Stats().Dropped)
	}
}

// The catalog summarizes live keys, with unkeyed events grouped.
func TestCatalogGroupsUnkeyed(t *testing.T) {
	f := newStartedFeed(t, 64, 64)
	f.tap("", "a", 1, 31, false)
	f.tap("interpreter.c:2154", "b", 2, 31, false)
	waitFor(t, "dispatched", func() bool { return f.LastSeq() == 2 })
	cat := f.Catalog()
	if len(cat) != 2 {
		t.Fatalf("catalog = %+v", cat)
	}
	if cat[0].Key != "unkeyed" || cat[0].Count != 1 || cat[1].Key != "interpreter.c:2154" {
		t.Fatalf("catalog = %+v", cat)
	}
}

// Stop may race taps from any goroutine (MudLog runs on the game loop and
// on sessions; shutdown is exactly when logging peaks). The queue is never
// closed, so a tap can at worst land in a full queue and drop — never send
// on a closed channel and panic in the caller (PR #1893 review, bug 2).
func TestStopRacesTapWithoutPanic(t *testing.T) {
	var wg sync.WaitGroup
	for round := 0; round < 40; round++ {
		f := NewFeed(4, 16)
		f.Start()
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Tap straight through Stop: the tappers must still be
				// mid-storm when the close lands, so the check-then-send
				// window is actually exercised.
				for i := 0; i < 2000; i++ {
					f.Tap("", "race", 1, 31, false)
				}
			}()
		}
		time.Sleep(time.Duration(rand.Intn(2000)) * time.Microsecond)
		f.Stop()
		wg.Wait()
	}
}
