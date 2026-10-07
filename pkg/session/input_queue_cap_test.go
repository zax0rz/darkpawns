package session

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestInputQueueCapWaitFlood(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	registerForDrain(t, m, s)
	closed := 0
	s.SetCloseFunc(func() {
		if !s.inputMu.TryLock() {
			t.Fatal("transport close holds inputMu")
		}
		s.inputMu.Unlock()
		closed++
	})
	for pulse := 0; pulse < 300; pulse++ {
		s.player.SetWaitState(3)
		if !s.tryExecuteNow("whoami", nil) {
			t.Fatal("flood command bypassed wait queue")
		}
		m.DrainInputQueues()
		if n := s.queueLen(); n > 128 {
			t.Fatalf("unbounded queue: %d", n)
		}
	}
	if closed != 1 {
		t.Fatalf("overflow close count = %d, want 1", closed)
	}
	if got := drainSendChannel(t, s); got != "" {
		t.Fatalf("overflow emitted or executed output: %q", got)
	}
	if s.queueLen() != 0 {
		t.Fatal("overflow retained queued input")
	}
	s.player.SetWaitState(0)
	if !s.tryExecuteNow("whoami", nil) || s.queueLen() != 0 {
		t.Fatal("overflow reopened zero-wait fast path")
	}
}

func TestInputQueueCapAliases(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	closed := 0
	s.SetCloseFunc(func() {
		if !s.inputMu.TryLock() {
			t.Fatal("alias overflow close holds inputMu")
		}
		s.inputMu.Unlock()
		closed++
	})
	s.player.SetWaitState(3)
	for i := 0; i < 127; i++ {
		s.tryExecuteNow("whoami", nil)
	}
	s.prependAliasedInputs([]string{"title First"})
	if n := s.queueLen(); n != 128 {
		t.Fatalf("at-bound queue = %d, want 128", n)
	}
	if closed != 0 {
		t.Fatal("closed at the accepted bound")
	}
	first, ok := s.dequeueInput()
	if !ok || first.cmd != "title" || !first.aliased {
		t.Fatalf("alias priority/marker lost: %+v", first)
	}
	s.prependAliasedInputs([]string{"whoami", "whoami"})
	if closed != 1 {
		t.Fatalf("alias overflow close count = %d, want 1", closed)
	}
	if s.queueLen() != 0 {
		t.Fatal("alias overflow retained queue")
	}
}

func TestInputQueueCapConcurrentAdmission(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	s.player.SetWaitState(3)
	var closed atomic.Int32
	s.SetCloseFunc(func() { closed.Add(1) })
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 40; i++ {
				s.tryExecuteNow("whoami", nil)
				if n := s.queueLen(); n > 128 {
					t.Errorf("concurrent queue exceeded cap: %d", n)
				}
			}
		}()
	}
	workers.Wait()
	if closed.Load() != 1 || s.queueLen() != 0 {
		t.Fatalf("overflow state: closes=%d queue=%d", closed.Load(), s.queueLen())
	}
}
