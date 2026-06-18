package session

import (
	"sync"
	"testing"
)

// TestSendMessage_AfterClose verifies that sending to a session whose send
// channel has been closed does not panic. Regression guard for the use-after-
// close race in cmdKick: the admin command copies a session reference, releases
// the manager lock, then calls Send — the session can be closed in between, and
// a send on a closed channel panics even inside a select.
func TestSendMessage_AfterClose(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "victim", 1001, true)

	s.CloseSend()

	// Must not panic, and should report no error (message silently dropped).
	if err := s.SendMessage("are you there?"); err != nil {
		t.Errorf("SendMessage after close returned error: %v", err)
	}
	// Send is the public wrapper used by admin commands like kick.
	s.Send("kicked")
}

// TestSendMessage_ConcurrentClose hammers Send and CloseSend concurrently. Run
// under -race; the test fails (panics) if the close/send guard regresses.
func TestSendMessage_ConcurrentClose(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "victim", 1001, true)

	var wg sync.WaitGroup
	// Drain the channel so sends don't all fall through to the default branch.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range s.SendChannel() {
		}
	}()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Send("spam")
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.CloseSend()
	}()

	wg.Wait()
}
